package docs_equipment

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

// captureRepo captures arguments passed to repo methods for verification.
type captureRepo struct {
	family string
	query  string
	page   int
}

func (r *captureRepo) GetAllPaginated(page int) (EquipmentDetailsPageResponse, error) {
	r.page = page
	return EquipmentDetailsPageResponse{}, nil
}
func (r *captureRepo) GetFamilies() (FamiliesResponse, error) {
	return FamiliesResponse{Families: []string{"aircraft"}, Count: 1}, nil
}
func (r *captureRepo) GetByFamilyPaginated(family string, page int) (EquipmentDetailsPageResponse, error) {
	r.family = family
	r.page = page
	return EquipmentDetailsPageResponse{}, nil
}
func (r *captureRepo) SearchPaginated(query string, page int) (EquipmentDetailsPageResponse, error) {
	r.query = query
	r.page = page
	return EquipmentDetailsPageResponse{}, nil
}

func TestServiceTrimsFamily(t *testing.T) {
	repo := &captureRepo{}
	svc := NewService(repo, nil)

	_, err := svc.GetByFamilyPaginated("  aircraft  ", 1)
	require.NoError(t, err)
	require.Equal(t, "aircraft", repo.family)
}

func TestServiceTrimsSearch(t *testing.T) {
	repo := &captureRepo{}
	svc := NewService(repo, nil)

	_, err := svc.SearchPaginated("  AH-64  ", 2)
	require.NoError(t, err)
	require.Equal(t, strings.TrimSpace("  AH-64  "), repo.query)
	require.Equal(t, 2, repo.page)
}

func TestServiceDelegatesGetAll(t *testing.T) {
	repo := &captureRepo{}
	svc := NewService(repo, nil)

	_, err := svc.GetAllPaginated(3)
	require.NoError(t, err)
	require.Equal(t, 3, repo.page)
}

func TestIsImageFile(t *testing.T) {
	require.True(t, isImageFile("test.jpg"))
	require.True(t, isImageFile("test.JPEG"))
	require.True(t, isImageFile("test.png"))
	require.True(t, isImageFile("test.gif"))
	require.True(t, isImageFile("test.webp"))
	require.False(t, isImageFile("test.pdf"))
	require.False(t, isImageFile("test.exe"))
	require.False(t, isImageFile("test"))
}
