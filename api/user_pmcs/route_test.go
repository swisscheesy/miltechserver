package user_pmcs

import (
	"compress/gzip"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
)

func TestGzipGETResponsesPreservesRepresentationHeaders(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.Use(gzipGETResponses())
	router.GET("/tree", func(context *gin.Context) {
		context.Header("ETag", `"tree-version"`)
		context.Header("Cache-Control", "private, no-cache")
		context.JSON(http.StatusOK, gin.H{"data": strings.Repeat("compressible", 50)})
	})

	request := httptest.NewRequest(http.MethodGet, "/tree", nil)
	request.Header.Set("Accept-Encoding", "gzip")
	recorder := httptest.NewRecorder()
	router.ServeHTTP(recorder, request)

	require.Equal(t, http.StatusOK, recorder.Code)
	require.Equal(t, "gzip", recorder.Header().Get("Content-Encoding"))
	require.Equal(t, `"tree-version"`, recorder.Header().Get("ETag"))
	require.Equal(t, "private, no-cache", recorder.Header().Get("Cache-Control"))
	reader, err := gzip.NewReader(recorder.Body)
	require.NoError(t, err)
	body, err := io.ReadAll(reader)
	require.NoError(t, err)
	require.NoError(t, reader.Close())
	require.Contains(t, string(body), "compressible")
}

func TestGzipGETResponsesPreservesConditional304(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.Use(gzipGETResponses())
	router.GET("/tree", func(context *gin.Context) {
		context.Header("ETag", `"tree-version"`)
		context.Status(http.StatusNotModified)
	})

	request := httptest.NewRequest(http.MethodGet, "/tree", nil)
	request.Header.Set("Accept-Encoding", "gzip")
	recorder := httptest.NewRecorder()
	router.ServeHTTP(recorder, request)

	require.Equal(t, http.StatusNotModified, recorder.Code)
	require.Equal(t, `"tree-version"`, recorder.Header().Get("ETag"))
	require.Empty(t, recorder.Body.Bytes())
}

func TestGzipGETResponsesDoesNotCompressMutations(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.Use(gzipGETResponses())
	router.PUT("/tree", func(context *gin.Context) {
		context.JSON(http.StatusOK, gin.H{"data": strings.Repeat("mutation", 50)})
	})

	request := httptest.NewRequest(http.MethodPut, "/tree", nil)
	request.Header.Set("Accept-Encoding", "gzip")
	recorder := httptest.NewRecorder()
	router.ServeHTTP(recorder, request)

	require.Equal(t, http.StatusOK, recorder.Code)
	require.Empty(t, recorder.Header().Get("Content-Encoding"))
	require.Contains(t, recorder.Body.String(), "mutation")
}
