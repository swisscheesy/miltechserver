package user_pmcs_test

import (
	"context"
	"database/sql"
	"errors"
	"testing"

	"github.com/google/uuid"
	"github.com/lib/pq"
	"github.com/stretchr/testify/require"

	"miltechserver/api/user_pmcs/persistence"
	"miltechserver/api/user_pmcs/shared"
)

func TestTreePersistenceRoundTripAndReplacement(t *testing.T) {
	ownerUID := newUserPmcsTestUser(t)
	checklistID := uuid.New()
	revisionID := uuid.New()
	insertChecklistAndRevision(t, ownerUID, checklistID, revisionID, "draft")

	first := preparedTree(t, revisionID)
	require.NoError(t, replaceTreeInTransaction(t, checklistID, first))

	loaded, err := persistence.LoadRevisionTrees(
		context.Background(),
		testDB,
		[]uuid.UUID{revisionID},
	)
	require.NoError(t, err)
	assertLoadedTree(t, first, loaded[revisionID])

	second := preparedTree(t, revisionID)
	second.Input.Name = "Updated draft"
	second, err = shared.PrepareDraft(second.Input, shared.DefaultConfig())
	require.NoError(t, err)
	require.NoError(t, replaceTreeInTransaction(t, checklistID, second))

	loaded, err = persistence.LoadRevisionTrees(
		context.Background(),
		testDB,
		[]uuid.UUID{revisionID},
	)
	require.NoError(t, err)
	require.Len(t, loaded, 1)
	assertLoadedTree(t, second, loaded[revisionID])

	var storedHash []byte
	require.NoError(t, testDB.QueryRowContext(
		context.Background(),
		`SELECT content_hash FROM user_pmcs_revisions WHERE id = $1`,
		revisionID,
	).Scan(&storedHash))
	require.Equal(t, second.Hash[:], storedHash)

	var oldSectionCount int
	require.NoError(t, testDB.QueryRowContext(
		context.Background(),
		`SELECT count(*) FROM user_pmcs_sections WHERE id = $1`,
		first.Input.Sections[0].ID,
	).Scan(&oldSectionCount))
	require.Zero(t, oldSectionCount)
}

func TestTreePersistenceRejectsCrossRevisionUUIDGraftBeforeDeletion(t *testing.T) {
	ownerUID := newUserPmcsTestUser(t)

	foreignChecklistID := uuid.New()
	foreignRevisionID := uuid.New()
	insertChecklistAndRevision(
		t,
		ownerUID,
		foreignChecklistID,
		foreignRevisionID,
		"draft",
	)
	foreignTree := preparedTree(t, foreignRevisionID)
	require.NoError(
		t,
		replaceTreeInTransaction(t, foreignChecklistID, foreignTree),
	)

	targetChecklistID := uuid.New()
	targetRevisionID := uuid.New()
	insertChecklistAndRevision(
		t,
		ownerUID,
		targetChecklistID,
		targetRevisionID,
		"draft",
	)
	targetTree := preparedTree(t, targetRevisionID)
	require.NoError(
		t,
		replaceTreeInTransaction(t, targetChecklistID, targetTree),
	)

	graftedInput := preparedTree(t, targetRevisionID).Input
	graftedInput.Sections[0].ID = foreignTree.Input.Sections[0].ID
	grafted, err := shared.PrepareDraft(graftedInput, shared.DefaultConfig())
	require.NoError(t, err)

	err = replaceTreeInTransaction(t, targetChecklistID, grafted)
	require.Error(t, err)
	var apiError *shared.APIError
	require.ErrorAs(t, err, &apiError)
	require.Equal(t, "validation_failed", apiError.Code)

	loaded, err := persistence.LoadRevisionTrees(
		context.Background(),
		testDB,
		[]uuid.UUID{targetRevisionID, foreignRevisionID},
	)
	require.NoError(t, err)
	assertLoadedTree(t, targetTree, loaded[targetRevisionID])
	assertLoadedTree(t, foreignTree, loaded[foreignRevisionID])
}

func TestTreePersistenceRejectsCrossNodeTypeUUIDGraft(t *testing.T) {
	ownerUID := newUserPmcsTestUser(t)

	foreignChecklistID := uuid.New()
	foreignRevisionID := uuid.New()
	insertChecklistAndRevision(
		t,
		ownerUID,
		foreignChecklistID,
		foreignRevisionID,
		"draft",
	)
	foreignTree := preparedTree(t, foreignRevisionID)
	require.NoError(
		t,
		replaceTreeInTransaction(t, foreignChecklistID, foreignTree),
	)

	targetChecklistID := uuid.New()
	targetRevisionID := uuid.New()
	insertChecklistAndRevision(
		t,
		ownerUID,
		targetChecklistID,
		targetRevisionID,
		"draft",
	)
	targetTree := preparedTree(t, targetRevisionID)
	require.NoError(
		t,
		replaceTreeInTransaction(t, targetChecklistID, targetTree),
	)

	graftedInput := preparedTree(t, targetRevisionID).Input
	graftedInput.Sections[0].Items[0].ID =
		foreignTree.Input.Sections[0].ID
	grafted, err := shared.PrepareDraft(graftedInput, shared.DefaultConfig())
	require.NoError(t, err)

	err = replaceTreeInTransaction(t, targetChecklistID, grafted)
	require.Error(t, err)
	var apiError *shared.APIError
	require.ErrorAs(t, err, &apiError)
	require.Equal(t, "validation_failed", apiError.Code)

	loaded, err := persistence.LoadRevisionTrees(
		context.Background(),
		testDB,
		[]uuid.UUID{targetRevisionID, foreignRevisionID},
	)
	require.NoError(t, err)
	assertLoadedTree(t, targetTree, loaded[targetRevisionID])
	assertLoadedTree(t, foreignTree, loaded[foreignRevisionID])
}

func TestTreePersistenceRejectsExistingUUIDMovedToAnotherParent(t *testing.T) {
	ownerUID := newUserPmcsTestUser(t)
	checklistID := uuid.New()
	revisionID := uuid.New()
	insertChecklistAndRevision(t, ownerUID, checklistID, revisionID, "draft")
	storedTree := preparedTree(t, revisionID)
	require.NoError(t, replaceTreeInTransaction(t, checklistID, storedTree))

	movedInput := preparedTree(t, revisionID).Input
	secondSection := preparedTree(t, revisionID).Input.Sections[0]
	secondSection.Position = 2
	secondSection.Items[0].ID = storedTree.Input.Sections[0].Items[0].ID
	movedInput.Sections = append(movedInput.Sections, secondSection)
	moved, err := shared.PrepareDraft(movedInput, shared.DefaultConfig())
	require.NoError(t, err)

	err = replaceTreeInTransaction(t, checklistID, moved)
	require.Error(t, err)
	var apiError *shared.APIError
	require.ErrorAs(t, err, &apiError)
	require.Equal(t, "validation_failed", apiError.Code)

	loaded, err := persistence.LoadRevisionTrees(
		context.Background(),
		testDB,
		[]uuid.UUID{revisionID},
	)
	require.NoError(t, err)
	assertLoadedTree(t, storedTree, loaded[revisionID])
}

func TestTreePersistenceRejectsImmutableRevisionReplacement(t *testing.T) {
	ownerUID := newUserPmcsTestUser(t)
	checklistID := uuid.New()
	revisionID := uuid.New()
	insertChecklistAndRevision(
		t,
		ownerUID,
		checklistID,
		revisionID,
		"published",
	)

	prepared := preparedTree(t, revisionID)
	err := replaceTreeInTransaction(t, checklistID, prepared)
	require.Error(t, err)

	var state string
	require.NoError(t, testDB.QueryRowContext(
		context.Background(),
		`SELECT state FROM user_pmcs_revisions WHERE id = $1`,
		revisionID,
	).Scan(&state))
	require.Equal(t, "published", state)
}

func TestTreeLoadingUsesFixedQueryCountForOneAndTwentyFiveRoots(t *testing.T) {
	ownerUID := newUserPmcsTestUser(t)
	revisionIDs := make([]uuid.UUID, 25)
	for index := range revisionIDs {
		checklistID := uuid.New()
		revisionIDs[index] = uuid.New()
		insertChecklistAndRevision(
			t,
			ownerUID,
			checklistID,
			revisionIDs[index],
			"draft",
		)
	}

	for _, selectedIDs := range [][]uuid.UUID{
		revisionIDs[:1],
		revisionIDs,
	} {
		queryer := &countingQueryer{database: testDB}
		loaded, err := persistence.LoadRevisionTrees(
			context.Background(),
			queryer,
			selectedIDs,
		)
		require.NoError(t, err)
		require.Len(t, loaded, len(selectedIDs))
		require.Equal(t, 7, queryer.queryCount)
	}

	queryer := &countingQueryer{database: testDB}
	loaded, err := persistence.LoadRevisionTrees(
		context.Background(),
		queryer,
		nil,
	)
	require.NoError(t, err)
	require.Empty(t, loaded)
	require.Zero(t, queryer.queryCount)
}

func TestTreeDeletionIsScopedToRequestedRevisionIDs(t *testing.T) {
	ownerUID := newUserPmcsTestUser(t)
	firstChecklistID := uuid.New()
	firstRevisionID := uuid.New()
	insertChecklistAndRevision(
		t,
		ownerUID,
		firstChecklistID,
		firstRevisionID,
		"draft",
	)
	secondChecklistID := uuid.New()
	secondRevisionID := uuid.New()
	insertChecklistAndRevision(
		t,
		ownerUID,
		secondChecklistID,
		secondRevisionID,
		"draft",
	)

	tx, err := testDB.BeginTx(context.Background(), nil)
	require.NoError(t, err)
	require.NoError(t, persistence.DeleteRevisionTrees(
		context.Background(),
		tx,
		[]uuid.UUID{firstRevisionID},
	))
	require.NoError(t, tx.Commit())

	var remaining []uuid.UUID
	rows, err := testDB.QueryContext(
		context.Background(),
		`SELECT id
		 FROM user_pmcs_revisions
		 WHERE id = ANY($1)
		 ORDER BY id`,
		pq.Array([]uuid.UUID{firstRevisionID, secondRevisionID}),
	)
	require.NoError(t, err)
	for rows.Next() {
		var revisionID uuid.UUID
		require.NoError(t, rows.Scan(&revisionID))
		remaining = append(remaining, revisionID)
	}
	require.NoError(t, rows.Err())
	require.NoError(t, rows.Close())
	require.Equal(t, []uuid.UUID{secondRevisionID}, remaining)

	tx, err = testDB.BeginTx(context.Background(), nil)
	require.NoError(t, err)
	require.NoError(t, persistence.DeleteRevisionTrees(
		context.Background(),
		tx,
		nil,
	))
	require.NoError(t, tx.Commit())
}

func TestTreeAccountVersionRequiresInitializedAccountAndRollsBack(t *testing.T) {
	missingUID := uniqueFixtureName("missing-user")
	tx, err := testDB.BeginTx(context.Background(), nil)
	require.NoError(t, err)
	_, err = persistence.LockAccountVersion(context.Background(), tx, missingUID)
	require.Error(t, err)
	var apiError *shared.APIError
	require.ErrorAs(t, err, &apiError)
	require.Equal(t, "account_not_initialized", apiError.Code)
	require.NoError(t, tx.Rollback())

	var missingStateCount int
	require.NoError(t, testDB.QueryRowContext(
		context.Background(),
		`SELECT count(*) FROM user_pmcs_sync_state WHERE user_uid = $1`,
		missingUID,
	).Scan(&missingStateCount))
	require.Zero(t, missingStateCount)

	ownerUID := newUserPmcsTestUser(t)
	tx, err = testDB.BeginTx(context.Background(), nil)
	require.NoError(t, err)
	version, err := persistence.LockAccountVersion(
		context.Background(),
		tx,
		ownerUID,
	)
	require.NoError(t, err)
	require.Zero(t, version)
	version, err = persistence.AdvanceAccountVersion(
		context.Background(),
		tx,
		ownerUID,
	)
	require.NoError(t, err)
	require.Equal(t, int64(1), version)
	require.NoError(t, tx.Rollback())

	var persistedVersion int64
	err = testDB.QueryRowContext(
		context.Background(),
		`SELECT current_version
		 FROM user_pmcs_sync_state
		 WHERE user_uid = $1`,
		ownerUID,
	).Scan(&persistedVersion)
	require.True(t, errors.Is(err, sql.ErrNoRows))

	tx, err = testDB.BeginTx(context.Background(), nil)
	require.NoError(t, err)
	version, err = persistence.LockAccountVersion(
		context.Background(),
		tx,
		ownerUID,
	)
	require.NoError(t, err)
	require.Zero(t, version)
	version, err = persistence.AdvanceAccountVersion(
		context.Background(),
		tx,
		ownerUID,
	)
	require.NoError(t, err)
	require.Equal(t, int64(1), version)
	require.NoError(t, tx.Commit())

	require.NoError(t, testDB.QueryRowContext(
		context.Background(),
		`SELECT current_version
		 FROM user_pmcs_sync_state
		 WHERE user_uid = $1`,
		ownerUID,
	).Scan(&persistedVersion))
	require.Equal(t, int64(1), persistedVersion)
}

func replaceTreeInTransaction(
	t *testing.T,
	checklistID uuid.UUID,
	prepared shared.PreparedRevision,
) error {
	t.Helper()

	tx, err := testDB.BeginTx(context.Background(), nil)
	require.NoError(t, err)
	if err := persistence.ReplaceDraftTree(
		context.Background(),
		tx,
		checklistID,
		prepared,
	); err != nil {
		require.NoError(t, tx.Rollback())
		return err
	}
	return tx.Commit()
}
