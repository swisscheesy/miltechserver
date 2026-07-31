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
	scenario          string
	latencies         []time.Duration
	dbDuration        time.Duration
	lockWait          time.Duration
	encodeDuration    time.Duration
	peakAllocated     uint64
	uncompressedBytes int
	gzipBytes         int
	queryCount        int
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

func TestPerformanceScenarios(t *testing.T) {
	requireUserPmcsTestDatabase(t, testDB)
	config := shared.DefaultConfig()
	performanceDB, queryCounter := newQueryCountingDatabase(t)

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
		before := readMemoryStats()
		queryCounter.reset()
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
		after := readMemoryStats()

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
		logPerformanceEvidence(t, performanceEvidence{
			scenario:          "maximum_draft_replacement",
			latencies:         []time.Duration{draftDuration},
			dbDuration:        draftDuration,
			encodeDuration:    encodeDuration,
			peakAllocated:     after.TotalAlloc - before.TotalAlloc,
			uncompressedBytes: len(encodedDraft),
			gzipBytes:         compressedSize(t, encodedDraft),
			queryCount:        draftQueryCount,
		})

		number := int32(1)
		publicationInput := maximumDraft.Input
		publicationInput.RevisionNumber = &number
		publication, err := shared.PreparePublication(
			publicationInput,
			config,
		)
		require.NoError(t, err)
		before = readMemoryStats()
		queryCounter.reset()
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
		after = readMemoryStats()
		encodeStarted = time.Now()
		encodedPublication, err := json.Marshal(published.Aggregate)
		require.NoError(t, err)
		encodeDuration = time.Since(encodeStarted)
		require.Less(
			t,
			len(encodedPublication),
			config.MaxDeltaResponseBytes,
		)
		logPerformanceEvidence(t, performanceEvidence{
			scenario:          "maximum_publication",
			latencies:         []time.Duration{publishDuration},
			dbDuration:        publishDuration,
			encodeDuration:    encodeDuration,
			peakAllocated:     after.TotalAlloc - before.TotalAlloc,
			uncompressedBytes: len(encodedPublication),
			gzipBytes:         compressedSize(t, encodedPublication),
			queryCount:        publishQueryCount,
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
		logPerformanceEvidence(t, performanceEvidence{
			scenario:          "25_root_embedded_delta",
			latencies:         []time.Duration{dbDuration},
			dbDuration:        dbDuration,
			encodeDuration:    encodeDuration,
			uncompressedBytes: len(payload),
			gzipBytes:         compressedSize(t, payload),
			queryCount:        deltaQueryCount,
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
		start := make(chan struct{})
		durations := make(chan time.Duration, userCount)
		errorsChannel := make(chan error, userCount)
		repository := owned.NewRepository(
			persistence.NewStore(
				performanceDB,
				config.TransactionMaxAttempts,
			),
			config,
		)
		queryCounter.reset()
		var workers sync.WaitGroup
		for index := range users {
			index := index
			workers.Add(1)
			go func() {
				defer workers.Done()
				<-start
				started := time.Now()
				_, err := repository.Create(
					ctx,
					users[index],
					uuid.New(),
					drafts[index],
					shared.Precondition{
						Mode: shared.PreconditionCreate,
					},
				)
				durations <- time.Since(started)
				errorsChannel <- err
			}()
		}
		close(start)
		waitForWorkers(t, &workers, 25*time.Second)
		close(durations)
		close(errorsChannel)
		for err := range errorsChannel {
			require.NoError(t, err)
		}
		samples := make([]time.Duration, 0, userCount)
		for duration := range durations {
			samples = append(samples, duration)
		}
		require.Len(t, samples, userCount)
		concurrentQueryCount := queryCounter.value()
		require.Equal(t, userCount*30, concurrentQueryCount)
		logPerformanceEvidence(t, performanceEvidence{
			scenario:   "20_concurrent_independent_users",
			latencies:  samples,
			queryCount: concurrentQueryCount,
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
		payload, err := json.Marshal(
			[]any{unfiltered, filtered},
		)
		require.NoError(t, err)
		logPerformanceEvidence(t, performanceEvidence{
			scenario: "community_browse_filtered_unfiltered",
			latencies: []time.Duration{
				unfilteredDuration,
				filteredDuration,
			},
			dbDuration:        unfilteredDuration + filteredDuration,
			uncompressedBytes: len(payload),
			gzipBytes:         compressedSize(t, payload),
			queryCount:        browseQueryCount,
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
		payload, err := json.Marshal(page)
		require.NoError(t, err)
		require.NotContains(t, string(payload), `"sections"`)
		logPerformanceEvidence(t, performanceEvidence{
			scenario:          "500_subscription_update_discovery",
			latencies:         []time.Duration{duration},
			dbDuration:        duration,
			uncompressedBytes: len(payload),
			gzipBytes:         compressedSize(t, payload),
			queryCount:        updateQueryCount,
		})
	})

	t.Run("50 publication first sync interruption and resume", func(t *testing.T) {
		ctx, cancel := context.WithTimeout(
			context.Background(),
			2*time.Minute,
		)
		defer cancel()
		ownerUID := newUserPmcsTestUser(t)
		checklistID := uuid.New()
		repository := owned.NewRepository(
			persistence.NewStore(
				performanceDB,
				config.TransactionMaxAttempts,
			),
			config,
		)
		initial := preparedTree(t, uuid.New())
		created, err := repository.Create(
			ctx,
			ownerUID,
			checklistID,
			initial,
			shared.Precondition{Mode: shared.PreconditionCreate},
		)
		require.NoError(t, err)
		currentVersion := created.Aggregate.SyncVersion
		samples := make([]time.Duration, 0, 50)
		queryCounter.reset()
		for number := int32(1); number <= 50; number++ {
			input := preparedTree(t, uuid.New()).Input
			publication := preparePublication(t, input, number)
			if number == 26 {
				interruptedContext, interrupt := context.WithCancel(ctx)
				interrupt()
				_, interruptErr := repository.Publish(
					interruptedContext,
					ownerUID,
					checklistID,
					publication,
					checklistPrecondition(
						checklistID,
						currentVersion,
					),
				)
				require.ErrorIs(
					t,
					interruptErr,
					context.Canceled,
				)
			}
			started := time.Now()
			result, publishErr := repository.Publish(
				ctx,
				ownerUID,
				checklistID,
				publication,
				checklistPrecondition(
					checklistID,
					currentVersion,
				),
			)
			samples = append(samples, time.Since(started))
			require.NoError(t, publishErr)
			currentVersion = result.Aggregate.SyncVersion
		}
		var publications int
		require.NoError(
			t,
			testDB.QueryRowContext(
				ctx,
				`SELECT count(*)
				 FROM user_pmcs_revisions
				 WHERE checklist_id = $1
				   AND revision_number IS NOT NULL`,
				checklistID,
			).Scan(&publications),
		)
		require.Equal(t, 50, publications)
		require.Equal(t, int64(51), accountVersion(t, ownerUID))
		historyQueryCount := queryCounter.value()
		require.Positive(t, historyQueryCount)
		require.LessOrEqual(t, historyQueryCount, 2_500)
		logPerformanceEvidence(t, performanceEvidence{
			scenario:   "50_publication_first_sync_resume",
			latencies:  samples,
			queryCount: historyQueryCount,
		})
	})

	t.Run("approved index plans", func(t *testing.T) {
		captureUserPmcsQueryPlans(t, fixture)
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

func readMemoryStats() runtime.MemStats {
	var stats runtime.MemStats
	runtime.ReadMemStats(&stats)
	return stats
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
			"peak_allocated_bytes=%d uncompressed_bytes=%d gzip_bytes=%d "+
			"query_count=%d",
		evidence.scenario,
		p50,
		p95,
		evidence.dbDuration,
		evidence.lockWait,
		evidence.encodeDuration,
		evidence.peakAllocated,
		evidence.uncompressedBytes,
		evidence.gzipBytes,
		evidence.queryCount,
	)
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
) {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	missingOwner := "missing-plan-owner-" + uuid.NewString()
	plans := []struct {
		name            string
		query           string
		args            []any
		approvedIndexes []string
	}{
		{
			name: "owner delta branch",
			query: `SELECT id, account_change_version
			        FROM user_pmcs_checklists
			        WHERE owner_uid = $1
			          AND account_change_version > 0
			        ORDER BY account_change_version
			        LIMIT 25`,
			args: []any{missingOwner},
			approvedIndexes: []string{
				"user_pmcs_checklists_owner_delta_idx",
			},
		},
		{
			name: "subscription delta branch",
			query: `SELECT checklist_id, account_change_version
			        FROM user_pmcs_subscriptions
			        WHERE subscriber_uid = $1
			          AND account_change_version > 0
			        ORDER BY account_change_version
			        LIMIT 25`,
			args: []any{fixture.subscriberUID},
			approvedIndexes: []string{
				"user_pmcs_subscriptions_delta_idx",
			},
		},
		{
			name: "batched tree loader",
			query: `SELECT id, checklist_id
			        FROM user_pmcs_revisions
			        WHERE id = ANY($1)`,
			args: []any{pq.Array(uuidStrings(fixture.currentIDs[:25]))},
			approvedIndexes: []string{
				"user_pmcs_revisions_pkey",
			},
		},
		{
			name: "active recent browse",
			query: `SELECT checklist_id
			        FROM user_pmcs_community_sources
			        WHERE status = 'active'
			        ORDER BY updated_at DESC, checklist_id
			        LIMIT 50`,
			approvedIndexes: []string{
				"user_pmcs_community_sources_recent_idx",
			},
		},
		{
			name: "exact model browse",
			query: `SELECT revision_id
			        FROM user_pmcs_revision_models
			        WHERE normalized_text = $1`,
			args: []any{fixture.normalizedName},
			approvedIndexes: []string{
				"user_pmcs_revision_models_lookup_idx",
			},
		},
		{
			name: "subscription updates",
			query: `SELECT checklist_id, installed_revision_id
			        FROM user_pmcs_subscriptions
			        WHERE subscriber_uid = $1
			          AND deleted_at IS NULL
			        ORDER BY checklist_id
			        LIMIT 101`,
			args: []any{fixture.subscriberUID},
			approvedIndexes: []string{
				"user_pmcs_subscriptions_active_update_idx",
			},
		},
		{
			name: "active pin lookup",
			query: `SELECT subscriber_uid, checklist_id
			        FROM user_pmcs_subscriptions
			        WHERE installed_revision_id = $1
			          AND deleted_at IS NULL`,
			args: []any{fixture.installedIDs[0]},
			approvedIndexes: []string{
				"user_pmcs_subscriptions_active_pin_idx",
			},
		},
		{
			name: "account limit counts",
			query: `SELECT count(*)
			        FROM user_pmcs_checklists
			        WHERE owner_uid = $1
			          AND deleted_at IS NULL`,
			args: []any{missingOwner},
			approvedIndexes: []string{
				"user_pmcs_checklists_owner_delta_idx",
			},
		},
	}
	for _, planCase := range plans {
		plan := explainAnalyzePlan(
			t,
			ctx,
			planCase.query,
			planCase.args...,
		)
		t.Logf(
			"EXPLAIN (ANALYZE, BUFFERS) %s:\n%s",
			planCase.name,
			plan,
		)
		require.Contains(t, plan, "Buffers:")
		require.Contains(t, plan, "Execution Time:")
		requirePlanUsesApprovedIndex(
			t,
			plan,
			planCase.approvedIndexes,
		)
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

func requirePlanUsesApprovedIndex(
	t *testing.T,
	plan string,
	approvedIndexes []string,
) {
	t.Helper()
	for _, index := range approvedIndexes {
		if strings.Contains(plan, index) {
			return
		}
	}
	require.Failf(
		t,
		"unexpected sequential scan",
		"plan did not use an approved index %v:\n%s",
		approvedIndexes,
		plan,
	)
}

type queryCounter struct {
	count atomic.Int64
}

func (counter *queryCounter) increment() {
	counter.count.Add(1)
}

func (counter *queryCounter) reset() {
	counter.count.Store(0)
}

func (counter *queryCounter) value() int {
	return int(counter.count.Load())
}

type queryCountingConnector struct {
	base    driver.Connector
	counter *queryCounter
}

func (connector *queryCountingConnector) Connect(
	ctx context.Context,
) (driver.Conn, error) {
	connection, err := connector.base.Connect(ctx)
	if err != nil {
		return nil, err
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
	connection.counter.increment()
	return executor.ExecContext(ctx, query, arguments)
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
	connection.counter.increment()
	return queryer.QueryContext(ctx, query, arguments)
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
) (*sql.DB, *queryCounter) {
	t.Helper()
	dsn := disableSSLWhenUnspecified(os.Getenv("TEST_DATABASE_URL"))
	base, err := pq.NewConnector(dsn)
	require.NoError(t, err)
	counter := &queryCounter{}
	database := sql.OpenDB(&queryCountingConnector{
		base:    base,
		counter: counter,
	})
	require.NoError(t, database.Ping())
	requireUserPmcsTestDatabase(t, database)
	counter.reset()
	t.Cleanup(func() {
		require.NoError(t, database.Close())
	})
	return database, counter
}

func uuidStrings(values []uuid.UUID) []string {
	result := make([]string, len(values))
	for index, value := range values {
		result[index] = value.String()
	}
	return result
}
