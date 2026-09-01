package response

// NoItemFoundResponse is a type alias retained for existing call sites;
// it is structurally identical to StandardResponse and carries no
// distinct fields.
type NoItemFoundResponse = StandardResponse

func NoItemFoundResponseMessage() NoItemFoundResponse {
	return NoItemFoundResponse{
		Status:  404,
		Data:    nil,
		Message: "no item(s) found",
	}
}

// ErrorResponse is a type alias retained for existing call sites; it is
// structurally identical to StandardResponse and carries no distinct
// fields.
type ErrorResponse = StandardResponse

func InternalErrorResponseMessage() ErrorResponse {
	return ErrorResponse{
		Status:  500,
		Data:    nil,
		Message: "internal Server Error",
	}
}
