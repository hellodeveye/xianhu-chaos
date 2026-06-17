package httpserver

import (
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"xianhu-chaos/internal/chaos"
	"xianhu-chaos/internal/provider"
)

func newTestServer(t *testing.T) *httptest.Server {
	t.Helper()
	reg, err := provider.LoadRegistry("../../configs/providers")
	if err != nil {
		t.Fatalf("LoadRegistry failed: %v", err)
	}
	engine := chaos.New(reg, 20)
	return httptest.NewServer(New(engine).Handler())
}

func TestHealthAndDefaultLogin(t *testing.T) {
	ts := newTestServer(t)
	defer ts.Close()

	resp, err := http.Get(ts.URL + "/health")
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("health status = %d", resp.StatusCode)
	}

	resp, err = http.Post(ts.URL+"/open/login", "application/json", strings.NewReader(`{"email":"x","password":"y"}`))
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(resp.Body)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("login status = %d body=%s", resp.StatusCode, body)
	}
	if !strings.Contains(string(body), "chaos-mock-token") {
		t.Fatalf("login token missing: %s", body)
	}
}

func TestWebUIAssets(t *testing.T) {
	ts := newTestServer(t)
	defer ts.Close()

	resp, err := http.Get(ts.URL + "/")
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(resp.Body)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("root status = %d body=%s", resp.StatusCode, body)
	}
	if !strings.Contains(string(body), "xianhu-chaos") {
		t.Fatalf("root does not look like UI: %s", body)
	}

	resp, err = http.Get(ts.URL + "/static/app.js")
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("static app status = %d", resp.StatusCode)
	}
}

func TestCouponCodeRuleSelectsVerifyFalse(t *testing.T) {
	ts := newTestServer(t)
	defer ts.Close()

	resp, err := http.Post(ts.URL+"/open/user/meituan/coupon/verify", "application/json", strings.NewReader(`{"store_id":"8674228","coupon_code":"VERIFY_FALSE"}`))
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(resp.Body)
	if !strings.Contains(string(body), `"verify_success": false`) {
		t.Fatalf("verify_false fixture not selected: %s", body)
	}
}

func TestHeaderScenarioTakesPrecedence(t *testing.T) {
	ts := newTestServer(t)
	defer ts.Close()

	req, err := http.NewRequest(http.MethodGet, ts.URL+"/open/user/meituan/coupon/detail?store_id=8674228&coupon_code=0109760017002", nil)
	if err != nil {
		t.Fatal(err)
	}
	req.Header.Set(chaos.ScenarioHeader, "detail_bad_json")
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(resp.Body)
	if !strings.Contains(string(body), `{"code":10000,"msg":"broken"`) {
		t.Fatalf("header scenario not selected: %s", body)
	}
}

func TestAdminSetsGlobalScenario(t *testing.T) {
	ts := newTestServer(t)
	defer ts.Close()

	req, err := http.NewRequest(http.MethodPut, ts.URL+"/__admin/providers/umember/scenario", strings.NewReader(`{"scenario":"login_http_500"}`))
	if err != nil {
		t.Fatal(err)
	}
	req.Header.Set("Content-Type", "application/json")
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("admin status = %d", resp.StatusCode)
	}

	resp, err = http.Post(ts.URL+"/open/login", "application/json", strings.NewReader(`{}`))
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusInternalServerError {
		t.Fatalf("login status after global scenario = %d", resp.StatusCode)
	}
}
