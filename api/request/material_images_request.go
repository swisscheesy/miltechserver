package request

type UploadImageRequest struct {
	NIIN string `form:"niin" binding:"required,len=9"`
}

type VoteImageRequest struct {
	VoteType string `json:"vote_type" binding:"required,oneof=upvote downvote"`
}

type FlagImageRequest struct {
	//Reason      string `json:"reason" binding:"required,oneof=incorrect_item inappropriate poor_quality duplicate other"`
	Reason      string `json:"reason" binding:"required"`
	Description string `json:"description" binding:"max=500"`
}

type GetImagesRequest struct {
	Page     int `form:"page,default=1" binding:"min=1"`
	PageSize int `form:"page_size,default=20" binding:"min=1,max=100"`
}
