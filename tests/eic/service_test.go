package eic_test

import (
	"strings"
	"testing"

	"miltechserver/api/eic"
	"miltechserver/api/response"

	"github.com/stretchr/testify/require"
)

type captureRepository struct {
	niin   string
	lin    string
	fsc    string
	page   int
	search string
}

func (repo *captureRepository) GetByNIIN(niin string) ([]response.EICConsolidatedItem, error) {
	repo.niin = niin
	return []response.EICConsolidatedItem{{Niin: "TEST"}}, nil
}

func (repo *captureRepository) GetByLIN(lin string) ([]response.EICConsolidatedItem, error) {
	repo.lin = lin
	return []response.EICConsolidatedItem{{Niin: "TEST"}}, nil
}

func (repo *captureRepository) GetByFSCPaginated(fsc string, page int) (response.EICPageResponse, error) {
	repo.fsc = fsc
	repo.page = page
	return response.EICPageResponse{}, nil
}

func (repo *captureRepository) GetAllPaginated(page int, search string) (response.EICPageResponse, error) {
	repo.page = page
	repo.search = search
	return response.EICPageResponse{}, nil
}

func TestEICServiceTrimsAndUppercases(t *testing.T) {
	repo := &captureRepository{}
	svc := eic.NewService(repo)

	_, err := svc.LookupByNIIN("  abcd ")
	require.NoError(t, err)
	require.Equal(t, "ABCD", repo.niin)

	_, err = svc.LookupByLIN("  l123 ")
	require.NoError(t, err)
	require.Equal(t, "L123", repo.lin)

	_, err = svc.LookupByFSCPaginated("  fsc ", 2)
	require.NoError(t, err)
	require.Equal(t, "FSC", repo.fsc)
	require.Equal(t, 2, repo.page)

	_, err = svc.LookupAllPaginated(3, "  search  ")
	require.NoError(t, err)
	require.Equal(t, 3, repo.page)
	require.Equal(t, strings.TrimSpace("  search  "), repo.search)
}
