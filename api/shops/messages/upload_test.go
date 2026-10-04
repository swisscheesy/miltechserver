package messages

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	"io"
	"miltechserver/bootstrap"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
)

type uploadHandlerService struct {
	Service
	calls int
	size  int
}

func (s *uploadHandlerService) UploadMessageImage(_ context.Context, _ *bootstrap.User, _ string, data []byte, _ string) (ImageUpload, error) {
	s.size = len(data)
	s.calls++
	return ImageUpload{MessageID: uuid.NewString(), FileExtension: ".jpg", ImageURL: "https://owned/image.jpg"}, nil
}

type countedUploadBody struct {
	io.Reader
	consumed int64
}

func (b *countedUploadBody) Read(p []byte) (int, error) {
	n, e := b.Reader.Read(p)
	b.consumed += int64(n)
	return n, e
}
func (*countedUploadBody) Close() error { return nil }
func uploadRequest(t *testing.T, files int, size int, fieldSize int) (*http.Request, *countedUploadBody) {
	t.Helper()
	var body bytes.Buffer
	w := multipart.NewWriter(&body)
	require.NoError(t, w.WriteField("shop_id", uuid.NewString()))
	if fieldSize > 0 {
		require.NoError(t, w.WriteField("extra", string(bytes.Repeat([]byte("x"), fieldSize))))
	}
	for i := 0; i < files; i++ {
		f, e := w.CreateFormFile("file", "image.jpg")
		require.NoError(t, e)
		_, e = f.Write(bytes.Repeat([]byte("x"), size))
		require.NoError(t, e)
	}
	require.NoError(t, w.Close())
	counted := &countedUploadBody{Reader: bytes.NewReader(body.Bytes())}
	req := httptest.NewRequest(http.MethodPost, "/upload", counted)
	req.Header.Set("Content-Type", w.FormDataContentType())
	return req, counted
}
func runUploadHandler(s Service, req *http.Request) *httptest.ResponseRecorder {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.POST("/upload", func(c *gin.Context) {
		c.Set("user", &bootstrap.User{UserID: "actor"})
		(&Handler{service: s}).UploadMessageImage(c)
	})
	out := httptest.NewRecorder()
	r.ServeHTTP(out, req)
	return out
}
func TestUploadBodyBoundBeforeParse(t *testing.T) {
	for _, tc := range []struct {
		name               string
		files, size, field int
	}{{"total", 1, 7 * 1024 * 1024, 0}, {"file", 1, 5*1024*1024 + 1, 0}, {"fields", 1, 16, 2 * 1024 * 1024}, {"parts", 2, 16, 0}} {
		t.Run(tc.name, func(t *testing.T) {
			dir := t.TempDir()
			t.Setenv("TMPDIR", dir)
			s := &uploadHandlerService{}
			req, body := uploadRequest(t, tc.files, tc.size, tc.field)
			out := runUploadHandler(s, req)
			require.LessOrEqual(t, body.consumed, int64(6*1024*1024+1))
			require.GreaterOrEqual(t, out.Code, 400)
			require.Zero(t, s.calls)
			files, e := os.ReadDir(dir)
			require.NoError(t, e)
			require.Empty(t, files)
		})
	}
}
func TestUploadLegacyResponseKeys(t *testing.T) {
	s := &uploadHandlerService{}
	req, _ := uploadRequest(t, 1, 16, 0)
	out := runUploadHandler(s, req)
	require.Equal(t, 200, out.Code)
	var envelope struct {
		Data map[string]any `json:"data"`
	}
	require.NoError(t, json.Unmarshal(out.Body.Bytes(), &envelope))
	keys := []string{}
	for k := range envelope.Data {
		keys = append(keys, k)
	}
	require.ElementsMatch(t, []string{"message_id", "shop_id", "image_url", "file_extension"}, keys)
}

type brokenUploadReader struct {
	data []byte
	err  error
}

func (r *brokenUploadReader) Read(p []byte) (int, error) {
	if len(r.data) == 0 {
		return 0, r.err
	}
	n := copy(p, r.data)
	r.data = r.data[n:]
	return n, nil
}
func TestUploadCanceledOrTruncatedRead(t *testing.T) {
	for _, name := range []string{"canceled", "truncated", "reader_error"} {
		t.Run(name, func(t *testing.T) {
			dir := t.TempDir()
			t.Setenv("TMPDIR", dir)
			req, _ := uploadRequest(t, 1, 2*1024*1024, 0)
			if name == "canceled" {
				ctx, cancel := context.WithCancel(req.Context())
				cancel()
				req = req.WithContext(ctx)
			} else {
				raw, e := io.ReadAll(req.Body)
				require.NoError(t, e)
				readErr := io.EOF
				if name == "reader_error" {
					readErr = errors.New("read interrupted")
				}
				req.Body = io.NopCloser(&brokenUploadReader{data: raw[:len(raw)-100], err: readErr})
			}
			s := &uploadHandlerService{}
			out := runUploadHandler(s, req)
			require.GreaterOrEqual(t, out.Code, 400)
			require.Zero(t, s.calls)
			files, e := os.ReadDir(dir)
			require.NoError(t, e)
			require.Empty(t, files)
		})
	}
}
func TestUploadRejectsInvalidShopBeforeService(t *testing.T) {
	for _, shop := range []string{"bad", "00000000-0000-0000-0000-000000000000"} {
		req, _ := uploadRequest(t, 1, 16, 0)
		raw, e := io.ReadAll(req.Body)
		require.NoError(t, e)
		// Replace the field value, preserving the multipart structure.
		start := bytes.Index(raw, []byte("name=\"shop_id\"\r\n\r\n")) + len("name=\"shop_id\"\r\n\r\n")
		raw = append(append(append([]byte{}, raw[:start]...), []byte(shop)...), raw[start+36:]...)
		req.Body = io.NopCloser(bytes.NewReader(raw))
		s := &uploadHandlerService{}
		out := runUploadHandler(s, req)
		require.Equal(t, 400, out.Code)
		require.Zero(t, s.calls)
	}
}

func TestUploadFileLimitAndTemporaryCleanup(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("TMPDIR", dir)
	req, body := uploadRequest(t, 1, 5*1024*1024, 0)
	s := &uploadHandlerService{}
	out := runUploadHandler(s, req)
	require.Equal(t, 200, out.Code)
	require.Equal(t, 5*1024*1024, s.size)
	require.Equal(t, 1, s.calls)
	require.LessOrEqual(t, body.consumed, int64(6*1024*1024+1))
	files, e := os.ReadDir(dir)
	require.NoError(t, e)
	require.Empty(t, files)
}
func TestUploadMissingStoreFailsWithoutRepository(t *testing.T) {
	result, e := NewService(nil, nil).UploadMessageImage(context.Background(), &bootstrap.User{UserID: "actor"}, uuid.NewString(), []byte("image"), "image/jpeg")
	require.Error(t, e)
	require.Empty(t, result)
}
