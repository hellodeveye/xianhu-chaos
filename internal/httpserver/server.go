package httpserver

import (
	"log/slog"
	"net/http"
	"time"

	"xianhu-chaos/internal/admin"
	"xianhu-chaos/internal/chaos"
	"xianhu-chaos/internal/provider"
	"xianhu-chaos/internal/ui"
)

type Server struct {
	engine *chaos.Engine
	mux    *http.ServeMux
}

func New(engine *chaos.Engine) *Server {
	s := &Server{
		engine: engine,
		mux:    http.NewServeMux(),
	}
	ui.Register(s.mux)
	admin.New(engine).Register(s.mux)
	s.registerProviderRoutes()
	return s
}

func (s *Server) Handler() http.Handler {
	return loggingMiddleware(s.mux)
}

func (s *Server) registerProviderRoutes() {
	for path, routes := range s.engine.Registry().Routes {
		path := path
		routes := routes
		s.mux.HandleFunc(path, func(w http.ResponseWriter, r *http.Request) {
			for _, route := range routes {
				if r.Method == route.Method {
					s.serveRoute(w, r, route)
					return
				}
			}
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		})
	}
}

func (s *Server) serveRoute(w http.ResponseWriter, r *http.Request, route provider.Route) {
	body, err := chaos.ReadAndRestoreBody(r)
	if err != nil {
		http.Error(w, "failed to read request body", http.StatusBadRequest)
		return
	}
	selection := s.engine.Select(r, route, body)
	s.engine.ApplyDelay(selection.Scenario)
	s.engine.Log(route, selection)
	w.Header().Set("Content-Type", selection.Scenario.ContentType)
	w.WriteHeader(selection.Scenario.Status)
	_, _ = w.Write(selection.Scenario.Body)
}

func loggingMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		next.ServeHTTP(w, r)
		slog.Info("request",
			"method", r.Method,
			"path", r.URL.Path,
			"duration", time.Since(start).String(),
		)
	})
}
