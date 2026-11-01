package server

import (
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestWebHandler(t *testing.T) {
	t.Run("serves the index page at the root", func(t *testing.T) {
		srv := newWebServer(t, map[string]string{"index.html": "<html>app</html>"})
		defer srv.Close()

		res := get(t, srv.URL+"/")

		require.Equal(t, http.StatusOK, res.StatusCode)
		require.Equal(t, "<html>app</html>", readBody(t, res))
	})

	t.Run("serves static files", func(t *testing.T) {
		srv := newWebServer(t, map[string]string{
			"index.html":    "<html>app</html>",
			"assets/app.js": "console.log('hi')",
		})
		defer srv.Close()

		res := get(t, srv.URL+"/assets/app.js")

		require.Equal(t, http.StatusOK, res.StatusCode)
		require.Equal(t, "console.log('hi')", readBody(t, res))
	})

	t.Run("falls back to the index page for unknown routes", func(t *testing.T) {
		srv := newWebServer(t, map[string]string{"index.html": "<html>app</html>"})
		defer srv.Close()

		res := get(t, srv.URL+"/library")

		require.Equal(t, http.StatusOK, res.StatusCode)
		require.Equal(t, "<html>app</html>", readBody(t, res))
	})

	t.Run("rejects non-GET requests", func(t *testing.T) {
		srv := newWebServer(t, map[string]string{"index.html": "<html>app</html>"})
		defer srv.Close()

		res, err := http.Post(srv.URL+"/", "text/plain", nil)
		require.NoError(t, err)
		defer res.Body.Close()

		require.Equal(t, http.StatusNotFound, res.StatusCode)
	})
}

func newWebServer(t *testing.T, files map[string]string) *httptest.Server {
	t.Helper()
	dir := t.TempDir()
	for name, content := range files {
		path := filepath.Join(dir, name)
		require.NoError(t, os.MkdirAll(filepath.Dir(path), 0o755))
		require.NoError(t, os.WriteFile(path, []byte(content), 0o644))
	}
	return httptest.NewServer(webHandler(dir))
}

func get(t *testing.T, url string) *http.Response {
	t.Helper()
	res, err := http.Get(url)
	require.NoError(t, err)
	t.Cleanup(func() { res.Body.Close() })
	return res
}

func readBody(t *testing.T, res *http.Response) string {
	t.Helper()
	b, err := io.ReadAll(res.Body)
	require.NoError(t, err)
	return string(b)
}
