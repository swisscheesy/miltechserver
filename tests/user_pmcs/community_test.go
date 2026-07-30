package user_pmcs_test

import (
	"context"
	"fmt"
	"sync"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/lib/pq"
	"github.com/stretchr/testify/require"

	"miltechserver/api/user_pmcs/community"
	"miltechserver/api/user_pmcs/owned"
	"miltechserver/api/user_pmcs/persistence"
	"miltechserver/api/user_pmcs/shared"
)

type releasedChecklistFixture struct {
	ownerUID   string
	checklist  uuid.UUID
	repository community.Repository
	owned      owned.Repository
	aggregate  shared.ChecklistAggregate
	revisions  []shared.PreparedRevision
}

func TestReleaseFirstHigherSameAndRollback(t *testing.T) {
	fixture := newReleasedChecklistFixture(t, 3)

	first, err := fixture.repository.Release(
		context.Background(),
		fixture.ownerUID,
		fixture.checklist,
		fixture.revisions[0].Input.ID,
		checklistPrecondition(fixture.checklist, fixture.aggregate.SyncVersion),
	)
	require.NoError(t, err)
	require.False(t, first.Idempotent)
	requireCommunityState(
		t,
		first.Aggregate,
		"active",
		fixture.revisions[0].Input.ID,
		1,
	)
	require.Equal(t, fixture.aggregate.SyncVersion+1, first.Aggregate.SyncVersion)
	require.Equal(t, first.Aggregate.AccountChangeVersion, accountVersion(t, fixture.ownerUID))

	same, err := fixture.repository.Release(
		context.Background(),
		fixture.ownerUID,
		fixture.checklist,
		fixture.revisions[0].Input.ID,
		checklistPrecondition(fixture.checklist, fixture.aggregate.SyncVersion),
	)
	require.NoError(t, err)
	require.True(t, same.Idempotent)
	require.Equal(t, first.Aggregate.SyncVersion, same.Aggregate.SyncVersion)
	require.Equal(t, first.Aggregate.AccountChangeVersion, accountVersion(t, fixture.ownerUID))

	higher, err := fixture.repository.Release(
		context.Background(),
		fixture.ownerUID,
		fixture.checklist,
		fixture.revisions[2].Input.ID,
		checklistPrecondition(fixture.checklist, first.Aggregate.SyncVersion),
	)
	require.NoError(t, err)
	requireCommunityState(
		t,
		higher.Aggregate,
		"active",
		fixture.revisions[2].Input.ID,
		3,
	)
	require.Equal(t, first.Aggregate.SyncVersion+1, higher.Aggregate.SyncVersion)

	for _, target := range fixture.revisions[:2] {
		_, err = fixture.repository.Release(
			context.Background(),
			fixture.ownerUID,
			fixture.checklist,
			target.Input.ID,
			checklistPrecondition(fixture.checklist, higher.Aggregate.SyncVersion),
		)
		requireAPIIntegrationError(t, err, 409, "invalid_transition")
	}
	require.Equal(t, higher.Aggregate.SyncVersion, checklistVersion(t, fixture.checklist))
	require.Equal(t, higher.Aggregate.AccountChangeVersion, accountVersion(t, fixture.ownerUID))
	require.Equal(t, 2, communityReleaseCount(t, fixture.checklist))
}

func TestReleaseOwnerOnlyCurrentETagAndPublishedTarget(t *testing.T) {
	fixture := newReleasedChecklistFixture(t, 1)
	otherUID := newUserPmcsTestUser(t)

	_, err := fixture.repository.Release(
		context.Background(),
		otherUID,
		fixture.checklist,
		fixture.revisions[0].Input.ID,
		checklistPrecondition(fixture.checklist, fixture.aggregate.SyncVersion),
	)
	requireAPIIntegrationError(t, err, 404, "resource_not_found")
	require.Equal(t, int64(0), accountVersion(t, otherUID))

	_, err = fixture.repository.Release(
		context.Background(),
		fixture.ownerUID,
		fixture.checklist,
		fixture.revisions[0].Input.ID,
		checklistPrecondition(fixture.checklist, fixture.aggregate.SyncVersion+1),
	)
	requireAPIIntegrationError(t, err, 412, "stale_precondition")

	draft := preparedTree(t, uuid.New())
	draftResult, err := fixture.owned.PutDraft(
		context.Background(),
		fixture.ownerUID,
		fixture.checklist,
		draft,
		checklistPrecondition(fixture.checklist, fixture.aggregate.SyncVersion),
	)
	require.NoError(t, err)
	fixture.aggregate = draftResult.Aggregate
	_, err = fixture.repository.Release(
		context.Background(),
		fixture.ownerUID,
		fixture.checklist,
		draft.Input.ID,
		checklistPrecondition(fixture.checklist, fixture.aggregate.SyncVersion),
	)
	requireAPIIntegrationError(t, err, 409, "invalid_transition")
}

func TestReleaseRevalidatesHistoricalTree(t *testing.T) {
	fixture := newReleasedChecklistFixture(t, 2)
	target := fixture.revisions[0]
	_, err := testDB.ExecContext(
		context.Background(),
		`UPDATE user_pmcs_procedure_steps
		 SET step_text = ''
		 WHERE item_id IN (
		     SELECT item.id
		     FROM user_pmcs_items AS item
		     JOIN user_pmcs_sections AS section ON section.id = item.section_id
		     WHERE section.revision_id = $1
		 )`,
		target.Input.ID,
	)
	require.NoError(t, err)

	_, err = fixture.repository.Release(
		context.Background(),
		fixture.ownerUID,
		fixture.checklist,
		target.Input.ID,
		checklistPrecondition(fixture.checklist, fixture.aggregate.SyncVersion),
	)

	requireAPIIntegrationError(t, err, 422, "validation_failed")
	require.Equal(t, 0, communityReleaseCount(t, fixture.checklist))
	require.Equal(t, fixture.aggregate.SyncVersion, checklistVersion(t, fixture.checklist))
}

func TestRetirePreservesHistoryAndReactivatesOnlyHigher(t *testing.T) {
	fixture := newReleasedChecklistFixture(t, 2)
	released, err := fixture.repository.Release(
		context.Background(),
		fixture.ownerUID,
		fixture.checklist,
		fixture.revisions[0].Input.ID,
		checklistPrecondition(fixture.checklist, fixture.aggregate.SyncVersion),
	)
	require.NoError(t, err)

	retired, err := fixture.repository.Retire(
		context.Background(),
		fixture.ownerUID,
		fixture.checklist,
		checklistPrecondition(fixture.checklist, released.Aggregate.SyncVersion),
	)
	require.NoError(t, err)
	require.NotNil(t, retired.Aggregate.Community)
	require.Equal(t, "retired", retired.Aggregate.Community.Status)
	require.Nil(t, retired.Aggregate.Community.CurrentReleaseRevisionID)
	require.NotNil(t, retired.Aggregate.Community.RetiredAt)
	require.Equal(t, int32(1), retired.Aggregate.Community.LatestReleaseRevisionNumber)
	require.Equal(t, 1, communityReleaseCount(t, fixture.checklist))

	retried, err := fixture.repository.Retire(
		context.Background(),
		fixture.ownerUID,
		fixture.checklist,
		checklistPrecondition(fixture.checklist, released.Aggregate.SyncVersion),
	)
	require.NoError(t, err)
	require.True(t, retried.Idempotent)
	require.Equal(t, retired.Aggregate.SyncVersion, retried.Aggregate.SyncVersion)

	_, err = fixture.repository.Release(
		context.Background(),
		fixture.ownerUID,
		fixture.checklist,
		fixture.revisions[0].Input.ID,
		checklistPrecondition(fixture.checklist, retired.Aggregate.SyncVersion),
	)
	requireAPIIntegrationError(t, err, 409, "invalid_transition")

	reactivated, err := fixture.repository.Release(
		context.Background(),
		fixture.ownerUID,
		fixture.checklist,
		fixture.revisions[1].Input.ID,
		checklistPrecondition(fixture.checklist, retired.Aggregate.SyncVersion),
	)
	require.NoError(t, err)
	requireCommunityState(
		t,
		reactivated.Aggregate,
		"active",
		fixture.revisions[1].Input.ID,
		2,
	)
	require.Nil(t, reactivated.Aggregate.Community.RetiredAt)
	require.Equal(t, 2, communityReleaseCount(t, fixture.checklist))
}

func TestReleaseTombstoneNeverReactivates(t *testing.T) {
	fixture := newReleasedChecklistFixture(t, 2)
	released, err := fixture.repository.Release(
		context.Background(),
		fixture.ownerUID,
		fixture.checklist,
		fixture.revisions[0].Input.ID,
		checklistPrecondition(fixture.checklist, fixture.aggregate.SyncVersion),
	)
	require.NoError(t, err)
	deleted, err := fixture.owned.DeleteChecklist(
		context.Background(),
		fixture.ownerUID,
		fixture.checklist,
		checklistPrecondition(fixture.checklist, released.Aggregate.SyncVersion),
	)
	require.NoError(t, err)

	_, err = fixture.repository.Release(
		context.Background(),
		fixture.ownerUID,
		fixture.checklist,
		fixture.revisions[1].Input.ID,
		checklistPrecondition(fixture.checklist, deleted.Aggregate.SyncVersion),
	)

	requireAPIIntegrationError(t, err, 412, "stale_precondition")
	require.Equal(t, "retired", communitySourceStatus(t, fixture.checklist))
}

func TestReleaseDoesNotFanOutToSubscribers(t *testing.T) {
	fixture := newReleasedChecklistFixture(t, 2)
	first, err := fixture.repository.Release(
		context.Background(),
		fixture.ownerUID,
		fixture.checklist,
		fixture.revisions[0].Input.ID,
		checklistPrecondition(fixture.checklist, fixture.aggregate.SyncVersion),
	)
	require.NoError(t, err)
	subscriberUIDs := insertCommunitySubscribers(
		t,
		fixture.checklist,
		fixture.revisions[0].Input.ID,
		100,
	)
	before := subscriptionSnapshots(t, fixture.checklist)

	_, err = fixture.repository.Release(
		context.Background(),
		fixture.ownerUID,
		fixture.checklist,
		fixture.revisions[1].Input.ID,
		checklistPrecondition(fixture.checklist, first.Aggregate.SyncVersion),
	)
	require.NoError(t, err)

	require.Equal(t, before, subscriptionSnapshots(t, fixture.checklist))
	for _, subscriberUID := range subscriberUIDs {
		require.Equal(t, int64(1), accountVersion(t, subscriberUID))
	}
}

func TestReleaseConcurrentHigherRevisionsSerialize(t *testing.T) {
	fixture := newReleasedChecklistFixture(t, 3)
	first, err := fixture.repository.Release(
		context.Background(),
		fixture.ownerUID,
		fixture.checklist,
		fixture.revisions[0].Input.ID,
		checklistPrecondition(fixture.checklist, fixture.aggregate.SyncVersion),
	)
	require.NoError(t, err)

	start := make(chan struct{})
	results := make(chan error, 2)
	var waitGroup sync.WaitGroup
	for _, revision := range fixture.revisions[1:] {
		revision := revision
		waitGroup.Add(1)
		go func() {
			defer waitGroup.Done()
			<-start
			_, releaseErr := fixture.repository.Release(
				context.Background(),
				fixture.ownerUID,
				fixture.checklist,
				revision.Input.ID,
				checklistPrecondition(
					fixture.checklist,
					first.Aggregate.SyncVersion,
				),
			)
			results <- releaseErr
		}()
	}
	close(start)
	waitGroup.Wait()
	close(results)

	var succeeded, stale int
	for releaseErr := range results {
		if releaseErr == nil {
			succeeded++
			continue
		}
		var apiError *shared.APIError
		require.ErrorAs(t, releaseErr, &apiError)
		require.Equal(t, "stale_precondition", apiError.Code)
		stale++
	}
	require.Equal(t, 1, succeeded)
	require.Equal(t, 1, stale)
	require.Equal(t, first.Aggregate.SyncVersion+1, checklistVersion(t, fixture.checklist))
}

func TestCommunityBrowseStaticKeysetCurrentOnlyAndModelFilter(t *testing.T) {
	baseTime := time.Date(2026, time.July, 30, 12, 0, 0, 0, time.UTC)
	normalizedModel := "task10-" + uuid.NewString()
	historicalModel := "task10-historical-" + uuid.NewString()
	fixtures := []*releasedChecklistFixture{
		newReleasedChecklistFixture(t, 1),
		newReleasedChecklistFixture(t, 2),
		newReleasedChecklistFixture(t, 1),
		newReleasedChecklistFixture(t, 1),
	}
	for index, fixture := range fixtures {
		revisionIndex := len(fixture.revisions) - 1
		_, err := fixture.repository.Release(
			context.Background(),
			fixture.ownerUID,
			fixture.checklist,
			fixture.revisions[revisionIndex].Input.ID,
			checklistPrecondition(
				fixture.checklist,
				fixture.aggregate.SyncVersion,
			),
		)
		require.NoError(t, err)
		_, err = testDB.ExecContext(
			context.Background(),
			`UPDATE user_pmcs_revision_models
			 SET display_text = $1, normalized_text = $1
			 WHERE revision_id = $2`,
			normalizedModel,
			fixture.revisions[revisionIndex].Input.ID,
		)
		require.NoError(t, err)
		if len(fixture.revisions) > 1 {
			_, err = testDB.ExecContext(
				context.Background(),
				`UPDATE user_pmcs_revision_models
				 SET display_text = $1, normalized_text = $1
				 WHERE revision_id = $2`,
				historicalModel,
				fixture.revisions[0].Input.ID,
			)
			require.NoError(t, err)
		}
		updatedAt := baseTime.Add(-time.Duration(index/2) * time.Hour)
		_, err = testDB.ExecContext(
			context.Background(),
			`UPDATE user_pmcs_community_sources
			 SET updated_at = $1
			 WHERE checklist_id = $2`,
			updatedAt,
			fixture.checklist,
		)
		require.NoError(t, err)
		_, err = testDB.ExecContext(
			context.Background(),
			`UPDATE users SET username = $1 WHERE uid = $2`,
			fmt.Sprintf("Creator %d", index),
			fixture.ownerUID,
		)
		require.NoError(t, err)
	}
	_, err := testDB.ExecContext(
		context.Background(),
		`UPDATE users SET username = NULL WHERE uid = $1`,
		fixtures[2].ownerUID,
	)
	require.NoError(t, err)

	retired := fixtures[3]
	_, err = retired.repository.Retire(
		context.Background(),
		retired.ownerUID,
		retired.checklist,
		checklistPrecondition(
			retired.checklist,
			checklistVersion(t, retired.checklist),
		),
	)
	require.NoError(t, err)

	repository := fixtures[0].repository
	firstPage, err := repository.Browse(
		context.Background(),
		shared.CommunityBrowseFilter{
			Limit:           2,
			NormalizedModel: normalizedModel,
		},
	)
	require.NoError(t, err)
	require.Len(t, firstPage.Items, 2)
	require.True(t, firstPage.HasMore)
	require.NotNil(t, firstPage.NextCursor)

	firstCursor, err := shared.DecodeCommunityCursor(*firstPage.NextCursor)
	require.NoError(t, err)
	secondPage, err := repository.Browse(
		context.Background(),
		shared.CommunityBrowseFilter{
			After:           &firstCursor,
			Limit:           2,
			NormalizedModel: normalizedModel,
		},
	)
	require.NoError(t, err)
	require.Len(t, secondPage.Items, 1)
	require.False(t, secondPage.HasMore)
	require.Nil(t, secondPage.NextCursor)

	allItems := append(
		append([]shared.PublicCommunitySummary{}, firstPage.Items...),
		secondPage.Items...,
	)
	require.Len(t, allItems, 3)
	seen := make(map[uuid.UUID]struct{}, len(allItems))
	for _, item := range allItems {
		_, duplicate := seen[item.ChecklistID]
		require.False(t, duplicate)
		seen[item.ChecklistID] = struct{}{}
		require.NotEqual(t, retired.checklist, item.ChecklistID)
		require.NotEmpty(t, item.Models)
		require.NotEmpty(t, item.CreatorDisplayName)
	}
	require.Equal(
		t,
		"Creator 0",
		summaryCreator(t, allItems, fixtures[0].checklist),
	)
	require.Equal(
		t,
		"Deleted user",
		summaryCreator(t, allItems, fixtures[2].checklist),
	)
	require.True(t, baseTime.Equal(allItems[0].UpdatedAt))
	require.True(t, baseTime.Equal(allItems[1].UpdatedAt))
	require.Less(
		t,
		allItems[0].ChecklistID.String(),
		allItems[1].ChecklistID.String(),
	)
	require.Equal(t, fixtures[1].revisions[1].Input.ID, summaryRevision(
		t,
		allItems,
		fixtures[1].checklist,
	))

	filtered, err := repository.Browse(
		context.Background(),
		shared.CommunityBrowseFilter{
			Limit:           50,
			NormalizedModel: normalizedModel,
		},
	)
	require.NoError(t, err)
	require.Len(t, filtered.Items, 3)
	none, err := repository.Browse(
		context.Background(),
		shared.CommunityBrowseFilter{
			Limit:           50,
			NormalizedModel: normalizedModel + "-other",
		},
	)
	require.NoError(t, err)
	require.Empty(t, none.Items)
	historicalOnly, err := repository.Browse(
		context.Background(),
		shared.CommunityBrowseFilter{
			Limit:           50,
			NormalizedModel: historicalModel,
		},
	)
	require.NoError(t, err)
	require.Empty(t, historicalOnly.Items)
}

func TestCommunityDetailLoadsCompleteCurrentReleaseAndVisibility(t *testing.T) {
	fixture := newReleasedChecklistFixture(t, 2)
	first, err := fixture.repository.Release(
		context.Background(),
		fixture.ownerUID,
		fixture.checklist,
		fixture.revisions[0].Input.ID,
		checklistPrecondition(fixture.checklist, fixture.aggregate.SyncVersion),
	)
	require.NoError(t, err)
	current, err := fixture.repository.Release(
		context.Background(),
		fixture.ownerUID,
		fixture.checklist,
		fixture.revisions[1].Input.ID,
		checklistPrecondition(fixture.checklist, first.Aggregate.SyncVersion),
	)
	require.NoError(t, err)
	_, err = testDB.ExecContext(
		context.Background(),
		`UPDATE users SET username = NULL WHERE uid = $1`,
		fixture.ownerUID,
	)
	require.NoError(t, err)

	release, err := fixture.repository.GetCurrentRelease(
		context.Background(),
		fixture.checklist,
	)
	require.NoError(t, err)
	require.Equal(t, fixture.checklist, release.ChecklistID)
	require.Equal(t, fixture.revisions[1].Input.ID, release.Revision.ID)
	require.Equal(t, fixture.revisions[1].Input.Name, release.Revision.Name)
	require.Equal(t, "Deleted user", release.CreatorDisplayName)
	require.NotZero(t, release.ReleasedAt)
	require.NotEmpty(t, release.Revision.Sections)
	require.NotEmpty(t, release.Revision.Sections[0].Items)
	require.NotEmpty(t, release.Revision.Sections[0].Items[0].Notices)
	require.NotEmpty(t, release.Revision.Sections[0].Items[0].ProcedureSteps)

	service := community.NewService(fixture.repository, shared.DefaultConfig())
	serviceRelease, etag, err := service.GetCurrentRelease(
		context.Background(),
		fixture.checklist.String(),
	)
	require.NoError(t, err)
	require.Equal(t, release, serviceRelease)
	require.NotEmpty(t, etag)
	require.Equal(t, '"', rune(etag[0]))
	require.Equal(t, '"', rune(etag[len(etag)-1]))

	_, err = testDB.ExecContext(
		context.Background(),
		`UPDATE user_pmcs_revisions
		 SET content_hash = decode(repeat('00', 32), 'hex')
		 WHERE id = $1`,
		fixture.revisions[1].Input.ID,
	)
	require.NoError(t, err)
	_, err = fixture.repository.GetCurrentRelease(
		context.Background(),
		fixture.checklist,
	)
	require.ErrorContains(t, err, "content hash mismatch")
	currentHash := fixture.revisions[1].Hash
	_, err = testDB.ExecContext(
		context.Background(),
		`UPDATE user_pmcs_revisions SET content_hash = $1 WHERE id = $2`,
		currentHash[:],
		fixture.revisions[1].Input.ID,
	)
	require.NoError(t, err)

	neverReleased := newReleasedChecklistFixture(t, 1)
	_, _, err = service.GetCurrentRelease(
		context.Background(),
		neverReleased.checklist.String(),
	)
	requireAPIIntegrationError(t, err, 404, "resource_not_found")

	retired, err := fixture.repository.Retire(
		context.Background(),
		fixture.ownerUID,
		fixture.checklist,
		checklistPrecondition(
			fixture.checklist,
			current.Aggregate.SyncVersion,
		),
	)
	require.NoError(t, err)
	_, err = fixture.repository.GetCurrentRelease(
		context.Background(),
		fixture.checklist,
	)
	requireAPIIntegrationError(t, err, 404, "resource_not_found")
	deleted, err := fixture.owned.DeleteChecklist(
		context.Background(),
		fixture.ownerUID,
		fixture.checklist,
		checklistPrecondition(
			fixture.checklist,
			retired.Aggregate.SyncVersion,
		),
	)
	require.NoError(t, err)
	require.NotNil(t, deleted.Aggregate.DeletedAt)
	_, err = fixture.repository.GetCurrentRelease(
		context.Background(),
		fixture.checklist,
	)
	requireAPIIntegrationError(t, err, 404, "resource_not_found")
}

func TestCommunityBrowseMovingReleaseAppearsAfterRestart(t *testing.T) {
	baseTime := time.Now().UTC().Add(-24 * time.Hour)
	normalizedModel := "task10-moving-" + uuid.NewString()
	older := newReleasedChecklistFixture(t, 2)
	newer := newReleasedChecklistFixture(t, 1)
	for index, fixture := range []*releasedChecklistFixture{older, newer} {
		first, err := fixture.repository.Release(
			context.Background(),
			fixture.ownerUID,
			fixture.checklist,
			fixture.revisions[0].Input.ID,
			checklistPrecondition(fixture.checklist, fixture.aggregate.SyncVersion),
		)
		require.NoError(t, err)
		fixture.aggregate = first.Aggregate
		_, err = testDB.ExecContext(
			context.Background(),
			`UPDATE user_pmcs_revision_models
			 SET display_text = $1, normalized_text = $1
			 WHERE revision_id = $2`,
			normalizedModel,
			fixture.revisions[0].Input.ID,
		)
		require.NoError(t, err)
		_, err = testDB.ExecContext(
			context.Background(),
			`UPDATE user_pmcs_community_sources
			 SET updated_at = $1
			 WHERE checklist_id = $2`,
			baseTime.Add(time.Duration(index)*time.Hour),
			fixture.checklist,
		)
		require.NoError(t, err)
	}

	firstPage, err := older.repository.Browse(
		context.Background(),
		shared.CommunityBrowseFilter{
			Limit:           1,
			NormalizedModel: normalizedModel,
		},
	)
	require.NoError(t, err)
	require.Equal(t, newer.checklist, firstPage.Items[0].ChecklistID)

	advanced, err := older.repository.Release(
		context.Background(),
		older.ownerUID,
		older.checklist,
		older.revisions[1].Input.ID,
		checklistPrecondition(older.checklist, older.aggregate.SyncVersion),
	)
	require.NoError(t, err)
	require.NotNil(t, advanced.Aggregate.Community)
	require.NotNil(t, advanced.Aggregate.Community.CurrentReleaseRevisionID)
	require.Equal(
		t,
		older.revisions[1].Input.ID,
		*advanced.Aggregate.Community.CurrentReleaseRevisionID,
	)
	_, err = testDB.ExecContext(
		context.Background(),
		`UPDATE user_pmcs_revision_models
		 SET display_text = $1, normalized_text = $1
		 WHERE revision_id = $2`,
		normalizedModel,
		older.revisions[1].Input.ID,
	)
	require.NoError(t, err)

	restarted, err := older.repository.Browse(
		context.Background(),
		shared.CommunityBrowseFilter{
			Limit:           1,
			NormalizedModel: normalizedModel,
		},
	)
	require.NoError(t, err)
	require.Equal(t, older.checklist, restarted.Items[0].ChecklistID)
	require.Equal(t, older.revisions[1].Input.ID, restarted.Items[0].RevisionID)
}

func summaryRevision(
	t *testing.T,
	items []shared.PublicCommunitySummary,
	checklistID uuid.UUID,
) uuid.UUID {
	t.Helper()
	for _, item := range items {
		if item.ChecklistID == checklistID {
			return item.RevisionID
		}
	}
	t.Fatalf("summary for checklist %s was not returned", checklistID)
	return uuid.Nil
}

func summaryCreator(
	t *testing.T,
	items []shared.PublicCommunitySummary,
	checklistID uuid.UUID,
) string {
	t.Helper()
	for _, item := range items {
		if item.ChecklistID == checklistID {
			return item.CreatorDisplayName
		}
	}
	t.Fatalf("summary for checklist %s was not returned", checklistID)
	return ""
}

func newReleasedChecklistFixture(
	t *testing.T,
	publicationCount int,
) *releasedChecklistFixture {
	t.Helper()
	ownerUID := newUserPmcsTestUser(t)
	checklistID := uuid.New()
	config := shared.DefaultConfig()
	store := persistence.NewStore(testDB, config.TransactionMaxAttempts)
	ownedRepository := owned.NewRepository(store, config)
	communityRepository := community.NewRepository(store, config)
	firstDraft := preparedTree(t, uuid.New())
	created, err := ownedRepository.Create(
		context.Background(),
		ownerUID,
		checklistID,
		firstDraft,
		shared.Precondition{Mode: shared.PreconditionCreate},
	)
	require.NoError(t, err)

	revisions := make([]shared.PreparedRevision, 0, publicationCount)
	current := created.Aggregate
	for number := 1; number <= publicationCount; number++ {
		input := firstDraft.Input
		if number > 1 {
			input = preparedTree(t, uuid.New()).Input
			input.Name = fmt.Sprintf("Publication %d", number)
		}
		publication := preparePublication(t, input, int32(number))
		published, publishErr := ownedRepository.Publish(
			context.Background(),
			ownerUID,
			checklistID,
			publication,
			checklistPrecondition(checklistID, current.SyncVersion),
		)
		require.NoError(t, publishErr)
		current = published.Aggregate
		revisions = append(revisions, publication)
	}

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
	})
	return &releasedChecklistFixture{
		ownerUID:   ownerUID,
		checklist:  checklistID,
		repository: communityRepository,
		owned:      ownedRepository,
		aggregate:  current,
		revisions:  revisions,
	}
}

func requireCommunityState(
	t *testing.T,
	aggregate shared.ChecklistAggregate,
	status string,
	currentRevision uuid.UUID,
	latest int32,
) {
	t.Helper()
	require.NotNil(t, aggregate.Community)
	require.Equal(t, status, aggregate.Community.Status)
	require.NotNil(t, aggregate.Community.CurrentReleaseRevisionID)
	require.Equal(t, currentRevision, *aggregate.Community.CurrentReleaseRevisionID)
	require.Equal(t, latest, aggregate.Community.LatestReleaseRevisionNumber)
}

func communityReleaseCount(t *testing.T, checklistID uuid.UUID) int {
	t.Helper()
	var count int
	require.NoError(t, testDB.QueryRowContext(
		context.Background(),
		`SELECT count(*) FROM user_pmcs_community_releases WHERE checklist_id = $1`,
		checklistID,
	).Scan(&count))
	return count
}

func communitySourceStatus(t *testing.T, checklistID uuid.UUID) string {
	t.Helper()
	var status string
	require.NoError(t, testDB.QueryRowContext(
		context.Background(),
		`SELECT status FROM user_pmcs_community_sources WHERE checklist_id = $1`,
		checklistID,
	).Scan(&status))
	return status
}

type subscriptionSnapshot struct {
	SubscriberUID       string
	InstalledRevisionID uuid.UUID
	SyncVersion         int64
	AccountVersion      int64
	UpdatedAt           time.Time
}

func insertCommunitySubscribers(
	t *testing.T,
	checklistID uuid.UUID,
	revisionID uuid.UUID,
	count int,
) []string {
	t.Helper()
	tx, err := testDB.BeginTx(context.Background(), nil)
	require.NoError(t, err)
	uids := make([]string, 0, count)
	for index := 0; index < count; index++ {
		uid := "community-subscriber-" + uuid.NewString()
		uids = append(uids, uid)
		_, err = tx.ExecContext(
			context.Background(),
			`INSERT INTO users (uid, email, username, created_at, is_enabled)
			 VALUES ($1, $2, $3, now(), TRUE)`,
			uid,
			uid+"@example.com",
			"community subscriber",
		)
		require.NoError(t, err)
		_, err = tx.ExecContext(
			context.Background(),
			`INSERT INTO user_pmcs_sync_state (user_uid, current_version)
			 VALUES ($1, 1)`,
			uid,
		)
		require.NoError(t, err)
		_, err = tx.ExecContext(
			context.Background(),
			`INSERT INTO user_pmcs_subscriptions
			     (subscriber_uid, checklist_id, installed_revision_id,
			      sync_version, account_change_version)
			 VALUES ($1, $2, $3, 1, 1)`,
			uid,
			checklistID,
			revisionID,
		)
		require.NoError(t, err)
	}
	require.NoError(t, tx.Commit())
	t.Cleanup(func() {
		_, cleanupError := testDB.ExecContext(
			context.Background(),
			`DELETE FROM users WHERE uid = ANY($1)`,
			pq.Array(uids),
		)
		require.NoError(t, cleanupError)
	})
	return uids
}

func subscriptionSnapshots(
	t *testing.T,
	checklistID uuid.UUID,
) []subscriptionSnapshot {
	t.Helper()
	rows, err := testDB.QueryContext(
		context.Background(),
		`SELECT subscriber_uid, installed_revision_id, sync_version,
		        account_change_version, updated_at
		 FROM user_pmcs_subscriptions
		 WHERE checklist_id = $1
		 ORDER BY subscriber_uid`,
		checklistID,
	)
	require.NoError(t, err)
	defer rows.Close()
	var snapshots []subscriptionSnapshot
	for rows.Next() {
		var snapshot subscriptionSnapshot
		require.NoError(t, rows.Scan(
			&snapshot.SubscriberUID,
			&snapshot.InstalledRevisionID,
			&snapshot.SyncVersion,
			&snapshot.AccountVersion,
			&snapshot.UpdatedAt,
		))
		snapshots = append(snapshots, snapshot)
	}
	require.NoError(t, rows.Err())
	return snapshots
}
