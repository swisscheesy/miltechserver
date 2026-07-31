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
	requireUserPmcsTestDatabase(t, testDB)

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

func requireUserPmcsTestDatabase(t *testing.T, database *sql.DB) {
	t.Helper()

	require.NotNil(t, database)
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	var databaseName string
	require.NoError(
		t,
		database.QueryRowContext(ctx, `SELECT current_database()`).Scan(&databaseName),
	)
	require.Equal(t, "miltech_ng_test", databaseName)
	t.Logf("database safety proof: current_database()=%s", databaseName)
}

func insertChecklistAndRevision(
	t *testing.T,
	ownerUID string,
	checklistID uuid.UUID,
	revisionID uuid.UUID,
	state string,
) {
	t.Helper()
	requireUserPmcsTestDatabase(t, testDB)

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

type deterministicTreeSize struct {
	Sections int
	Items    int
	Notices  int
	Steps    int
}

func maximumDeterministicTree(
	t testing.TB,
	fixtureID string,
	revisionID uuid.UUID,
) shared.RevisionInput {
	t.Helper()
	return deterministicRevisionTree(
		t,
		fixtureID,
		revisionID,
		deterministicTreeSize{
			Sections: 100,
			Items:    2_000,
			Notices:  4_000,
			Steps:    10_000,
		},
	)
}

func deterministicRevisionTree(
	t testing.TB,
	fixtureID string,
	revisionID uuid.UUID,
	size deterministicTreeSize,
) shared.RevisionInput {
	t.Helper()
	require.Positive(t, size.Sections)
	require.GreaterOrEqual(t, size.Items, size.Sections)
	require.LessOrEqual(t, size.Items, size.Sections*500)
	require.LessOrEqual(t, size.Notices, size.Items*100)
	require.LessOrEqual(t, size.Steps, size.Items*250)

	input := shared.RevisionInput{
		ID:          revisionID,
		Name:        "n",
		Description: "d",
		Models: []shared.ModelInput{
			{DisplayText: "m"},
		},
		Sections: make([]shared.SectionInput, 0, size.Sections),
	}
	noticeType := "warning"
	itemIndex := 0
	for sectionIndex := 0; sectionIndex < size.Sections; sectionIndex++ {
		itemCount := distributedCount(
			size.Items,
			size.Sections,
			sectionIndex,
		)
		section := shared.SectionInput{
			ID: deterministicFixtureUUID(
				fixtureID,
				fmt.Sprintf("section-%d", sectionIndex),
			),
			Position: int32(sectionIndex + 1),
			Title:    "s",
			Items:    make([]shared.ItemInput, 0, itemCount),
		}
		for itemPosition := 0; itemPosition < itemCount; itemPosition++ {
			noticeCount := distributedCount(
				size.Notices,
				size.Items,
				itemIndex,
			)
			stepCount := distributedCount(
				size.Steps,
				size.Items,
				itemIndex,
			)
			item := shared.ItemInput{
				ID: deterministicFixtureUUID(
					fixtureID,
					fmt.Sprintf("item-%d", itemIndex),
				),
				Position:                  int32(itemPosition + 1),
				Interval:                  "i",
				ItemToBeCheckedOrServiced: "c",
				PerformedBy:               "p",
				Notices: make(
					[]shared.NoticeInput,
					0,
					noticeCount,
				),
				ProcedureSteps: make(
					[]shared.ProcedureStepInput,
					0,
					stepCount,
				),
			}
			for noticeIndex := 0; noticeIndex < noticeCount; noticeIndex++ {
				item.Notices = append(item.Notices, shared.NoticeInput{
					ID: deterministicFixtureUUID(
						fixtureID,
						fmt.Sprintf(
							"item-%d-notice-%d",
							itemIndex,
							noticeIndex,
						),
					),
					Position:   int32(noticeIndex + 1),
					Type:       &noticeType,
					NoticeText: "w",
				})
			}
			for stepIndex := 0; stepIndex < stepCount; stepIndex++ {
				item.ProcedureSteps = append(
					item.ProcedureSteps,
					shared.ProcedureStepInput{
						ID: deterministicFixtureUUID(
							fixtureID,
							fmt.Sprintf(
								"item-%d-step-%d",
								itemIndex,
								stepIndex,
							),
						),
						Position:     int32(stepIndex + 1),
						StepText:     "x",
						FaultFoundIf: "f",
					},
				)
			}
			section.Items = append(section.Items, item)
			itemIndex++
		}
		input.Sections = append(input.Sections, section)
	}
	return input
}

func distributedCount(total, buckets, index int) int {
	base := total / buckets
	if index < total%buckets {
		return base + 1
	}
	return base
}

func deterministicFixtureUUID(fixtureID, nodeID string) uuid.UUID {
	return uuid.NewSHA1(
		uuid.NameSpaceOID,
		[]byte("user-pmcs-test/"+fixtureID+"/"+nodeID),
	)
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
