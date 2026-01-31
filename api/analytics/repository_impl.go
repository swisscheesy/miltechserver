package analytics

import (
	"database/sql"
	"fmt"
	"time"

	. "github.com/go-jet/jet/v2/postgres"
	"github.com/google/uuid"

	"miltechserver/.gen/miltech_ng/public/table"
)

type RepositoryImpl struct {
	db *sql.DB
}

func NewRepository(db *sql.DB) *RepositoryImpl {
	return &RepositoryImpl{db: db}
}

func (repo *RepositoryImpl) IncrementCounter(eventType string, entityKey string, entityLabel string) error {
	now := time.Now()

	stmt := table.AnalyticsEventCounters.INSERT(
		table.AnalyticsEventCounters.ID,
		table.AnalyticsEventCounters.EventType,
		table.AnalyticsEventCounters.EntityKey,
		table.AnalyticsEventCounters.EntityLabel,
		table.AnalyticsEventCounters.Count,
		table.AnalyticsEventCounters.LastSeenAt,
	).VALUES(
		uuid.NewString(),
		eventType,
		entityKey,
		entityLabel,
		1,
		TimestampT(now),
	).ON_CONFLICT(
		table.AnalyticsEventCounters.EventType,
		table.AnalyticsEventCounters.EntityKey,
	).DO_UPDATE(
		SET(
			table.AnalyticsEventCounters.Count.SET(
				table.AnalyticsEventCounters.Count.ADD(Int(1)),
			),
			table.AnalyticsEventCounters.EntityLabel.SET(String(entityLabel)),
			table.AnalyticsEventCounters.LastSeenAt.SET(TimestampT(now)),
		),
	)

	_, err := stmt.Exec(repo.db)
	if err != nil {
		return fmt.Errorf("failed to increment analytics counter: %w", err)
	}

	return nil
}
