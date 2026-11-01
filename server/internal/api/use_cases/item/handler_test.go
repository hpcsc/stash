//go:build unit

package item_test

import (
	"bytes"
	"context"
	"errors"
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
	"github.com/hpcsc/stash/internal/api/middleware"
	"github.com/hpcsc/stash/internal/api/response"
	"github.com/hpcsc/stash/internal/api/session"
	apiTest "github.com/hpcsc/stash/internal/api/test"
	"github.com/hpcsc/stash/internal/api/use_cases/item"
	commonTest "github.com/hpcsc/stash/internal/common/test"
	"github.com/hpcsc/stash/internal/domain"
	"github.com/hpcsc/stash/internal/domain/command"
	"github.com/hpcsc/stash/internal/domain/command/fake"
	"github.com/hpcsc/stash/internal/storage"
	itemCommand "github.com/hpcsc/stash/internal/use_cases/item/command"
	"github.com/hpcsc/stash/internal/use_cases/item/items"
	trashCommand "github.com/hpcsc/stash/internal/use_cases/trash/command"
	"github.com/stretchr/testify/require"
)

var testUser = domain.UserID("test-user")

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

		t.Run("marks a captured link as a duplicate when its URL is already saved", func(t *testing.T) {
			handler := fake.NewHandler[itemCommand.CaptureLink]()
			bus := command.NewBus()
			bus.Register[itemCommand.CaptureLink](handler)
			router := newRouter(bus, fakeReader{existing: &items.Item{ID: "existing", State: "kept"}})

			w := send(router, "POST", "/items", `{"body": "https://example.com", "type": "link"}`)

			require.Equal(t, http.StatusOK, w.Code)
			body := commonTest.UnmarshalJson[response.Response](t, w.Body.Bytes())
			data := body.Data.(map[string]any)
			require.Equal(t, true, data["duplicate"])
			require.Equal(t, "kept", data["duplicateState"])
		})

		t.Run("marks captured text as a duplicate when its body is already saved", func(t *testing.T) {
			handler := fake.NewHandler[itemCommand.CaptureText]()
			bus := command.NewBus()
			bus.Register[itemCommand.CaptureText](handler)
			router := newRouter(bus, fakeReader{existing: &items.Item{ID: "existing", State: "inbox"}})

			w := send(router, "POST", "/items", `{"body": "remember this"}`)

			require.Equal(t, http.StatusOK, w.Code)
			body := commonTest.UnmarshalJson[response.Response](t, w.Body.Bytes())
			data := body.Data.(map[string]any)
			require.Equal(t, true, data["duplicate"])
			require.Equal(t, "inbox", data["duplicateState"])
		})

		t.Run("does not mark a new capture as a duplicate", func(t *testing.T) {
			handler := fake.NewHandler[itemCommand.CaptureText]()
			bus := command.NewBus()
			bus.Register[itemCommand.CaptureText](handler)
			router := newRouter(bus, fakeReader{})

			w := send(router, "POST", "/items", `{"body": "brand new"}`)

			require.Equal(t, http.StatusOK, w.Code)
			body := commonTest.UnmarshalJson[response.Response](t, w.Body.Bytes())
			data := body.Data.(map[string]any)
			require.Equal(t, false, data["duplicate"])
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
			require.True(t, strings.HasPrefix(payload.FileRef, "user/test-user/images/"))
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
			require.True(t, strings.HasPrefix(payload.FileRef, "user/test-user/files/"))
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

		t.Run("dispatches a capture note command with text and images in order", func(t *testing.T) {
			handler := fake.NewHandler[itemCommand.CaptureNote]()
			bus := command.NewBus()
			bus.Register[itemCommand.CaptureNote](handler)
			router := newRouter(bus, fakeReader{})

			body, contentType := noteMultipartBody(t,
				`{"blocks":[{"type":"text","text":"before"},{"type":"image","index":0},{"type":"text","text":"after"}]}`,
				imagePart{filename: "photo.png", contentType: "image/png", data: []byte("png-bytes")},
			)
			w := sendMultipart(router, body, contentType)

			require.Equal(t, http.StatusOK, w.Code)
			payload := handler.TriggeredWithCommand().Payload
			require.Len(t, payload.Blocks, 3)
			require.Equal(t, "before", payload.Blocks[0].Text)
			require.True(t, strings.HasPrefix(payload.Blocks[1].Image.FileRef, "user/test-user/images/"))
			require.True(t, strings.HasSuffix(payload.Blocks[1].Image.FileRef, ".png"))
			require.Equal(t, "photo.png", payload.Blocks[1].Image.FileName)
			require.Equal(t, "image/png", payload.Blocks[1].Image.MimeType)
			require.Equal(t, int64(len("png-bytes")), payload.Blocks[1].Image.FileSize)
			require.Equal(t, "after", payload.Blocks[2].Text)

			responseBody := commonTest.UnmarshalJson[response.Response](t, w.Body.Bytes())
			data := responseBody.Data.(map[string]any)
			require.Equal(t, payload.ItemID.String(), data["id"])
			require.Equal(t, "note", data["type"])
			require.Equal(t, "before\nafter", data["body"])
			require.Len(t, data["images"], 1)
			require.Len(t, data["blocks"], 3)
		})

		t.Run("captures a note with images only", func(t *testing.T) {
			handler := fake.NewHandler[itemCommand.CaptureNote]()
			bus := command.NewBus()
			bus.Register[itemCommand.CaptureNote](handler)
			router := newRouter(bus, fakeReader{})

			body, contentType := noteMultipartBody(t,
				`{"blocks":[{"type":"image","index":0}]}`,
				imagePart{filename: "photo.png", contentType: "image/png", data: []byte("png-bytes")},
			)
			w := sendMultipart(router, body, contentType)

			require.Equal(t, http.StatusOK, w.Code)
			require.Len(t, handler.TriggeredWithCommand().Payload.Blocks, 1)
		})

		t.Run("rejects a note without a document", func(t *testing.T) {
			router := newRouter(command.NewBus(), fakeReader{})

			body, contentType := noteMultipartBody(t, "",
				imagePart{filename: "photo.png", contentType: "image/png", data: []byte("png-bytes")},
			)
			w := sendMultipart(router, body, contentType)

			require.Equal(t, http.StatusBadRequest, w.Code)
			require.Contains(t, w.Body.String(), "missing note document")
		})

		t.Run("rejects an image that the note does not reference", func(t *testing.T) {
			router := newRouter(command.NewBus(), fakeReader{})

			body, contentType := noteMultipartBody(t,
				`{"blocks":[{"type":"text","text":"hello"}]}`,
				imagePart{filename: "photo.png", contentType: "image/png", data: []byte("png-bytes")},
			)
			w := sendMultipart(router, body, contentType)

			require.Equal(t, http.StatusBadRequest, w.Code)
			require.Contains(t, w.Body.String(), "referenced exactly once")
		})

		t.Run("rejects an image referenced twice", func(t *testing.T) {
			router := newRouter(command.NewBus(), fakeReader{})

			body, contentType := noteMultipartBody(t,
				`{"blocks":[{"type":"image","index":0},{"type":"image","index":0}]}`,
				imagePart{filename: "photo.png", contentType: "image/png", data: []byte("png-bytes")},
			)
			w := sendMultipart(router, body, contentType)

			require.Equal(t, http.StatusBadRequest, w.Code)
			require.Contains(t, w.Body.String(), "referenced more than once")
		})

		t.Run("rejects a non-image note upload", func(t *testing.T) {
			router := newRouter(command.NewBus(), fakeReader{})

			body, contentType := noteMultipartBody(t,
				`{"blocks":[{"type":"image","index":0}]}`,
				imagePart{filename: "report.pdf", contentType: "application/pdf", data: []byte("pdf-bytes")},
			)
			w := sendMultipart(router, body, contentType)

			require.Equal(t, http.StatusBadRequest, w.Code)
			require.Contains(t, w.Body.String(), "only images can be added to a note")
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

	t.Run("restore", func(t *testing.T) {
		t.Run("dispatches a restore command", func(t *testing.T) {
			handler := fake.NewHandler[itemCommand.Restore]()
			bus := command.NewBus()
			bus.Register[itemCommand.Restore](handler)
			router := newRouter(bus, fakeReader{})

			w := send(router, "POST", "/items/6f1c2a54-7b0e-4f6a-9a51-2d8f0c3e9b71/restore", "")

			require.Equal(t, http.StatusOK, w.Code)
			require.Equal(t, uuid.MustParse("6f1c2a54-7b0e-4f6a-9a51-2d8f0c3e9b71"), handler.TriggeredWithCommand().Payload.ItemID)
		})

		t.Run("rejects an invalid item id", func(t *testing.T) {
			bus := command.NewBus()
			bus.Register[itemCommand.Restore](fake.NewHandler[itemCommand.Restore]())
			router := newRouter(bus, fakeReader{})

			w := send(router, "POST", "/items/not-a-uuid/restore", "")

			require.Equal(t, http.StatusBadRequest, w.Code)
		})
	})

	t.Run("delete", func(t *testing.T) {
		t.Run("dispatches a delete command", func(t *testing.T) {
			handler := fake.NewHandler[itemCommand.Delete]()
			bus := command.NewBus()
			bus.Register[itemCommand.Delete](handler)
			router := newRouter(bus, fakeReader{})

			w := send(router, "POST", "/items/6f1c2a54-7b0e-4f6a-9a51-2d8f0c3e9b71/delete", "")

			require.Equal(t, http.StatusOK, w.Code)
			require.Equal(t, uuid.MustParse("6f1c2a54-7b0e-4f6a-9a51-2d8f0c3e9b71"), handler.TriggeredWithCommand().Payload.ItemID)
		})

		t.Run("rejects an invalid item id", func(t *testing.T) {
			bus := command.NewBus()
			bus.Register[itemCommand.Delete](fake.NewHandler[itemCommand.Delete]())
			router := newRouter(bus, fakeReader{})

			w := send(router, "POST", "/items/not-a-uuid/delete", "")

			require.Equal(t, http.StatusBadRequest, w.Code)
		})
	})

	t.Run("empty trash", func(t *testing.T) {
		t.Run("dispatches an empty trash command with the dropped items", func(t *testing.T) {
			handler := fake.NewHandler[trashCommand.EmptyTrash]()
			bus := command.NewBus()
			bus.Register[trashCommand.EmptyTrash](handler)
			router := newRouter(bus, fakeReader{items: []items.Item{
				{ID: "6f1c2a54-7b0e-4f6a-9a51-2d8f0c3e9b71"},
				{ID: "c2d5a1e8-3f47-4b90-8e61-7a4c9d2b5f03"},
			}})

			w := send(router, "POST", "/trash/empty", "")

			require.Equal(t, http.StatusOK, w.Code)
			require.Equal(t, []uuid.UUID{
				uuid.MustParse("6f1c2a54-7b0e-4f6a-9a51-2d8f0c3e9b71"),
				uuid.MustParse("c2d5a1e8-3f47-4b90-8e61-7a4c9d2b5f03"),
			}, handler.TriggeredWithCommand().Payload.ItemIDs)
		})

		t.Run("rejects an empty trash", func(t *testing.T) {
			bus := command.NewBus()
			bus.Register[trashCommand.EmptyTrash](fake.NewHandler[trashCommand.EmptyTrash]())
			router := newRouter(bus, fakeReader{})

			w := send(router, "POST", "/trash/empty", "")

			require.Equal(t, http.StatusBadRequest, w.Code)
			require.Contains(t, w.Body.String(), "trash is empty")
		})
	})

	t.Run("serve file", func(t *testing.T) {
		store := fakeStore{files: map[string]string{
			"user/test-user/files/report.pdf":  "pdf-bytes",
			"user/other-user/files/report.pdf": "someone else's bytes",
		}}
		router := newRouterWithStore(command.NewBus(), fakeReader{}, store)
		issuer := session.New([]byte("test-secret"), time.Hour)
		token, err := issuer.Issue(string(testUser))
		require.NoError(t, err)

		t.Run("rejects a request without a token", func(t *testing.T) {
			w := sendFile(router, "/files/user/test-user/files/report.pdf")

			require.Equal(t, http.StatusUnauthorized, w.Code)
		})

		t.Run("rejects another user's file", func(t *testing.T) {
			w := sendFile(router, "/files/user/other-user/files/report.pdf?token="+token)

			require.Equal(t, http.StatusNotFound, w.Code)
			require.NotContains(t, w.Body.String(), "someone else's bytes")
		})

		t.Run("serves the signed-in user's file", func(t *testing.T) {
			w := sendFile(router, "/files/user/test-user/files/report.pdf?token="+token)

			require.Equal(t, http.StatusOK, w.Code)
			require.Equal(t, "pdf-bytes", w.Body.String())
		})
	})
}

type fakeReader struct {
	items    []items.Item
	item     *items.Item
	existing *items.Item
}

func (f fakeReader) For(domain.UserID) items.View {
	return f
}

func (f fakeReader) List(_ context.Context, _ string, _ ...string) ([]items.Item, error) {
	return f.items, nil
}

func (f fakeReader) ByID(context.Context, string) (*items.Item, error) {
	return f.item, nil
}

func (f fakeReader) Existing(_ context.Context, _ string, _ string) (*items.Item, error) {
	return f.existing, nil
}

func newRouter(bus *command.Bus, reader item.ItemReader) http.Handler {
	return newRouterWithStore(bus, reader, fakeStore{})
}

func newRouterWithStore(bus *command.Bus, reader item.ItemReader, store storage.Storage) http.Handler {
	issuer := session.New([]byte("test-secret"), time.Hour)
	return apiTest.NewRouterWithRoutable(item.NewHandler(bus, reader, store, issuer))
}

func send(router http.Handler, method, path, body string) *httptest.ResponseRecorder {
	w := httptest.NewRecorder()
	req := httptest.NewRequest(method, path, strings.NewReader(body))
	req = req.WithContext(middleware.WithUser(req.Context(), testUser))
	router.ServeHTTP(w, req)
	return w
}

func sendFile(router http.Handler, path string) *httptest.ResponseRecorder {
	w := httptest.NewRecorder()
	router.ServeHTTP(w, httptest.NewRequest("GET", path, nil))
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

type imagePart struct {
	filename    string
	contentType string
	data        []byte
}

func noteMultipartBody(t *testing.T, document string, images ...imagePart) (*bytes.Buffer, string) {
	t.Helper()

	var body bytes.Buffer
	writer := multipart.NewWriter(&body)
	if document != "" {
		require.NoError(t, writer.WriteField("document", document))
	}
	for _, image := range images {
		part, err := writer.CreatePart(textproto.MIMEHeader{
			"Content-Disposition": []string{fmt.Sprintf(`form-data; name="images"; filename=%q`, image.filename)},
			"Content-Type":        []string{image.contentType},
		})
		require.NoError(t, err)
		_, err = part.Write(image.data)
		require.NoError(t, err)
	}
	require.NoError(t, writer.Close())

	return &body, writer.FormDataContentType()
}

func sendMultipart(router http.Handler, body *bytes.Buffer, contentType string) *httptest.ResponseRecorder {
	w := httptest.NewRecorder()
	req := httptest.NewRequest("POST", "/items", body)
	req.Header.Set("Content-Type", contentType)
	req = req.WithContext(middleware.WithUser(req.Context(), testUser))
	router.ServeHTTP(w, req)
	return w
}

type fakeStore struct {
	files map[string]string
}

func (fakeStore) Save(_ context.Context, _ string, _ io.Reader) (string, error) { return "key", nil }

func (f fakeStore) Open(_ context.Context, key string) (io.ReadCloser, error) {
	content, ok := f.files[key]
	if !ok {
		return nil, errors.New("file not found")
	}
	return io.NopCloser(strings.NewReader(content)), nil
}
