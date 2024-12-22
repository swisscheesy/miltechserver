package response

type NoItemFoundResponse struct {
	Status  int         `json:"status"`
	Data    interface{} `json:"data"`
	Message string      `json:"message"`
}

func NoItemFoundResponseMessage() NoItemFoundResponse {
	return NoItemFoundResponse{
		Status:  404,
		Data:    nil,
		Message: "no item(s) found",
	}
}

type ErrorResponse struct {
	Status  int         `json:"status"`
	Data    interface{} `json:"data"`
	Message string      `json:"message"`
}

func InternalErrorResponseMessage() ErrorResponse {
	return ErrorResponse{
		Status:  500,
		Data:    nil,
		Message: "internal Server Error",
	}
}
