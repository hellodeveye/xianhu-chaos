package provider

const ScenarioScopeShared = "shared"

type Manifest struct {
	Name      string              `json:"name"`
	Enabled   bool                `json:"enabled"`
	BasePath  string              `json:"basePath"`
	Routes    []Route             `json:"routes"`
	Scenarios map[string]Scenario `json:"scenarios"`
	Rules     []Rule              `json:"rules"`
}

type Route struct {
	ID              string `json:"id"`
	Method          string `json:"method"`
	Path            string `json:"path"`
	DefaultScenario string `json:"defaultScenario"`
	CodeParam       string `json:"codeParam"`
	Provider        string `json:"-"`
}

type Scenario struct {
	Status      int     `json:"status"`
	Fixture     string  `json:"fixture"`
	ContentType string  `json:"contentType"`
	RouteID     string  `json:"routeId"`
	Scope       string  `json:"scope"`
	DelayMS     int     `json:"delayMs"`
	JitterMS    int     `json:"jitterMs"`
	ErrorRate   float64 `json:"errorRate"`
	FailOnNth   int64   `json:"failOnNth"`
	Body        []byte  `json:"-"`
}

type Rule struct {
	Name     string `json:"name"`
	Match    Match  `json:"match"`
	Scenario string `json:"scenario"`
}

type Match struct {
	Code   string            `json:"code"`
	Header map[string]string `json:"header"`
}

type Registry struct {
	Providers map[string]*Manifest
	Routes    map[string][]Route
}
