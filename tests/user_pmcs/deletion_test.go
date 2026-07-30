package user_pmcs_test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/stretchr/testify/require"

	"miltechserver/api/user_pmcs/owned"
	"miltechserver/api/user_pmcs/shared"
	"miltechserver/bootstrap"
)

func TestDeleteChecklistPrivateRetainsOnlyPermanentTombstone(t *testing.T) {
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

	result, err := repository.DeleteChecklist(
		context.Background(),
		ownerUID,
		checklistID,
		checklistPrecondition(checklistID, created.Aggregate.SyncVersion),
	)

	require.NoError(t, err)
	require.False(t, result.Idempotent)
	require.Equal(t, int64(2), result.Aggregate.SyncVersion)
	require.Equal(t, int64(2), result.Aggregate.AccountChangeVersion)
	require.NotNil(t, result.Aggregate.DeletedAt)
	require.Nil(t, result.Aggregate.Draft)
	require.Nil(t, result.Aggregate.Publication)
	require.Nil(t, result.Aggregate.Community)
	require.Equal(t, int64(2), accountVersion(t, ownerUID))
	require.Equal(t, 0, deletionRowCount(t, "user_pmcs_revisions", "checklist_id", checklistID))
	require.Equal(t, 1, checklistCount(t, checklistID))
}

func TestDeleteChecklistReleasedWithoutPinsRemovesReleaseBeforeRevision(t *testing.T) {
	ownerUID := newUserPmcsTestUser(t)
	repository := newOwnedRepository()
	checklistID := uuid.New()
	publication := createPublishedChecklistForDeletion(
		t,
		repository,
		ownerUID,
		checklistID,
	)
	insertCommunityReleaseForDeletion(t, checklistID, publication.Aggregate.Publication.ID, 1)
	registerCommunityDeletionCleanup(t, checklistID)

	result, err := repository.DeleteChecklist(
		context.Background(),
		ownerUID,
		checklistID,
		checklistPrecondition(checklistID, publication.Aggregate.SyncVersion),
	)

	require.NoError(t, err)
	require.NotNil(t, result.Aggregate.DeletedAt)
	require.Nil(t, result.Aggregate.Community)
	require.Equal(t, 0, deletionRowCount(
		t,
		"user_pmcs_community_releases",
		"checklist_id",
		checklistID,
	))
	require.Equal(t, 0, deletionRowCount(t, "user_pmcs_revisions", "checklist_id", checklistID))
	requireCommunitySourceRetired(t, checklistID, 1)
}

func TestDeleteChecklistReleasedRetainsOnlyExplicitActiveSubscriberPins(t *testing.T) {
	ownerUID := newUserPmcsTestUser(t)
	subscriberUID := newUserPmcsTestUser(t)
	repository := newOwnedRepository()
	checklistID := uuid.New()
	first := createPublishedChecklistForDeletion(
		t,
		repository,
		ownerUID,
		checklistID,
	)
	firstRevisionID := first.Aggregate.Publication.ID
	insertCommunityReleaseForDeletion(t, checklistID, firstRevisionID, 1)
	registerCommunityDeletionCleanup(t, checklistID)

	secondDraft := preparedTree(t, uuid.New())
	secondDraftResult, err := repository.PutDraft(
		context.Background(),
		ownerUID,
		checklistID,
		secondDraft,
		checklistPrecondition(checklistID, first.Aggregate.SyncVersion),
	)
	require.NoError(t, err)
	second, err := repository.Publish(
		context.Background(),
		ownerUID,
		checklistID,
		preparePublication(t, secondDraft.Input, 2),
		checklistPrecondition(checklistID, secondDraftResult.Aggregate.SyncVersion),
	)
	require.NoError(t, err)
	secondRevisionID := second.Aggregate.Publication.ID
	insertNextCommunityReleaseForDeletion(t, checklistID, secondRevisionID, 2)
	insertActiveSubscriptionForDeletion(
		t,
		subscriberUID,
		checklistID,
		firstRevisionID,
	)

	unpublishedDraft := preparedTree(t, uuid.New())
	withDraft, err := repository.PutDraft(
		context.Background(),
		ownerUID,
		checklistID,
		unpublishedDraft,
		checklistPrecondition(checklistID, second.Aggregate.SyncVersion),
	)
	require.NoError(t, err)

	result, err := repository.DeleteChecklist(
		context.Background(),
		ownerUID,
		checklistID,
		checklistPrecondition(checklistID, withDraft.Aggregate.SyncVersion),
	)

	require.NoError(t, err)
	require.NotNil(t, result.Aggregate.DeletedAt)
	require.Equal(t, []uuid.UUID{firstRevisionID}, deletionUUIDs(
		t,
		`SELECT revision_id
		 FROM user_pmcs_community_releases
		 WHERE checklist_id = $1
		 ORDER BY revision_id`,
		checklistID,
	))
	require.Equal(t, []uuid.UUID{firstRevisionID}, deletionUUIDs(
		t,
		`SELECT id
		 FROM user_pmcs_revisions
		 WHERE checklist_id = $1
		 ORDER BY id`,
		checklistID,
	))
	require.Equal(t, 1, deletionRowCount(
		t,
		"user_pmcs_sections",
		"revision_id",
		firstRevisionID,
	))
	require.Equal(t, 0, deletionRowCount(
		t,
		"user_pmcs_sections",
		"revision_id",
		secondRevisionID,
	))
	require.Equal(t, 0, deletionRowCount(
		t,
		"user_pmcs_sections",
		"revision_id",
		unpublishedDraft.Input.ID,
	))
	requireCommunitySourceRetired(t, checklistID, 2)
	require.Equal(t, int64(6), result.Aggregate.SyncVersion)
	require.Equal(t, int64(6), accountVersion(t, ownerUID))
}

func TestDeleteChecklistRejectsStaleAndCrossOwnerWithoutVersionConsumption(t *testing.T) {
	ownerUID := newUserPmcsTestUser(t)
	otherUID := newUserPmcsTestUser(t)
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

	_, err = repository.DeleteChecklist(
		context.Background(),
		ownerUID,
		checklistID,
		checklistPrecondition(checklistID, created.Aggregate.SyncVersion+1),
	)
	requireAPIIntegrationError(t, err, 412, "stale_precondition")
	require.Equal(t, int64(1), accountVersion(t, ownerUID))
	require.Equal(t, int64(1), checklistVersion(t, checklistID))

	_, err = repository.DeleteChecklist(
		context.Background(),
		otherUID,
		checklistID,
		checklistPrecondition(checklistID, created.Aggregate.SyncVersion),
	)
	requireAPIIntegrationError(t, err, 404, "resource_not_found")
	require.Equal(t, int64(0), accountVersion(t, otherUID))
	require.Equal(t, int64(1), checklistVersion(t, checklistID))
}

func TestDeleteChecklistRepeatedOwnerDeleteIsIdempotent(t *testing.T) {
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
	originalPrecondition := checklistPrecondition(
		checklistID,
		created.Aggregate.SyncVersion,
	)
	deleted, err := repository.DeleteChecklist(
		context.Background(),
		ownerUID,
		checklistID,
		originalPrecondition,
	)
	require.NoError(t, err)

	repeated, err := repository.DeleteChecklist(
		context.Background(),
		ownerUID,
		checklistID,
		originalPrecondition,
	)

	require.NoError(t, err)
	require.True(t, repeated.Idempotent)
	require.Equal(t, deleted.Aggregate.SyncVersion, repeated.Aggregate.SyncVersion)
	require.Equal(
		t,
		deleted.Aggregate.AccountChangeVersion,
		repeated.Aggregate.AccountChangeVersion,
	)
	require.Equal(t, deleted.Aggregate.DeletedAt, repeated.Aggregate.DeletedAt)
	require.Equal(t, int64(2), accountVersion(t, ownerUID))
}

func TestDeleteChecklistTombstoneDefeatsLaterCreateDraftAndPublication(t *testing.T) {
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
	deleted, err := repository.DeleteChecklist(
		context.Background(),
		ownerUID,
		checklistID,
		checklistPrecondition(checklistID, created.Aggregate.SyncVersion),
	)
	require.NoError(t, err)

	_, err = repository.Create(
		context.Background(),
		ownerUID,
		checklistID,
		preparedTree(t, uuid.New()),
		shared.Precondition{Mode: shared.PreconditionCreate},
	)
	requireAPIIntegrationError(t, err, 412, "stale_precondition")

	currentPrecondition := checklistPrecondition(
		checklistID,
		deleted.Aggregate.SyncVersion,
	)
	_, err = repository.PutDraft(
		context.Background(),
		ownerUID,
		checklistID,
		preparedTree(t, uuid.New()),
		currentPrecondition,
	)
	requireAPIIntegrationError(t, err, 412, "stale_precondition")

	publication := preparedTree(t, uuid.New())
	_, err = repository.Publish(
		context.Background(),
		ownerUID,
		checklistID,
		preparePublication(t, publication.Input, 1),
		currentPrecondition,
	)
	requireAPIIntegrationError(t, err, 412, "stale_precondition")

	aggregate, err := repository.Get(context.Background(), ownerUID, checklistID)
	require.NoError(t, err)
	require.NotNil(t, aggregate.DeletedAt)
	require.Nil(t, aggregate.Draft)
	require.Nil(t, aggregate.Publication)
	require.Nil(t, aggregate.Community)
	require.Equal(t, int64(2), accountVersion(t, ownerUID))
	require.Equal(t, 0, deletionRowCount(t, "user_pmcs_revisions", "checklist_id", checklistID))
}

func TestDeleteChecklistHTTPReturnsVersionedPrivateTombstone(t *testing.T) {
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

	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.Use(func(context *gin.Context) {
		context.Set("user", &bootstrap.User{UserID: ownerUID})
		context.Next()
	})
	owned.RegisterRoutes(
		router.Group("/api/v1/auth"),
		owned.NewService(repository, shared.DefaultConfig()),
		shared.DefaultConfig(),
	)
	request := httptest.NewRequest(
		http.MethodDelete,
		"/api/v1/auth/user-pmcs/checklists/"+checklistID.String(),
		nil,
	)
	request.Header.Set(
		"If-Match",
		shared.MakeChecklistETag(checklistID, created.Aggregate.SyncVersion),
	)
	response := httptest.NewRecorder()

	router.ServeHTTP(response, request)

	require.Equal(t, http.StatusOK, response.Code)
	require.Equal(
		t,
		shared.MakeChecklistETag(checklistID, 2),
		response.Header().Get("ETag"),
	)
	require.Equal(
		t,
		"private, max-age=31536000, immutable",
		response.Header().Get("Cache-Control"),
	)
	var envelope struct {
		Data map[string]json.RawMessage `json:"data"`
	}
	require.NoError(t, json.Unmarshal(response.Body.Bytes(), &envelope))
	require.Contains(t, envelope.Data, "id")
	require.Contains(t, envelope.Data, "sync_version")
	require.Contains(t, envelope.Data, "account_change_version")
	require.Contains(t, envelope.Data, "deleted_at")
	require.NotContains(t, envelope.Data, "draft")
	require.NotContains(t, envelope.Data, "publication")
	require.NotContains(t, envelope.Data, "community")
	require.NotContains(t, response.Body.String(), draft.Input.Name)
	require.NotContains(t, response.Body.String(), draft.Input.Description)
}

func createPublishedChecklistForDeletion(
	t *testing.T,
	repository owned.Repository,
	ownerUID string,
	checklistID uuid.UUID,
) *owned.MutationResult {
	t.Helper()
	draft := preparedTree(t, uuid.New())
	created, err := repository.Create(
		context.Background(),
		ownerUID,
		checklistID,
		draft,
		shared.Precondition{Mode: shared.PreconditionCreate},
	)
	require.NoError(t, err)
	publication, err := repository.Publish(
		context.Background(),
		ownerUID,
		checklistID,
		preparePublication(t, draft.Input, 1),
		checklistPrecondition(checklistID, created.Aggregate.SyncVersion),
	)
	require.NoError(t, err)
	return publication
}

func insertCommunityReleaseForDeletion(
	t *testing.T,
	checklistID uuid.UUID,
	revisionID uuid.UUID,
	revisionNumber int32,
) {
	t.Helper()
	_, err := testDB.ExecContext(
		context.Background(),
		`INSERT INTO user_pmcs_community_releases
		     (revision_id, checklist_id)
		 VALUES ($1, $2)`,
		revisionID,
		checklistID,
	)
	require.NoError(t, err)
	_, err = testDB.ExecContext(
		context.Background(),
		`INSERT INTO user_pmcs_community_sources
		     (checklist_id, status, current_release_revision_id,
		      latest_release_revision_number, first_released_at, updated_at)
		 VALUES ($1, 'active', $2, $3, now(), now())`,
		checklistID,
		revisionID,
		revisionNumber,
	)
	require.NoError(t, err)
}

func insertNextCommunityReleaseForDeletion(
	t *testing.T,
	checklistID uuid.UUID,
	revisionID uuid.UUID,
	revisionNumber int32,
) {
	t.Helper()
	_, err := testDB.ExecContext(
		context.Background(),
		`INSERT INTO user_pmcs_community_releases
		     (revision_id, checklist_id)
		 VALUES ($1, $2)`,
		revisionID,
		checklistID,
	)
	require.NoError(t, err)
	_, err = testDB.ExecContext(
		context.Background(),
		`UPDATE user_pmcs_community_sources
		 SET current_release_revision_id = $1,
		     latest_release_revision_number = $2,
		     updated_at = now()
		 WHERE checklist_id = $3`,
		revisionID,
		revisionNumber,
		checklistID,
	)
	require.NoError(t, err)
}

func insertActiveSubscriptionForDeletion(
	t *testing.T,
	subscriberUID string,
	checklistID uuid.UUID,
	revisionID uuid.UUID,
) {
	t.Helper()
	_, err := testDB.ExecContext(
		context.Background(),
		`INSERT INTO user_pmcs_subscriptions
		     (subscriber_uid, checklist_id, installed_revision_id,
		      sync_version, account_change_version)
		 VALUES ($1, $2, $3, 1, 1)`,
		subscriberUID,
		checklistID,
		revisionID,
	)
	require.NoError(t, err)
}

func requireCommunitySourceRetired(
	t *testing.T,
	checklistID uuid.UUID,
	wantLatest int32,
) {
	t.Helper()
	var (
		status         string
		currentRelease uuid.NullUUID
		latest         int32
		retiredAt      any
	)
	require.NoError(t, testDB.QueryRowContext(
		context.Background(),
		`SELECT status, current_release_revision_id,
		        latest_release_revision_number, retired_at
		 FROM user_pmcs_community_sources
		 WHERE checklist_id = $1`,
		checklistID,
	).Scan(&status, &currentRelease, &latest, &retiredAt))
	require.Equal(t, "retired", status)
	require.False(t, currentRelease.Valid)
	require.Equal(t, wantLatest, latest)
	require.NotNil(t, retiredAt)
}

func registerCommunityDeletionCleanup(t *testing.T, checklistID uuid.UUID) {
	t.Helper()
	t.Cleanup(func() {
		_, cleanupError := testDB.ExecContext(
			context.Background(),
			`DELETE FROM user_pmcs_subscriptions WHERE checklist_id = $1`,
			checklistID,
		)
		require.NoError(t, cleanupError)
		_, cleanupError = testDB.ExecContext(
			context.Background(),
			`DELETE FROM user_pmcs_community_sources WHERE checklist_id = $1`,
			checklistID,
		)
		require.NoError(t, cleanupError)
		_, cleanupError = testDB.ExecContext(
			context.Background(),
			`DELETE FROM user_pmcs_community_releases WHERE checklist_id = $1`,
			checklistID,
		)
		require.NoError(t, cleanupError)
		_, cleanupError = testDB.ExecContext(
			context.Background(),
			`DELETE FROM user_pmcs_checklists WHERE id = $1`,
			checklistID,
		)
		require.NoError(t, cleanupError)
	})
}

func deletionRowCount(
	t *testing.T,
	table string,
	column string,
	id uuid.UUID,
) int {
	t.Helper()
	allowed := map[string]bool{
		"user_pmcs_revisions.checklist_id":          true,
		"user_pmcs_community_releases.checklist_id": true,
		"user_pmcs_sections.revision_id":            true,
	}
	require.True(t, allowed[table+"."+column])
	var count int
	require.NoError(t, testDB.QueryRowContext(
		context.Background(),
		"SELECT count(*) FROM "+table+" WHERE "+column+" = $1",
		id,
	).Scan(&count))
	return count
}

func deletionUUIDs(t *testing.T, query string, checklistID uuid.UUID) []uuid.UUID {
	t.Helper()
	rows, err := testDB.QueryContext(context.Background(), query, checklistID)
	require.NoError(t, err)
	defer func() {
		require.NoError(t, rows.Close())
	}()
	var values []uuid.UUID
	for rows.Next() {
		var value uuid.UUID
		require.NoError(t, rows.Scan(&value))
		values = append(values, value)
	}
	require.NoError(t, rows.Err())
	return values
}
