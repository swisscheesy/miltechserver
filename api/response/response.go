package response

import "github.com/gin-gonic/gin"

// OK writes a 200 response with the given data in the standard envelope.
func OK(c *gin.Context, data interface{}) {
	c.JSON(200, StandardResponse{
		Status: 200,
		Data:   data,
	})
}

// Error writes an error response in the standard envelope at the given
// status code.
func Error(c *gin.Context, status int, message string) {
	c.JSON(status, StandardResponse{
		Status:  status,
		Data:    nil,
		Message: message,
	})
}
