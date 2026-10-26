//go:build unit

package item_test

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"net/textproto"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/hpcsc/stash/internal/api/response"
	apiTest "github.com/hpcsc/stash/internal/api/test"
	"github.com/hpcsc/stash/internal/api/use_cases/item"
	commonTest "github.com/hpcsc/stash/internal/common/test"
	"github.com/hpcsc/stash/internal/domain/command"
	"github.com/hpcsc/stash/internal/domain/command/fake"
	itemCommand "github.com/hpcsc/stash/internal/use_cases/item/command"
	"github.com/hpcsc/stash/internal/use_cases/item/items"
	"github.com/stretchr/testify/require"
)

func TestHandler(t *testing.T) {
	t.Run("capture", func(t *testing.T) {
		t.Run("dispatches a capture text command and returns the item ID", func(t *testing.T) {
			handler := fake.NewHandler[itemCommand.CaptureText]()
			bus := command.NewBus()
			bus.Register[itemCommand.CaptureText](handler)
			router := newRouter(bus, fakeReader{})

			w := send(router, "POST", "/items", `{"body": "hello", "sourceApp": "chrome"}`)

			require.Equal(t, http.StatusOK, w.Code)
			payload := handler.TriggeredWithCommand().Payload
			require.Equal(t, "hello", payload.Body)
			require.Equal(t, "chrome", payload.SourceApp)
			require.False(t, payload.CapturedAt.IsZero())
			body := commonTest.UnmarshalJson[response.Response](t, w.Body.Bytes())
			require.Equal(t, payload.ItemID.String(), body.Data.(map[string]any)["id"])
		})

		t.Run("dispatches a capture link command and returns the item ID", func(t *testing.T) {
			handler := fake.NewHandler[itemCommand.CaptureLink]()
			bus := command.NewBus()
			bus.Register[itemCommand.CaptureLink](handler)
			router := newRouter(bus, fakeReader{})

			w := send(router, "POST", "/items", `{"body": "https://example.com", "type": "link"}`)

			require.Equal(t, http.StatusOK, w.Code)
			payload := handler.TriggeredWithCommand().Payload
			require.Equal(t, "https://example.com", payload.URL)
			require.False(t, payload.CapturedAt.IsZero())
			body := commonTest.UnmarshalJson[response.Response](t, w.Body.Bytes())
			require.Equal(t, payload.ItemID.String(), body.Data.(map[string]any)["id"])
		})

		t.Run("rejects a body without text", func(t *testing.T) {
			bus := command.NewBus()
			bus.Register[itemCommand.CaptureText](fake.NewHandler[itemCommand.CaptureText]())
			router := newRouter(bus, fakeReader{})

			w := send(router, "POST", "/items", `{}`)

			require.Equal(t, http.StatusBadRequest, w.Code)
		})

		t.Run("dispatches a capture image command for an image upload", func(t *testing.T) {
			handler := fake.NewHandler[itemCommand.CaptureImage]()
			bus := command.NewBus()
			bus.Register[itemCommand.CaptureImage](handler)
			router := newRouter(bus, fakeReader{})

			body, contentType := multipartBody(t, "photo.png", "image/png", []byte("png-bytes"))
			w := sendMultipart(router, body, contentType)

			require.Equal(t, http.StatusOK, w.Code)
			payload := handler.TriggeredWithCommand().Payload
			require.True(t, strings.HasPrefix(payload.FileRef, "images/"))
			require.True(t, strings.HasSuffix(payload.FileRef, ".png"))
		})

		t.Run("dispatches a capture file command for a non-image upload", func(t *testing.T) {
			handler := fake.NewHandler[itemCommand.CaptureFile]()
			bus := command.NewBus()
			bus.Register[itemCommand.CaptureFile](handler)
			router := newRouter(bus, fakeReader{})

			body, contentType := multipartBody(t, "report.pdf", "application/pdf", []byte("pdf-bytes"))
			w := sendMultipart(router, body, contentType)

			require.Equal(t, http.StatusOK, w.Code)
			payload := handler.TriggeredWithCommand().Payload
			require.Equal(t, "report.pdf", payload.FileName)
			require.Equal(t, int64(len("pdf-bytes")), payload.FileSize)
			require.Equal(t, "application/pdf", payload.MimeType)
			require.True(t, strings.HasPrefix(payload.FileRef, "files/"))
			require.True(t, strings.HasSuffix(payload.FileRef, ".pdf"))

			responseBody := commonTest.UnmarshalJson[response.Response](t, w.Body.Bytes())
			data := responseBody.Data.(map[string]any)
			require.Equal(t, payload.ItemID.String(), data["id"])
			require.Equal(t, "file", data["type"])
			require.Equal(t, payload.FileRef, data["filePath"])
			require.Equal(t, "report.pdf", data["fileName"])
		})

		t.Run("rejects a video upload", func(t *testing.T) {
			router := newRouter(command.NewBus(), fakeReader{})

			body, contentType := multipartBody(t, "clip.mp4", "video/mp4", []byte("video-bytes"))
			w := sendMultipart(router, body, contentType)

			require.Equal(t, http.StatusBadRequest, w.Code)
			require.Contains(t, w.Body.String(), "videos are not accepted")
		})
	})

	t.Run("list", func(t *testing.T) {
		t.Run("returns the items from the read model", func(t *testing.T) {
			router := newRouter(command.NewBus(), fakeReader{items: []items.Item{{ID: "1", Body: "hello", CapturedAt: time.Now()}}})

			w := send(router, "GET", "/items", "")

			require.Equal(t, http.StatusOK, w.Code)
			require.Contains(t, w.Body.String(), `"body":"hello"`)
		})
	})

	t.Run("get", func(t *testing.T) {
		t.Run("returns the item from the read model", func(t *testing.T) {
			router := newRouter(command.NewBus(), fakeReader{item: &items.Item{ID: "1", Body: "hello"}})

			w := send(router, "GET", "/items/1", "")

			require.Equal(t, http.StatusOK, w.Code)
			require.Contains(t, w.Body.String(), `"body":"hello"`)
		})

		t.Run("returns not found when the read model has no item", func(t *testing.T) {
			router := newRouter(command.NewBus(), fakeReader{})

			w := send(router, "GET", "/items/missing", "")

			require.Equal(t, http.StatusNotFound, w.Code)
		})
	})

	t.Run("keep", func(t *testing.T) {
		t.Run("dispatches a keep command", func(t *testing.T) {
			handler := fake.NewHandler[itemCommand.Keep]()
			bus := command.NewBus()
			bus.Register[itemCommand.Keep](handler)
			router := newRouter(bus, fakeReader{})

			w := send(router, "POST", "/items/6f1c2a54-7b0e-4f6a-9a51-2d8f0c3e9b71/keep", "")

			require.Equal(t, http.StatusOK, w.Code)
			require.Equal(t, uuid.MustParse("6f1c2a54-7b0e-4f6a-9a51-2d8f0c3e9b71"), handler.TriggeredWithCommand().Payload.ItemID)
		})

		t.Run("rejects an invalid item id", func(t *testing.T) {
			bus := command.NewBus()
			bus.Register[itemCommand.Keep](fake.NewHandler[itemCommand.Keep]())
			router := newRouter(bus, fakeReader{})

			w := send(router, "POST", "/items/not-a-uuid/keep", "")

			require.Equal(t, http.StatusBadRequest, w.Code)
		})
	})

	t.Run("drop", func(t *testing.T) {
		t.Run("dispatches a drop command", func(t *testing.T) {
			handler := fake.NewHandler[itemCommand.Drop]()
			bus := command.NewBus()
			bus.Register[itemCommand.Drop](handler)
			router := newRouter(bus, fakeReader{})

			w := send(router, "POST", "/items/6f1c2a54-7b0e-4f6a-9a51-2d8f0c3e9b71/drop", "")

			require.Equal(t, http.StatusOK, w.Code)
			require.Equal(t, uuid.MustParse("6f1c2a54-7b0e-4f6a-9a51-2d8f0c3e9b71"), handler.TriggeredWithCommand().Payload.ItemID)
		})

		t.Run("rejects an invalid item id", func(t *testing.T) {
			bus := command.NewBus()
			bus.Register[itemCommand.Drop](fake.NewHandler[itemCommand.Drop]())
			router := newRouter(bus, fakeReader{})

			w := send(router, "POST", "/items/not-a-uuid/drop", "")

			require.Equal(t, http.StatusBadRequest, w.Code)
		})
	})
}

type fakeReader struct {
	items []items.Item
	item  *items.Item
}

func (f fakeReader) List(_ context.Context, _ string, _ ...string) ([]items.Item, error) {
	return f.items, nil
}

func (f fakeReader) ByID(context.Context, string) (*items.Item, error) {
	return f.item, nil
}

func newRouter(bus *command.Bus, reader item.ItemReader) http.Handler {
	return apiTest.NewRouterWithRoutable(item.NewHandler(bus, reader, fakeStore{}))
}

func send(router http.Handler, method, path, body string) *httptest.ResponseRecorder {
	w := httptest.NewRecorder()
	router.ServeHTTP(w, httptest.NewRequest(method, path, strings.NewReader(body)))
	return w
}

func multipartBody(t *testing.T, filename, contentType string, data []byte) (*bytes.Buffer, string) {
	t.Helper()

	var body bytes.Buffer
	writer := multipart.NewWriter(&body)
	part, err := writer.CreatePart(textproto.MIMEHeader{
		"Content-Disposition": []string{fmt.Sprintf(`form-data; name="file"; filename=%q`, filename)},
		"Content-Type":        []string{contentType},
	})
	require.NoError(t, err)
	_, err = part.Write(data)
	require.NoError(t, err)
	require.NoError(t, writer.Close())

	return &body, writer.FormDataContentType()
}

func sendMultipart(router http.Handler, body *bytes.Buffer, contentType string) *httptest.ResponseRecorder {
	w := httptest.NewRecorder()
	req := httptest.NewRequest("POST", "/items", body)
	req.Header.Set("Content-Type", contentType)
	router.ServeHTTP(w, req)
	return w
}

type fakeStore struct{}

func (fakeStore) Save(_ context.Context, _ string, _ io.Reader) (string, error) { return "key", nil }
func (fakeStore) Open(_ context.Context, _ string) (io.ReadCloser, error)       { return nil, nil }