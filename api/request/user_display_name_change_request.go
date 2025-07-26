package request

type UserDisplayNameChangeRequest struct {
	UID         string `json:"uid" binding:"required"`
	DisplayName string `json:"displayName" binding:"required"`
}