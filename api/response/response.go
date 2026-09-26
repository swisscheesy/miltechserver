package response

import "github.com/gin-gonic/gin"

// OK writes a 200 response with the given data in the standard envelope.
func OK(c *gin.Context, data interface{}) {
	c.JSON(200, StandardResponse{
		Status: 200,
		Data:   data,
	})
}

const ErrorWriterKey = "response.scopedErrorWriter"

// Error writes an error response in the standard envelope at the given
// status code.
func Error(c *gin.Context, status int, message string) {
	if writer, ok := c.Get(ErrorWriterKey); ok {
		writer.(func(int, string))(status, message)
		return
	}
	c.JSON(status, StandardResponse{
		Status:  status,
		Data:    nil,
		Message: message,
	})
}
