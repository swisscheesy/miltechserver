package response

type StandardResponse struct {
	Status  int         `json:"status"`
	Data    interface{} `json:"data"`
	Message string      `json:"message"`
}

type EmptyResponse struct {
	Status  int      `json:"status"`
	Message string   `json:"message"`
	Data    struct{} `json:"data"`
}

func EmptyResponseMessage() EmptyResponse {
	return EmptyResponse{
		Status:  404,
		Message: "No item found",
		Data:    struct{}{},
	}
}
