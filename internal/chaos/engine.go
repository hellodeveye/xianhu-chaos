package chaos

import (
	"bytes"
	"encoding/json"
	"io"
	"math/rand/v2"
	"net/http"
	"sort"
	"strings"
	"sync"
	"time"

	"xianhu-chaos/internal/provider"
)

const ScenarioHeader = "X-Chaos-Scenario"

type Override struct {
	Status      int    `json:"status"`
	ContentType string `json:"contentType"`
	Body        []byte `json:"-"`
	HasBody     bool   `json:"hasBody"`
}

type overrideKey struct {
	Provider string
	Scenario string
}

type routeKey struct {
	Provider string
	RouteID  string
}

type Engine struct {
	registry    *provider.Registry
	logLimit    int
	mu          sync.Mutex
	global      map[string]string
	route       map[routeKey]string
	counts      map[string]int64
	recent      []RequestLog
	perProvider map[string]ProviderState
	overrides   map[overrideKey]Override
}

type ProviderState struct {
	Name            string               `json:"name"`
	Enabled         bool                 `json:"enabled"`
	GlobalScenario  string               `json:"globalScenario,omitempty"`
	Routes          []string             `json:"routes"`
	Scenarios       []string             `json:"scenarios"`
	RouteGroups     []RouteScenarioGroup `json:"routeGroups"`
	SharedScenarios []string             `json:"sharedScenarios"`
}

// RouteScenarioGroup bundles a route with the scenarios that belong to it, so
// the UI can present scenarios grouped under their owning route instead of one
// flat list.
type RouteScenarioGroup struct {
	RouteID         string   `json:"routeId"`
	Method          string   `json:"method"`
	Path            string   `json:"path"`
	DefaultScenario string   `json:"defaultScenario"`
	ActiveScenario  string   `json:"activeScenario,omitempty"`
	Scenarios       []string `json:"scenarios"`
}

type RequestLog struct {
	Time         string `json:"time"`
	Provider     string `json:"provider"`
	RouteID      string `json:"routeId"`
	Method       string `json:"method"`
	Path         string `json:"path"`
	Query        string `json:"query,omitempty"`
	RequestBody  string `json:"requestBody,omitempty"`
	Code         string `json:"code,omitempty"`
	Scenario     string `json:"scenario"`
	Status       int    `json:"status"`
	ContentType  string `json:"contentType,omitempty"`
	ResponseBody string `json:"responseBody,omitempty"`
}

type Selection struct {
	ScenarioName string
	Scenario     provider.Scenario
	Code         string
	Count        int64
}

type ScenarioDetail struct {
	Provider    string `json:"provider"`
	Scenario    string `json:"scenario"`
	Status      int    `json:"status"`
	ContentType string `json:"contentType"`
	Body        string `json:"body"`
	HasOverride bool   `json:"hasOverride"`
	Overridden  *struct {
		Status      int    `json:"status"`
		ContentType string `json:"contentType"`
		Body        string `json:"body"`
	} `json:"overridden,omitempty"`
}

func New(reg *provider.Registry, logLimit int) *Engine {
	if logLimit <= 0 {
		logLimit = 100
	}
	return &Engine{
		registry:    reg,
		logLimit:    logLimit,
		global:      make(map[string]string),
		route:       make(map[routeKey]string),
		counts:      make(map[string]int64),
		perProvider: make(map[string]ProviderState),
		overrides:   make(map[overrideKey]Override),
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
		if scenario, ok := manifest.Scenarios[requested]; ok && scenarioAppliesToRoute(scenario, route) {
			scenarioName = requested
			selected = true
		}
	}
	if !selected {
		if matched := matchRule(r, manifest.Rules, code); matched != "" {
			if scenarioAppliesToRoute(manifest.Scenarios[matched], route) {
				scenarioName = matched
				selected = true
			}
		}
	}
	if !selected {
		if routeScenario := e.RouteScenario(route.Provider, route.ID); routeScenario != "" {
			scenarioName = routeScenario
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
	if ov, ok := e.GetOverride(route.Provider, scenarioName); ok {
		scenario.Status = ov.Status
		if ov.ContentType != "" {
			scenario.ContentType = ov.ContentType
		}
		if ov.HasBody {
			scenario.Body = ov.Body
		}
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

func (e *Engine) Log(r *http.Request, route provider.Route, sel Selection, requestBody []byte) {
	e.mu.Lock()
	defer e.mu.Unlock()
	entry := RequestLog{
		Time:         time.Now().Format(time.RFC3339),
		Provider:     route.Provider,
		RouteID:      route.ID,
		Method:       route.Method,
		Path:         route.Path,
		Query:        r.URL.RawQuery,
		RequestBody:  string(requestBody),
		Code:         sel.Code,
		Scenario:     sel.ScenarioName,
		Status:       sel.Scenario.Status,
		ContentType:  sel.Scenario.ContentType,
		ResponseBody: string(sel.Scenario.Body),
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
		groups, shared := groupRouteScenarios(manifest, e.route, name)
		state := ProviderState{
			Name:            name,
			Enabled:         manifest.Enabled,
			GlobalScenario:  e.global[name],
			Routes:          make([]string, 0, len(manifest.Routes)),
			Scenarios:       make([]string, 0, len(manifest.Scenarios)),
			RouteGroups:     groups,
			SharedScenarios: shared,
		}
		for _, route := range manifest.Routes {
			state.Routes = append(state.Routes, route.Method+" "+route.Path)
		}
		for scenario := range manifest.Scenarios {
			state.Scenarios = append(state.Scenarios, scenario)
		}
		sort.Strings(state.Routes)
		sort.Strings(state.Scenarios)
		states = append(states, state)
	}
	sort.Slice(states, func(i, j int) bool { return states[i].Name < states[j].Name })
	return states
}

// groupRouteScenarios assigns each scenario to its explicit manifest owner:
// either a route via routeId, or the shared bucket via scope: "shared".
func groupRouteScenarios(manifest *provider.Manifest, active map[routeKey]string, providerName string) ([]RouteScenarioGroup, []string) {
	names := make([]string, 0, len(manifest.Scenarios))
	for name := range manifest.Scenarios {
		names = append(names, name)
	}
	sort.Strings(names)

	groups := make([]RouteScenarioGroup, len(manifest.Routes))
	routeIndex := make(map[string]int, len(manifest.Routes))
	for i, route := range manifest.Routes {
		groups[i] = RouteScenarioGroup{
			RouteID:         route.ID,
			Method:          route.Method,
			Path:            route.Path,
			DefaultScenario: route.DefaultScenario,
			ActiveScenario:  active[routeKey{providerName, route.ID}],
			Scenarios:       []string{},
		}
		routeIndex[route.ID] = i
	}

	shared := []string{}
	for _, name := range names {
		scenario := manifest.Scenarios[name]
		if scenario.Scope == provider.ScenarioScopeShared {
			shared = append(shared, name)
			continue
		}
		if idx, ok := routeIndex[scenario.RouteID]; ok {
			groups[idx].Scenarios = append(groups[idx].Scenarios, name)
		}
	}
	return groups, shared
}

func (e *Engine) ScenarioDetail(providerName, scenarioName string) (ScenarioDetail, bool) {
	e.mu.Lock()
	defer e.mu.Unlock()
	manifest, ok := e.registry.Providers[providerName]
	if !ok {
		return ScenarioDetail{}, false
	}
	base, ok := manifest.Scenarios[scenarioName]
	if !ok {
		return ScenarioDetail{}, false
	}
	detail := ScenarioDetail{
		Provider:    providerName,
		Scenario:    scenarioName,
		Status:      base.Status,
		ContentType: base.ContentType,
		Body:        string(base.Body),
	}
	if ov, ok := e.overrides[overrideKey{providerName, scenarioName}]; ok {
		detail.HasOverride = true
		detail.Overridden = &struct {
			Status      int    `json:"status"`
			ContentType string `json:"contentType"`
			Body        string `json:"body"`
		}{
			Status:      ov.Status,
			ContentType: ov.ContentType,
			Body:        string(ov.Body),
		}
	}
	return detail, true
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

func (e *Engine) RouteScenario(providerName, routeID string) string {
	e.mu.Lock()
	defer e.mu.Unlock()
	return e.route[routeKey{providerName, routeID}]
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
	if !isSharedScenario(manifest.Scenarios[scenarioName]) {
		return false
	}
	e.global[providerName] = scenarioName
	return true
}

func (e *Engine) SetRouteScenario(providerName, routeID, scenarioName string) bool {
	e.mu.Lock()
	defer e.mu.Unlock()
	manifest, ok := e.registry.Providers[providerName]
	if !ok {
		return false
	}
	if !manifestHasRoute(manifest, routeID) {
		return false
	}
	key := routeKey{providerName, routeID}
	if scenarioName == "" {
		delete(e.route, key)
		return true
	}
	scenario, ok := manifest.Scenarios[scenarioName]
	if !ok {
		return false
	}
	if scenario.RouteID != routeID {
		return false
	}
	e.route[key] = scenarioName
	return true
}

func manifestHasRoute(manifest *provider.Manifest, routeID string) bool {
	for _, route := range manifest.Routes {
		if route.ID == routeID {
			return true
		}
	}
	return false
}

func scenarioAppliesToRoute(scenario provider.Scenario, route provider.Route) bool {
	return isSharedScenario(scenario) || scenario.RouteID == route.ID
}

func isSharedScenario(scenario provider.Scenario) bool {
	return scenario.Scope == provider.ScenarioScopeShared
}

func (e *Engine) GetOverride(providerName, scenarioName string) (Override, bool) {
	e.mu.Lock()
	defer e.mu.Unlock()
	ov, ok := e.overrides[overrideKey{providerName, scenarioName}]
	return ov, ok
}

func (e *Engine) SetOverride(providerName, scenarioName string, ov Override) bool {
	e.mu.Lock()
	defer e.mu.Unlock()
	manifest, ok := e.registry.Providers[providerName]
	if !ok {
		return false
	}
	base, ok := manifest.Scenarios[scenarioName]
	if !ok {
		return false
	}
	if ov.Status == 0 {
		ov.Status = base.Status
	}
	if ov.ContentType == "" {
		ov.ContentType = base.ContentType
	}
	if !ov.HasBody {
		ov.Body = base.Body
		ov.HasBody = true
	}
	e.overrides[overrideKey{providerName, scenarioName}] = ov
	return true
}

func (e *Engine) ClearOverride(providerName, scenarioName string) bool {
	e.mu.Lock()
	defer e.mu.Unlock()
	key := overrideKey{providerName, scenarioName}
	if _, ok := e.overrides[key]; !ok {
		return false
	}
	delete(e.overrides, key)
	return true
}

func (e *Engine) Reset() {
	e.mu.Lock()
	defer e.mu.Unlock()
	e.global = make(map[string]string)
	e.route = make(map[routeKey]string)
	e.counts = make(map[string]int64)
	e.recent = nil
	e.overrides = make(map[overrideKey]Override)
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
