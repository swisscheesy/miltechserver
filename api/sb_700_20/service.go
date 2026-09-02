package sb_700_20

import (
	"context"
	"miltechserver/.gen/miltech_ng/public/model"
)

type Service interface {
	GetAppBByLIN(ctx context.Context, lin string) ([]model.Sb70020AppB, error)
	GetAppBPaginated(ctx context.Context, page int) (PageResponse[model.Sb70020AppB], error)
	GetAppCByLIN(ctx context.Context, lin string) (model.Sb70020AppC, error)
	GetAppCPaginated(ctx context.Context, page int) (PageResponse[model.Sb70020AppC], error)
	GetAppDByLIN(ctx context.Context, lin string) ([]model.Sb70020AppD, error)
	GetAppDPaginated(ctx context.Context, page int) (PageResponse[model.Sb70020AppD], error)
	GetAppEByLIN(ctx context.Context, lin string) ([]model.Sb70020AppE, error)
	GetAppEPaginated(ctx context.Context, page int) (PageResponse[model.Sb70020AppE], error)
	GetAppFByLIN(ctx context.Context, lin string) (model.Sb70020AppF, error)
	GetAppFPaginated(ctx context.Context, page int) (PageResponse[model.Sb70020AppF], error)
	GetAppGByLIN(ctx context.Context, lin string) (model.Sb70020AppG, error)
	GetAppGPaginated(ctx context.Context, page int) (PageResponse[model.Sb70020AppG], error)
	GetAppH1ByLIN(ctx context.Context, lin string) ([]model.Sb70020AppH1, error)
	GetAppH1Paginated(ctx context.Context, page int) (PageResponse[model.Sb70020AppH1], error)
	GetAppH2ByLIN(ctx context.Context, lin string) ([]model.Sb70020AppH2, error)
	GetAppH2Paginated(ctx context.Context, page int) (PageResponse[model.Sb70020AppH2], error)
	GetAppIByLIN(ctx context.Context, lin string) (model.Sb70020AppI, error)
	GetAppIPaginated(ctx context.Context, page int) (PageResponse[model.Sb70020AppI], error)
	GetAppJByLIN(ctx context.Context, lin string) (model.Sb70020AppJ, error)
	GetAppJPaginated(ctx context.Context, page int) (PageResponse[model.Sb70020AppJ], error)
	GetChp4ByLIN(ctx context.Context, lin string) (model.Sb70020Chp4, error)
	GetChp4Paginated(ctx context.Context, page int) (PageResponse[model.Sb70020Chp4], error)
	GetChp6ByLIN(ctx context.Context, lin string) ([]model.Sb70020Chp6, error)
	GetChp6Paginated(ctx context.Context, page int) (PageResponse[model.Sb70020Chp6], error)
	GetChp8ByLIN(ctx context.Context, lin string) ([]model.Sb70020Chp8, error)
	GetChp8Paginated(ctx context.Context, page int) (PageResponse[model.Sb70020Chp8], error)
	GetAppEByNewLIN(ctx context.Context, newLin string) ([]model.Sb70020AppE, error)
	GetAppGByNewLIN(ctx context.Context, newLin string) ([]model.Sb70020AppG, error)
	GetAppH1BySubLIN(ctx context.Context, sublin string) ([]model.Sb70020AppH1, error)
	GetAppH2BySubLIN(ctx context.Context, sublin string) ([]model.Sb70020AppH2, error)
	GetChp4ByRIC(ctx context.Context, ric string) ([]model.Sb70020Chp4, error)
	GetChp6ByRIC(ctx context.Context, ric string) ([]model.Sb70020Chp6, error)
	GetChp8ByRIC(ctx context.Context, ric string) ([]model.Sb70020Chp8, error)
}
