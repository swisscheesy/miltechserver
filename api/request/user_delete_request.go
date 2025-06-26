package request

type UserDeleteRequest struct {
	UID string `json:"uid" binding:"required"`
}
