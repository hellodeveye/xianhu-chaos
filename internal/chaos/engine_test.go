package chaos

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"sort"
	"testing"

	"xianhu-chaos/internal/provider"
)

func newTestEngine(t *testing.T) *Engine {
	t.Helper()
	reg, err := provider.LoadRegistry("../../configs/providers")
	if err != nil {
		t.Fatalf("LoadRegistry failed: %v", err)
	}
	return New(reg, 20)
}

func umemberLoginRoute(t *testing.T, engine *Engine) provider.Route {
	t.Helper()
	routes := engine.Registry().Routes["/open/login"]
	if len(routes) != 1 {
		t.Fatalf("expected 1 /open/login route, got %d", len(routes))
	}
	return routes[0]
}

func newRequest(t *testing.T, method, path string, body string) *http.Request {
	t.Helper()
	var buf *bytes.Buffer
	if body != "" {
		buf = bytes.NewBufferString(body)
	}
	req := httptest.NewRequest(method, path, buf)
	return req
}

func TestSetOverrideChangesSelectResponse(t *testing.T) {
	engine := newTestEngine(t)
	route := umemberLoginRoute(t, engine)

	before := engine.Select(newRequest(t, "POST", "/open/login", `{}`), route, []byte(`{}`))
	if before.Scenario.Status != 200 {
		t.Fatalf("baseline status = %d, want 200", before.Scenario.Status)
	}
	if !bytes.Contains(before.Scenario.Body, []byte("chaos-mock-token")) {
		t.Fatalf("baseline body missing token: %s", before.Scenario.Body)
	}

	overrideBody := []byte(`{"overridden": true}`)
	ok := engine.SetOverride("umember", "login_success", Override{
		Status:      500,
		ContentType: "application/json",
		Body:        overrideBody,
		HasBody:     true,
	})
	if !ok {
		t.Fatalf("SetOverride returned false")
	}

	after := engine.Select(newRequest(t, "POST", "/open/login", `{}`), route, []byte(`{}`))
	if after.Scenario.Status != 500 {
		t.Fatalf("overridden status = %d, want 500", after.Scenario.Status)
	}
	if after.Scenario.ContentType != "application/json" {
		t.Fatalf("overridden contentType = %q", after.Scenario.ContentType)
	}
	if !bytes.Equal(after.Scenario.Body, overrideBody) {
		t.Fatalf("overridden body = %s, want %s", after.Scenario.Body, overrideBody)
	}
}

func TestSetOverridePartialUpdateKeepsBaseFields(t *testing.T) {
	engine := newTestEngine(t)
	route := umemberLoginRoute(t, engine)

	base := engine.Select(newRequest(t, "POST", "/open/login", `{}`), route, []byte(`{}`)).Scenario

	ok := engine.SetOverride("umember", "login_success", Override{
		Status: 503,
	})
	if !ok {
		t.Fatalf("SetOverride returned false")
	}
	after := engine.Select(newRequest(t, "POST", "/open/login", `{}`), route, []byte(`{}`)).Scenario
	if after.Status != 503 {
		t.Fatalf("partial override status = %d, want 503", after.Status)
	}
	if after.ContentType != base.ContentType {
		t.Fatalf("partial override contentType = %q, want base %q", after.ContentType, base.ContentType)
	}
	if !bytes.Equal(after.Body, base.Body) {
		t.Fatalf("partial override body changed; expected base fixture preserved")
	}
}

func TestClearOverrideRestoresFixture(t *testing.T) {
	engine := newTestEngine(t)
	route := umemberLoginRoute(t, engine)

	engine.SetOverride("umember", "login_success", Override{
		Status:  500,
		Body:    []byte(`{"overridden": true}`),
		HasBody: true,
	})

	engine.ClearOverride("umember", "login_success")

	after := engine.Select(newRequest(t, "POST", "/open/login", `{}`), route, []byte(`{}`))
	if after.Scenario.Status != 200 {
		t.Fatalf("after clear status = %d, want 200", after.Scenario.Status)
	}
	if !bytes.Contains(after.Scenario.Body, []byte("chaos-mock-token")) {
		t.Fatalf("after clear body should be original fixture: %s", after.Scenario.Body)
	}
}

func TestResetClearsOverrides(t *testing.T) {
	engine := newTestEngine(t)
	route := umemberLoginRoute(t, engine)

	engine.SetOverride("umember", "login_success", Override{
		Status:  500,
		Body:    []byte(`{"overridden": true}`),
		HasBody: true,
	})

	engine.Reset()

	after := engine.Select(newRequest(t, "POST", "/open/login", `{}`), route, []byte(`{}`))
	if after.Scenario.Status != 200 {
		t.Fatalf("after reset status = %d, want 200", after.Scenario.Status)
	}
}

func TestSetOverrideRejectsUnknownScenario(t *testing.T) {
	engine := newTestEngine(t)
	if ok := engine.SetOverride("umember", "nope", Override{Status: 500}); ok {
		t.Fatalf("expected SetOverride false for unknown scenario")
	}
	if ok := engine.SetOverride("nope", "login_success", Override{Status: 500}); ok {
		t.Fatalf("expected SetOverride false for unknown provider")
	}
}

func TestSetGlobalScenarioAllowsOnlySharedScenarios(t *testing.T) {
	engine := newTestEngine(t)

	if ok := engine.SetGlobalScenario("umember", "login_http_500"); ok {
		t.Fatalf("expected route-specific global scenario to be rejected")
	}
	if got := engine.GlobalScenario("umember"); got != "" {
		t.Fatalf("global scenario after rejected route scenario = %q, want empty", got)
	}

	if ok := engine.SetGlobalScenario("umember", "random_http_500"); !ok {
		t.Fatalf("expected shared global scenario to be accepted")
	}
	if got := engine.GlobalScenario("umember"); got != "random_http_500" {
		t.Fatalf("global scenario = %q, want random_http_500", got)
	}

	if ok := engine.SetGlobalScenario("umember", ""); !ok {
		t.Fatalf("expected empty global scenario to clear state")
	}
	if got := engine.GlobalScenario("umember"); got != "" {
		t.Fatalf("global scenario after clear = %q, want empty", got)
	}
}

func TestSetRouteScenarioAllowsOnlyRouteOwnedScenarios(t *testing.T) {
	engine := newTestEngine(t)
	route := umemberLoginRoute(t, engine)

	if ok := engine.SetRouteScenario("umember", "login", "detail_http_500"); ok {
		t.Fatalf("expected other route scenario to be rejected")
	}
	if ok := engine.SetRouteScenario("umember", "login", "random_http_500"); ok {
		t.Fatalf("expected shared scenario to be rejected for route override")
	}
	if ok := engine.SetRouteScenario("umember", "login", "login_http_500"); !ok {
		t.Fatalf("expected route-owned scenario to be accepted")
	}

	selected := engine.Select(newRequest(t, "POST", "/open/login", `{}`), route, []byte(`{}`))
	if selected.ScenarioName != "login_http_500" || selected.Scenario.Status != 500 {
		t.Fatalf("route scenario selection = %q status=%d, want login_http_500 500", selected.ScenarioName, selected.Scenario.Status)
	}

	if ok := engine.SetRouteScenario("umember", "login", ""); !ok {
		t.Fatalf("expected empty route scenario to clear state")
	}
	selected = engine.Select(newRequest(t, "POST", "/open/login", `{}`), route, []byte(`{}`))
	if selected.ScenarioName != "login_success" {
		t.Fatalf("scenario after clear = %q, want login_success", selected.ScenarioName)
	}
}

func TestRouteScenarioPrecedenceBetweenRulesAndGlobal(t *testing.T) {
	engine := newTestEngine(t)
	route := umemberLoginRoute(t, engine)

	engine.SetGlobalScenario("umember", "random_http_500")
	engine.SetRouteScenario("umember", "login", "login_business_fail")

	selected := engine.Select(newRequest(t, "POST", "/open/login", `{}`), route, []byte(`{}`))
	if selected.ScenarioName != "login_business_fail" {
		t.Fatalf("route scenario should beat global scenario, got %q", selected.ScenarioName)
	}

	req := newRequest(t, "POST", "/open/login", `{}`)
	req.Header.Set(ScenarioHeader, "login_http_500")
	selected = engine.Select(req, route, []byte(`{}`))
	if selected.ScenarioName != "login_http_500" {
		t.Fatalf("header scenario should beat route scenario, got %q", selected.ScenarioName)
	}
}

func TestProviderStatesGroupScenariosByRoute(t *testing.T) {
	engine := newTestEngine(t)
	engine.SetRouteScenario("douyin", "certificate_prepare", "prepare_http_500")
	states := engine.ProviderStates()

	if len(states) != 2 || states[0].Name != "douyin" || states[1].Name != "umember" {
		t.Fatalf("expected providers sorted [douyin umember], got %v", stateNames(states))
	}

	douyin := states[0]

	// Every scenario must land in exactly one place (a route group or shared).
	seen := map[string]int{}
	for _, g := range douyin.RouteGroups {
		if !sort.StringsAreSorted(g.Scenarios) {
			t.Fatalf("route %s scenarios not sorted: %v", g.RouteID, g.Scenarios)
		}
		for _, s := range g.Scenarios {
			seen[s]++
		}
	}
	for _, s := range douyin.SharedScenarios {
		seen[s]++
	}
	for _, s := range douyin.Scenarios {
		if seen[s] != 1 {
			t.Fatalf("scenario %q assigned %d times, want exactly 1", s, seen[s])
		}
	}
	if len(seen) != len(douyin.Scenarios) {
		t.Fatalf("grouped %d scenarios, want %d", len(seen), len(douyin.Scenarios))
	}

	prepare := findGroup(t, douyin.RouteGroups, "certificate_prepare")
	if prepare.DefaultScenario != "prepare_success" {
		t.Fatalf("prepare default = %q, want prepare_success", prepare.DefaultScenario)
	}
	if prepare.ActiveScenario != "prepare_http_500" {
		t.Fatalf("prepare active scenario = %q, want prepare_http_500", prepare.ActiveScenario)
	}
	if !contains(prepare.Scenarios, "prepare_success") || !contains(prepare.Scenarios, "prepare_verify_bad_json_token") {
		t.Fatalf("prepare group missing expected scenarios: %v", prepare.Scenarios)
	}

	// token_* must not leak into the prepare or verify groups.
	if contains(prepare.Scenarios, "token_success") {
		t.Fatalf("token_success leaked into prepare group: %v", prepare.Scenarios)
	}

	// Cross-cutting scenarios are explicitly marked as shared in the manifest.
	for _, want := range []string{"slow_2s", "random_http_500", "third_request_http_500"} {
		if !contains(douyin.SharedScenarios, want) {
			t.Fatalf("expected %q in shared, got %v", want, douyin.SharedScenarios)
		}
	}
}

func stateNames(states []ProviderState) []string {
	out := make([]string, len(states))
	for i, s := range states {
		out[i] = s.Name
	}
	return out
}

func findGroup(t *testing.T, groups []RouteScenarioGroup, routeID string) RouteScenarioGroup {
	t.Helper()
	for _, g := range groups {
		if g.RouteID == routeID {
			return g
		}
	}
	t.Fatalf("route group %q not found", routeID)
	return RouteScenarioGroup{}
}

func contains(values []string, want string) bool {
	for _, v := range values {
		if v == want {
			return true
		}
	}
	return false
}

func TestScenarioDetailReportsOverride(t *testing.T) {
	engine := newTestEngine(t)

	detail, ok := engine.ScenarioDetail("umember", "login_success")
	if !ok {
		t.Fatalf("ScenarioDetail returned false")
	}
	if detail.HasOverride {
		t.Fatalf("fresh engine should have no override")
	}
	if detail.Body == "" {
		t.Fatalf("detail body should contain fixture content")
	}

	engine.SetOverride("umember", "login_success", Override{
		Status:  500,
		Body:    []byte(`{"overridden": true}`),
		HasBody: true,
	})

	detail, ok = engine.ScenarioDetail("umember", "login_success")
	if !ok {
		t.Fatalf("ScenarioDetail returned false after override")
	}
	if !detail.HasOverride || detail.Overridden == nil {
		t.Fatalf("detail should report override")
	}
	if detail.Overridden.Status != 500 {
		t.Fatalf("overridden status = %d, want 500", detail.Overridden.Status)
	}
	if detail.Overridden.Body != `{"overridden": true}` {
		t.Fatalf("overridden body = %q", detail.Overridden.Body)
	}
}
