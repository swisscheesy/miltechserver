package shared

import "errors"

var (
	ErrShopNotFound      = errors.New("shop not found")
	ErrShopAccessDenied  = errors.New("access denied: not a member of this shop")
	ErrShopAdminRequired = errors.New("access denied: admin privileges required")
	ErrShopCreatorOnly   = errors.New("access denied: only shop creator can perform this action")
)

var (
	ErrMemberNotFound      = errors.New("member not found")
	ErrAlreadyMember       = errors.New("user is already a member of this shop")
	ErrCannotRemoveSelf    = errors.New("cannot remove yourself from shop")
	ErrCannotRemoveCreator = errors.New("cannot remove shop creator")
)

var (
	ErrInviteCodeInvalid = errors.New("invalid invite code")
	ErrInviteCodeExpired = errors.New("invite code has expired")
	ErrInviteCodeUsed    = errors.New("invite code has already been used")
)

var (
	ErrVehicleNotFound     = errors.New("vehicle not found")
	ErrVehicleAccessDenied = errors.New("access denied to vehicle")
)

var (
	ErrListNotFound    = errors.New("list not found")
	ErrListAccessDenied = errors.New("access denied to list")
	ErrAdminOnlyLists  = errors.New("only admins can create lists in this shop")
)

var (
	ErrNotificationNotFound = errors.New("notification not found")
)
