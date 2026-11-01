package server

import (
	"net/http"
	"path/filepath"
)

// webHandler serves the built web app from dir. Requests for existing files are
// served directly; every other GET falls back to index.html so the app loads.
func webHandler(dir string) http.HandlerFunc {
	fs := http.Dir(dir)
	fileServer := http.FileServer(fs)
	index := filepath.Join(dir, "index.html")

	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet && r.Method != http.MethodHead {
			http.NotFound(w, r)
			return
		}

		if f, err := fs.Open(r.URL.Path); err == nil {
			_ = f.Close()
			fileServer.ServeHTTP(w, r)
			return
		}

		http.ServeFile(w, r, index)
	}
}
