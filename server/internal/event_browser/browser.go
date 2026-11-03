package event_browser

import (
	"embed"
	"errors"
	"fmt"
	"io/fs"
	"log/slog"
	"net/http"
	"net/http/httputil"
	"net/url"
	"path"
	"strings"
)

//go:embed all:assets
var assets embed.FS

var ErrUINotBuilt = errors.New("the event browser UI is not built, run the event browser build task")

func embeddedUI() (fs.FS, error) {
	root, err := fs.Sub(assets, "assets")
	if err != nil {
		return nil, err
	}
	if _, err := indexHTML(root); err != nil {
		return nil, ErrUINotBuilt
	}
	return root, nil
}

func indexHTML(ui fs.FS) (string, error) {
	entries, err := fs.ReadDir(ui, ".")
	if err != nil {
		return "", err
	}
	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".html") {
			continue
		}
		if entry.Name() == "index.html" {
			return entry.Name(), nil
		}
	}
	for _, entry := range entries {
		if !entry.IsDir() && strings.HasSuffix(entry.Name(), ".html") {
			return entry.Name(), nil
		}
	}
	return "", ErrUINotBuilt
}

func NewHandler(backend string, session Session, ui fs.FS, logger *slog.Logger) (http.Handler, error) {
	target, err := url.Parse(backend)
	if err != nil {
		return nil, fmt.Errorf("failed to parse the backend URL: %w", err)
	}

	proxy := newProxy(target, session, logger)
	sessions := newSessionHandler(target, session, logger)
	serveUI := newUIHandler(ui)

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.URL.Path == "/session":
			sessions.ServeHTTP(w, r)
		case strings.HasPrefix(r.URL.Path, "/api/"):
			proxy.ServeHTTP(w, r)
		default:
			serveUI(w, r)
		}
	}), nil
}

func newProxy(target *url.URL, session Session, logger *slog.Logger) *httputil.ReverseProxy {
	return &httputil.ReverseProxy{
		Rewrite: func(r *httputil.ProxyRequest) {
			r.SetURL(target)
			r.Out.URL.Path, r.Out.URL.RawPath = joinPath(target, r.In.URL)
			r.Out.Host = target.Host

			// the token stays in this process, so the browser never sees it
			r.Out.Header.Set("Authorization", "Bearer "+session.Token())
			r.SetXForwarded()
		},
		ModifyResponse: func(resp *http.Response) error {
			if resp.StatusCode != http.StatusUnauthorized || session.Fixed() {
				return nil
			}
			if err := session.Clear(); err != nil {
				logger.Warn("failed to clear the session", "error", err)
			}
			return nil
		},
		FlushInterval: -1,
		ErrorHandler: func(w http.ResponseWriter, r *http.Request, err error) {
			logger.Warn("failed to reach the backend", "path", r.URL.Path, "error", err)
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusBadGateway)
			_, _ = fmt.Fprintf(w, `{"successful":false,"messages":["the backend did not answer: %s"]}`, err)
		},
	}
}

func newUIHandler(ui fs.FS) http.HandlerFunc {
	fileServer := http.FileServer(http.FS(ui))
	index, err := indexHTML(ui)
	if err != nil {
		index = "index.html"
	}

	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet && r.Method != http.MethodHead {
			http.NotFound(w, r)
			return
		}

		name := strings.TrimPrefix(path.Clean(r.URL.Path), "/")
		if name != "" && name != "." {
			if _, err := fs.Stat(ui, name); err == nil {
				fileServer.ServeHTTP(w, r)
				return
			}
		}

		http.ServeFileFS(w, r, ui, index)
	}
}

func joinPath(target *url.URL, in *url.URL) (string, string) {
	if target.RawPath == "" && in.RawPath == "" {
		return singleJoiningSlash(target.Path, in.Path), ""
	}

	outPath := singleJoiningSlash(target.Path, in.Path)
	outRawPath := singleJoiningSlash(target.EscapedPath(), in.EscapedPath())
	if outPath == outRawPath {
		outRawPath = ""
	}
	return outPath, outRawPath
}

func singleJoiningSlash(a, b string) string {
	aslash := strings.HasSuffix(a, "/")
	bslash := strings.HasPrefix(b, "/")
	switch {
	case aslash && bslash:
		return a + b[1:]
	case !aslash && !bslash:
		return a + "/" + b
	}
	return a + b
}
