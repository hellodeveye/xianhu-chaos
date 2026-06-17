package ui

import (
	"embed"
	"io/fs"
	"net/http"
)

//go:embed static
var assets embed.FS

func Register(mux *http.ServeMux) {
	staticFS, err := fs.Sub(assets, "static")
	if err != nil {
		panic(err)
	}
	mux.HandleFunc("GET /{$}", func(w http.ResponseWriter, r *http.Request) {
		http.ServeFileFS(w, r, staticFS, "index.html")
	})
	mux.Handle("GET /static/", http.StripPrefix("/static/", http.FileServerFS(staticFS)))
}
