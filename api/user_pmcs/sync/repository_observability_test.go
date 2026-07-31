package sync

import (
	"context"
	"database/sql"
	"database/sql/driver"
	"errors"
	"fmt"
	"io"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"

	"miltechserver/api/user_pmcs/persistence"
	"miltechserver/api/user_pmcs/shared"
)

func TestGetDeltaAttributesPostCommitSizingToEncodeDuration(t *testing.T) {
	checklistID := uuid.New()
	database := openDeltaObservabilityDatabase(t, checklistID)
	clock := &steppingClock{
		current: time.Unix(10_000, 0),
		step:    10 * time.Millisecond,
	}
	repository := &RepositoryImpl{
		store: persistence.NewStore(database, 1),
		now:   clock.Now,
	}
	requestContext, measurements := shared.WithRequestMeasurements(
		context.Background(),
	)

	delta, err := repository.GetDelta(
		requestContext,
		"observer-user",
		0,
		10,
		1_000_000,
	)

	require.NoError(t, err)
	require.Len(t, delta.Changes, 1)
	snapshot := measurements.Snapshot()
	require.Equal(t, 10*time.Millisecond, snapshot.DBDuration)
	require.Equal(t, 10*time.Millisecond, snapshot.EncodeDuration)
}

type steppingClock struct {
	mu      sync.Mutex
	current time.Time
	step    time.Duration
}

func (clock *steppingClock) Now() time.Time {
	clock.mu.Lock()
	defer clock.mu.Unlock()
	current := clock.current
	clock.current = clock.current.Add(clock.step)
	return current
}

type deltaObservabilityDriver struct {
	checklistID uuid.UUID
}

func (databaseDriver deltaObservabilityDriver) Open(string) (driver.Conn, error) {
	return &deltaObservabilityConnection{
		checklistID: databaseDriver.checklistID,
	}, nil
}

type deltaObservabilityConnection struct {
	checklistID uuid.UUID
}

func (connection *deltaObservabilityConnection) Prepare(string) (driver.Stmt, error) {
	return nil, errors.New("not implemented")
}

func (connection *deltaObservabilityConnection) Close() error {
	return nil
}

func (connection *deltaObservabilityConnection) Begin() (driver.Tx, error) {
	return deltaObservabilityTransaction{}, nil
}

func (connection *deltaObservabilityConnection) BeginTx(
	context.Context,
	driver.TxOptions,
) (driver.Tx, error) {
	return deltaObservabilityTransaction{}, nil
}

func (connection *deltaObservabilityConnection) QueryContext(
	_ context.Context,
	query string,
	_ []driver.NamedValue,
) (driver.Rows, error) {
	switch {
	case strings.Contains(query, "SELECT EXISTS"):
		return &deltaObservabilityRows{
			columns: []string{"initialized", "current_version"},
			values:  [][]driver.Value{{true, int64(1)}},
		}, nil
	case strings.Contains(query, "user_pmcs_account_delta_roots"):
		return &deltaObservabilityRows{
			columns: []string{
				"account_change_version",
				"kind",
				"identity",
			},
			values: [][]driver.Value{{
				int64(1),
				ChangeKindChecklist,
				connection.checklistID.String(),
			}},
		}, nil
	case strings.Contains(query, "FROM user_pmcs_checklists AS checklist"):
		now := time.Unix(9_000, 0)
		return &deltaObservabilityRows{
			columns: []string{
				"id",
				"sync_version",
				"account_change_version",
				"created_at",
				"updated_at",
				"deleted_at",
				"draft_id",
				"publication_id",
				"source_status",
				"source_revision_id",
				"source_latest",
				"source_first",
				"source_updated",
				"source_retired",
			},
			values: [][]driver.Value{{
				connection.checklistID.String(),
				int64(1),
				int64(1),
				now,
				now,
				now,
				nil,
				nil,
				nil,
				nil,
				nil,
				nil,
				nil,
				nil,
			}},
		}, nil
	default:
		return nil, fmt.Errorf("unexpected query: %s", query)
	}
}

type deltaObservabilityTransaction struct{}

func (deltaObservabilityTransaction) Commit() error {
	return nil
}

func (deltaObservabilityTransaction) Rollback() error {
	return nil
}

type deltaObservabilityRows struct {
	columns []string
	values  [][]driver.Value
	index   int
}

func (rows *deltaObservabilityRows) Columns() []string {
	return rows.columns
}

func (rows *deltaObservabilityRows) Close() error {
	return nil
}

func (rows *deltaObservabilityRows) Next(destination []driver.Value) error {
	if rows.index >= len(rows.values) {
		return io.EOF
	}
	copy(destination, rows.values[rows.index])
	rows.index++
	return nil
}

var deltaObservabilityDriverCounter atomic.Uint64

func openDeltaObservabilityDatabase(
	t *testing.T,
	checklistID uuid.UUID,
) *sql.DB {
	t.Helper()
	driverName := fmt.Sprintf(
		"user-pmcs-delta-observability-%d",
		deltaObservabilityDriverCounter.Add(1),
	)
	sql.Register(
		driverName,
		deltaObservabilityDriver{checklistID: checklistID},
	)
	database, err := sql.Open(driverName, "")
	require.NoError(t, err)
	t.Cleanup(func() {
		require.NoError(t, database.Close())
	})
	return database
}
