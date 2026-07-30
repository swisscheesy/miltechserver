package user_pmcs_test

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/lib/pq"
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

func TestCreateOwnedChecklistConcurrentSameTableUUIDCollisionSerializes(t *testing.T) {
	sharedSectionID := uuid.New()
	firstDraft := preparedTree(t, uuid.New())
	firstDraft.Input.Sections[0].ID = sharedSectionID
	firstDraft = prepareOwnedDraft(t, firstDraft.Input)
	secondDraft := preparedTree(t, uuid.New())
	secondDraft.Input.Sections[0].ID = sharedSectionID
	secondDraft = prepareOwnedDraft(t, secondDraft.Input)

	assertConcurrentCreateUUIDCollision(
		t,
		firstDraft,
		secondDraft,
	)
}

func TestCreateOwnedChecklistConcurrentCrossTableUUIDCollisionSerializes(t *testing.T) {
	sharedNodeID := uuid.New()
	firstDraft := preparedTree(t, uuid.New())
	firstDraft.Input.Sections[0].ID = sharedNodeID
	firstDraft = prepareOwnedDraft(t, firstDraft.Input)
	secondDraft := preparedTree(t, uuid.New())
	secondDraft.Input.Sections[0].Items[0].ID = sharedNodeID
	secondDraft = prepareOwnedDraft(t, secondDraft.Input)

	assertConcurrentCreateUUIDCollision(
		t,
		firstDraft,
		secondDraft,
	)
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

func TestGetOwnedChecklistConcurrentPutReturnsOneRevisionSnapshot(t *testing.T) {
	ownerUID := newUserPmcsTestUser(t)
	repository := newOwnedRepository()
	checklistID := uuid.New()
	current := wideOwnedDraft(t, uuid.New(), 100, 20)
	created, err := repository.Create(
		context.Background(),
		ownerUID,
		checklistID,
		current,
		shared.Precondition{Mode: shared.PreconditionCreate},
	)
	require.NoError(t, err)
	replacement := preparedTree(t, uuid.New())
	releaseWriter := installChecklistUpdateBarrier(t, checklistID)
	type getOutcome struct {
		aggregate *shared.ChecklistAggregate
		err       error
	}
	type putOutcome struct {
		result *owned.MutationResult
		err    error
	}
	getResult := make(chan getOutcome, 1)
	putResult := make(chan putOutcome, 1)
	go func() {
		result, putErr := repository.PutDraft(
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
		putResult <- putOutcome{result: result, err: putErr}
	}()
	require.True(
		t,
		waitForDatabaseActivity(
			15*time.Second,
			"lower(wait_event) = 'advisory' AND query LIKE '%UPDATE user_pmcs_checklists%'",
		),
		"PutDraft did not reach the root-update barrier",
	)

	go func() {
		aggregate, getErr := repository.Get(
			context.Background(),
			ownerUID,
			checklistID,
		)
		getResult <- getOutcome{aggregate: aggregate, err: getErr}
	}()
	require.True(
		t,
		waitForDatabaseActivity(
			5*time.Second,
			"state = 'active' AND query LIKE '%FROM user_pmcs_%'",
		),
		"Get did not reach tree loading after reading the root",
	)
	releaseWriter()

	read := <-getResult
	written := <-putResult
	require.NoError(t, read.err)
	require.NoError(t, written.err)
	require.Equal(t, created.Aggregate.SyncVersion, read.aggregate.SyncVersion)
	require.NotNil(t, read.aggregate.Draft)
	require.Equal(t, current.Input.ID, read.aggregate.Draft.ID)
	require.Len(t, read.aggregate.Draft.Sections, len(current.Input.Sections))
	for _, section := range read.aggregate.Draft.Sections {
		require.Len(t, section.Items, 20)
		for _, item := range section.Items {
			require.Len(t, item.ProcedureSteps, 1)
		}
	}
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

func assertConcurrentCreateUUIDCollision(
	t *testing.T,
	firstDraft shared.PreparedRevision,
	secondDraft shared.PreparedRevision,
) {
	t.Helper()
	firstOwner := newUserPmcsTestUser(t)
	secondOwner := newUserPmcsTestUser(t)
	type createInput struct {
		ownerUID    string
		checklistID uuid.UUID
		draft       shared.PreparedRevision
	}
	inputs := []createInput{
		{
			ownerUID:    firstOwner,
			checklistID: uuid.New(),
			draft:       firstDraft,
		},
		{
			ownerUID:    secondOwner,
			checklistID: uuid.New(),
			draft:       secondDraft,
		},
	}
	type createOutcome struct {
		input  createInput
		result *owned.MutationResult
		err    error
	}
	start := make(chan struct{})
	outcomes := make(chan createOutcome, len(inputs))
	var waitGroup sync.WaitGroup
	for _, input := range inputs {
		input := input
		waitGroup.Add(1)
		go func() {
			defer waitGroup.Done()
			<-start
			result, err := newOwnedRepository().Create(
				context.Background(),
				input.ownerUID,
				input.checklistID,
				input.draft,
				shared.Precondition{Mode: shared.PreconditionCreate},
			)
			outcomes <- createOutcome{
				input:  input,
				result: result,
				err:    err,
			}
		}()
	}
	close(start)
	waitGroup.Wait()
	close(outcomes)

	var successCount, validationCount int
	for outcome := range outcomes {
		if outcome.err == nil {
			successCount++
			require.NotNil(t, outcome.result)
			require.Equal(t, int64(1), accountVersion(t, outcome.input.ownerUID))
			require.Equal(t, 1, checklistCount(t, outcome.input.checklistID))
			continue
		}
		var apiError *shared.APIError
		require.ErrorAs(t, outcome.err, &apiError)
		require.Equal(t, 422, apiError.Status)
		require.Equal(t, "validation_failed", apiError.Code)
		validationCount++
		require.Equal(t, int64(0), accountVersion(t, outcome.input.ownerUID))
		require.Equal(t, 0, checklistCount(t, outcome.input.checklistID))
	}
	require.Equal(t, 1, successCount)
	require.Equal(t, 1, validationCount)
}

func prepareOwnedDraft(
	t *testing.T,
	input shared.RevisionInput,
) shared.PreparedRevision {
	t.Helper()
	prepared, err := shared.PrepareDraft(input, shared.DefaultConfig())
	require.NoError(t, err)
	return prepared
}

func wideOwnedDraft(
	t *testing.T,
	revisionID uuid.UUID,
	sectionCount int,
	itemCount int,
) shared.PreparedRevision {
	t.Helper()
	input := validOwnedRevisionInput(revisionID)
	input.Sections = make([]shared.SectionInput, 0, sectionCount)
	for sectionIndex := 0; sectionIndex < sectionCount; sectionIndex++ {
		section := shared.SectionInput{
			ID:       uuid.New(),
			Position: int32(sectionIndex + 1),
			Title:    "Section",
			Items:    make([]shared.ItemInput, 0, itemCount),
		}
		for itemIndex := 0; itemIndex < itemCount; itemIndex++ {
			section.Items = append(section.Items, shared.ItemInput{
				ID:                        uuid.New(),
				Position:                  int32(itemIndex + 1),
				Interval:                  "Before",
				ItemToBeCheckedOrServiced: "Component",
				ProcedureSteps: []shared.ProcedureStepInput{
					{
						ID:       uuid.New(),
						Position: 1,
						StepText: "Inspect",
					},
				},
			})
		}
		input.Sections = append(input.Sections, section)
	}
	return prepareOwnedDraft(t, input)
}

func validOwnedRevisionInput(revisionID uuid.UUID) shared.RevisionInput {
	return shared.RevisionInput{
		ID:          revisionID,
		Name:        "Vehicle PMCS",
		Description: "Snapshot fixture",
		Models: []shared.ModelInput{
			{DisplayText: "M998"},
		},
	}
}

func installChecklistUpdateBarrier(
	t *testing.T,
	checklistID uuid.UUID,
) func() {
	t.Helper()
	ctx := context.Background()
	connection, err := testDB.Conn(ctx)
	require.NoError(t, err)

	lockKey := time.Now().UnixNano()
	_, err = connection.ExecContext(
		ctx,
		`SELECT pg_advisory_lock($1)`,
		lockKey,
	)
	require.NoError(t, err)

	suffix := strings.ReplaceAll(uuid.NewString(), "-", "")
	functionName := "user_pmcs_test_barrier_function_" + suffix
	triggerName := "user_pmcs_test_barrier_trigger_" + suffix
	quotedFunction := pq.QuoteIdentifier(functionName)
	quotedTrigger := pq.QuoteIdentifier(triggerName)
	_, err = testDB.ExecContext(
		ctx,
		fmt.Sprintf(
			`CREATE FUNCTION %s() RETURNS trigger
			 LANGUAGE plpgsql AS $barrier$
			 BEGIN
			   IF NEW.id = '%s'::uuid THEN
			     PERFORM pg_advisory_xact_lock(%d);
			   END IF;
			   RETURN NEW;
			 END
			 $barrier$`,
			quotedFunction,
			checklistID.String(),
			lockKey,
		),
	)
	require.NoError(t, err)
	_, err = testDB.ExecContext(
		ctx,
		fmt.Sprintf(
			`CREATE TRIGGER %s
			 BEFORE UPDATE ON user_pmcs_checklists
			 FOR EACH ROW EXECUTE FUNCTION %s()`,
			quotedTrigger,
			quotedFunction,
		),
	)
	require.NoError(t, err)

	var releaseOnce sync.Once
	release := func() {
		releaseOnce.Do(func() {
			_, unlockErr := connection.ExecContext(
				context.Background(),
				`SELECT pg_advisory_unlock($1)`,
				lockKey,
			)
			require.NoError(t, unlockErr)
		})
	}
	t.Cleanup(func() {
		release()
		_, dropTriggerErr := testDB.ExecContext(
			context.Background(),
			fmt.Sprintf(
				`DROP TRIGGER IF EXISTS %s ON user_pmcs_checklists`,
				quotedTrigger,
			),
		)
		require.NoError(t, dropTriggerErr)
		_, dropFunctionErr := testDB.ExecContext(
			context.Background(),
			fmt.Sprintf(`DROP FUNCTION IF EXISTS %s()`, quotedFunction),
		)
		require.NoError(t, dropFunctionErr)
		require.NoError(t, connection.Close())
	})
	return release
}

func waitForDatabaseActivity(
	timeout time.Duration,
	predicate string,
) bool {
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		var active bool
		err := testDB.QueryRowContext(
			context.Background(),
			`SELECT EXISTS (
			     SELECT 1
			     FROM pg_stat_activity
			     WHERE datname = current_database()
			       AND pid <> pg_backend_pid()
			       AND `+predicate+`
			 )`,
		).Scan(&active)
		if err == nil && active {
			return true
		}
	}
	return false
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
