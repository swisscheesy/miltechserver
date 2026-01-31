package user_general

type DeleteRequest struct {
	UID string `json:"uid" binding:"required"`
}

type DisplayNameChangeRequest struct {
	UID         string `json:"uid" binding:"required"`
	DisplayName string `json:"displayName" binding:"required"`
}
