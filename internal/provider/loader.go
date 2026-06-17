package provider

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

func LoadRegistry(dir string) (*Registry, error) {
	matches, err := filepath.Glob(filepath.Join(dir, "*.yaml"))
	if err != nil {
		return nil, fmt.Errorf("scan providers %q: %w", dir, err)
	}
	reg := &Registry{
		Providers: make(map[string]*Manifest),
		Routes:    make(map[string][]Route),
	}
	seenRoutes := make(map[string]string)
	for _, path := range matches {
		manifest, err := loadManifest(path)
		if err != nil {
			return nil, err
		}
		if manifest.Name == "" {
			return nil, fmt.Errorf("provider %q missing name", path)
		}
		if _, exists := reg.Providers[manifest.Name]; exists {
			return nil, fmt.Errorf("duplicate provider name %q", manifest.Name)
		}
		if err := validateManifest(manifest); err != nil {
			return nil, fmt.Errorf("provider %q: %w", manifest.Name, err)
		}
		reg.Providers[manifest.Name] = manifest
		if !manifest.Enabled {
			continue
		}
		for i := range manifest.Routes {
			route := manifest.Routes[i]
			route.Provider = manifest.Name
			key := route.Method + " " + route.Path
			if prior, exists := seenRoutes[key]; exists {
				return nil, fmt.Errorf("route %s duplicated by providers %q and %q", key, prior, manifest.Name)
			}
			seenRoutes[key] = manifest.Name
			reg.Routes[route.Path] = append(reg.Routes[route.Path], route)
		}
	}
	return reg, nil
}

func loadManifest(path string) (*Manifest, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read provider %q: %w", path, err)
	}
	var manifest Manifest
	if err := json.Unmarshal(data, &manifest); err != nil {
		return nil, fmt.Errorf("parse provider %q: %w", path, err)
	}
	for name, scenario := range manifest.Scenarios {
		if scenario.Status == 0 {
			scenario.Status = 200
		}
		if scenario.ContentType == "" {
			scenario.ContentType = "application/json"
		}
		if scenario.Fixture == "" {
			return nil, fmt.Errorf("provider %q scenario %q missing fixture", manifest.Name, name)
		}
		body, err := readFixture(path, scenario.Fixture)
		if err != nil {
			return nil, fmt.Errorf("provider %q scenario %q read fixture %q: %w", manifest.Name, name, scenario.Fixture, err)
		}
		scenario.Body = body
		manifest.Scenarios[name] = scenario
	}
	return &manifest, nil
}

func readFixture(manifestPath, fixture string) ([]byte, error) {
	if filepath.IsAbs(fixture) {
		return os.ReadFile(fixture)
	}
	if body, err := os.ReadFile(fixture); err == nil {
		return body, nil
	}
	projectRelative := filepath.Join(filepath.Dir(manifestPath), "..", "..", fixture)
	return os.ReadFile(projectRelative)
}

func validateManifest(manifest *Manifest) error {
	for i, route := range manifest.Routes {
		if route.ID == "" {
			return fmt.Errorf("route[%d] missing id", i)
		}
		if route.Method == "" || route.Path == "" {
			return fmt.Errorf("route %q missing method or path", route.ID)
		}
		manifest.Routes[i].Method = strings.ToUpper(route.Method)
		if _, ok := manifest.Scenarios[route.DefaultScenario]; !ok {
			return fmt.Errorf("route %q references missing default scenario %q", route.ID, route.DefaultScenario)
		}
	}
	for _, rule := range manifest.Rules {
		if rule.Scenario == "" {
			return fmt.Errorf("rule %q missing scenario", rule.Name)
		}
		if _, ok := manifest.Scenarios[rule.Scenario]; !ok {
			return fmt.Errorf("rule %q references missing scenario %q", rule.Name, rule.Scenario)
		}
	}
	return nil
}
