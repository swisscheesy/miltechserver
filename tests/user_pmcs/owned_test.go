package user_pmcs_test

import (
	"context"
	"errors"
	"sync"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"

	"miltechserver/api/user_pmcs/owned"
	"miltechserver/api/user_pmcs/persistence"
	"miltechserver/api/user_pmcs/shared"
)

func TestCreateOwnedChecklistPersistsCompleteDraftAndVersions(t *testing.T) {
	ownerUID := newUserPmcsTestUser(t)
	repository := newOwnedRepository()
	checklistID := uuid.New()
	draft := preparedTree(t, uuid.New())

	result, err := repository.Create(
		context.Background(),
		ownerUID,
		checklistID,
		draft,
		shared.Precondition{Mode: shared.PreconditionCreate},
	)

	require.NoError(t, err)
	require.True(t, result.Created)
	require.False(t, result.Idempotent)
	require.Equal(t, int64(1), result.Aggregate.SyncVersion)
	require.Equal(t, int64(1), result.Aggregate.AccountChangeVersion)
	require.NotNil(t, result.Aggregate.Draft)
	assertLoadedTree(t, draft, *result.Aggregate.Draft)
	require.Nil(t, result.Aggregate.Publication)
	require.Nil(t, result.Aggregate.Community)
	require.Nil(t, result.Aggregate.DeletedAt)

	require.Equal(t, int64(1), accountVersion(t, ownerUID))
}

func TestCreateOwnedChecklistRepositoryRejectsWrongPrecondition(t *testing.T) {
	ownerUID := newUserPmcsTestUser(t)
	repository := newOwnedRepository()
	checklistID := uuid.New()

	_, err := repository.Create(
		context.Background(),
		ownerUID,
		checklistID,
		preparedTree(t, uuid.New()),
		shared.Precondition{
			Mode: shared.PreconditionMatch,
			ETag: `"not-a-create-condition"`,
		},
	)

	requireAPIIntegrationError(t, err, 412, "stale_precondition")
	require.Equal(t, int64(0), accountVersion(t, ownerUID))
	require.Equal(t, 0, checklistCount(t, checklistID))
}

func TestCreateOwnedChecklistIdempotentRetryDoesNotConsumeVersions(t *testing.T) {
	ownerUID := newUserPmcsTestUser(t)
	repository := newOwnedRepository()
	checklistID := uuid.New()
	draft := preparedTree(t, uuid.New())

	created, err := repository.Create(
		context.Background(),
		ownerUID,
		checklistID,
		draft,
		shared.Precondition{Mode: shared.PreconditionCreate},
	)
	require.NoError(t, err)

	retried, err := repository.Create(
		context.Background(),
		ownerUID,
		checklistID,
		draft,
		shared.Precondition{Mode: shared.PreconditionCreate},
	)

	require.NoError(t, err)
	require.False(t, retried.Created)
	require.True(t, retried.Idempotent)
	require.Equal(t, created.Aggregate.SyncVersion, retried.Aggregate.SyncVersion)
	require.Equal(
		t,
		created.Aggregate.AccountChangeVersion,
		retried.Aggregate.AccountChangeVersion,
	)
	require.Equal(t, int64(1), accountVersion(t, ownerUID))
}

func TestCreateOwnedChecklistRejectsDifferentRetryWithoutVersionConsumption(t *testing.T) {
	ownerUID := newUserPmcsTestUser(t)
	repository := newOwnedRepository()
	checklistID := uuid.New()
	draft := preparedTree(t, uuid.New())
	_, err := repository.Create(
		context.Background(),
		ownerUID,
		checklistID,
		draft,
		shared.Precondition{Mode: shared.PreconditionCreate},
	)
	require.NoError(t, err)

	different := draft
	different.Input.Description = "different byte-exact metadata"
	different, err = shared.PrepareDraft(different.Input, shared.DefaultConfig())
	require.NoError(t, err)
	_, err = repository.Create(
		context.Background(),
		ownerUID,
		checklistID,
		different,
		shared.Precondition{Mode: shared.PreconditionCreate},
	)

	requireAPIIntegrationError(t, err, 412, "stale_precondition")
	require.Equal(t, int64(1), accountVersion(t, ownerUID))
}

func TestCreateOwnedChecklistRequiresInitializedAccount(t *testing.T) {
	repository := newOwnedRepository()
	_, err := repository.Create(
		context.Background(),
		"missing-"+uuid.NewString(),
		uuid.New(),
		preparedTree(t, uuid.New()),
		shared.Precondition{Mode: shared.PreconditionCreate},
	)

	requireAPIIntegrationError(t, err, 409, "account_not_initialized")
}

func TestCreateOwnedChecklistHidesCrossOwnerExistingID(t *testing.T) {
	firstOwner := newUserPmcsTestUser(t)
	secondOwner := newUserPmcsTestUser(t)
	repository := newOwnedRepository()
	checklistID := uuid.New()
	_, err := repository.Create(
		context.Background(),
		firstOwner,
		checklistID,
		preparedTree(t, uuid.New()),
		shared.Precondition{Mode: shared.PreconditionCreate},
	)
	require.NoError(t, err)

	_, err = repository.Create(
		context.Background(),
		secondOwner,
		checklistID,
		preparedTree(t, uuid.New()),
		shared.Precondition{Mode: shared.PreconditionCreate},
	)

	requireAPIIntegrationError(t, err, 404, "resource_not_found")
	require.Equal(t, int64(0), accountVersion(t, secondOwner))
}

func TestCreateOwnedChecklistConcurrentCrossOwnerCollisionStaysHidden(t *testing.T) {
	firstOwner := newUserPmcsTestUser(t)
	secondOwner := newUserPmcsTestUser(t)
	checklistID := uuid.New()
	start := make(chan struct{})
	drafts := map[string]shared.PreparedRevision{
		firstOwner:  preparedTree(t, uuid.New()),
		secondOwner: preparedTree(t, uuid.New()),
	}
	type createOutcome struct {
		ownerUID string
		err      error
	}
	outcomes := make(chan createOutcome, 2)
	var waitGroup sync.WaitGroup
	for _, ownerUID := range []string{firstOwner, secondOwner} {
		waitGroup.Add(1)
		go func(uid string) {
			defer waitGroup.Done()
			<-start
			_, err := newOwnedRepository().Create(
				context.Background(),
				uid,
				checklistID,
				drafts[uid],
				shared.Precondition{Mode: shared.PreconditionCreate},
			)
			outcomes <- createOutcome{ownerUID: uid, err: err}
		}(ownerUID)
	}
	close(start)
	waitGroup.Wait()
	close(outcomes)

	var successCount, hiddenCount int
	for outcome := range outcomes {
		if outcome.err == nil {
			successCount++
			require.Equal(t, int64(1), accountVersion(t, outcome.ownerUID))
			continue
		}
		var apiError *shared.APIError
		require.ErrorAs(t, outcome.err, &apiError)
		require.Equal(t, 404, apiError.Status)
		require.Equal(t, "resource_not_found", apiError.Code)
		hiddenCount++
		require.Equal(t, int64(0), accountVersion(t, outcome.ownerUID))
	}
	require.Equal(t, 1, successCount)
	require.Equal(t, 1, hiddenCount)
}

func TestCreateOwnedChecklistEnforcesActiveCeiling(t *testing.T) {
	ownerUID := newUserPmcsTestUser(t)
	config := shared.DefaultConfig()
	config.MaxOwnedChecklists = 1
	repository := owned.NewRepository(
		persistence.NewStore(testDB, config.TransactionMaxAttempts),
		config,
	)
	_, err := repository.Create(
		context.Background(),
		ownerUID,
		uuid.New(),
		preparedTree(t, uuid.New()),
		shared.Precondition{Mode: shared.PreconditionCreate},
	)
	require.NoError(t, err)

	_, err = repository.Create(
		context.Background(),
		ownerUID,
		uuid.New(),
		preparedTree(t, uuid.New()),
		shared.Precondition{Mode: shared.PreconditionCreate},
	)

	requireAPIIntegrationError(t, err, 413, "content_too_large")
	require.Equal(t, int64(1), accountVersion(t, ownerUID))
}

func TestGetOwnedChecklistReturnsCompleteAggregateAndHidesCrossOwner(t *testing.T) {
	ownerUID := newUserPmcsTestUser(t)
	otherUID := newUserPmcsTestUser(t)
	repository := newOwnedRepository()
	checklistID := uuid.New()
	draft := preparedTree(t, uuid.New())
	_, err := repository.Create(
		context.Background(),
		ownerUID,
		checklistID,
		draft,
		shared.Precondition{Mode: shared.PreconditionCreate},
	)
	require.NoError(t, err)

	aggregate, err := repository.Get(context.Background(), ownerUID, checklistID)
	require.NoError(t, err)
	require.NotNil(t, aggregate.Draft)
	assertLoadedTree(t, draft, *aggregate.Draft)

	_, err = repository.Get(context.Background(), otherUID, checklistID)
	requireAPIIntegrationError(t, err, 404, "resource_not_found")
}

func TestGetOwnedChecklistRequiresInitializedAccount(t *testing.T) {
	repository := newOwnedRepository()

	_, err := repository.Get(
		context.Background(),
		"missing-"+uuid.NewString(),
		uuid.New(),
	)

	requireAPIIntegrationError(t, err, 409, "account_not_initialized")
}

func TestGetOwnedChecklistTombstoneContainsNoAuthoredContent(t *testing.T) {
	ownerUID := newUserPmcsTestUser(t)
	repository := newOwnedRepository()
	checklistID := uuid.New()
	_, err := repository.Create(
		context.Background(),
		ownerUID,
		checklistID,
		preparedTree(t, uuid.New()),
		shared.Precondition{Mode: shared.PreconditionCreate},
	)
	require.NoError(t, err)
	_, err = testDB.ExecContext(
		context.Background(),
		`UPDATE user_pmcs_checklists SET deleted_at = now() WHERE id = $1`,
		checklistID,
	)
	require.NoError(t, err)

	aggregate, err := repository.Get(
		context.Background(),
		ownerUID,
		checklistID,
	)

	require.NoError(t, err)
	require.NotNil(t, aggregate.DeletedAt)
	require.Nil(t, aggregate.Draft)
	require.Nil(t, aggregate.Publication)
	require.Nil(t, aggregate.Community)
}

func TestPutDraftReplacesDraftUUIDAndIncrementsVersionsOnce(t *testing.T) {
	ownerUID := newUserPmcsTestUser(t)
	repository := newOwnedRepository()
	checklistID := uuid.New()
	initial := preparedTree(t, uuid.New())
	created, err := repository.Create(
		context.Background(),
		ownerUID,
		checklistID,
		initial,
		shared.Precondition{Mode: shared.PreconditionCreate},
	)
	require.NoError(t, err)
	replacement := preparedTree(t, uuid.New())

	result, err := repository.PutDraft(
		context.Background(),
		ownerUID,
		checklistID,
		replacement,
		shared.Precondition{
			Mode: shared.PreconditionMatch,
			ETag: shared.MakeChecklistETag(
				checklistID,
				created.Aggregate.SyncVersion,
			),
		},
	)

	require.NoError(t, err)
	require.Equal(t, int64(2), result.Aggregate.SyncVersion)
	require.Equal(t, int64(2), result.Aggregate.AccountChangeVersion)
	require.NotNil(t, result.Aggregate.Draft)
	assertLoadedTree(t, replacement, *result.Aggregate.Draft)
	require.Equal(t, int64(2), accountVersion(t, ownerUID))

	var oldCount int
	require.NoError(t, testDB.QueryRowContext(
		context.Background(),
		`SELECT count(*) FROM user_pmcs_revisions WHERE id = $1`,
		initial.Input.ID,
	).Scan(&oldCount))
	require.Zero(t, oldCount)
}

func TestPutDraftStalePreconditionConsumesNoVersions(t *testing.T) {
	ownerUID := newUserPmcsTestUser(t)
	repository := newOwnedRepository()
	checklistID := uuid.New()
	created, err := repository.Create(
		context.Background(),
		ownerUID,
		checklistID,
		preparedTree(t, uuid.New()),
		shared.Precondition{Mode: shared.PreconditionCreate},
	)
	require.NoError(t, err)

	_, err = repository.PutDraft(
		context.Background(),
		ownerUID,
		checklistID,
		preparedTree(t, uuid.New()),
		shared.Precondition{
			Mode: shared.PreconditionMatch,
			ETag: shared.MakeChecklistETag(checklistID, 99),
		},
	)

	requireAPIIntegrationError(t, err, 412, "stale_precondition")
	require.Equal(t, created.Aggregate.SyncVersion, checklistVersion(t, checklistID))
	require.Equal(t, int64(1), accountVersion(t, ownerUID))
}

func TestPutDraftRejectsUUIDGraftFromReplacedDraft(t *testing.T) {
	ownerUID := newUserPmcsTestUser(t)
	repository := newOwnedRepository()
	checklistID := uuid.New()
	initial := preparedTree(t, uuid.New())
	created, err := repository.Create(
		context.Background(),
		ownerUID,
		checklistID,
		initial,
		shared.Precondition{Mode: shared.PreconditionCreate},
	)
	require.NoError(t, err)
	replacementInput := preparedTree(t, uuid.New()).Input
	replacementInput.Sections[0].ID = initial.Input.Sections[0].ID
	replacement, err := shared.PrepareDraft(
		replacementInput,
		shared.DefaultConfig(),
	)
	require.NoError(t, err)

	_, err = repository.PutDraft(
		context.Background(),
		ownerUID,
		checklistID,
		replacement,
		shared.Precondition{
			Mode: shared.PreconditionMatch,
			ETag: shared.MakeChecklistETag(
				checklistID,
				created.Aggregate.SyncVersion,
			),
		},
	)

	requireAPIIntegrationError(t, err, 422, "validation_failed")
	require.Equal(t, int64(1), accountVersion(t, ownerUID))
	aggregate, getErr := repository.Get(
		context.Background(),
		ownerUID,
		checklistID,
	)
	require.NoError(t, getErr)
	require.Equal(t, initial.Input.ID, aggregate.Draft.ID)
	require.Equal(t, initial.Input.Sections[0].ID, aggregate.Draft.Sections[0].ID)
}

func TestDeleteDraftWithoutPublicationReturnsInvalidTransition(t *testing.T) {
	ownerUID := newUserPmcsTestUser(t)
	repository := newOwnedRepository()
	checklistID := uuid.New()
	draft := preparedTree(t, uuid.New())
	created, err := repository.Create(
		context.Background(),
		ownerUID,
		checklistID,
		draft,
		shared.Precondition{Mode: shared.PreconditionCreate},
	)
	require.NoError(t, err)

	_, err = repository.DeleteDraft(
		context.Background(),
		ownerUID,
		checklistID,
		draft.Input.ID,
		shared.Precondition{
			Mode: shared.PreconditionMatch,
			ETag: shared.MakeChecklistETag(
				checklistID,
				created.Aggregate.SyncVersion,
			),
		},
	)

	requireAPIIntegrationError(t, err, 409, "invalid_transition")
	require.Equal(t, int64(1), accountVersion(t, ownerUID))
	require.Equal(t, int64(1), checklistVersion(t, checklistID))
}

func TestDeleteDraftWithPublicationIncrementsVersionsOnce(t *testing.T) {
	ownerUID := newUserPmcsTestUser(t)
	repository := newOwnedRepository()
	checklistID := uuid.New()
	draft := preparedTree(t, uuid.New())
	created, err := repository.Create(
		context.Background(),
		ownerUID,
		checklistID,
		draft,
		shared.Precondition{Mode: shared.PreconditionCreate},
	)
	require.NoError(t, err)
	publicationID := uuid.New()
	_, err = testDB.ExecContext(
		context.Background(),
		`INSERT INTO user_pmcs_revisions
		     (id, checklist_id, state, revision_number, name, description,
		      content_hash, published_at)
		 VALUES ($1, $2, 'published', 1, 'Published', '', $3, now())`,
		publicationID,
		checklistID,
		make([]byte, 32),
	)
	require.NoError(t, err)

	result, err := repository.DeleteDraft(
		context.Background(),
		ownerUID,
		checklistID,
		draft.Input.ID,
		shared.Precondition{
			Mode: shared.PreconditionMatch,
			ETag: shared.MakeChecklistETag(
				checklistID,
				created.Aggregate.SyncVersion,
			),
		},
	)

	require.NoError(t, err)
	require.Nil(t, result.Aggregate.Draft)
	require.NotNil(t, result.Aggregate.Publication)
	require.Equal(t, publicationID, result.Aggregate.Publication.ID)
	require.Equal(t, int64(2), result.Aggregate.SyncVersion)
	require.Equal(t, int64(2), result.Aggregate.AccountChangeVersion)
	require.Equal(t, int64(2), accountVersion(t, ownerUID))
}

func newOwnedRepository() owned.Repository {
	config := shared.DefaultConfig()
	return owned.NewRepository(
		persistence.NewStore(testDB, config.TransactionMaxAttempts),
		config,
	)
}

func accountVersion(t *testing.T, ownerUID string) int64 {
	t.Helper()
	var version int64
	err := testDB.QueryRowContext(
		context.Background(),
		`SELECT COALESCE(
		     (SELECT current_version
		      FROM user_pmcs_sync_state
		      WHERE user_uid = $1),
		     0
		 )`,
		ownerUID,
	).Scan(&version)
	require.NoError(t, err)
	return version
}

func checklistVersion(t *testing.T, checklistID uuid.UUID) int64 {
	t.Helper()
	var version int64
	require.NoError(t, testDB.QueryRowContext(
		context.Background(),
		`SELECT sync_version FROM user_pmcs_checklists WHERE id = $1`,
		checklistID,
	).Scan(&version))
	return version
}

func checklistCount(t *testing.T, checklistID uuid.UUID) int {
	t.Helper()
	var count int
	require.NoError(t, testDB.QueryRowContext(
		context.Background(),
		`SELECT count(*) FROM user_pmcs_checklists WHERE id = $1`,
		checklistID,
	).Scan(&count))
	return count
}

func requireAPIIntegrationError(
	t *testing.T,
	err error,
	status int,
	code string,
) {
	t.Helper()
	var apiError *shared.APIError
	require.True(t, errors.As(err, &apiError), "error = %v", err)
	require.Equal(t, status, apiError.Status)
	require.Equal(t, code, apiError.Code)
}
