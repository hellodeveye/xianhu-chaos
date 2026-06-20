package chaos

import (
	"bytes"
	"net/http"
	"net/http/httptest"
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
