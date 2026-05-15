package sb_700_20_test

import (
	"database/sql"
	"strings"
	"testing"

	sb700 "miltechserver/api/sb_700_20"

	"github.com/stretchr/testify/require"
)

func TestRepositoryValidationErrors(t *testing.T) {
	repo := sb700.NewRepository((*sql.DB)(nil))

	cases := []struct {
		name string
		fn   func() error
	}{
		{"GetAppBByLIN blank", func() error { _, err := repo.GetAppBByLIN("  "); return err }},
		{"GetAppBPaginated zero", func() error { _, err := repo.GetAppBPaginated(0); return err }},
		{"GetAppBPaginated neg", func() error { _, err := repo.GetAppBPaginated(-1); return err }},
		{"GetAppCByLIN blank", func() error { _, err := repo.GetAppCByLIN("  "); return err }},
		{"GetAppCPaginated zero", func() error { _, err := repo.GetAppCPaginated(0); return err }},
		{"GetAppDByLIN blank", func() error { _, err := repo.GetAppDByLIN("  "); return err }},
		{"GetAppDPaginated zero", func() error { _, err := repo.GetAppDPaginated(0); return err }},
		{"GetAppEByLIN blank", func() error { _, err := repo.GetAppEByLIN("  "); return err }},
		{"GetAppEPaginated zero", func() error { _, err := repo.GetAppEPaginated(0); return err }},
		{"GetAppFByLIN blank", func() error { _, err := repo.GetAppFByLIN("  "); return err }},
		{"GetAppFPaginated zero", func() error { _, err := repo.GetAppFPaginated(0); return err }},
		{"GetAppGByLIN blank", func() error { _, err := repo.GetAppGByLIN("  "); return err }},
		{"GetAppGPaginated zero", func() error { _, err := repo.GetAppGPaginated(0); return err }},
		{"GetAppH1ByLIN blank", func() error { _, err := repo.GetAppH1ByLIN("  "); return err }},
		{"GetAppH1Paginated zero", func() error { _, err := repo.GetAppH1Paginated(0); return err }},
		{"GetAppH2ByLIN blank", func() error { _, err := repo.GetAppH2ByLIN("  "); return err }},
		{"GetAppH2Paginated zero", func() error { _, err := repo.GetAppH2Paginated(0); return err }},
		{"GetAppIByLIN blank", func() error { _, err := repo.GetAppIByLIN("  "); return err }},
		{"GetAppIPaginated zero", func() error { _, err := repo.GetAppIPaginated(0); return err }},
		{"GetAppJByLIN blank", func() error { _, err := repo.GetAppJByLIN("  "); return err }},
		{"GetAppJPaginated zero", func() error { _, err := repo.GetAppJPaginated(0); return err }},
		{"GetChp4ByLIN blank", func() error { _, err := repo.GetChp4ByLIN("  "); return err }},
		{"GetChp4Paginated zero", func() error { _, err := repo.GetChp4Paginated(0); return err }},
		{"GetChp6ByLIN blank", func() error { _, err := repo.GetChp6ByLIN("  "); return err }},
		{"GetChp6Paginated zero", func() error { _, err := repo.GetChp6Paginated(0); return err }},
		{"GetChp8ByLIN blank", func() error { _, err := repo.GetChp8ByLIN("  "); return err }},
		{"GetChp8Paginated zero", func() error { _, err := repo.GetChp8Paginated(0); return err }},
		{"GetAppEByNewLIN blank", func() error { _, err := repo.GetAppEByNewLIN("  "); return err }},
		{"GetAppGByNewLIN blank", func() error { _, err := repo.GetAppGByNewLIN("  "); return err }},
		{"GetAppH1BySubLIN blank", func() error { _, err := repo.GetAppH1BySubLIN("  "); return err }},
		{"GetAppH2BySubLIN blank", func() error { _, err := repo.GetAppH2BySubLIN("  "); return err }},
		{"GetChp4ByRIC blank", func() error { _, err := repo.GetChp4ByRIC("  "); return err }},
		{"GetChp6ByRIC blank", func() error { _, err := repo.GetChp6ByRIC("  "); return err }},
		{"GetChp8ByRIC blank", func() error { _, err := repo.GetChp8ByRIC("  "); return err }},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			err := tc.fn()
			if strings.Contains(tc.name, "blank") {
				require.ErrorIs(t, err, sb700.ErrEmptyParam)
			} else {
				require.ErrorIs(t, err, sb700.ErrInvalidPage)
			}
		})
	}
}
