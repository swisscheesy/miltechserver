package detailed

import (
	"context"
	"database/sql"
	"log/slog"

	"github.com/go-jet/jet/v2/qrm"
	"golang.org/x/sync/errgroup"

	"miltechserver/api/item_query/detailed/queries"
	"miltechserver/api/response"
)

type RepositoryImpl struct {
	Db *sql.DB
}

func NewRepository(db *sql.DB) *RepositoryImpl {
	return &RepositoryImpl{Db: db}
}

// GetDetailedItemData fetches comprehensive item data by NIIN using parallel queries.
// All 10 query functions execute concurrently, reducing response time from ~45 sequential
// round-trips to the latency of the slowest query function (~8 parallel round-trips max).
// Errors in individual queries are logged but don't fail the entire request - partial data is returned.
// Uses plain errgroup (no context) to prevent cascading cancellations when tables have no data.
func (repo *RepositoryImpl) GetDetailedItemData(ctx context.Context, niin string) (response.DetailedResponse, error) {
	var result response.DetailedResponse
	var g errgroup.Group

	g.Go(func() error {
		data, err := queries.GetAmdfData(ctx, repo.Db, niin)
		if err != nil {
			repo.logQueryError("amdf", niin, err)
			return nil // Continue with partial data
		}
		result.Amdf = data
		return nil
	})

	g.Go(func() error {
		data, err := queries.GetArmyPackagingAndFreight(ctx, repo.Db, niin)
		if err != nil {
			repo.logQueryError("army_packaging", niin, err)
			return nil
		}
		result.ArmyPackagingAndFreight = data
		return nil
	})

	g.Go(func() error {
		data, err := queries.GetSarsscat(ctx, repo.Db, niin)
		if err != nil {
			repo.logQueryError("sarsscat", niin, err)
			return nil
		}
		result.Sarsscat = data
		return nil
	})

	g.Go(func() error {
		data, err := queries.GetIdentification(ctx, repo.Db, niin)
		if err != nil {
			repo.logQueryError("identification", niin, err)
			return nil
		}
		result.Identification = data
		return nil
	})

	g.Go(func() error {
		data, err := queries.GetManagement(ctx, repo.Db, niin)
		if err != nil {
			repo.logQueryError("management", niin, err)
			return nil
		}
		result.Management = data
		return nil
	})

	g.Go(func() error {
		data, err := queries.GetReference(ctx, repo.Db, niin)
		if err != nil {
			repo.logQueryError("reference", niin, err)
			return nil
		}
		result.Reference = data
		return nil
	})

	g.Go(func() error {
		data, err := queries.GetFreight(ctx, repo.Db, niin)
		if err != nil {
			repo.logQueryError("freight", niin, err)
			return nil
		}
		result.Freight = data
		return nil
	})

	g.Go(func() error {
		data, err := queries.GetPackaging(ctx, repo.Db, niin)
		if err != nil {
			repo.logQueryError("packaging", niin, err)
			return nil
		}
		result.Packaging = data
		return nil
	})

	g.Go(func() error {
		data, err := queries.GetCharacteristics(ctx, repo.Db, niin)
		if err != nil {
			repo.logQueryError("characteristics", niin, err)
			return nil
		}
		result.Characteristics = data
		return nil
	})

	g.Go(func() error {
		data, err := queries.GetDisposition(ctx, repo.Db, niin)
		if err != nil {
			repo.logQueryError("disposition", niin, err)
			return nil
		}
		result.Disposition = data
		return nil
	})

	// Wait for all goroutines to complete
	// Since we return nil from all goroutines, this will never return an error
	_ = g.Wait()

	return result, nil
}

func (repo *RepositoryImpl) logQueryError(source string, niin string, err error) {
	if err == nil {
		return
	}
	// Don't log "no rows" as an error - it's expected when tables don't have data for this NIIN
	if err == qrm.ErrNoRows {
		slog.Debug("No data found", "source", source, "niin", niin)
		return
	}
	slog.Error("Detailed item query failed", "source", source, "niin", niin, "error", err)
}
