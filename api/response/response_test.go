package response

import (
	"encoding/json"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
)

func TestOK(t *testing.T) {
	gin.SetMode(gin.TestMode)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)

	OK(c, map[string]string{"foo": "bar"})

	require.Equal(t, 200, w.Code)

	var body StandardResponse
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &body))
	require.Equal(t, 200, body.Status)
	require.Equal(t, "", body.Message)
}

func TestError(t *testing.T) {
	gin.SetMode(gin.TestMode)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)

	Error(c, 404, "no item found")

	require.Equal(t, 404, w.Code)

	var body StandardResponse
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &body))
	require.Equal(t, 404, body.Status)
	require.Nil(t, body.Data)
	require.Equal(t, "no item found", body.Message)
}
