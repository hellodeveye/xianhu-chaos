package httpserver

import (
	"encoding/json"
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

func jsonQuote(s string) string {
	b, _ := json.Marshal(s)
	return string(b)
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

func TestDouyinDefaultToken(t *testing.T) {
	ts := newTestServer(t)
	defer ts.Close()

	resp, err := http.Post(ts.URL+"/oauth/client_token/", "application/json", strings.NewReader(`{"client_key":"x","client_secret":"y","grant_type":"client_credential"}`))
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(resp.Body)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("douyin token status = %d body=%s", resp.StatusCode, body)
	}
	if !strings.Contains(string(body), "chaos-douyin-token") {
		t.Fatalf("douyin token missing: %s", body)
	}
}

func TestDouyinPrepareRuleCanDriveVerifyFailure(t *testing.T) {
	ts := newTestServer(t)
	defer ts.Close()

	resp, err := http.Get(ts.URL + "/goodlife/v1/fulfilment/certificate/prepare/?poi_id=7630290236999731263&code=DY_VERIFY_FAIL")
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(resp.Body)
	if !strings.Contains(string(body), `"verify_token": "DY_VERIFY_FAIL_TOKEN"`) {
		t.Fatalf("prepare fixture did not return verify failure token: %s", body)
	}

	resp, err = http.Post(ts.URL+"/goodlife/v1/fulfilment/certificate/verify/", "application/json", strings.NewReader(`{"verify_token":"DY_VERIFY_FAIL_TOKEN","poi_id":"7630290236999731263","encrypted_codes":["chaos-verify-fail-code"]}`))
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	body, _ = io.ReadAll(resp.Body)
	if !strings.Contains(string(body), `"msg": "券码已核销"`) {
		t.Fatalf("verify failure fixture not selected: %s", body)
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

func TestScenarioDetailReturnsFixtureBody(t *testing.T) {
	ts := newTestServer(t)
	defer ts.Close()

	resp, err := http.Get(ts.URL + "/__admin/providers/umember/scenarios/login_success")
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("detail status = %d", resp.StatusCode)
	}
	var detail struct {
		Provider    string `json:"provider"`
		Scenario    string `json:"scenario"`
		Status      int    `json:"status"`
		ContentType string `json:"contentType"`
		Body        string `json:"body"`
		HasOverride bool   `json:"hasOverride"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&detail); err != nil {
		t.Fatal(err)
	}
	if detail.Provider != "umember" || detail.Scenario != "login_success" {
		t.Fatalf("detail identity = %s/%s", detail.Provider, detail.Scenario)
	}
	if detail.Status != 200 {
		t.Fatalf("detail status = %d, want 200", detail.Status)
	}
	if detail.HasOverride {
		t.Fatalf("fresh engine detail should have no override")
	}
	if !strings.Contains(detail.Body, "chaos-mock-token") {
		t.Fatalf("detail body should contain fixture token: %s", detail.Body)
	}
}

func TestScenarioOverrideChangesProviderResponse(t *testing.T) {
	ts := newTestServer(t)
	defer ts.Close()

	overrideBody := `{"overridden": true}`
	payload := `{"status":500,"body":` + jsonQuote(overrideBody) + `}`
	req, err := http.NewRequest(http.MethodPut, ts.URL+"/__admin/providers/umember/scenarios/login_success", strings.NewReader(payload))
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
		t.Fatalf("override PUT status = %d", resp.StatusCode)
	}

	resp, err = http.Post(ts.URL+"/open/login", "application/json", strings.NewReader(`{}`))
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(resp.Body)
	if resp.StatusCode != http.StatusInternalServerError {
		t.Fatalf("login status after override = %d, want 500", resp.StatusCode)
	}
	if string(body) != overrideBody {
		t.Fatalf("login body after override = %s, want %s", body, overrideBody)
	}
}

func TestScenarioOverrideRoutePrecedence(t *testing.T) {
	ts := newTestServer(t)
	defer ts.Close()

	req, err := http.NewRequest(http.MethodPut, ts.URL+"/__admin/providers/umember/scenarios/login_success", strings.NewReader(`{"status":503,"body":"{}"}`))
	if err != nil {
		t.Fatal(err)
	}
	req.Header.Set("Content-Type", "application/json")
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("PUT /scenarios/{scenario} should hit override handler, got status %d (route precedence bug)", resp.StatusCode)
	}

	detailResp, err := http.Get(ts.URL + "/__admin/providers/umember/scenarios/login_success")
	if err != nil {
		t.Fatal(err)
	}
	defer detailResp.Body.Close()
	var detail struct {
		HasOverride bool `json:"hasOverride"`
		Overridden  *struct {
			Status int `json:"status"`
		} `json:"overridden"`
	}
	if err := json.NewDecoder(detailResp.Body).Decode(&detail); err != nil {
		t.Fatal(err)
	}
	if !detail.HasOverride || detail.Overridden == nil || detail.Overridden.Status != 503 {
		t.Fatalf("override not applied; hasOverride=%v overridden=%+v", detail.HasOverride, detail.Overridden)
	}
}

func TestScenarioOverrideDeleteRestoresFixture(t *testing.T) {
	ts := newTestServer(t)
	defer ts.Close()

	putReq, err := http.NewRequest(http.MethodPut, ts.URL+"/__admin/providers/umember/scenarios/login_success", strings.NewReader(`{"status":500,"body":"{}"}`))
	if err != nil {
		t.Fatal(err)
	}
	putReq.Header.Set("Content-Type", "application/json")
	putResp, err := http.DefaultClient.Do(putReq)
	if err != nil {
		t.Fatal(err)
	}
	putResp.Body.Close()

	delReq, err := http.NewRequest(http.MethodDelete, ts.URL+"/__admin/providers/umember/scenarios/login_success", nil)
	if err != nil {
		t.Fatal(err)
	}
	delResp, err := http.DefaultClient.Do(delReq)
	if err != nil {
		t.Fatal(err)
	}
	defer delResp.Body.Close()
	if delResp.StatusCode != http.StatusOK {
		t.Fatalf("DELETE override status = %d", delResp.StatusCode)
	}

	resp, err := http.Post(ts.URL+"/open/login", "application/json", strings.NewReader(`{}`))
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(resp.Body)
	if resp.StatusCode != 200 {
		t.Fatalf("login status after delete override = %d, want 200", resp.StatusCode)
	}
	if !strings.Contains(string(body), "chaos-mock-token") {
		t.Fatalf("login body after delete should be original fixture: %s", body)
	}
}

func TestResetClearsScenarioOverrides(t *testing.T) {
	ts := newTestServer(t)
	defer ts.Close()

	putReq, err := http.NewRequest(http.MethodPut, ts.URL+"/__admin/providers/umember/scenarios/login_success", strings.NewReader(`{"status":500,"body":"{}"}`))
	if err != nil {
		t.Fatal(err)
	}
	putReq.Header.Set("Content-Type", "application/json")
	putResp, err := http.DefaultClient.Do(putReq)
	if err != nil {
		t.Fatal(err)
	}
	putResp.Body.Close()

	resetResp, err := http.Post(ts.URL+"/__admin/reset", "application/json", nil)
	if err != nil {
		t.Fatal(err)
	}
	resetResp.Body.Close()

	resp, err := http.Post(ts.URL+"/open/login", "application/json", strings.NewReader(`{}`))
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		t.Fatalf("login status after reset = %d, want 200 (override should be cleared)", resp.StatusCode)
	}
}
