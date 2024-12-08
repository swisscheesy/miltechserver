package service

import "context"

type ItemLookupService interface {
	LookupLINByPage(ctx context.Context, page int) (string, error)
	//LookupSpecificLIN(ctx context.Context, lin string) (string, error)
	//
	//LookupUOCByPage(ctx context.Context, page int) (string, error)
	//LookupSpecificUOC(ctx context.Context, uoc string) (string, error)
}
