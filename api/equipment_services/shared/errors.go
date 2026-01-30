package shared

import "errors"

var (
	ErrUnauthorizedUser     = errors.New("unauthorized user")
	ErrServiceHoursNegative = errors.New("service_hours must be non-negative")
	ErrAccessDenied         = errors.New("access denied: user is not a member of this shop")
	ErrModifyDenied         = errors.New("access denied: only service creators or shop admins can modify services")
	ErrDeleteDenied         = errors.New("access denied: only service creators or shop admins can delete services")
	ErrShopMismatch         = errors.New("equipment and list must belong to the same shop")
	ErrEquipmentNotFound    = errors.New("equipment not found or access denied")
	ErrListNotFound         = errors.New("list not found or access denied")
	ErrServiceNotFound      = errors.New("service not found")
)
