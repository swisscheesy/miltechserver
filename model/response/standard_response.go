package response

type StandardResponse struct {
	Code    int         "json:status"
	Data    interface{} "json:data"
	Message string      "json:message"
}
