package user_saves_test

import (
	"bytes"
	"io"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"testing"

	"miltechserver/api/user_saves/images"
	"miltechserver/bootstrap"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
)

type imagesServiceStub struct {
	lastItemID   string
	lastTable    string
	lastImage    []byte
	returnURL    string
	returnData   []byte
	returnType   string
	deleteCalled bool
}

func (service *imagesServiceStub) Upload(user *bootstrap.User, itemID string, tableType string, imageData []byte) (string, error) {
	service.lastItemID = itemID
	service.lastTable = tableType
	service.lastImage = imageData
	return service.returnURL, nil
}

func (service *imagesServiceStub) Delete(user *bootstrap.User, itemID string, tableType string) error {
	service.deleteCalled = true
	service.lastItemID = itemID
	service.lastTable = tableType
	return nil
}

func (service *imagesServiceStub) Get(user *bootstrap.User, itemID string, tableType string) ([]byte, string, error) {
	service.lastItemID = itemID
	service.lastTable = tableType
	return service.returnData, service.returnType, nil
}

func TestImagesHandlersUploadAndGet(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.Use(testUserMiddleware())

	service := &imagesServiceStub{
		returnURL:  "https://example.com/img.jpg",
		returnData: []byte("img"),
		returnType: "image/jpeg",
	}
	group := router.Group("/api/v1/auth")
	images.RegisterRoutes(group, service)

	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)
	part, err := writer.CreateFormFile("file", "test.jpg")
	require.NoError(t, err)
	_, err = io.Copy(part, bytes.NewBuffer([]byte("payload")))
	require.NoError(t, err)
	require.NoError(t, writer.Close())

	req, err := http.NewRequest(http.MethodPost, "/api/v1/auth/user/saves/items/image/upload/quick?item_id=123", body)
	require.NoError(t, err)
	req.Header.Set("Content-Type", writer.FormDataContentType())
	req.Header.Set("X-User-ID", "user")

	resp := httptest.NewRecorder()
	router.ServeHTTP(resp, req)
	require.Equal(t, http.StatusOK, resp.Code)
	require.Equal(t, "123", service.lastItemID)
	require.Equal(t, "quick", service.lastTable)

	getReq, err := http.NewRequest(http.MethodGet, "/api/v1/auth/user/saves/items/image/quick?item_id=123", nil)
	require.NoError(t, err)
	getReq.Header.Set("X-User-ID", "user")

	getResp := httptest.NewRecorder()
	router.ServeHTTP(getResp, getReq)
	require.Equal(t, http.StatusOK, getResp.Code)
	require.Equal(t, "image/jpeg", getResp.Header().Get("Content-Type"))
}

func TestImagesHandlersDeleteUnauthorized(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.Use(testUserMiddleware())

	service := &imagesServiceStub{}
	group := router.Group("/api/v1/auth")
	images.RegisterRoutes(group, service)

	req, err := http.NewRequest(http.MethodDelete, "/api/v1/auth/user/saves/items/image/quick?item_id=123", nil)
	require.NoError(t, err)

	resp := httptest.NewRecorder()
	router.ServeHTTP(resp, req)
	require.Equal(t, http.StatusUnauthorized, resp.Code)
}
