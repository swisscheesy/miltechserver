package sb_700_20

import (
	"strings"

	"miltechserver/.gen/miltech_ng/public/model"
)

type service struct {
	repository Repository
}

func NewService(repo Repository) Service {
	return &service{repository: repo}
}

func (s *service) GetAppBByLIN(lin string) ([]model.Sb70020AppB, error) {
	return s.repository.GetAppBByLIN(strings.TrimSpace(strings.ToUpper(lin)))
}
func (s *service) GetAppBPaginated(page int) (PageResponse[model.Sb70020AppB], error) {
	return s.repository.GetAppBPaginated(page)
}
func (s *service) GetAppCByLIN(lin string) (model.Sb70020AppC, error) {
	return s.repository.GetAppCByLIN(strings.TrimSpace(strings.ToUpper(lin)))
}
func (s *service) GetAppCPaginated(page int) (PageResponse[model.Sb70020AppC], error) {
	return s.repository.GetAppCPaginated(page)
}
func (s *service) GetAppDByLIN(lin string) ([]model.Sb70020AppD, error) {
	return s.repository.GetAppDByLIN(strings.TrimSpace(strings.ToUpper(lin)))
}
func (s *service) GetAppDPaginated(page int) (PageResponse[model.Sb70020AppD], error) {
	return s.repository.GetAppDPaginated(page)
}
func (s *service) GetAppEByLIN(lin string) ([]model.Sb70020AppE, error) {
	return s.repository.GetAppEByLIN(strings.TrimSpace(strings.ToUpper(lin)))
}
func (s *service) GetAppEPaginated(page int) (PageResponse[model.Sb70020AppE], error) {
	return s.repository.GetAppEPaginated(page)
}
func (s *service) GetAppFByLIN(lin string) (model.Sb70020AppF, error) {
	return s.repository.GetAppFByLIN(strings.TrimSpace(strings.ToUpper(lin)))
}
func (s *service) GetAppFPaginated(page int) (PageResponse[model.Sb70020AppF], error) {
	return s.repository.GetAppFPaginated(page)
}
func (s *service) GetAppGByLIN(lin string) (model.Sb70020AppG, error) {
	return s.repository.GetAppGByLIN(strings.TrimSpace(strings.ToUpper(lin)))
}
func (s *service) GetAppGPaginated(page int) (PageResponse[model.Sb70020AppG], error) {
	return s.repository.GetAppGPaginated(page)
}
func (s *service) GetAppH1ByLIN(lin string) ([]model.Sb70020AppH1, error) {
	return s.repository.GetAppH1ByLIN(strings.TrimSpace(strings.ToUpper(lin)))
}
func (s *service) GetAppH1Paginated(page int) (PageResponse[model.Sb70020AppH1], error) {
	return s.repository.GetAppH1Paginated(page)
}
func (s *service) GetAppH2ByLIN(lin string) ([]model.Sb70020AppH2, error) {
	return s.repository.GetAppH2ByLIN(strings.TrimSpace(strings.ToUpper(lin)))
}
func (s *service) GetAppH2Paginated(page int) (PageResponse[model.Sb70020AppH2], error) {
	return s.repository.GetAppH2Paginated(page)
}
func (s *service) GetAppIByLIN(lin string) (model.Sb70020AppI, error) {
	return s.repository.GetAppIByLIN(strings.TrimSpace(strings.ToUpper(lin)))
}
func (s *service) GetAppIPaginated(page int) (PageResponse[model.Sb70020AppI], error) {
	return s.repository.GetAppIPaginated(page)
}
func (s *service) GetAppJByLIN(lin string) (model.Sb70020AppJ, error) {
	return s.repository.GetAppJByLIN(strings.TrimSpace(strings.ToUpper(lin)))
}
func (s *service) GetAppJPaginated(page int) (PageResponse[model.Sb70020AppJ], error) {
	return s.repository.GetAppJPaginated(page)
}
func (s *service) GetChp4ByLIN(lin string) (model.Sb70020Chp4, error) {
	return s.repository.GetChp4ByLIN(strings.TrimSpace(strings.ToUpper(lin)))
}
func (s *service) GetChp4Paginated(page int) (PageResponse[model.Sb70020Chp4], error) {
	return s.repository.GetChp4Paginated(page)
}
func (s *service) GetChp6ByLIN(lin string) ([]model.Sb70020Chp6, error) {
	return s.repository.GetChp6ByLIN(strings.TrimSpace(strings.ToUpper(lin)))
}
func (s *service) GetChp6Paginated(page int) (PageResponse[model.Sb70020Chp6], error) {
	return s.repository.GetChp6Paginated(page)
}
func (s *service) GetChp8ByLIN(lin string) ([]model.Sb70020Chp8, error) {
	return s.repository.GetChp8ByLIN(strings.TrimSpace(strings.ToUpper(lin)))
}
func (s *service) GetChp8Paginated(page int) (PageResponse[model.Sb70020Chp8], error) {
	return s.repository.GetChp8Paginated(page)
}
