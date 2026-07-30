package user_pmcs_test

import (
	"context"
	"database/sql"
	"fmt"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"

	"miltechserver/api/user_pmcs/shared"
)

func newUserPmcsTestUser(t *testing.T) string {
	t.Helper()

	userUID := "user-pmcs-" + uuid.NewString()
	_, err := testDB.ExecContext(
		context.Background(),
		`INSERT INTO users (uid, email, username, created_at, is_enabled)
		 VALUES ($1, $2, $3, $4, TRUE)`,
		userUID,
		userUID+"@example.com",
		"user-pmcs-test",
		time.Now().UTC(),
	)
	require.NoError(t, err)

	t.Cleanup(func() {
		_, cleanupError := testDB.ExecContext(
			context.Background(),
			`DELETE FROM user_pmcs_checklists WHERE owner_uid = $1`,
			userUID,
		)
		require.NoError(t, cleanupError)
		_, cleanupError = testDB.ExecContext(
			context.Background(),
			`DELETE FROM users WHERE uid = $1`,
			userUID,
		)
		require.NoError(t, cleanupError)
	})
	return userUID
}

func insertChecklistAndRevision(
	t *testing.T,
	ownerUID string,
	checklistID uuid.UUID,
	revisionID uuid.UUID,
	state string,
) {
	t.Helper()

	revisionNumber := any(nil)
	publishedAt := any(nil)
	if state != "draft" {
		revisionNumber = int32(1)
		publishedAt = time.Now().UTC()
	}

	tx, err := testDB.BeginTx(context.Background(), nil)
	require.NoError(t, err)
	t.Cleanup(func() {
		_ = tx.Rollback()
	})

	_, err = tx.ExecContext(
		context.Background(),
		`INSERT INTO user_pmcs_checklists
		     (id, owner_uid, sync_version, account_change_version)
		 VALUES ($1, $2, 1, 1)`,
		checklistID,
		ownerUID,
	)
	require.NoError(t, err)
	_, err = tx.ExecContext(
		context.Background(),
		`INSERT INTO user_pmcs_revisions
		     (id, checklist_id, state, revision_number, name, description,
		      content_hash, published_at)
		 VALUES ($1, $2, $3, $4, '', '', $5, $6)`,
		revisionID,
		checklistID,
		state,
		revisionNumber,
		make([]byte, 32),
		publishedAt,
	)
	require.NoError(t, err)
	require.NoError(t, tx.Commit())
}

func preparedTree(t *testing.T, revisionID uuid.UUID) shared.PreparedRevision {
	t.Helper()

	noticeType := "warning"
	prepared, err := shared.PrepareDraft(
		shared.RevisionInput{
			ID:          revisionID,
			Name:        "Vehicle PMCS",
			Description: "Complete draft round trip",
			Models: []shared.ModelInput{
				{DisplayText: " M998  HMMWV "},
			},
			Sections: []shared.SectionInput{
				{
					ID:       uuid.New(),
					Position: 1,
					Title:    "Before operation",
					Models: []shared.ModelInput{
						{DisplayText: "M998"},
					},
					Items: []shared.ItemInput{
						{
							ID:                        uuid.New(),
							Position:                  1,
							Interval:                  "Before",
							ItemToBeCheckedOrServiced: "Engine compartment",
							PerformedBy:               "Operator",
							Notices: []shared.NoticeInput{
								{
									ID:         uuid.New(),
									Position:   1,
									Type:       &noticeType,
									NoticeText: "Use caution",
								},
							},
							ProcedureSteps: []shared.ProcedureStepInput{
								{
									ID:           uuid.New(),
									Position:     1,
									StepText:     "Inspect for leaks",
									FaultFoundIf: "Any leak is present",
								},
							},
						},
					},
				},
			},
		},
		shared.DefaultConfig(),
	)
	require.NoError(t, err)
	return prepared
}

type countingQueryer struct {
	database   *sql.DB
	queryCount int
}

func (queryer *countingQueryer) ExecContext(
	ctx context.Context,
	query string,
	args ...any,
) (sql.Result, error) {
	return queryer.database.ExecContext(ctx, query, args...)
}

func (queryer *countingQueryer) QueryContext(
	ctx context.Context,
	query string,
	args ...any,
) (*sql.Rows, error) {
	queryer.queryCount++
	return queryer.database.QueryContext(ctx, query, args...)
}

func (queryer *countingQueryer) QueryRowContext(
	ctx context.Context,
	query string,
	args ...any,
) *sql.Row {
	return queryer.database.QueryRowContext(ctx, query, args...)
}

func assertLoadedTree(
	t *testing.T,
	prepared shared.PreparedRevision,
	actual shared.Revision,
) {
	t.Helper()

	require.Equal(t, prepared.Input.ID, actual.ID)
	require.Equal(t, "draft", actual.State)
	require.Nil(t, actual.RevisionNumber)
	require.Equal(t, prepared.Input.Name, actual.Name)
	require.Equal(t, prepared.Input.Description, actual.Description)
	require.Equal(t, []shared.ModelValue{
		{
			DisplayText:    prepared.Input.Models[0].DisplayText,
			NormalizedText: prepared.Input.Models[0].NormalizedText,
		},
	}, actual.Models)
	require.Len(t, actual.Sections, 1)

	expectedSection := prepared.Input.Sections[0]
	actualSection := actual.Sections[0]
	require.Equal(t, expectedSection.ID, actualSection.ID)
	require.Equal(t, expectedSection.Position, actualSection.Position)
	require.Equal(t, expectedSection.Title, actualSection.Title)
	require.Equal(t, []shared.ModelValue{
		{
			DisplayText:    expectedSection.Models[0].DisplayText,
			NormalizedText: expectedSection.Models[0].NormalizedText,
		},
	}, actualSection.Models)
	require.Len(t, actualSection.Items, 1)

	expectedItem := expectedSection.Items[0]
	actualItem := actualSection.Items[0]
	require.Equal(t, expectedItem.ID, actualItem.ID)
	require.Equal(t, expectedItem.Position, actualItem.Position)
	require.Equal(t, expectedItem.Interval, actualItem.Interval)
	require.Equal(
		t,
		expectedItem.ItemToBeCheckedOrServiced,
		actualItem.ItemToBeCheckedOrServiced,
	)
	require.Equal(t, expectedItem.PerformedBy, actualItem.PerformedBy)
	require.Equal(t, expectedItem.Notices, actualItem.Notices)
	require.Equal(t, expectedItem.ProcedureSteps, actualItem.ProcedureSteps)
}

func uniqueFixtureName(prefix string) string {
	return fmt.Sprintf("%s-%s", prefix, uuid.NewString())
}
