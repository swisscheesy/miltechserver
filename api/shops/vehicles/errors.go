package vehicles

import "errors"

var (
	ErrInvalidUsageAdjustment = errors.New("invalid usage adjustment")
	ErrUsageOutOfRange        = errors.New("usage adjustment would move tracked usage outside the supported range")
)
