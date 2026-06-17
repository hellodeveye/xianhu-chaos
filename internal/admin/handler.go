package admin

import (
	"encoding/json"
	"net/http"
	"strings"

	"xianhu-chaos/internal/chaos"
)

type Handler struct {
	engine *chaos.Engine
}

func New(engine *chaos.Engine) *Handler {
	return &Handler{engine: engine}
}

func (h *Handler) Register(mux *http.ServeMux) {
	mux.HandleFunc("GET /health", h.health)
	mux.HandleFunc("GET /__admin/providers", h.providers)
	mux.HandleFunc("GET /__admin/scenarios", h.scenarios)
	mux.HandleFunc("GET /__admin/state", h.state)
	mux.HandleFunc("PUT /__admin/providers/", h.setProviderScenario)
	mux.HandleFunc("POST /__admin/reset", h.reset)
}

func (h *Handler) health(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, map[string]any{"status": "ok"})
}

func (h *Handler) providers(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, h.engine.ProviderStates())
}

func (h *Handler) scenarios(w http.ResponseWriter, r *http.Request) {
	out := make(map[string][]string)
	for _, state := range h.engine.ProviderStates() {
		out[state.Name] = state.Scenarios
	}
	writeJSON(w, http.StatusOK, out)
}

func (h *Handler) state(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, map[string]any{
		"providers":      h.engine.ProviderStates(),
		"recentRequests": h.engine.RecentRequests(),
	})
}

func (h *Handler) setProviderScenario(w http.ResponseWriter, r *http.Request) {
	trimmed := strings.TrimPrefix(r.URL.Path, "/__admin/providers/")
	providerName, ok := strings.CutSuffix(trimmed, "/scenario")
	if !ok || providerName == "" {
		writeJSON(w, http.StatusNotFound, map[string]any{"error": "admin path not found"})
		return
	}
	var req struct {
		Scenario string `json:"scenario"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]any{"error": "invalid json body"})
		return
	}
	if ok := h.engine.SetGlobalScenario(providerName, req.Scenario); !ok {
		writeJSON(w, http.StatusBadRequest, map[string]any{"error": "unknown provider or scenario"})
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"provider": providerName,
		"scenario": req.Scenario,
	})
}

func (h *Handler) reset(w http.ResponseWriter, r *http.Request) {
	h.engine.Reset()
	writeJSON(w, http.StatusOK, map[string]any{"status": "reset"})
}

func writeJSON(w http.ResponseWriter, status int, data any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(data)
}
