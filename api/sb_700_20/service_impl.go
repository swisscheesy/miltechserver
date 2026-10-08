package sb_700_20

import (
	"context"
	"strings"

	"miltechserver/.gen/miltech_ng/public/model"
)

type service struct {
	repository Repository
}

func NewService(repo Repository) Service {
	return &service{repository: repo}
}

func (s *service) GetAppBByLIN(ctx context.Context, lin string) ([]model.Sb70020AppB, error) {
	return s.repository.GetAppBByLIN(strings.TrimSpace(strings.ToUpper(lin)))
}
func (s *service) GetAppBPaginated(ctx context.Context, page int) (PageResponse[model.Sb70020AppB], error) {
	return s.repository.GetAppBPaginated(page)
}
func (s *service) GetAppCByLIN(ctx context.Context, lin string) (model.Sb70020AppC, error) {
	return s.repository.GetAppCByLIN(strings.TrimSpace(strings.ToUpper(lin)))
}
func (s *service) GetAppCPaginated(ctx context.Context, page int) (PageResponse[model.Sb70020AppC], error) {
	return s.repository.GetAppCPaginated(page)
}
func (s *service) GetAppDByLIN(ctx context.Context, lin string) ([]model.Sb70020AppD, error) {
	return s.repository.GetAppDByLIN(strings.TrimSpace(strings.ToUpper(lin)))
}
func (s *service) GetAppDPaginated(ctx context.Context, page int) (PageResponse[model.Sb70020AppD], error) {
	return s.repository.GetAppDPaginated(page)
}
func (s *service) GetAppEByLIN(ctx context.Context, lin string) ([]model.Sb70020AppE, error) {
	return s.repository.GetAppEByLIN(strings.TrimSpace(strings.ToUpper(lin)))
}
func (s *service) GetAppEPaginated(ctx context.Context, page int) (PageResponse[model.Sb70020AppE], error) {
	return s.repository.GetAppEPaginated(page)
}
func (s *service) GetAppFByLIN(ctx context.Context, lin string) (model.Sb70020AppF, error) {
	return s.repository.GetAppFByLIN(strings.TrimSpace(strings.ToUpper(lin)))
}
func (s *service) GetAppFPaginated(ctx context.Context, page int) (PageResponse[model.Sb70020AppF], error) {
	return s.repository.GetAppFPaginated(page)
}
func (s *service) GetAppGByLIN(ctx context.Context, lin string) (model.Sb70020AppG, error) {
	return s.repository.GetAppGByLIN(strings.TrimSpace(strings.ToUpper(lin)))
}
func (s *service) GetAppGPaginated(ctx context.Context, page int) (PageResponse[model.Sb70020AppG], error) {
	return s.repository.GetAppGPaginated(page)
}
func (s *service) GetAppH1ByLIN(ctx context.Context, lin string) ([]model.Sb70020AppH1, error) {
	return s.repository.GetAppH1ByLIN(strings.TrimSpace(strings.ToUpper(lin)))
}
func (s *service) GetAppH1Paginated(ctx context.Context, page int) (PageResponse[model.Sb70020AppH1], error) {
	return s.repository.GetAppH1Paginated(page)
}
func (s *service) GetAppH2ByLIN(ctx context.Context, lin string) ([]model.Sb70020AppH2, error) {
	return s.repository.GetAppH2ByLIN(strings.TrimSpace(strings.ToUpper(lin)))
}
func (s *service) GetAppH2Paginated(ctx context.Context, page int) (PageResponse[model.Sb70020AppH2], error) {
	return s.repository.GetAppH2Paginated(page)
}
func (s *service) GetAppIByLIN(ctx context.Context, lin string) (model.Sb70020AppI, error) {
	return s.repository.GetAppIByLIN(strings.TrimSpace(strings.ToUpper(lin)))
}
func (s *service) GetAppIPaginated(ctx context.Context, page int) (PageResponse[model.Sb70020AppI], error) {
	return s.repository.GetAppIPaginated(page)
}
func (s *service) GetAppJByLIN(ctx context.Context, lin string) (model.Sb70020AppJ, error) {
	return s.repository.GetAppJByLIN(strings.TrimSpace(strings.ToUpper(lin)))
}
func (s *service) GetAppJPaginated(ctx context.Context, page int) (PageResponse[model.Sb70020AppJ], error) {
	return s.repository.GetAppJPaginated(page)
}
func (s *service) GetChp4ByLIN(ctx context.Context, lin string) (model.Sb70020Chp4, error) {
	return s.repository.GetChp4ByLIN(strings.TrimSpace(strings.ToUpper(lin)))
}
func (s *service) GetChp4Paginated(ctx context.Context, page int) (PageResponse[model.Sb70020Chp4], error) {
	return s.repository.GetChp4Paginated(page)
}
func (s *service) GetChp6ByLIN(ctx context.Context, lin string) ([]model.Sb70020Chp6, error) {
	return s.repository.GetChp6ByLIN(strings.TrimSpace(strings.ToUpper(lin)))
}
func (s *service) GetChp6Paginated(ctx context.Context, page int) (PageResponse[model.Sb70020Chp6], error) {
	return s.repository.GetChp6Paginated(page)
}
func (s *service) GetChp8ByLIN(ctx context.Context, lin string) ([]model.Sb70020Chp8, error) {
	return s.repository.GetChp8ByLIN(strings.TrimSpace(strings.ToUpper(lin)))
}
func (s *service) GetChp8Paginated(ctx context.Context, page int) (PageResponse[model.Sb70020Chp8], error) {
	return s.repository.GetChp8Paginated(page)
}
func (s *service) GetAppEByNewLIN(ctx context.Context, newLin string) ([]model.Sb70020AppE, error) {
	return s.repository.GetAppEByNewLIN(strings.TrimSpace(strings.ToUpper(newLin)))
}
func (s *service) GetAppGByNewLIN(ctx context.Context, newLin string) ([]model.Sb70020AppG, error) {
	return s.repository.GetAppGByNewLIN(strings.TrimSpace(strings.ToUpper(newLin)))
}
func (s *service) GetAppH1BySubLIN(ctx context.Context, sublin string) ([]model.Sb70020AppH1, error) {
	return s.repository.GetAppH1BySubLIN(strings.TrimSpace(strings.ToUpper(sublin)))
}
func (s *service) GetAppH2BySubLIN(ctx context.Context, sublin string) ([]model.Sb70020AppH2, error) {
	return s.repository.GetAppH2BySubLIN(strings.TrimSpace(strings.ToUpper(sublin)))
}
func (s *service) GetChp4ByRIC(ctx context.Context, ric string) ([]model.Sb70020Chp4, error) {
	return s.repository.GetChp4ByRIC(strings.TrimSpace(strings.ToUpper(ric)))
}
func (s *service) GetChp6ByRIC(ctx context.Context, ric string) ([]model.Sb70020Chp6, error) {
	return s.repository.GetChp6ByRIC(strings.TrimSpace(strings.ToUpper(ric)))
}
func (s *service) GetChp8ByRIC(ctx context.Context, ric string) ([]model.Sb70020Chp8, error) {
	return s.repository.GetChp8ByRIC(strings.TrimSpace(strings.ToUpper(ric)))
}
