package user_pmcs_test

import (
	"bytes"
	"context"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/stretchr/testify/require"

	"miltechserver/api/user_pmcs/owned"
	"miltechserver/api/user_pmcs/persistence"
	"miltechserver/api/user_pmcs/shared"
	"miltechserver/bootstrap"
)

func TestPublishPromotesDraftWithExactClientIdentityAndNumber(t *testing.T) {
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

	publication := preparePublication(t, draft.Input, 1)
	result, err := repository.Publish(
		context.Background(),
		ownerUID,
		checklistID,
		publication,
		checklistPrecondition(checklistID, created.Aggregate.SyncVersion),
	)

	require.NoError(t, err)
	require.Nil(t, result.Aggregate.Draft)
	require.NotNil(t, result.Aggregate.Publication)
	require.Equal(t, draft.Input.ID, result.Aggregate.Publication.ID)
	require.Equal(t, int32(1), *result.Aggregate.Publication.RevisionNumber)
	require.Equal(t, "published", result.Aggregate.Publication.State)
	require.Equal(t, int64(2), result.Aggregate.SyncVersion)
	require.Equal(t, int64(2), accountVersion(t, ownerUID))
}

func TestPublishRejectsRevisionNumberGapWithoutVersionConsumption(t *testing.T) {
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

	_, err = repository.Publish(
		context.Background(),
		ownerUID,
		checklistID,
		preparePublication(t, draft.Input, 3),
		checklistPrecondition(checklistID, created.Aggregate.SyncVersion),
	)

	requireAPIIntegrationError(t, err, 409, "invalid_transition")
	require.Equal(t, int64(1), checklistVersion(t, checklistID))
	require.Equal(t, int64(1), accountVersion(t, ownerUID))
}

func TestPublishWithoutPriorDraftSupersedesImmutableHistory(t *testing.T) {
	ownerUID := newUserPmcsTestUser(t)
	repository := newOwnedRepository()
	checklistID := uuid.New()
	first := preparedTree(t, uuid.New())
	created, err := repository.Create(
		context.Background(),
		ownerUID,
		checklistID,
		first,
		shared.Precondition{Mode: shared.PreconditionCreate},
	)
	require.NoError(t, err)
	firstPublication := preparePublication(t, first.Input, 1)
	publishedFirst, err := repository.Publish(
		context.Background(),
		ownerUID,
		checklistID,
		firstPublication,
		checklistPrecondition(checklistID, created.Aggregate.SyncVersion),
	)
	require.NoError(t, err)
	firstUpdatedAt := publishedFirst.Aggregate.Publication.UpdatedAt

	second := preparedTree(t, uuid.New())
	second.Input.Name = "Second publication"
	secondPublication := preparePublication(t, second.Input, 2)
	publishedSecond, err := repository.Publish(
		context.Background(),
		ownerUID,
		checklistID,
		secondPublication,
		checklistPrecondition(
			checklistID,
			publishedFirst.Aggregate.SyncVersion,
		),
	)

	require.NoError(t, err)
	require.Equal(t, second.Input.ID, publishedSecond.Aggregate.Publication.ID)
	historical, err := repository.GetRevision(
		context.Background(),
		ownerUID,
		checklistID,
		first.Input.ID,
	)
	require.NoError(t, err)
	require.Equal(t, "superseded", revisionState(t, first.Input.ID))
	require.Equal(t, first.Input.Name, historical.Revision.Name)
	require.Equal(
		t,
		first.Input.Sections[0].ID,
		historical.Revision.Sections[0].ID,
	)
	require.Equal(t, int32(1), *historical.Revision.RevisionNumber)
	require.Equal(t, firstUpdatedAt, historical.Revision.UpdatedAt)
}

func TestPublishReplacesDifferentCurrentDraftBeforePromotion(t *testing.T) {
	ownerUID := newUserPmcsTestUser(t)
	repository := newOwnedRepository()
	checklistID := uuid.New()
	currentDraft := preparedTree(t, uuid.New())
	created, err := repository.Create(
		context.Background(), ownerUID, checklistID, currentDraft,
		shared.Precondition{Mode: shared.PreconditionCreate},
	)
	require.NoError(t, err)
	submitted := preparedTree(t, uuid.New())
	submitted.Input.Name = "Offline publication"
	publication := preparePublication(t, submitted.Input, 1)

	result, err := repository.Publish(
		context.Background(), ownerUID, checklistID, publication,
		checklistPrecondition(checklistID, created.Aggregate.SyncVersion),
	)

	require.NoError(t, err)
	require.Nil(t, result.Aggregate.Draft)
	require.Equal(t, submitted.Input.ID, result.Aggregate.Publication.ID)
	require.Equal(t, submitted.Input.Name, result.Aggregate.Publication.Name)
	require.Equal(t, int32(1), *result.Aggregate.Publication.RevisionNumber)
	require.Equal(t, 0, revisionCount(t, currentDraft.Input.ID))
}

func TestPublishIdempotentRetryAfterSupersessionConsumesNoVersion(t *testing.T) {
	ownerUID := newUserPmcsTestUser(t)
	repository := newOwnedRepository()
	checklistID := uuid.New()
	first := preparedTree(t, uuid.New())
	created, err := repository.Create(
		context.Background(), ownerUID, checklistID, first,
		shared.Precondition{Mode: shared.PreconditionCreate},
	)
	require.NoError(t, err)
	firstPublication := preparePublication(t, first.Input, 1)
	firstResult, err := repository.Publish(
		context.Background(), ownerUID, checklistID, firstPublication,
		checklistPrecondition(checklistID, created.Aggregate.SyncVersion),
	)
	require.NoError(t, err)
	second := preparePublication(t, preparedTree(t, uuid.New()).Input, 2)
	secondResult, err := repository.Publish(
		context.Background(), ownerUID, checklistID, second,
		checklistPrecondition(checklistID, firstResult.Aggregate.SyncVersion),
	)
	require.NoError(t, err)

	retried, err := repository.Publish(
		context.Background(), ownerUID, checklistID, firstPublication,
		checklistPrecondition(checklistID, created.Aggregate.SyncVersion),
	)

	require.NoError(t, err)
	require.True(t, retried.Idempotent)
	require.Equal(t, secondResult.Aggregate, retried.Aggregate)
	require.Equal(t, secondResult.Aggregate.SyncVersion, retried.Aggregate.SyncVersion)
	require.Equal(t, int64(3), accountVersion(t, ownerUID))
}

func TestPublishImmediateLostResponseRetryAcceptsOriginalParentETag(t *testing.T) {
	ownerUID := newUserPmcsTestUser(t)
	repository := newOwnedRepository()
	checklistID := uuid.New()
	draft := preparedTree(t, uuid.New())
	created, err := repository.Create(
		context.Background(), ownerUID, checklistID, draft,
		shared.Precondition{Mode: shared.PreconditionCreate},
	)
	require.NoError(t, err)
	publication := preparePublication(t, draft.Input, 1)
	published, err := repository.Publish(
		context.Background(), ownerUID, checklistID, publication,
		checklistPrecondition(checklistID, created.Aggregate.SyncVersion),
	)
	require.NoError(t, err)

	retried, err := repository.Publish(
		context.Background(), ownerUID, checklistID, publication,
		checklistPrecondition(checklistID, created.Aggregate.SyncVersion),
	)

	require.NoError(t, err)
	require.True(t, retried.Idempotent)
	require.Equal(t, published.Aggregate, retried.Aggregate)
	require.Equal(t, int64(2), checklistVersion(t, checklistID))
	require.Equal(t, int64(2), accountVersion(t, ownerUID))
}

func TestPublishDivergentHistoricalRetryConsumesNoVersion(t *testing.T) {
	ownerUID := newUserPmcsTestUser(t)
	repository := newOwnedRepository()
	checklistID := uuid.New()
	first := preparedTree(t, uuid.New())
	created, err := repository.Create(
		context.Background(), ownerUID, checklistID, first,
		shared.Precondition{Mode: shared.PreconditionCreate},
	)
	require.NoError(t, err)
	firstPublication := preparePublication(t, first.Input, 1)
	_, err = repository.Publish(
		context.Background(), ownerUID, checklistID, firstPublication,
		checklistPrecondition(checklistID, created.Aggregate.SyncVersion),
	)
	require.NoError(t, err)

	divergentInput := firstPublication.Input
	divergentInput.Description = "divergent immutable content"
	divergent := preparePublication(t, divergentInput, 1)
	_, err = repository.Publish(
		context.Background(), ownerUID, checklistID, divergent,
		checklistPrecondition(checklistID, created.Aggregate.SyncVersion),
	)

	requireAPIIntegrationError(t, err, 412, "stale_precondition")
	require.Equal(t, int64(2), checklistVersion(t, checklistID))
	require.Equal(t, int64(2), accountVersion(t, ownerUID))

	reusedNumber := firstPublication
	number := int32(2)
	reusedNumber.Input.RevisionNumber = &number
	_, err = repository.Publish(
		context.Background(), ownerUID, checklistID, reusedNumber,
		checklistPrecondition(checklistID, created.Aggregate.SyncVersion),
	)
	requireAPIIntegrationError(t, err, 409, "invalid_transition")
	require.Equal(t, int64(2), checklistVersion(t, checklistID))
	require.Equal(t, int64(2), accountVersion(t, ownerUID))
}

func TestPublishConcurrentExactNextAttemptsSerialize(t *testing.T) {
	ownerUID := newUserPmcsTestUser(t)
	repository := newOwnedRepository()
	checklistID := uuid.New()
	initial := preparedTree(t, uuid.New())
	created, err := repository.Create(
		context.Background(), ownerUID, checklistID, initial,
		shared.Precondition{Mode: shared.PreconditionCreate},
	)
	require.NoError(t, err)
	first, err := repository.Publish(
		context.Background(), ownerUID, checklistID,
		preparePublication(t, initial.Input, 1),
		checklistPrecondition(checklistID, created.Aggregate.SyncVersion),
	)
	require.NoError(t, err)

	inputs := []shared.PreparedRevision{
		preparePublication(t, preparedTree(t, uuid.New()).Input, 2),
		preparePublication(t, preparedTree(t, uuid.New()).Input, 2),
	}
	start := make(chan struct{})
	errorsChannel := make(chan error, len(inputs))
	var waitGroup sync.WaitGroup
	for _, input := range inputs {
		input := input
		waitGroup.Add(1)
		go func() {
			defer waitGroup.Done()
			<-start
			_, publishErr := repository.Publish(
				context.Background(), ownerUID, checklistID, input,
				checklistPrecondition(
					checklistID,
					first.Aggregate.SyncVersion,
				),
			)
			errorsChannel <- publishErr
		}()
	}
	close(start)
	waitGroup.Wait()
	close(errorsChannel)

	var successes, stale int
	for publishErr := range errorsChannel {
		if publishErr == nil {
			successes++
			continue
		}
		var apiError *shared.APIError
		require.ErrorAs(t, publishErr, &apiError)
		require.Equal(t, "stale_precondition", apiError.Code)
		stale++
	}
	require.Equal(t, 1, successes)
	require.Equal(t, 1, stale)
	require.Equal(t, int64(3), accountVersion(t, ownerUID))
}

func TestHistoricalOwnerOnlyAndDraftHidden(t *testing.T) {
	ownerUID := newUserPmcsTestUser(t)
	otherUID := newUserPmcsTestUser(t)
	repository := newOwnedRepository()
	checklistID := uuid.New()
	draft := preparedTree(t, uuid.New())
	created, err := repository.Create(
		context.Background(), ownerUID, checklistID, draft,
		shared.Precondition{Mode: shared.PreconditionCreate},
	)
	require.NoError(t, err)

	_, err = repository.GetRevision(
		context.Background(), ownerUID, checklistID, draft.Input.ID,
	)
	requireAPIIntegrationError(t, err, 404, "resource_not_found")
	published, err := repository.Publish(
		context.Background(), ownerUID, checklistID,
		preparePublication(t, draft.Input, 1),
		checklistPrecondition(checklistID, created.Aggregate.SyncVersion),
	)
	require.NoError(t, err)
	_, err = repository.GetRevision(
		context.Background(), otherUID, checklistID, draft.Input.ID,
	)
	requireAPIIntegrationError(t, err, 404, "resource_not_found")
	require.Equal(t, int64(0), accountVersion(t, otherUID))
	require.Equal(t, int64(2), published.Aggregate.SyncVersion)
}

func TestHistoricalHTTPBodyAndETagRemainByteExactAfterSupersession(t *testing.T) {
	ownerUID := newUserPmcsTestUser(t)
	config := shared.DefaultConfig()
	repository := owned.NewRepository(
		persistence.NewStore(testDB, config.TransactionMaxAttempts),
		config,
	)
	service := owned.NewService(repository, config)
	router := historicalTestRouter(ownerUID, service, config)
	checklistID := uuid.New()
	first := preparedTree(t, uuid.New())
	created, err := repository.Create(
		context.Background(), ownerUID, checklistID, first,
		shared.Precondition{Mode: shared.PreconditionCreate},
	)
	require.NoError(t, err)
	firstPublication := preparePublication(t, first.Input, 1)
	publishedFirst, err := repository.Publish(
		context.Background(), ownerUID, checklistID, firstPublication,
		checklistPrecondition(checklistID, created.Aggregate.SyncVersion),
	)
	require.NoError(t, err)

	path := "/api/v1/auth/user-pmcs/checklists/" + checklistID.String() +
		"/revisions/" + first.Input.ID.String()
	before := httptest.NewRecorder()
	router.ServeHTTP(
		before,
		httptest.NewRequest(http.MethodGet, path, nil),
	)
	require.Equal(t, http.StatusOK, before.Code)
	beforeBody := bytes.Clone(before.Body.Bytes())
	beforeETag := before.Header().Get("ETag")
	require.NotEmpty(t, beforeETag)
	require.NotContains(t, string(beforeBody), `"state"`)
	require.NotContains(t, string(beforeBody), ownerUID)

	second := preparePublication(t, preparedTree(t, uuid.New()).Input, 2)
	_, err = repository.Publish(
		context.Background(), ownerUID, checklistID, second,
		checklistPrecondition(
			checklistID,
			publishedFirst.Aggregate.SyncVersion,
		),
	)
	require.NoError(t, err)

	after := httptest.NewRecorder()
	router.ServeHTTP(
		after,
		httptest.NewRequest(http.MethodGet, path, nil),
	)
	require.Equal(t, http.StatusOK, after.Code)
	require.Equal(t, beforeBody, after.Body.Bytes())
	require.Equal(t, beforeETag, after.Header().Get("ETag"))

	conditionalRequest := httptest.NewRequest(http.MethodGet, path, nil)
	conditionalRequest.Header.Set("If-None-Match", beforeETag)
	notModified := httptest.NewRecorder()
	router.ServeHTTP(notModified, conditionalRequest)
	require.Equal(t, http.StatusNotModified, notModified.Code)
	require.Empty(t, notModified.Body.Bytes())
	require.Equal(t, beforeETag, notModified.Header().Get("ETag"))
	require.Equal(
		t,
		"private, max-age=31536000, immutable",
		notModified.Header().Get("Cache-Control"),
	)
}

func TestOfflineReplayResumesWithExactUUIDsAndNumbers(t *testing.T) {
	ownerUID := newUserPmcsTestUser(t)
	repository := newOwnedRepository()
	checklistID := uuid.New()
	local := []shared.PreparedRevision{
		preparedTree(t, uuid.New()),
		preparedTree(t, uuid.New()),
		preparedTree(t, uuid.New()),
	}
	created, err := repository.Create(
		context.Background(), ownerUID, checklistID, local[0],
		shared.Precondition{Mode: shared.PreconditionCreate},
	)
	require.NoError(t, err)
	version := created.Aggregate.SyncVersion

	for index := 0; index < 1; index++ {
		published, publishErr := repository.Publish(
			context.Background(), ownerUID, checklistID,
			preparePublication(t, local[index].Input, int32(index+1)),
			checklistPrecondition(checklistID, version),
		)
		require.NoError(t, publishErr)
		version = published.Aggregate.SyncVersion
	}

	interruptedContext, cancel := context.WithCancel(context.Background())
	cancel()
	_, err = repository.Publish(
		interruptedContext, ownerUID, checklistID,
		preparePublication(t, local[1].Input, 2),
		checklistPrecondition(checklistID, version),
	)
	require.ErrorIs(t, err, context.Canceled)

	current, err := repository.Get(context.Background(), ownerUID, checklistID)
	require.NoError(t, err)
	require.Equal(t, local[0].Input.ID, current.Publication.ID)
	require.Equal(t, int32(1), *current.Publication.RevisionNumber)

	version = current.SyncVersion
	var final *owned.MutationResult
	for index := 1; index < len(local); index++ {
		final, err = repository.Publish(
			context.Background(), ownerUID, checklistID,
			preparePublication(t, local[index].Input, int32(index+1)),
			checklistPrecondition(checklistID, version),
		)
		require.NoError(t, err)
		version = final.Aggregate.SyncVersion
	}
	require.Equal(t, local[2].Input.ID, final.Aggregate.Publication.ID)
	require.Equal(t, int32(3), *final.Aggregate.Publication.RevisionNumber)
}

func checklistPrecondition(
	checklistID uuid.UUID,
	version int64,
) shared.Precondition {
	return shared.Precondition{
		Mode: shared.PreconditionMatch,
		ETag: shared.MakeChecklistETag(checklistID, version),
	}
}

func preparePublication(
	t *testing.T,
	input shared.RevisionInput,
	number int32,
) shared.PreparedRevision {
	t.Helper()
	input.RevisionNumber = &number
	prepared, err := shared.PreparePublication(input, shared.DefaultConfig())
	require.NoError(t, err)
	return prepared
}

func revisionCount(t *testing.T, revisionID uuid.UUID) int {
	t.Helper()
	var count int
	require.NoError(t, testDB.QueryRowContext(
		context.Background(),
		`SELECT count(*) FROM user_pmcs_revisions WHERE id = $1`,
		revisionID,
	).Scan(&count))
	return count
}

func revisionState(t *testing.T, revisionID uuid.UUID) string {
	t.Helper()
	var state string
	require.NoError(t, testDB.QueryRowContext(
		context.Background(),
		`SELECT state FROM user_pmcs_revisions WHERE id = $1`,
		revisionID,
	).Scan(&state))
	return state
}

func historicalTestRouter(
	ownerUID string,
	service owned.Service,
	config shared.Config,
) *gin.Engine {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.Use(func(context *gin.Context) {
		context.Set("user", &bootstrap.User{UserID: ownerUID})
		context.Next()
	})
	owned.RegisterRoutes(router.Group("/api/v1/auth"), service, config)
	return router
}
