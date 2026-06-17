package provider

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoadRegistryLoadsEnabledProviderRoutes(t *testing.T) {
	reg, err := LoadRegistry("../../configs/providers")
	if err != nil {
		t.Fatalf("LoadRegistry failed: %v", err)
	}
	if _, ok := reg.Providers["umember"]; !ok {
		t.Fatalf("umember provider missing")
	}
	if provider, ok := reg.Providers["douyin"]; !ok || !provider.Enabled {
		t.Fatalf("douyin provider should be enabled")
	}
	routes := reg.Routes["/open/login"]
	if len(routes) != 1 {
		t.Fatalf("expected one /open/login route, got %d", len(routes))
	}
	if routes[0].Provider != "umember" || routes[0].Method != "POST" {
		t.Fatalf("unexpected route: %+v", routes[0])
	}
	douyinRoutes := reg.Routes["/oauth/client_token/"]
	if len(douyinRoutes) != 1 {
		t.Fatalf("expected one /oauth/client_token/ route, got %d", len(douyinRoutes))
	}
	if douyinRoutes[0].Provider != "douyin" || douyinRoutes[0].Method != "POST" {
		t.Fatalf("unexpected douyin route: %+v", douyinRoutes[0])
	}
}

func TestLoadRegistryFailsForMissingScenario(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "broken.yaml"), []byte(`{
		"name": "broken",
		"enabled": true,
		"routes": [{"id":"r","method":"GET","path":"/x","defaultScenario":"missing"}],
		"scenarios": {},
		"rules": []
	}`), 0644); err != nil {
		t.Fatal(err)
	}
	if _, err := LoadRegistry(dir); err == nil {
		t.Fatalf("expected missing scenario error")
	}
}
