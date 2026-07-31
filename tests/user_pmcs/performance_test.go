package user_pmcs_test

import (
	"bytes"
	"compress/gzip"
	"context"
	"database/sql"
	"database/sql/driver"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"runtime"
	"sort"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/lib/pq"
	"github.com/stretchr/testify/require"

	"miltechserver/api/user_pmcs/community"
	"miltechserver/api/user_pmcs/owned"
	"miltechserver/api/user_pmcs/persistence"
	"miltechserver/api/user_pmcs/shared"
	"miltechserver/api/user_pmcs/subscriptions"
	userpmcssync "miltechserver/api/user_pmcs/sync"
)

type performanceEvidence struct {
	scenario           string
	latencies          []time.Duration
	dbDuration         time.Duration
	lockWait           time.Duration
	encodeDuration     time.Duration
	peakLiveBytes      uint64
	uncompressedBytes  int
	gzipBytes          int
	queryCount         int
	latencyMeasured    bool
	dbMeasured         bool
	lockWaitMeasured   bool
	lockWaitWasZero    bool
	encodeMeasured     bool
	peakLiveMeasured   bool
	bytesMeasured      bool
	queryCountMeasured bool
}

type performanceSubscriptionFixture struct {
	ownerUID       string
	subscriberUID  string
	checklistIDs   []uuid.UUID
	installedIDs   []uuid.UUID
	currentIDs     []uuid.UUID
	noiseUserUIDs  []string
	normalizedName string
}

func TestPerformanceEvidenceRejectsUnmeasuredFields(t *testing.T) {
	err := validatePerformanceEvidence(performanceEvidence{
		scenario: "incomplete",
	})
	require.ErrorContains(t, err, "unmeasured")
}

func TestPerformanceScenarios(t *testing.T) {
	requireUserPmcsTestDatabase(t, testDB)
	config := shared.DefaultConfig()
	performanceDB, queryCounter, performanceApplicationName :=
		newQueryCountingDatabase(t)
	capturedProductionQueries := make(map[string]observedQuery)

	t.Run("maximum tree and body field boundaries", func(t *testing.T) {
		input := maximumDeterministicTree(
			t,
			"maximum-tree",
			deterministicFixtureUUID("maximum-tree", "revision"),
		)
		sections, items, notices, steps, maxItemsPerSection := treeInputCounts(input)
		require.Equal(t, 100, sections)
		require.Equal(t, 2_000, items)
		require.Equal(t, 4_000, notices)
		require.Equal(t, 10_000, steps)
		require.LessOrEqual(t, maxItemsPerSection, 500)
		_, err := shared.PrepareDraft(input, config)
		require.NoError(t, err)

		userUID := newUserPmcsTestUser(t)
		router := newUserPmcsContractRouter(config)
		exactPayload := paddedJSONAtSize(
			t,
			preparedTree(t, uuid.New()).Input,
			int(config.MaxMutationBodyBytes),
		)
		exactResponse := performContractRequest(
			router,
			http.MethodPut,
			"/auth/user-pmcs/checklists/"+uuid.NewString(),
			userUID,
			exactPayload,
			map[string]string{
				"Content-Type":  "application/json",
				"If-None-Match": "*",
			},
		)
		require.Equal(
			t,
			http.StatusCreated,
			exactResponse.Code,
			exactResponse.Body.String(),
		)
		overResponse := performContractRequest(
			router,
			http.MethodPut,
			"/auth/user-pmcs/checklists/"+uuid.NewString(),
			userUID,
			append(exactPayload, ' '),
			map[string]string{
				"Content-Type":  "application/json",
				"If-None-Match": "*",
			},
		)
		requireStableAPIError(
			t,
			overResponse,
			http.StatusRequestEntityTooLarge,
			"content_too_large",
		)
		assertPostgresTextFieldByteBoundaries(t, userUID)
	})

	t.Run("maximum draft replacement and publication", func(t *testing.T) {
		ctx, cancel := context.WithTimeout(
			context.Background(),
			2*time.Minute,
		)
		defer cancel()
		ownerUID := newUserPmcsTestUser(t)
		repository := owned.NewRepository(
			persistence.NewStore(
				performanceDB,
				config.TransactionMaxAttempts,
			),
			config,
		)
		checklistID := uuid.New()
		created, err := repository.Create(
			ctx,
			ownerUID,
			checklistID,
			preparedTree(t, uuid.New()),
			shared.Precondition{Mode: shared.PreconditionCreate},
		)
		require.NoError(t, err)

		maximumInput := maximumDeterministicTree(
			t,
			"maximum-write-"+checklistID.String(),
			deterministicFixtureUUID(
				"maximum-write-"+checklistID.String(),
				"revision",
			),
		)
		maximumDraft, err := shared.PrepareDraft(maximumInput, config)
		require.NoError(t, err)
		queryCounter.reset()
		draftPeakSampler := startPeakLiveSampler()
		draftLockSampler := startLockWaitSampler(
			ctx,
			performanceApplicationName,
		)
		draftStarted := time.Now()
		replaced, err := repository.PutDraft(
			ctx,
			ownerUID,
			checklistID,
			maximumDraft,
			checklistPrecondition(
				checklistID,
				created.Aggregate.SyncVersion,
			),
		)
		draftDuration := time.Since(draftStarted)
		draftQueryCount := queryCounter.value()
		require.NoError(t, err)
		require.Positive(t, draftQueryCount)
		require.LessOrEqual(t, draftQueryCount, 200)

		encodeStarted := time.Now()
		encodedDraft, err := json.Marshal(replaced.Aggregate)
		require.NoError(t, err)
		encodeDuration := time.Since(encodeStarted)
		require.Less(t, len(encodedDraft), config.MaxDeltaResponseBytes)
		queryer := &countingQueryer{database: testDB}
		loaded, err := persistence.LoadRevisionTrees(
			ctx,
			queryer,
			[]uuid.UUID{maximumDraft.Input.ID},
		)
		require.NoError(t, err)
		require.Len(t, loaded, 1)
		require.Equal(t, 7, queryer.queryCount)
		allocations := testing.AllocsPerRun(3, func() {
			_, marshalErr := json.Marshal(replaced.Aggregate)
			if marshalErr != nil {
				panic(marshalErr)
			}
		})
		t.Logf(
			"scenario=maximum_draft_replacement allocations_per_encode=%.0f",
			allocations,
		)
		draftGzipBytes := compressedSize(t, encodedDraft)
		draftPeakLiveBytes := draftPeakSampler.finish()
		draftLockWait := finishLockWaitSampler(t, draftLockSampler)
		for _, query := range queryCounter.snapshot() {
			if strings.Contains(query.query, "FROM user_pmcs_revisions") {
				capturedProductionQueries["batched revision loader"] = query
			}
			if strings.Contains(
				query.query,
				"FROM user_pmcs_subscriptions",
			) && strings.Contains(query.query, "FOR UPDATE") {
				capturedProductionQueries["active pin lookup"] = query
			}
		}
		logPerformanceEvidence(t, performanceEvidence{
			scenario:           "maximum_draft_replacement",
			latencies:          []time.Duration{draftDuration},
			dbDuration:         queryCounter.databaseDuration(),
			lockWait:           draftLockWait,
			encodeDuration:     encodeDuration,
			peakLiveBytes:      draftPeakLiveBytes,
			uncompressedBytes:  len(encodedDraft),
			gzipBytes:          draftGzipBytes,
			queryCount:         draftQueryCount,
			latencyMeasured:    true,
			dbMeasured:         true,
			lockWaitMeasured:   true,
			lockWaitWasZero:    draftLockWait == 0,
			encodeMeasured:     true,
			peakLiveMeasured:   true,
			bytesMeasured:      true,
			queryCountMeasured: true,
		})

		number := int32(1)
		publicationInput := maximumDraft.Input
		publicationInput.RevisionNumber = &number
		publication, err := shared.PreparePublication(
			publicationInput,
			config,
		)
		require.NoError(t, err)
		queryCounter.reset()
		publishPeakSampler := startPeakLiveSampler()
		publishLockSampler := startLockWaitSampler(
			ctx,
			performanceApplicationName,
		)
		publishStarted := time.Now()
		published, err := repository.Publish(
			ctx,
			ownerUID,
			checklistID,
			publication,
			checklistPrecondition(
				checklistID,
				replaced.Aggregate.SyncVersion,
			),
		)
		publishDuration := time.Since(publishStarted)
		publishQueryCount := queryCounter.value()
		require.NoError(t, err)
		require.Positive(t, publishQueryCount)
		require.LessOrEqual(t, publishQueryCount, 100)
		encodeStarted = time.Now()
		encodedPublication, err := json.Marshal(published.Aggregate)
		require.NoError(t, err)
		encodeDuration = time.Since(encodeStarted)
		require.Less(
			t,
			len(encodedPublication),
			config.MaxDeltaResponseBytes,
		)
		publishGzipBytes := compressedSize(t, encodedPublication)
		publishPeakLiveBytes := publishPeakSampler.finish()
		publishLockWait := finishLockWaitSampler(t, publishLockSampler)
		for _, query := range queryCounter.snapshot() {
			if strings.Contains(
				query.query,
				"FROM user_pmcs_subscriptions",
			) && strings.Contains(query.query, "FOR UPDATE") {
				capturedProductionQueries["active pin lookup"] = query
			}
		}
		logPerformanceEvidence(t, performanceEvidence{
			scenario:           "maximum_publication",
			latencies:          []time.Duration{publishDuration},
			dbDuration:         queryCounter.databaseDuration(),
			lockWait:           publishLockWait,
			encodeDuration:     encodeDuration,
			peakLiveBytes:      publishPeakLiveBytes,
			uncompressedBytes:  len(encodedPublication),
			gzipBytes:          publishGzipBytes,
			queryCount:         publishQueryCount,
			latencyMeasured:    true,
			dbMeasured:         true,
			lockWaitMeasured:   true,
			lockWaitWasZero:    publishLockWait == 0,
			encodeMeasured:     true,
			peakLiveMeasured:   true,
			bytesMeasured:      true,
			queryCountMeasured: true,
		})
	})

	t.Run("25 root embedded delta near byte ceiling", func(t *testing.T) {
		ctx, cancel := context.WithTimeout(
			context.Background(),
			90*time.Second,
		)
		defer cancel()
		ownerUID := newUserPmcsTestUser(t)
		repository := owned.NewRepository(
			persistence.NewStore(
				performanceDB,
				config.TransactionMaxAttempts,
			),
			config,
		)
		revisionIDs := make([]uuid.UUID, 0, 25)
		for rootIndex := 0; rootIndex < 25; rootIndex++ {
			fixtureID := fmt.Sprintf(
				"embedded-delta-%s-%d",
				ownerUID,
				rootIndex,
			)
			revisionID := deterministicFixtureUUID(
				fixtureID,
				"revision",
			)
			input := deterministicRevisionTree(
				t,
				fixtureID,
				revisionID,
				deterministicTreeSize{
					Sections: 1,
					Items:    10,
					Notices:  10,
					Steps:    200,
				},
			)
			for sectionIndex := range input.Sections {
				for itemIndex := range input.Sections[sectionIndex].Items {
					for stepIndex := range input.Sections[sectionIndex].
						Items[itemIndex].ProcedureSteps {
						input.Sections[sectionIndex].
							Items[itemIndex].
							ProcedureSteps[stepIndex].
							StepText = strings.Repeat("x", 3_700)
					}
				}
			}
			prepared, err := shared.PrepareDraft(input, config)
			require.NoError(t, err)
			_, err = repository.Create(
				ctx,
				ownerUID,
				uuid.New(),
				prepared,
				shared.Precondition{
					Mode: shared.PreconditionCreate,
				},
			)
			require.NoError(t, err)
			revisionIDs = append(revisionIDs, revisionID)
		}
		queryCounter.reset()
		peakSampler := startPeakLiveSampler()
		lockSampler := startLockWaitSampler(ctx, performanceApplicationName)
		started := time.Now()
		delta, err := userpmcssync.NewRepository(
			persistence.NewStore(performanceDB, 3),
		).GetDelta(
			ctx,
			ownerUID,
			0,
			25,
			config.MaxDeltaResponseBytes,
		)
		dbDuration := time.Since(started)
		deltaQueryCount := queryCounter.value()
		require.NoError(t, err)
		require.Positive(t, deltaQueryCount)
		require.LessOrEqual(t, deltaQueryCount, 20)
		require.False(t, delta.HasMore)
		require.Equal(t, int64(25), delta.ThroughCursor)
		encodeStarted := time.Now()
		payload, err := json.Marshal(delta)
		require.NoError(t, err)
		encodeDuration := time.Since(encodeStarted)
		require.Len(t, delta.Changes, 25)
		require.Greater(t, len(payload), 18*1024*1024)
		require.LessOrEqual(t, len(payload), config.MaxDeltaResponseBytes)
		for _, change := range delta.Changes {
			require.NotNil(t, change.Checklist)
			require.NotNil(t, change.Checklist.Draft)
			require.NotEmpty(t, change.Checklist.Draft.Sections)
		}
		queryer := &countingQueryer{database: testDB}
		loaded, err := persistence.LoadRevisionTrees(
			ctx,
			queryer,
			revisionIDs,
		)
		require.NoError(t, err)
		require.Len(t, loaded, 25)
		require.Equal(t, 7, queryer.queryCount)
		gzipBytes := compressedSize(t, payload)
		peakLiveBytes := peakSampler.finish()
		lockWait := finishLockWaitSampler(t, lockSampler)
		capturedProductionQueries["merged account delta"] =
			requireObservedQuery(
				t,
				queryCounter.snapshot(),
				"user_pmcs_account_delta_roots",
			)
		logPerformanceEvidence(t, performanceEvidence{
			scenario:           "25_root_embedded_delta",
			latencies:          []time.Duration{dbDuration},
			dbDuration:         queryCounter.databaseDuration(),
			lockWait:           lockWait,
			encodeDuration:     encodeDuration,
			peakLiveBytes:      peakLiveBytes,
			uncompressedBytes:  len(payload),
			gzipBytes:          gzipBytes,
			queryCount:         deltaQueryCount,
			latencyMeasured:    true,
			dbMeasured:         true,
			lockWaitMeasured:   true,
			lockWaitWasZero:    lockWait == 0,
			encodeMeasured:     true,
			peakLiveMeasured:   true,
			bytesMeasured:      true,
			queryCountMeasured: true,
		})
	})

	t.Run("20 concurrent independent users", func(t *testing.T) {
		ctx, cancel := context.WithTimeout(
			context.Background(),
			30*time.Second,
		)
		defer cancel()
		const userCount = 20
		users := make([]string, userCount)
		drafts := make([]shared.PreparedRevision, userCount)
		for index := range users {
			users[index] = newUserPmcsTestUser(t)
			drafts[index] = preparedTree(t, uuid.New())
		}
		type concurrentOutcome struct {
			duration time.Duration
			result   *owned.MutationResult
			err      error
		}
		start := make(chan struct{})
		outcomes := make(chan concurrentOutcome, userCount)
		repository := owned.NewRepository(
			persistence.NewStore(
				performanceDB,
				config.TransactionMaxAttempts,
			),
			config,
		)
		queryCounter.reset()
		peakSampler := startPeakLiveSampler()
		lockSampler := startLockWaitSampler(ctx, performanceApplicationName)
		var workers sync.WaitGroup
		for index := range users {
			index := index
			workers.Add(1)
			go func() {
				defer workers.Done()
				<-start
				started := time.Now()
				result, err := repository.Create(
					ctx,
					users[index],
					uuid.New(),
					drafts[index],
					shared.Precondition{
						Mode: shared.PreconditionCreate,
					},
				)
				outcomes <- concurrentOutcome{
					duration: time.Since(started),
					result:   result,
					err:      err,
				}
			}()
		}
		close(start)
		waitForWorkers(t, &workers, 25*time.Second)
		close(outcomes)
		samples := make([]time.Duration, 0, userCount)
		aggregates := make([]shared.ChecklistAggregate, 0, userCount)
		for outcome := range outcomes {
			require.NoError(t, outcome.err)
			samples = append(samples, outcome.duration)
			require.NotNil(t, outcome.result)
			aggregates = append(aggregates, outcome.result.Aggregate)
		}
		require.Len(t, samples, userCount)
		concurrentQueryCount := queryCounter.value()
		require.Equal(t, userCount*30, concurrentQueryCount)
		encodeStarted := time.Now()
		payload, err := json.Marshal(aggregates)
		require.NoError(t, err)
		encodeDuration := time.Since(encodeStarted)
		gzipBytes := compressedSize(t, payload)
		peakLiveBytes := peakSampler.finish()
		lockWait := finishLockWaitSampler(t, lockSampler)
		logPerformanceEvidence(t, performanceEvidence{
			scenario:           "20_concurrent_independent_users",
			latencies:          samples,
			dbDuration:         queryCounter.databaseDuration(),
			lockWait:           lockWait,
			encodeDuration:     encodeDuration,
			peakLiveBytes:      peakLiveBytes,
			uncompressedBytes:  len(payload),
			gzipBytes:          gzipBytes,
			queryCount:         concurrentQueryCount,
			latencyMeasured:    true,
			dbMeasured:         true,
			lockWaitMeasured:   true,
			lockWaitWasZero:    lockWait == 0,
			encodeMeasured:     true,
			peakLiveMeasured:   true,
			bytesMeasured:      true,
			queryCountMeasured: true,
		})
	})

	fixture := seedPerformanceSubscriptions(t, 500)
	t.Run("filtered and unfiltered browse", func(t *testing.T) {
		ctx, cancel := context.WithTimeout(
			context.Background(),
			20*time.Second,
		)
		defer cancel()
		repository := community.NewRepository(
			persistence.NewStore(performanceDB, 3),
			config,
		)
		queryCounter.reset()
		peakSampler := startPeakLiveSampler()
		lockSampler := startLockWaitSampler(ctx, performanceApplicationName)
		unfilteredStarted := time.Now()
		unfiltered, err := repository.Browse(
			ctx,
			shared.CommunityBrowseFilter{Limit: 50},
		)
		unfilteredDuration := time.Since(unfilteredStarted)
		require.NoError(t, err)
		require.Len(t, unfiltered.Items, 50)
		filteredStarted := time.Now()
		filtered, err := repository.Browse(
			ctx,
			shared.CommunityBrowseFilter{
				Limit:           50,
				NormalizedModel: fixture.normalizedName,
			},
		)
		filteredDuration := time.Since(filteredStarted)
		require.NoError(t, err)
		require.Len(t, filtered.Items, 1)
		browseQueryCount := queryCounter.value()
		require.Equal(t, 4, browseQueryCount)
		encodeStarted := time.Now()
		payload, err := json.Marshal([]any{unfiltered, filtered})
		require.NoError(t, err)
		encodeDuration := time.Since(encodeStarted)
		gzipBytes := compressedSize(t, payload)
		peakLiveBytes := peakSampler.finish()
		lockWait := finishLockWaitSampler(t, lockSampler)
		queries := queryCounter.snapshot()
		capturedProductionQueries["active recent browse"] =
			requireObservedQuery(
				t,
				queries,
				"WHERE source.status = 'active'",
			)
		capturedProductionQueries["exact model browse"] =
			requireObservedQuery(t, queries, "EXISTS")
		logPerformanceEvidence(t, performanceEvidence{
			scenario: "community_browse_filtered_unfiltered",
			latencies: []time.Duration{
				unfilteredDuration,
				filteredDuration,
			},
			dbDuration:         queryCounter.databaseDuration(),
			lockWait:           lockWait,
			encodeDuration:     encodeDuration,
			peakLiveBytes:      peakLiveBytes,
			uncompressedBytes:  len(payload),
			gzipBytes:          gzipBytes,
			queryCount:         browseQueryCount,
			latencyMeasured:    true,
			dbMeasured:         true,
			lockWaitMeasured:   true,
			lockWaitWasZero:    lockWait == 0,
			encodeMeasured:     true,
			peakLiveMeasured:   true,
			bytesMeasured:      true,
			queryCountMeasured: true,
		})
	})

	t.Run("500 subscription update discovery", func(t *testing.T) {
		ctx, cancel := context.WithTimeout(
			context.Background(),
			20*time.Second,
		)
		defer cancel()
		repository := subscriptions.NewRepository(
			persistence.NewStore(performanceDB, 3),
			config,
		)
		queryCounter.reset()
		peakSampler := startPeakLiveSampler()
		lockSampler := startLockWaitSampler(ctx, performanceApplicationName)
		started := time.Now()
		page, err := repository.ListUpdates(
			ctx,
			fixture.subscriberUID,
			nil,
			100,
		)
		duration := time.Since(started)
		updateQueryCount := queryCounter.value()
		require.NoError(t, err)
		require.Equal(t, 1, updateQueryCount)
		require.Len(t, page.Items, 100)
		require.True(t, page.HasMore)
		require.NotNil(t, page.NextCursor)
		for _, item := range page.Items {
			require.True(t, item.UpdateAvailable)
		}
		encodeStarted := time.Now()
		payload, err := json.Marshal(page)
		require.NoError(t, err)
		encodeDuration := time.Since(encodeStarted)
		require.NotContains(t, string(payload), `"sections"`)
		gzipBytes := compressedSize(t, payload)
		peakLiveBytes := peakSampler.finish()
		lockWait := finishLockWaitSampler(t, lockSampler)
		capturedProductionQueries["subscription updates"] =
			requireObservedQuery(
				t,
				queryCounter.snapshot(),
				"FROM user_pmcs_subscriptions AS subscription",
				"installed.revision_number",
			)
		logPerformanceEvidence(t, performanceEvidence{
			scenario:           "500_subscription_update_discovery",
			latencies:          []time.Duration{duration},
			dbDuration:         queryCounter.databaseDuration(),
			lockWait:           lockWait,
			encodeDuration:     encodeDuration,
			peakLiveBytes:      peakLiveBytes,
			uncompressedBytes:  len(payload),
			gzipBytes:          gzipBytes,
			queryCount:         updateQueryCount,
			latencyMeasured:    true,
			dbMeasured:         true,
			lockWaitMeasured:   true,
			lockWaitWasZero:    lockWait == 0,
			encodeMeasured:     true,
			peakLiveMeasured:   true,
			bytesMeasured:      true,
			queryCountMeasured: true,
		})
	})

	t.Run("50 publication first sync interruption and resume", func(t *testing.T) {
		ctx, cancel := context.WithTimeout(
			context.Background(),
			2*time.Minute,
		)
		defer cancel()
		ownerUID := newUserPmcsTestUser(t)
		repository := owned.NewRepository(
			persistence.NewStore(
				performanceDB,
				config.TransactionMaxAttempts,
			),
			config,
		)
		expectedChecklistIDs := make(map[uuid.UUID]struct{}, 50)
		for index := 0; index < 50; index++ {
			checklistID := uuid.New()
			draft := preparedTree(t, uuid.New())
			created, err := repository.Create(
				ctx,
				ownerUID,
				checklistID,
				draft,
				shared.Precondition{Mode: shared.PreconditionCreate},
			)
			require.NoError(t, err)
			_, err = repository.Publish(
				ctx,
				ownerUID,
				checklistID,
				preparePublication(t, draft.Input, 1),
				checklistPrecondition(
					checklistID,
					created.Aggregate.SyncVersion,
				),
			)
			require.NoError(t, err)
			expectedChecklistIDs[checklistID] = struct{}{}
		}

		queryCounter.reset()
		peakSampler := startPeakLiveSampler()
		deltaRepository := userpmcssync.NewRepository(
			persistence.NewStore(performanceDB, 3),
		)
		samples := make([]time.Duration, 0, 3)
		firstStarted := time.Now()
		firstPage, err := deltaRepository.GetDelta(
			ctx,
			ownerUID,
			0,
			25,
			config.MaxDeltaResponseBytes,
		)
		samples = append(samples, time.Since(firstStarted))
		require.NoError(t, err)
		require.True(t, firstPage.HasMore)
		require.Len(t, firstPage.Changes, 25)
		durableCursor := firstPage.ThroughCursor
		require.Positive(t, durableCursor)

		blocker, observer := dedicatedConnections(t, ctx)
		lockTx, err := blocker.BeginTx(ctx, nil)
		require.NoError(t, err)
		_, err = lockTx.ExecContext(
			ctx,
			`LOCK TABLE user_pmcs_subscriptions IN ACCESS EXCLUSIVE MODE`,
		)
		require.NoError(t, err)
		blockerPID := backendPID(t, ctx, blocker)
		interruptedContext, interrupt := context.WithCancel(ctx)
		interruptedResults := make(chan error, 1)
		interruptedStarted := time.Now()
		go func() {
			_, interruptErr := deltaRepository.GetDelta(
				interruptedContext,
				ownerUID,
				durableCursor,
				25,
				config.MaxDeltaResponseBytes,
			)
			interruptedResults <- interruptErr
		}()
		lockWait := waitForApplicationLockWait(
			t,
			ctx,
			observer,
			blockerPID,
			performanceApplicationName,
			"user_pmcs_account_delta_roots",
			1,
		)
		interrupt()
		interruptErr := <-interruptedResults
		samples = append(samples, time.Since(interruptedStarted))
		require.Error(t, interruptErr)
		require.ErrorIs(t, interruptedContext.Err(), context.Canceled)
		require.ErrorContains(
			t,
			interruptErr,
			"canceling statement due to user request",
		)
		require.NoError(t, lockTx.Rollback())

		resumeStarted := time.Now()
		secondPage, err := deltaRepository.GetDelta(
			ctx,
			ownerUID,
			durableCursor,
			25,
			config.MaxDeltaResponseBytes,
		)
		samples = append(samples, time.Since(resumeStarted))
		require.NoError(t, err)
		require.False(t, secondPage.HasMore)
		require.Len(t, secondPage.Changes, 25)
		require.Equal(t, int64(100), secondPage.AccountVersion)

		seen := make(map[uuid.UUID]int, 50)
		pages := []*userpmcssync.AccountDelta{firstPage, secondPage}
		var (
			encodeDuration    time.Duration
			uncompressedBytes int
			gzipBytes         int
		)
		for _, page := range pages {
			encodeStarted := time.Now()
			payload, marshalErr := json.Marshal(page)
			encodeDuration += time.Since(encodeStarted)
			require.NoError(t, marshalErr)
			require.LessOrEqual(
				t,
				len(payload),
				config.MaxDeltaResponseBytes,
			)
			uncompressedBytes += len(payload)
			gzipBytes += compressedSize(t, payload)
			for _, change := range page.Changes {
				require.NotNil(t, change.Checklist)
				require.NotNil(t, change.Checklist.Publication)
				require.NotEmpty(
					t,
					change.Checklist.Publication.Sections,
				)
				seen[change.Checklist.ID]++
			}
		}
		require.Len(t, seen, 50)
		for checklistID := range expectedChecklistIDs {
			require.Equal(t, 1, seen[checklistID])
		}
		syncQueryCount := queryCounter.value()
		require.Positive(t, syncQueryCount)
		require.LessOrEqual(t, syncQueryCount, 30)
		peakLiveBytes := peakSampler.finish()
		logPerformanceEvidence(t, performanceEvidence{
			scenario:           "50_publication_first_sync_resume",
			latencies:          samples,
			dbDuration:         queryCounter.databaseDuration(),
			lockWait:           lockWait,
			encodeDuration:     encodeDuration,
			peakLiveBytes:      peakLiveBytes,
			uncompressedBytes:  uncompressedBytes,
			gzipBytes:          gzipBytes,
			queryCount:         syncQueryCount,
			latencyMeasured:    true,
			dbMeasured:         true,
			lockWaitMeasured:   true,
			lockWaitWasZero:    false,
			encodeMeasured:     true,
			peakLiveMeasured:   true,
			bytesMeasured:      true,
			queryCountMeasured: true,
		})
	})

	t.Run("approved index plans", func(t *testing.T) {
		captureUserPmcsQueryPlans(
			t,
			fixture,
			capturedProductionQueries,
			performanceDB,
			queryCounter,
		)
	})
}

func BenchmarkMaximumTreeEncoding(b *testing.B) {
	input := maximumDeterministicTree(
		b,
		"benchmark-maximum-tree",
		deterministicFixtureUUID("benchmark-maximum-tree", "revision"),
	)
	b.ReportAllocs()
	b.ResetTimer()
	for range b.N {
		if _, err := json.Marshal(input); err != nil {
			b.Fatal(err)
		}
	}
}

func treeInputCounts(input shared.RevisionInput) (
	sections int,
	items int,
	notices int,
	steps int,
	maxItemsPerSection int,
) {
	sections = len(input.Sections)
	for _, section := range input.Sections {
		items += len(section.Items)
		maxItemsPerSection = max(maxItemsPerSection, len(section.Items))
		for _, item := range section.Items {
			notices += len(item.Notices)
			steps += len(item.ProcedureSteps)
		}
	}
	return sections, items, notices, steps, maxItemsPerSection
}

func paddedJSONAtSize(t *testing.T, value any, size int) []byte {
	t.Helper()
	payload := mustJSON(t, value)
	require.LessOrEqual(t, len(payload), size)
	return append(payload, bytes.Repeat([]byte{' '}, size-len(payload))...)
}

func assertPostgresTextFieldByteBoundaries(t *testing.T, ownerUID string) {
	t.Helper()
	requireUserPmcsTestDatabase(t, testDB)
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	insert := func(nameBytes int, descriptionBytes int) error {
		tx, err := testDB.BeginTx(ctx, nil)
		if err != nil {
			return err
		}
		defer tx.Rollback()
		checklistID := uuid.New()
		if _, err = tx.ExecContext(
			ctx,
			`INSERT INTO user_pmcs_checklists
			     (id, owner_uid, sync_version, account_change_version)
			 VALUES ($1, $2, 1, 1)`,
			checklistID,
			ownerUID,
		); err != nil {
			return err
		}
		_, err = tx.ExecContext(
			ctx,
			`INSERT INTO user_pmcs_revisions
			     (id, checklist_id, state, revision_number, name,
			      description, content_hash, published_at)
			 VALUES ($1, $2, 'published', 1, $3, $4, $5, now())`,
			uuid.New(),
			checklistID,
			strings.Repeat("n", nameBytes),
			strings.Repeat("d", descriptionBytes),
			make([]byte, 32),
		)
		return err
	}
	require.NoError(t, insert(8_192, 65_536))
	require.Error(t, insert(8_193, 1))
	require.Error(t, insert(1, 65_537))
}

type peakLiveSampler struct {
	stop chan struct{}
	done chan uint64
}

func startPeakLiveSampler() *peakLiveSampler {
	runtime.GC()
	var baseline runtime.MemStats
	runtime.ReadMemStats(&baseline)
	sampler := &peakLiveSampler{
		stop: make(chan struct{}),
		done: make(chan uint64, 1),
	}
	go func() {
		ticker := time.NewTicker(time.Millisecond)
		defer ticker.Stop()
		peak := baseline.HeapAlloc
		sample := func() {
			var current runtime.MemStats
			runtime.ReadMemStats(&current)
			if current.HeapAlloc > peak {
				peak = current.HeapAlloc
			}
		}
		for {
			select {
			case <-ticker.C:
				sample()
			case <-sampler.stop:
				sample()
				sampler.done <- peak - baseline.HeapAlloc
				return
			}
		}
	}()
	return sampler
}

func (sampler *peakLiveSampler) finish() uint64 {
	close(sampler.stop)
	return <-sampler.done
}

type lockWaitSample struct {
	duration time.Duration
	err      error
}

type lockWaitSampler struct {
	stop chan struct{}
	done chan lockWaitSample
}

func startLockWaitSampler(
	ctx context.Context,
	applicationName string,
) *lockWaitSampler {
	sampler := &lockWaitSampler{
		stop: make(chan struct{}),
		done: make(chan lockWaitSample, 1),
	}
	go func() {
		ticker := time.NewTicker(2 * time.Millisecond)
		defer ticker.Stop()
		lastSample := time.Now()
		wasWaiting := false
		var total time.Duration
		for {
			select {
			case <-ticker.C:
				now := time.Now()
				if wasWaiting {
					total += now.Sub(lastSample)
				}
				lastSample = now
				err := testDB.QueryRowContext(
					ctx,
					`SELECT EXISTS (
					     SELECT 1
					     FROM pg_stat_activity
					     WHERE application_name = $1
					       AND wait_event_type = 'Lock'
					 )`,
					applicationName,
				).Scan(&wasWaiting)
				if err != nil {
					sampler.done <- lockWaitSample{
						duration: total,
						err:      err,
					}
					return
				}
			case <-sampler.stop:
				now := time.Now()
				if wasWaiting {
					total += now.Sub(lastSample)
				}
				sampler.done <- lockWaitSample{duration: total}
				return
			case <-ctx.Done():
				sampler.done <- lockWaitSample{
					duration: total,
					err:      ctx.Err(),
				}
				return
			}
		}
	}()
	return sampler
}

func finishLockWaitSampler(
	t *testing.T,
	sampler *lockWaitSampler,
) time.Duration {
	t.Helper()
	close(sampler.stop)
	result := <-sampler.done
	require.NoError(t, result.err)
	return result.duration
}

func waitForApplicationLockWait(
	t *testing.T,
	ctx context.Context,
	observer *sql.Conn,
	blockerPID int,
	applicationName string,
	queryToken string,
	expectedWaiters int,
) time.Duration {
	t.Helper()
	var measured time.Duration
	require.Eventually(t, func() bool {
		rows, err := observer.QueryContext(
			ctx,
			`WITH RECURSIVE blocked_chain(pid) AS (
			     SELECT $1::integer
			     UNION
			     SELECT activity.pid
			     FROM pg_stat_activity AS activity
			     JOIN blocked_chain AS blocker
			       ON blocker.pid = ANY(pg_blocking_pids(activity.pid))
			 )
			 SELECT activity.pid,
			        EXTRACT(
			            EPOCH FROM clock_timestamp() - MIN(waiting.waitstart)
			        )
			 FROM pg_stat_activity AS activity
			 JOIN blocked_chain ON blocked_chain.pid = activity.pid
			 JOIN pg_locks AS waiting
			   ON waiting.pid = activity.pid
			  AND NOT waiting.granted
			 WHERE activity.application_name = $2
			   AND activity.wait_event_type = 'Lock'
			   AND activity.query LIKE '%' || $3 || '%'
			 GROUP BY activity.pid
			 ORDER BY activity.pid`,
			blockerPID,
			applicationName,
			queryToken,
		)
		if err != nil {
			return false
		}
		defer rows.Close()
		waiters := 0
		var longestSeconds float64
		for rows.Next() {
			var (
				pid     int
				seconds float64
			)
			if err := rows.Scan(&pid, &seconds); err != nil {
				return false
			}
			waiters++
			longestSeconds = max(longestSeconds, seconds)
		}
		if rows.Err() != nil || waiters != expectedWaiters {
			return false
		}
		measured = time.Duration(longestSeconds * float64(time.Second))
		return measured > 0
	}, 5*time.Second, 2*time.Millisecond)
	return measured
}

func requireObservedQuery(
	t *testing.T,
	queries []observedQuery,
	requiredTokens ...string,
) observedQuery {
	t.Helper()
	for _, query := range queries {
		matches := true
		for _, token := range requiredTokens {
			if !strings.Contains(query.query, token) {
				matches = false
				break
			}
		}
		if matches {
			return query
		}
	}
	require.Failf(
		t,
		"production query was not captured",
		"required tokens: %v",
		requiredTokens,
	)
	return observedQuery{}
}

func compressedSize(t *testing.T, payload []byte) int {
	t.Helper()
	var compressed bytes.Buffer
	writer := gzip.NewWriter(&compressed)
	_, err := writer.Write(payload)
	require.NoError(t, err)
	require.NoError(t, writer.Close())
	return compressed.Len()
}

func logPerformanceEvidence(t *testing.T, evidence performanceEvidence) {
	t.Helper()
	require.NoError(t, validatePerformanceEvidence(evidence))
	samples := append([]time.Duration(nil), evidence.latencies...)
	sort.Slice(samples, func(left, right int) bool {
		return samples[left] < samples[right]
	})
	var p50, p95 time.Duration
	if len(samples) > 0 {
		p50 = samples[percentileIndex(len(samples), 50)]
		p95 = samples[percentileIndex(len(samples), 95)]
	}
	t.Logf(
		"scenario=%s p50=%s p95=%s db=%s lock_wait=%s encode=%s "+
			"peak_live_bytes=%d uncompressed_bytes=%d gzip_bytes=%d "+
			"query_count=%d validity=latency:%t,db:%t,lock_wait:%t,"+
			"lock_wait_semantic_zero:%t,encode:%t,peak_live:%t,"+
			"bytes:%t,query_count:%t",
		evidence.scenario,
		p50,
		p95,
		evidence.dbDuration,
		evidence.lockWait,
		evidence.encodeDuration,
		evidence.peakLiveBytes,
		evidence.uncompressedBytes,
		evidence.gzipBytes,
		evidence.queryCount,
		evidence.latencyMeasured,
		evidence.dbMeasured,
		evidence.lockWaitMeasured,
		evidence.lockWaitWasZero,
		evidence.encodeMeasured,
		evidence.peakLiveMeasured,
		evidence.bytesMeasured,
		evidence.queryCountMeasured,
	)
}

func validatePerformanceEvidence(evidence performanceEvidence) error {
	measurements := []struct {
		name  string
		valid bool
	}{
		{"latency", evidence.latencyMeasured && len(evidence.latencies) > 0},
		{"db duration", evidence.dbMeasured},
		{"lock wait", evidence.lockWaitMeasured},
		{"encode duration", evidence.encodeMeasured},
		{"peak live bytes", evidence.peakLiveMeasured},
		{"payload bytes", evidence.bytesMeasured},
		{"query count", evidence.queryCountMeasured},
	}
	var missing []string
	for _, measurement := range measurements {
		if !measurement.valid {
			missing = append(missing, measurement.name)
		}
	}
	if len(missing) > 0 {
		return fmt.Errorf(
			"scenario %q has unmeasured fields: %s",
			evidence.scenario,
			strings.Join(missing, ", "),
		)
	}
	if evidence.lockWaitWasZero && evidence.lockWait != 0 {
		return fmt.Errorf(
			"scenario %q marks nonzero lock wait as semantic zero",
			evidence.scenario,
		)
	}
	return nil
}

func percentileIndex(sampleCount, percentile int) int {
	index := (sampleCount*percentile + 99) / 100
	return max(0, min(sampleCount-1, index-1))
}

func seedPerformanceSubscriptions(
	t *testing.T,
	count int,
) performanceSubscriptionFixture {
	t.Helper()
	requireUserPmcsTestDatabase(t, testDB)
	ctx, cancel := context.WithTimeout(context.Background(), 45*time.Second)
	defer cancel()

	fixture := performanceSubscriptionFixture{
		ownerUID:       "perf-owner-" + uuid.NewString(),
		subscriberUID:  "perf-subscriber-" + uuid.NewString(),
		checklistIDs:   make([]uuid.UUID, count),
		installedIDs:   make([]uuid.UUID, count),
		currentIDs:     make([]uuid.UUID, count),
		normalizedName: "exact-performance-model",
	}
	for index := 0; index < 4; index++ {
		fixture.noiseUserUIDs = append(
			fixture.noiseUserUIDs,
			fmt.Sprintf("perf-noise-%d-%s", index, uuid.NewString()),
		)
	}
	userUIDs := append(
		[]string{fixture.ownerUID, fixture.subscriberUID},
		fixture.noiseUserUIDs...,
	)
	tx, err := testDB.BeginTx(ctx, nil)
	require.NoError(t, err)
	defer func() {
		_ = tx.Rollback()
	}()
	for index, uid := range userUIDs {
		_, err = tx.ExecContext(
			ctx,
			`INSERT INTO users
			     (uid, email, username, created_at, is_enabled)
			 VALUES ($1, $2, $3, now(), TRUE)`,
			uid,
			uid+"@example.com",
			fmt.Sprintf("perf-%d", index),
		)
		require.NoError(t, err)
	}

	checklists := make([]string, count)
	revisionIDs := make([]string, 0, count*2)
	revisionChecklists := make([]string, 0, count*2)
	revisionStates := make([]string, 0, count*2)
	revisionNumbers := make([]int64, 0, count*2)
	releaseIDs := make([]string, 0, count*2)
	releaseChecklists := make([]string, 0, count*2)
	currentIDs := make([]string, count)
	installedIDs := make([]string, count)
	modelRevisionIDs := make([]string, count)
	modelDisplay := make([]string, count)
	modelNormalized := make([]string, count)
	for index := 0; index < count; index++ {
		fixture.checklistIDs[index] = uuid.New()
		fixture.installedIDs[index] = uuid.New()
		fixture.currentIDs[index] = uuid.New()
		checklists[index] = fixture.checklistIDs[index].String()
		installedIDs[index] = fixture.installedIDs[index].String()
		currentIDs[index] = fixture.currentIDs[index].String()
		for _, revision := range []struct {
			id     uuid.UUID
			state  string
			number int64
		}{
			{fixture.installedIDs[index], "superseded", 1},
			{fixture.currentIDs[index], "published", 2},
		} {
			revisionIDs = append(revisionIDs, revision.id.String())
			revisionChecklists = append(
				revisionChecklists,
				fixture.checklistIDs[index].String(),
			)
			revisionStates = append(revisionStates, revision.state)
			revisionNumbers = append(revisionNumbers, revision.number)
			releaseIDs = append(releaseIDs, revision.id.String())
			releaseChecklists = append(
				releaseChecklists,
				fixture.checklistIDs[index].String(),
			)
		}
		modelRevisionIDs[index] = fixture.currentIDs[index].String()
		model := fmt.Sprintf("performance-model-%d", index)
		if index == 0 {
			model = fixture.normalizedName
		}
		modelDisplay[index] = model
		modelNormalized[index] = model
	}
	_, err = tx.ExecContext(
		ctx,
		`INSERT INTO user_pmcs_checklists
		     (id, owner_uid, sync_version, account_change_version)
		 SELECT id::uuid, $1, 1, ordinal
		 FROM unnest($2::text[]) WITH ORDINALITY AS rows(id, ordinal)`,
		fixture.ownerUID,
		pq.Array(checklists),
	)
	require.NoError(t, err)
	_, err = tx.ExecContext(
		ctx,
		`INSERT INTO user_pmcs_revisions
		     (id, checklist_id, state, revision_number, name, description,
		      content_hash, published_at)
		 SELECT id::uuid, checklist_id::uuid, state, revision_number::integer,
		        'n', 'd', decode(repeat('00', 32), 'hex'), now()
		 FROM unnest($1::text[], $2::text[], $3::text[], $4::bigint[])
		      AS rows(id, checklist_id, state, revision_number)`,
		pq.Array(revisionIDs),
		pq.Array(revisionChecklists),
		pq.Array(revisionStates),
		pq.Array(revisionNumbers),
	)
	require.NoError(t, err)
	_, err = tx.ExecContext(
		ctx,
		`INSERT INTO user_pmcs_community_releases
		     (revision_id, checklist_id)
		 SELECT revision_id::uuid, checklist_id::uuid
		 FROM unnest($1::text[], $2::text[])
		      AS rows(revision_id, checklist_id)`,
		pq.Array(releaseIDs),
		pq.Array(releaseChecklists),
	)
	require.NoError(t, err)
	_, err = tx.ExecContext(
		ctx,
		`INSERT INTO user_pmcs_community_sources
		     (checklist_id, status, current_release_revision_id,
		      latest_release_revision_number, first_released_at, updated_at)
		 SELECT checklist_id::uuid, 'active', current_id::uuid,
		        2, now() - (ordinal * interval '1 second'), now()
		 FROM unnest($1::text[], $2::text[]) WITH ORDINALITY
		      AS rows(checklist_id, current_id, ordinal)`,
		pq.Array(checklists),
		pq.Array(currentIDs),
	)
	require.NoError(t, err)
	_, err = tx.ExecContext(
		ctx,
		`INSERT INTO user_pmcs_revision_models
		     (revision_id, display_text, normalized_text)
		 SELECT revision_id::uuid, display_text, normalized_text
		 FROM unnest($1::text[], $2::text[], $3::text[])
		      AS rows(revision_id, display_text, normalized_text)`,
		pq.Array(modelRevisionIDs),
		pq.Array(modelDisplay),
		pq.Array(modelNormalized),
	)
	require.NoError(t, err)
	_, err = tx.ExecContext(
		ctx,
		`INSERT INTO user_pmcs_checklists
		     (id, owner_uid, sync_version, account_change_version)
		 SELECT gen_random_uuid(), $1, 1, $2 + sequence
		 FROM generate_series(1, 5000) AS sequence`,
		fixture.ownerUID,
		count,
	)
	require.NoError(t, err)
	_, err = tx.ExecContext(
		ctx,
		`INSERT INTO user_pmcs_revisions
		     (id, checklist_id, state, revision_number, name, description,
		      content_hash, published_at)
		 SELECT gen_random_uuid(), checklist.id, 'draft', NULL, 'n', 'd',
		        decode(repeat('00', 32), 'hex'), NULL
		 FROM user_pmcs_checklists AS checklist
		 WHERE checklist.owner_uid = $1
		   AND checklist.account_change_version > $2`,
		fixture.ownerUID,
		count,
	)
	require.NoError(t, err)
	_, err = tx.ExecContext(
		ctx,
		`INSERT INTO user_pmcs_community_sources
		     (checklist_id, status, current_release_revision_id,
		      latest_release_revision_number, first_released_at, updated_at,
		      retired_at)
		 SELECT checklist.id, 'retired', NULL, 1, now(), now(), now()
		 FROM user_pmcs_checklists AS checklist
		 WHERE checklist.owner_uid = $1
		   AND checklist.account_change_version > $2`,
		fixture.ownerUID,
		count,
	)
	require.NoError(t, err)
	for _, subscriberUID := range append(
		[]string{fixture.subscriberUID},
		fixture.noiseUserUIDs...,
	) {
		_, err = tx.ExecContext(
			ctx,
			`INSERT INTO user_pmcs_subscriptions
			     (subscriber_uid, checklist_id, installed_revision_id,
			      sync_version, account_change_version)
			 SELECT $1, checklist_id::uuid, installed_id::uuid, 1, ordinal
			 FROM unnest($2::text[], $3::text[]) WITH ORDINALITY
			      AS rows(checklist_id, installed_id, ordinal)`,
			subscriberUID,
			pq.Array(checklists),
			pq.Array(installedIDs),
		)
		require.NoError(t, err)
	}
	_, err = tx.ExecContext(
		ctx,
		`INSERT INTO user_pmcs_subscriptions
		     (subscriber_uid, checklist_id, installed_revision_id,
		      sync_version, account_change_version, deleted_at)
		 SELECT $1, checklist.id, NULL, 1,
		        $2 + checklist.account_change_version, now()
		 FROM user_pmcs_checklists AS checklist
		 WHERE checklist.owner_uid = $3
		   AND checklist.account_change_version > $2`,
		fixture.subscriberUID,
		count,
		fixture.ownerUID,
	)
	require.NoError(t, err)
	require.NoError(t, tx.Commit())
	for _, table := range []string{
		"user_pmcs_checklists",
		"user_pmcs_revisions",
		"user_pmcs_revision_models",
		"user_pmcs_community_releases",
		"user_pmcs_community_sources",
		"user_pmcs_subscriptions",
	} {
		_, err = testDB.ExecContext(ctx, "ANALYZE "+table)
		require.NoError(t, err)
	}

	t.Cleanup(func() {
		cleanupCtx, cleanupCancel := context.WithTimeout(
			context.Background(),
			30*time.Second,
		)
		defer cleanupCancel()
		_, cleanupErr := testDB.ExecContext(
			cleanupCtx,
			`DELETE FROM user_pmcs_subscriptions
			 WHERE checklist_id = ANY($1::uuid[])`,
			pq.Array(checklists),
		)
		require.NoError(t, cleanupErr)
		_, cleanupErr = testDB.ExecContext(
			cleanupCtx,
			`DELETE FROM user_pmcs_community_sources AS source
			 USING user_pmcs_checklists AS checklist
			 WHERE source.checklist_id = checklist.id
			   AND checklist.owner_uid = $1`,
			fixture.ownerUID,
		)
		require.NoError(t, cleanupErr)
		_, cleanupErr = testDB.ExecContext(
			cleanupCtx,
			`DELETE FROM user_pmcs_community_releases AS release
			 USING user_pmcs_checklists AS checklist
			 WHERE release.checklist_id = checklist.id
			   AND checklist.owner_uid = $1`,
			fixture.ownerUID,
		)
		require.NoError(t, cleanupErr)
		_, cleanupErr = testDB.ExecContext(
			cleanupCtx,
			`DELETE FROM user_pmcs_checklists
			 WHERE owner_uid = $1`,
			fixture.ownerUID,
		)
		require.NoError(t, cleanupErr)
		_, cleanupErr = testDB.ExecContext(
			cleanupCtx,
			`DELETE FROM users WHERE uid = ANY($1)`,
			pq.Array(userUIDs),
		)
		require.NoError(t, cleanupErr)
	})
	return fixture
}

func captureUserPmcsQueryPlans(
	t *testing.T,
	fixture performanceSubscriptionFixture,
	captured map[string]observedQuery,
	performanceDB *sql.DB,
	counter *queryCounter,
) {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), 45*time.Second)
	defer cancel()

	planOwnerUID := newUserPmcsTestUser(t)
	planChecklistIDs := make([]string, 25)
	planRevisionIDs := make([]string, 25)
	for index := range planChecklistIDs {
		planChecklistIDs[index] = uuid.NewString()
		planRevisionIDs[index] = uuid.NewString()
	}
	_, err := testDB.ExecContext(
		ctx,
		`INSERT INTO user_pmcs_checklists
		     (id, owner_uid, sync_version, account_change_version)
		 SELECT id::uuid, $1, 1, ordinal
		 FROM unnest($2::text[]) WITH ORDINALITY AS rows(id, ordinal)`,
		planOwnerUID,
		pq.Array(planChecklistIDs),
	)
	require.NoError(t, err)
	_, err = testDB.ExecContext(
		ctx,
		`INSERT INTO user_pmcs_revisions
		     (id, checklist_id, state, revision_number, name, description,
		      content_hash, published_at)
		 SELECT revision_id::uuid, checklist_id::uuid, 'draft', NULL,
		        'plan draft', 'plan draft',
		        decode(repeat('00', 32), 'hex'), NULL
		 FROM unnest($1::text[], $2::text[])
		      AS rows(revision_id, checklist_id)`,
		pq.Array(planRevisionIDs),
		pq.Array(planChecklistIDs),
	)
	require.NoError(t, err)
	_, err = testDB.ExecContext(
		ctx,
		`INSERT INTO user_pmcs_sync_state (user_uid, current_version)
		 VALUES ($1, 25)`,
		planOwnerUID,
	)
	require.NoError(t, err)
	_, err = testDB.ExecContext(ctx, `ANALYZE user_pmcs_checklists`)
	require.NoError(t, err)

	planSubscriberUID := newUserPmcsTestUser(t)
	planSubscriptionIDs := uuidStrings(fixture.checklistIDs[:25])
	planInstalledIDs := uuidStrings(fixture.installedIDs[:25])
	_, err = testDB.ExecContext(
		ctx,
		`INSERT INTO user_pmcs_subscriptions
		     (subscriber_uid, checklist_id, installed_revision_id,
		      sync_version, account_change_version)
		 SELECT $1, checklist_id::uuid, installed_revision_id::uuid,
		        1, ordinal
		 FROM unnest($2::text[], $3::text[]) WITH ORDINALITY
		      AS rows(checklist_id, installed_revision_id, ordinal)`,
		planSubscriberUID,
		pq.Array(planSubscriptionIDs),
		pq.Array(planInstalledIDs),
	)
	require.NoError(t, err)
	t.Cleanup(func() {
		cleanupCtx, cleanupCancel := context.WithTimeout(
			context.Background(),
			10*time.Second,
		)
		defer cleanupCancel()
		_, cleanupErr := testDB.ExecContext(
			cleanupCtx,
			`DELETE FROM user_pmcs_subscriptions
			 WHERE subscriber_uid = $1`,
			planSubscriberUID,
		)
		require.NoError(t, cleanupErr)
	})
	_, err = testDB.ExecContext(ctx, `ANALYZE user_pmcs_subscriptions`)
	require.NoError(t, err)

	config := shared.DefaultConfig()
	planRepository := owned.NewRepository(
		persistence.NewStore(performanceDB, config.TransactionMaxAttempts),
		config,
	)
	planChecklistID := uuid.New()
	planDraft := preparedTree(t, uuid.New())
	counter.reset()
	created, err := planRepository.Create(
		ctx,
		planOwnerUID,
		planChecklistID,
		planDraft,
		shared.Precondition{Mode: shared.PreconditionCreate},
	)
	require.NoError(t, err)
	captured["account limit count"] = requireObservedQuery(
		t,
		counter.snapshot(),
		"SELECT count(*)",
		"FROM user_pmcs_checklists",
		"deleted_at IS NULL",
	)
	counter.reset()
	published, err := planRepository.Publish(
		ctx,
		planOwnerUID,
		planChecklistID,
		preparePublication(t, planDraft.Input, 1),
		checklistPrecondition(
			planChecklistID,
			created.Aggregate.SyncVersion,
		),
	)
	require.NoError(t, err)
	counter.reset()
	_, err = planRepository.DeleteChecklist(
		ctx,
		planOwnerUID,
		planChecklistID,
		checklistPrecondition(
			planChecklistID,
			published.Aggregate.SyncVersion,
		),
	)
	require.NoError(t, err)
	captured["active pin lookup"] = requireObservedQuery(
		t,
		counter.snapshot(),
		"FROM user_pmcs_subscriptions",
		"FOR UPDATE",
	)

	counter.reset()
	_, err = userpmcssync.NewRepository(
		persistence.NewStore(performanceDB, 3),
	).GetDelta(
		ctx,
		planOwnerUID,
		0,
		25,
		config.MaxDeltaResponseBytes,
	)
	require.NoError(t, err)
	captured["merged account delta"] = requireObservedQuery(
		t,
		counter.snapshot(),
		"user_pmcs_account_delta_roots",
	)

	counter.reset()
	_, err = persistence.LoadRevisionTrees(
		ctx,
		performanceDB,
		fixture.currentIDs[:25],
	)
	require.NoError(t, err)
	captured["batched revision loader"] = requireObservedQuery(
		t,
		counter.snapshot(),
		"FROM user_pmcs_revisions",
		"WHERE id = ANY",
	)

	communityRepository := community.NewRepository(
		persistence.NewStore(performanceDB, 3),
		config,
	)
	counter.reset()
	_, err = communityRepository.Browse(
		ctx,
		shared.CommunityBrowseFilter{Limit: 50},
	)
	require.NoError(t, err)
	_, err = communityRepository.Browse(
		ctx,
		shared.CommunityBrowseFilter{
			Limit:           50,
			NormalizedModel: fixture.normalizedName,
		},
	)
	require.NoError(t, err)
	browseQueries := counter.snapshot()
	captured["active recent browse"] = requireObservedQuery(
		t,
		browseQueries,
		"WHERE source.status = 'active'",
	)
	captured["exact model browse"] = requireObservedQuery(
		t,
		browseQueries,
		"EXISTS",
	)

	counter.reset()
	_, err = subscriptions.NewRepository(
		persistence.NewStore(performanceDB, 3),
		config,
	).ListUpdates(ctx, planSubscriberUID, nil, 100)
	require.NoError(t, err)
	captured["subscription updates"] = requireObservedQuery(
		t,
		counter.snapshot(),
		"FROM user_pmcs_subscriptions AS subscription",
		"installed.revision_number",
	)

	type relationPlanExpectation struct {
		relation        string
		approvedIndexes []string
	}
	plans := []struct {
		name         string
		observation  observedQuery
		args         []any
		expectations []relationPlanExpectation
	}{
		{
			name:        "owner delta branch",
			observation: captured["merged account delta"],
			args:        []any{planOwnerUID, int64(0), 26},
			expectations: []relationPlanExpectation{
				{
					relation: "user_pmcs_checklists",
					approvedIndexes: []string{
						"user_pmcs_checklists_owner_delta_idx",
					},
				},
			},
		},
		{
			name:        "subscription delta branch",
			observation: captured["merged account delta"],
			args:        []any{planSubscriberUID, int64(0), 26},
			expectations: []relationPlanExpectation{
				{
					relation: "user_pmcs_subscriptions",
					approvedIndexes: []string{
						"user_pmcs_subscriptions_delta_idx",
					},
				},
			},
		},
		{
			name:        "batched tree loader",
			observation: captured["batched revision loader"],
			expectations: []relationPlanExpectation{
				{
					relation: "user_pmcs_revisions",
					approvedIndexes: []string{
						"user_pmcs_revisions_pkey",
					},
				},
			},
		},
		{
			name:        "active recent browse",
			observation: captured["active recent browse"],
			expectations: []relationPlanExpectation{
				{
					relation: "user_pmcs_community_sources",
					approvedIndexes: []string{
						"user_pmcs_community_sources_recent_idx",
					},
				},
				{
					relation: "user_pmcs_community_releases",
					approvedIndexes: []string{
						"user_pmcs_community_releases_pkey",
					},
				},
				{
					relation: "user_pmcs_revisions",
					approvedIndexes: []string{
						"user_pmcs_revisions_checklist_id_id_key",
						"user_pmcs_revisions_pkey",
					},
				},
			},
		},
		{
			name:        "exact model browse",
			observation: captured["exact model browse"],
			expectations: []relationPlanExpectation{
				{
					relation: "user_pmcs_revision_models",
					approvedIndexes: []string{
						"user_pmcs_revision_models_lookup_idx",
					},
				},
			},
		},
		{
			name:        "subscription updates",
			observation: captured["subscription updates"],
			expectations: []relationPlanExpectation{
				{
					relation: "user_pmcs_subscriptions",
					approvedIndexes: []string{
						"user_pmcs_subscriptions_active_update_idx",
					},
				},
			},
		},
		{
			name:        "active pin lookup",
			observation: captured["active pin lookup"],
			args:        []any{fixture.checklistIDs[0]},
			expectations: []relationPlanExpectation{
				{
					relation: "user_pmcs_subscriptions",
					approvedIndexes: []string{
						"user_pmcs_subscriptions_source_idx",
					},
				},
			},
		},
		{
			name:        "account limit counts",
			observation: captured["account limit count"],
			args:        []any{planOwnerUID},
			expectations: []relationPlanExpectation{
				{
					relation: "user_pmcs_checklists",
					approvedIndexes: []string{
						"user_pmcs_checklists_owner_delta_idx",
					},
				},
			},
		},
	}
	for _, planCase := range plans {
		require.NotEmpty(
			t,
			strings.TrimSpace(planCase.observation.query),
			"missing captured production query for %s",
			planCase.name,
		)
		args := planCase.observation.args
		if planCase.args != nil {
			args = planCase.args
		}
		plan := explainAnalyzePlan(
			t,
			ctx,
			planCase.observation.query,
			args...,
		)
		t.Logf(
			"EXPLAIN (ANALYZE, BUFFERS) %s "+
				"[driver-captured production SQL]:\n%s",
			planCase.name,
			plan,
		)
		require.Contains(t, plan, "Buffers:")
		require.Contains(t, plan, "Execution Time:")
		for _, expectation := range planCase.expectations {
			requirePlanUsesApprovedRelationIndex(
				t,
				plan,
				expectation.relation,
				expectation.approvedIndexes,
			)
		}
	}
}

func explainAnalyzePlan(
	t *testing.T,
	ctx context.Context,
	query string,
	args ...any,
) string {
	t.Helper()
	rows, err := testDB.QueryContext(
		ctx,
		"EXPLAIN (ANALYZE, BUFFERS) "+query,
		args...,
	)
	require.NoError(t, err)
	defer rows.Close()
	var lines []string
	for rows.Next() {
		var line string
		require.NoError(t, rows.Scan(&line))
		lines = append(lines, line)
	}
	require.NoError(t, rows.Err())
	return strings.Join(lines, "\n")
}

func requirePlanUsesApprovedRelationIndex(
	t *testing.T,
	plan string,
	relation string,
	approvedIndexes []string,
) {
	t.Helper()
	require.NotContains(
		t,
		plan,
		"Seq Scan on "+relation,
		"unexpected sequential scan on %s:\n%s",
		relation,
		plan,
	)
	for _, index := range approvedIndexes {
		for _, line := range strings.Split(plan, "\n") {
			if strings.Contains(line, index) &&
				strings.Contains(line, relation) {
				return
			}
		}
	}
	require.Failf(
		t,
		"unexpected relation scan",
		"plan did not use approved index %v on %s:\n%s",
		approvedIndexes,
		relation,
		plan,
	)
}

type queryCounter struct {
	count    atomic.Int64
	duration atomic.Int64
	mu       sync.Mutex
	queries  []observedQuery
}

type observedQuery struct {
	query string
	args  []any
}

func (counter *queryCounter) record(
	query string,
	arguments []driver.NamedValue,
) {
	counter.count.Add(1)
	args := make([]any, len(arguments))
	for index, argument := range arguments {
		args[index] = argument.Value
	}
	counter.mu.Lock()
	counter.queries = append(
		counter.queries,
		observedQuery{query: query, args: args},
	)
	counter.mu.Unlock()
}

func (counter *queryCounter) addDuration(duration time.Duration) {
	counter.duration.Add(int64(duration))
}

func (counter *queryCounter) reset() {
	counter.count.Store(0)
	counter.duration.Store(0)
	counter.mu.Lock()
	counter.queries = nil
	counter.mu.Unlock()
}

func (counter *queryCounter) value() int {
	return int(counter.count.Load())
}

func (counter *queryCounter) databaseDuration() time.Duration {
	return time.Duration(counter.duration.Load())
}

func (counter *queryCounter) snapshot() []observedQuery {
	counter.mu.Lock()
	defer counter.mu.Unlock()
	queries := make([]observedQuery, len(counter.queries))
	copy(queries, counter.queries)
	return queries
}

type queryCountingConnector struct {
	base            driver.Connector
	counter         *queryCounter
	applicationName string
}

func (connector *queryCountingConnector) Connect(
	ctx context.Context,
) (driver.Conn, error) {
	connection, err := connector.base.Connect(ctx)
	if err != nil {
		return nil, err
	}
	if executor, ok := connection.(driver.ExecerContext); ok {
		_, err = executor.ExecContext(
			ctx,
			"SET application_name = '"+connector.applicationName+"'",
			nil,
		)
		if err != nil {
			_ = connection.Close()
			return nil, err
		}
	}
	return &queryCountingConnection{
		Conn:    connection,
		counter: connector.counter,
	}, nil
}

func (connector *queryCountingConnector) Driver() driver.Driver {
	return connector.base.Driver()
}

type queryCountingConnection struct {
	driver.Conn
	counter *queryCounter
}

func (connection *queryCountingConnection) ExecContext(
	ctx context.Context,
	query string,
	arguments []driver.NamedValue,
) (driver.Result, error) {
	executor, ok := connection.Conn.(driver.ExecerContext)
	if !ok {
		return nil, driver.ErrSkip
	}
	connection.counter.record(query, arguments)
	started := time.Now()
	result, err := executor.ExecContext(ctx, query, arguments)
	connection.counter.addDuration(time.Since(started))
	return result, err
}

func (connection *queryCountingConnection) QueryContext(
	ctx context.Context,
	query string,
	arguments []driver.NamedValue,
) (driver.Rows, error) {
	queryer, ok := connection.Conn.(driver.QueryerContext)
	if !ok {
		return nil, driver.ErrSkip
	}
	connection.counter.record(query, arguments)
	started := time.Now()
	rows, err := queryer.QueryContext(ctx, query, arguments)
	connection.counter.addDuration(time.Since(started))
	if err != nil {
		return nil, err
	}
	return &timedDriverRows{Rows: rows, counter: connection.counter}, nil
}

type timedDriverRows struct {
	driver.Rows
	counter *queryCounter
}

func (rows *timedDriverRows) Next(values []driver.Value) error {
	started := time.Now()
	err := rows.Rows.Next(values)
	rows.counter.addDuration(time.Since(started))
	return err
}

func (rows *timedDriverRows) Close() error {
	started := time.Now()
	err := rows.Rows.Close()
	rows.counter.addDuration(time.Since(started))
	return err
}

func (connection *queryCountingConnection) BeginTx(
	ctx context.Context,
	options driver.TxOptions,
) (driver.Tx, error) {
	beginner, ok := connection.Conn.(driver.ConnBeginTx)
	if !ok {
		return nil, driver.ErrSkip
	}
	return beginner.BeginTx(ctx, options)
}

func (connection *queryCountingConnection) Ping(
	ctx context.Context,
) error {
	pinger, ok := connection.Conn.(driver.Pinger)
	if !ok {
		return driver.ErrSkip
	}
	return pinger.Ping(ctx)
}

func (connection *queryCountingConnection) ResetSession(
	ctx context.Context,
) error {
	resetter, ok := connection.Conn.(driver.SessionResetter)
	if !ok {
		return nil
	}
	return resetter.ResetSession(ctx)
}

func (connection *queryCountingConnection) IsValid() bool {
	validator, ok := connection.Conn.(driver.Validator)
	if !ok {
		return true
	}
	return validator.IsValid()
}

func (connection *queryCountingConnection) CheckNamedValue(
	value *driver.NamedValue,
) error {
	checker, ok := connection.Conn.(driver.NamedValueChecker)
	if !ok {
		return driver.ErrSkip
	}
	return checker.CheckNamedValue(value)
}

func newQueryCountingDatabase(
	t *testing.T,
) (*sql.DB, *queryCounter, string) {
	t.Helper()
	dsn := disableSSLWhenUnspecified(os.Getenv("TEST_DATABASE_URL"))
	base, err := pq.NewConnector(dsn)
	require.NoError(t, err)
	counter := &queryCounter{}
	applicationName := "upmcs-performance-" + uuid.NewString()[:8]
	database := sql.OpenDB(&queryCountingConnector{
		base:            base,
		counter:         counter,
		applicationName: applicationName,
	})
	require.NoError(t, database.Ping())
	requireUserPmcsTestDatabase(t, database)
	counter.reset()
	t.Cleanup(func() {
		require.NoError(t, database.Close())
	})
	return database, counter, applicationName
}

func uuidStrings(values []uuid.UUID) []string {
	result := make([]string, len(values))
	for index, value := range values {
		result[index] = value.String()
	}
	return result
}
