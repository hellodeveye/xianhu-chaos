package chaos

import (
	"bytes"
	"encoding/json"
	"io"
	"math/rand/v2"
	"net/http"
	"strings"
	"sync"
	"time"

	"xianhu-chaos/internal/provider"
)

const ScenarioHeader = "X-Chaos-Scenario"

type Engine struct {
	registry    *provider.Registry
	logLimit    int
	mu          sync.Mutex
	global      map[string]string
	counts      map[string]int64
	recent      []RequestLog
	perProvider map[string]ProviderState
}

type ProviderState struct {
	Name           string   `json:"name"`
	Enabled        bool     `json:"enabled"`
	GlobalScenario string   `json:"globalScenario,omitempty"`
	Routes         []string `json:"routes"`
	Scenarios      []string `json:"scenarios"`
}

type RequestLog struct {
	Time     string `json:"time"`
	Provider string `json:"provider"`
	RouteID  string `json:"routeId"`
	Method   string `json:"method"`
	Path     string `json:"path"`
	Code     string `json:"code,omitempty"`
	Scenario string `json:"scenario"`
	Status   int    `json:"status"`
}

type Selection struct {
	ScenarioName string
	Scenario     provider.Scenario
	Code         string
	Count        int64
}

func New(reg *provider.Registry, logLimit int) *Engine {
	if logLimit <= 0 {
		logLimit = 100
	}
	return &Engine{
		registry:    reg,
		logLimit:    logLimit,
		global:      make(map[string]string),
		counts:      make(map[string]int64),
		perProvider: make(map[string]ProviderState),
	}
}

func (e *Engine) Registry() *provider.Registry {
	return e.registry
}

func (e *Engine) Select(r *http.Request, route provider.Route, body []byte) Selection {
	manifest := e.registry.Providers[route.Provider]
	code := extractCode(r, route.CodeParam, body)
	scenarioName := route.DefaultScenario

	selected := false
	if requested := strings.TrimSpace(r.Header.Get(ScenarioHeader)); requested != "" {
		if _, ok := manifest.Scenarios[requested]; ok {
			scenarioName = requested
			selected = true
		}
	}
	if !selected {
		if matched := matchRule(r, manifest.Rules, code); matched != "" {
			scenarioName = matched
			selected = true
		}
	}
	if !selected {
		if global := e.GlobalScenario(route.Provider); global != "" {
			scenarioName = global
		}
	}

	count := e.increment(route.Provider, route.ID, scenarioName)
	scenario := manifest.Scenarios[scenarioName]
	if scenario.ErrorRate > 0 && scenario.ErrorRate < 1 && rand.Float64() > scenario.ErrorRate {
		scenarioName = route.DefaultScenario
		scenario = manifest.Scenarios[scenarioName]
	}
	if scenario.FailOnNth > 0 && count != scenario.FailOnNth {
		scenarioName = route.DefaultScenario
		scenario = manifest.Scenarios[scenarioName]
	}

	return Selection{
		ScenarioName: scenarioName,
		Scenario:     scenario,
		Code:         code,
		Count:        count,
	}
}

func (e *Engine) ApplyDelay(s provider.Scenario) {
	delay := s.DelayMS
	if s.JitterMS > 0 {
		delay += rand.IntN(s.JitterMS + 1)
	}
	if delay > 0 {
		time.Sleep(time.Duration(delay) * time.Millisecond)
	}
}

func (e *Engine) Log(route provider.Route, sel Selection) {
	e.mu.Lock()
	defer e.mu.Unlock()
	entry := RequestLog{
		Time:     time.Now().Format(time.RFC3339),
		Provider: route.Provider,
		RouteID:  route.ID,
		Method:   route.Method,
		Path:     route.Path,
		Code:     sel.Code,
		Scenario: sel.ScenarioName,
		Status:   sel.Scenario.Status,
	}
	e.recent = append(e.recent, entry)
	if len(e.recent) > e.logLimit {
		e.recent = e.recent[len(e.recent)-e.logLimit:]
	}
}

func (e *Engine) ProviderStates() []ProviderState {
	e.mu.Lock()
	defer e.mu.Unlock()
	states := make([]ProviderState, 0, len(e.registry.Providers))
	for name, manifest := range e.registry.Providers {
		state := ProviderState{
			Name:           name,
			Enabled:        manifest.Enabled,
			GlobalScenario: e.global[name],
			Routes:         make([]string, 0, len(manifest.Routes)),
			Scenarios:      make([]string, 0, len(manifest.Scenarios)),
		}
		for _, route := range manifest.Routes {
			state.Routes = append(state.Routes, route.Method+" "+route.Path)
		}
		for scenario := range manifest.Scenarios {
			state.Scenarios = append(state.Scenarios, scenario)
		}
		states = append(states, state)
	}
	return states
}

func (e *Engine) RecentRequests() []RequestLog {
	e.mu.Lock()
	defer e.mu.Unlock()
	out := make([]RequestLog, len(e.recent))
	copy(out, e.recent)
	return out
}

func (e *Engine) GlobalScenario(providerName string) string {
	e.mu.Lock()
	defer e.mu.Unlock()
	return e.global[providerName]
}

func (e *Engine) SetGlobalScenario(providerName, scenarioName string) bool {
	e.mu.Lock()
	defer e.mu.Unlock()
	if scenarioName == "" {
		delete(e.global, providerName)
		return true
	}
	manifest, ok := e.registry.Providers[providerName]
	if !ok {
		return false
	}
	if _, ok := manifest.Scenarios[scenarioName]; !ok {
		return false
	}
	e.global[providerName] = scenarioName
	return true
}

func (e *Engine) Reset() {
	e.mu.Lock()
	defer e.mu.Unlock()
	e.global = make(map[string]string)
	e.counts = make(map[string]int64)
	e.recent = nil
}

func (e *Engine) increment(providerName, routeID, scenarioName string) int64 {
	e.mu.Lock()
	defer e.mu.Unlock()
	key := providerName + ":" + routeID + ":" + scenarioName
	e.counts[key]++
	return e.counts[key]
}

func ReadAndRestoreBody(r *http.Request) ([]byte, error) {
	if r.Body == nil {
		return nil, nil
	}
	body, err := io.ReadAll(r.Body)
	if err != nil {
		return nil, err
	}
	r.Body = io.NopCloser(bytes.NewReader(body))
	return body, nil
}

func extractCode(r *http.Request, param string, body []byte) string {
	if param == "" {
		return ""
	}
	if value := r.URL.Query().Get(param); value != "" {
		return value
	}
	if len(body) == 0 {
		return ""
	}
	var payload map[string]any
	if err := json.Unmarshal(body, &payload); err != nil {
		return ""
	}
	if value, ok := payload[param].(string); ok {
		return value
	}
	return ""
}

func matchRule(r *http.Request, rules []provider.Rule, code string) string {
	for _, rule := range rules {
		if rule.Match.Code != "" && rule.Match.Code != code {
			continue
		}
		if len(rule.Match.Header) > 0 {
			matched := true
			for key, want := range rule.Match.Header {
				if r.Header.Get(key) != want {
					matched = false
					break
				}
			}
			if !matched {
				continue
			}
		}
		if rule.Match.Code == "" && len(rule.Match.Header) == 0 {
			continue
		}
		return rule.Scenario
	}
	return ""
}
