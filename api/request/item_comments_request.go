package request

type ItemCommentCreateRequest struct {
	Text     string  `json:"text"`
	ParentID *string `json:"parent_id"`
}

type ItemCommentUpdateRequest struct {
	Text string `json:"text"`
}
