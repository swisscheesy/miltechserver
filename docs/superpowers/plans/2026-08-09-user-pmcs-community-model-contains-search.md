# User PMCS Community Model Contains Search Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Make `GET /api/v1/user-pmcs/community?model={text}` return active current community releases whose revision-level model contains the user's case-agnostic literal search text.

**Architecture:** Retain the existing handler, service normalization, current-release `EXISTS`, keyset cursor, and response contract. Convert the normalized model term into an escaped `%term%` `LIKE` parameter inside the repository, prove semantics with unit and real-PostgreSQL tests, and add `pg_trgm` only if the captured representative production plan cannot satisfy the existing approved-index gate.

**Tech Stack:** Go, Gin, `database/sql`, PostgreSQL, `testify/require`, existing User PMCS integration/performance harness, optional PostgreSQL `pg_trgm` with GIN.

## Global Constraints

- The approved design is `docs/superpowers/specs/2026-08-09-user-pmcs-community-model-contains-search-design.md`; do not redesign or weaken it during execution.
- Before edits, read `AGENTS.md` completely, verify branch/HEAD/status, and use `superpowers:using-git-worktrees` to create an isolated implementation worktree. Keep the controller checkout edit-free.
- Before the first implementation edit, set `user_pmcs_contains_base` to the exact starting `HEAD` and record that SHA in the progress ledger for final range verification.
- Use `rtk` for shell commands.
- Execute tasks strictly in order. In subagent-driven mode, use one fresh implementer per task followed by independent spec-compliance and code-quality review; do not run implementation agents in parallel.
- Use TDD: add the focused failing test, observe the expected failure, make the minimal implementation, then rerun focused verification before committing.
- Preserve literal substring matching anywhere in revision-level models. Do not add prefix-only, token, fuzzy, ranking, name/description, creator, or section-model search.
- Preserve `shared.NormalizeModel`, the route/query names, existing typed errors, one- and two-character searches, response shape, current-release visibility, keyset cursor format/order, gzip/cache headers, rate limits, and authored-text log redaction.
- Treat `%`, `_`, and `!` as literal search text. Keep every SQL value parameterized; never concatenate user text into SQL.
- The client must restart pagination without `after` when `model` changes; do not change the cursor payload in this work.
- Do not add a database index unless the captured 500-source production query performs a sequential scan of `user_pmcs_revision_models` or fails the approved relation-index assertion.
- If migration `012` is required, verify the exact database immediately before every write: forward/rollback/forward only on `miltech_ng_test`, forward-only on `miltech_ng`, and never touch production or roll back `miltech_ng`.
- Do not hand-edit Jet-generated artifacts. This change does not alter table shape, so a trigram index does not require Jet regeneration.
- Serialize database-backed acceptance with `go test -p 1 ./... -count=1`. Use Go 1.23.4 and an isolated `GOCACHE` for race verification if the local Go 1.23.3 race-runtime defect appears.
- Report every command, exit status, test count when available, accepted baseline failure, migration result, final HEAD, and worktree state exactly. Do not claim deployment, push, production migration, or device verification.

## File Structure

- Create `api/user_pmcs/community/repository_impl_test.go`: focused unit contract for escaped contains patterns and query arguments.
- Modify `api/user_pmcs/community/repository_impl.go`: repository-local pattern helper and parameterized current-release `LIKE` predicate.
- Modify `tests/user_pmcs/community_test.go`: real-PostgreSQL contains, case, wildcard, visibility, duplicate, and pagination coverage.
- Modify `tests/user_pmcs/performance_test.go`: representative partial term, contains-query capture, query-count assertion, and approved plan indexes.
- Conditionally create `migrations/012_add_user_pmcs_model_search_trigram_index.sql`: enable `pg_trgm` if needed and create the feature-owned GIN index concurrently.
- Conditionally create `migrations/012_rollback_user_pmcs_model_search_trigram_index.sql`: remove only the feature-owned index concurrently.
- Conditionally modify `tests/user_pmcs/migration_schema_test.go`: assert the exact trigram index definition when migration `012` is required.
- Modify `docs/client/2026-07-29-user-pmcs-server-api-contract.md`: server wire-contract wording.
- Modify `docs/client/2026-07-31-user-pmcs-mobile-api-implementation-guide.md`: mobile search and pagination guidance.

---

### Task 1: Implement the Escaped Literal Contains Predicate

**Files:**
- Create: `api/user_pmcs/community/repository_impl_test.go`
- Modify: `api/user_pmcs/community/repository_impl.go:3-15,345-405`

**Interfaces:**
- Consumes: `shared.CommunityBrowseFilter{After *shared.CommunityCursor, Limit int, NormalizedModel string}` and the existing `communityBrowseQuery(filter) (string, []any)`.
- Produces: unexported `containsModelPattern(normalizedModel string) string`; `communityBrowseQuery` binds its returned pattern and emits `model.normalized_text LIKE $n ESCAPE '!'`.

- [ ] **Step 1: Create the focused pattern/query tests**

Create `api/user_pmcs/community/repository_impl_test.go`:

```go
package community

import (
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"

	"miltechserver/api/user_pmcs/shared"
)

func TestContainsModelPatternEscapesLikeMetacharacters(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  string
	}{
		{name: "ordinary text", input: "m1165a1", want: "%m1165a1%"},
		{name: "percent", input: "m%1165", want: "%m!%1165%"},
		{name: "underscore", input: "m_1165", want: "%m!_1165%"},
		{name: "escape character", input: "m!1165", want: "%m!!1165%"},
		{
			name:  "combined",
			input: "m!_1165%a1",
			want:  "%m!!!_1165!%a1%",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			require.Equal(t, test.want, containsModelPattern(test.input))
		})
	}
}

func TestCommunityBrowseQueryUsesParameterizedContainsPattern(t *testing.T) {
	query, arguments := communityBrowseQuery(shared.CommunityBrowseFilter{
		Limit:           20,
		NormalizedModel: "m1165_1",
	})

	require.Contains(
		t,
		query,
		"model.normalized_text LIKE $1 ESCAPE '!'",
	)
	require.NotContains(t, query, "model.normalized_text = $1")
	require.Equal(t, []any{"%m1165!_1%", 21}, arguments)
}

func TestCommunityBrowseQueryKeepsCursorArgumentPositions(t *testing.T) {
	query, arguments := communityBrowseQuery(shared.CommunityBrowseFilter{
		After: &shared.CommunityCursor{
			Version:   1,
			UpdatedAt: time.Date(2026, time.August, 9, 12, 0, 0, 0, time.UTC),
			Checklist: uuid.MustParse("10000000-0000-4000-8000-000000000001"),
		},
		Limit:           7,
		NormalizedModel: "m1165",
	})

	require.Contains(
		t,
		query,
		"model.normalized_text LIKE $3 ESCAPE '!'",
	)
	require.Contains(t, query, "LIMIT $4")
	require.Equal(t, "%m1165%", arguments[2])
	require.Equal(t, 8, arguments[3])
}
```

- [ ] **Step 2: Run the focused tests and confirm the red state**

Run:

```bash
rtk go test ./api/user_pmcs/community \
  -run 'Test(ContainsModelPattern|CommunityBrowseQuery)' -count=1
```

Expected: compilation fails because `containsModelPattern` does not exist, and the query still uses equality.

- [ ] **Step 3: Add the minimal helper and change the predicate**

Add `strings` to `repository_impl.go` imports, then add this helper immediately before `communityBrowseQuery`:

```go
func containsModelPattern(normalizedModel string) string {
	escaped := strings.NewReplacer(
		"!", "!!",
		"%", "!%",
		"_", "!_",
	).Replace(normalizedModel)
	return "%" + escaped + "%"
}
```

Replace only the model-filter block's argument and predicate:

```go
if filter.NormalizedModel != "" {
	arguments = append(
		arguments,
		containsModelPattern(filter.NormalizedModel),
	)
	query += fmt.Sprintf(
		` AND EXISTS (
		      SELECT 1
		      FROM user_pmcs_revision_models AS model
		      WHERE model.normalized_text LIKE $%d ESCAPE '!'
		        AND model.revision_id =
		            source.current_release_revision_id
		  )`,
		len(arguments),
	)
}
```

Do not alter the handler, service, filter type, visibility predicates, cursor predicate, ordering, or `LIMIT + 1` behavior.

- [ ] **Step 4: Format and run focused/package verification**

Format the Go files with:

```bash
rtk gofmt -w api/user_pmcs/community/repository_impl.go \
  api/user_pmcs/community/repository_impl_test.go
```

Then run:

```bash
rtk go test ./api/user_pmcs/community \
  -run 'Test(ContainsModelPattern|CommunityBrowseQuery)' -count=1
rtk go test ./api/user_pmcs/community -count=1
```

Expected: both commands pass; the package remains at or above its current 27-test baseline plus the new tests.

- [ ] **Step 5: Review and commit Task 1**

Run `git diff --check`, then complete independent spec-compliance and code-quality review. Resolve all Critical and Important findings before committing:

```bash
rtk git add api/user_pmcs/community/repository_impl.go \
  api/user_pmcs/community/repository_impl_test.go
rtk git commit -m "feat(user-pmcs): add literal contains model query"
```

---

### Task 2: Prove Contains Semantics Against PostgreSQL

**Files:**
- Modify: `tests/user_pmcs/community_test.go:348-520,829-930`

**Interfaces:**
- Consumes: Task 1's `community.Repository.Browse` behavior and unchanged `community.NewService(repository, config).Browse(ctx, after, limit, model)` normalization boundary.
- Produces: integration proof for partial/exact/mixed-case/literal wildcard matching, current-only visibility, retired/deleted exclusion, duplicate prevention, and filtered keyset pagination.

- [ ] **Step 1: Verify the integration-test database before fixture writes**

Load `.env` into the shell without printing secrets, then verify the target:

```bash
set -a
source .env
set +a
rtk psql "$TEST_DATABASE_URL" -X -Atqc 'SELECT current_database()'
```

Expected output: exactly `miltech_ng_test`. Stop if it differs.

- [ ] **Step 2: Add reusable test helpers**

Add these helpers near `summaryRevision` in `tests/user_pmcs/community_test.go`:

```go
func setRevisionModel(
	t *testing.T,
	revisionID uuid.UUID,
	displayText string,
	normalizedText string,
) {
	t.Helper()
	_, err := testDB.ExecContext(
		context.Background(),
		`UPDATE user_pmcs_revision_models
		 SET display_text = $1, normalized_text = $2
		 WHERE revision_id = $3`,
		displayText,
		normalizedText,
		revisionID,
	)
	require.NoError(t, err)
}

func communityChecklistIDs(items []shared.PublicCommunitySummary) []uuid.UUID {
	checklistIDs := make([]uuid.UUID, len(items))
	for index, item := range items {
		checklistIDs[index] = item.ChecklistID
	}
	return checklistIDs
}
```

- [ ] **Step 3: Add the failing semantic integration test**

Add a new test after `TestCommunityBrowseStaticKeysetCurrentOnlyAndModelFilter`:

```go
func TestCommunityBrowseModelFilterUsesLiteralCaseAgnosticContains(t *testing.T) {
	models := []struct {
		displayText    string
		normalizedText string
	}{
		{displayText: "M1165", normalizedText: "m1165"},
		{displayText: "M1165A1", normalizedText: "m1165a1"},
		{displayText: "M1165A2", normalizedText: "m1165a2"},
		{displayText: "XM1165A1-R2", normalizedText: "xm1165a1-r2"},
		{displayText: "M_1165%!", normalizedText: "m_1165%!"},
		{displayText: "M1200", normalizedText: "m1200"},
	}

	fixtures := make([]*releasedChecklistFixture, len(models))
	for index, model := range models {
		fixture := newReleasedChecklistFixture(t, 1)
		_, err := fixture.repository.Release(
			context.Background(),
			fixture.ownerUID,
			fixture.checklist,
			fixture.revisions[0].Input.ID,
			checklistPrecondition(
				fixture.checklist,
				fixture.aggregate.SyncVersion,
			),
		)
		require.NoError(t, err)
		setRevisionModel(
			t,
			fixture.revisions[0].Input.ID,
			model.displayText,
			model.normalizedText,
		)
		fixtures[index] = fixture
	}

	_, err := testDB.ExecContext(
		context.Background(),
		`INSERT INTO user_pmcs_revision_models
		     (revision_id, display_text, normalized_text)
		 VALUES ($1, $2, $3)`,
		fixtures[1].revisions[0].Input.ID,
		"Alternate M1165A1",
		"alternate m1165a1",
	)
	require.NoError(t, err)

	service := community.NewService(
		fixtures[0].repository,
		shared.DefaultConfig(),
	)
	tests := []struct {
		name   string
		search string
		want   []uuid.UUID
	}{
		{
			name:   "partial family",
			search: "m1165",
			want: []uuid.UUID{
				fixtures[0].checklist,
				fixtures[1].checklist,
				fixtures[2].checklist,
				fixtures[3].checklist,
			},
		},
		{
			name:   "mixed case exact text remains contains",
			search: "M1165A1",
			want: []uuid.UUID{
				fixtures[1].checklist,
				fixtures[3].checklist,
			},
		},
		{
			name:   "middle substring",
			search: "1165a",
			want: []uuid.UUID{
				fixtures[1].checklist,
				fixtures[2].checklist,
				fixtures[3].checklist,
			},
		},
		{
			name:   "normalized surrounding whitespace",
			search: "  M1165A1\u00a0 ",
			want: []uuid.UUID{
				fixtures[1].checklist,
				fixtures[3].checklist,
			},
		},
		{name: "suffix", search: "r2", want: []uuid.UUID{fixtures[3].checklist}},
		{name: "literal underscore", search: "_", want: []uuid.UUID{fixtures[4].checklist}},
		{name: "literal percent", search: "%", want: []uuid.UUID{fixtures[4].checklist}},
		{name: "literal escape", search: "!", want: []uuid.UUID{fixtures[4].checklist}},
		{name: "no match", search: "m1165a9", want: []uuid.UUID{}},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			page, browseErr := service.Browse(
				context.Background(),
				"",
				"50",
				test.search,
			)
			require.NoError(t, browseErr)
			require.ElementsMatch(
				t,
				test.want,
				communityChecklistIDs(page.Items),
			)
		})
	}
}
```

Run only this test against verified `miltech_ng_test`:

```bash
rtk go test ./tests/user_pmcs \
  -run '^TestCommunityBrowseModelFilterUsesLiteralCaseAgnosticContains$' \
  -count=1
```

Expected: PASS with Task 1 applied. Task 1's focused tests already established the red state for the behavior change; this test supplies the real-PostgreSQL acceptance boundary.

- [ ] **Step 4: Convert the existing keyset/current-only test to a partial term**

In `TestCommunityBrowseStaticKeysetCurrentOnlyAndModelFilter`, separate the stored and searched values:

```go
storedModel := "task10-m1165a1-" + uuid.NewString()
modelSearch := "task10-m1165"
historicalModel := "task10-historical-m1165a1-" + uuid.NewString()
historicalSearch := "task10-historical-m1165"
```

Use `storedModel` when updating current revision rows, `modelSearch` in every current filtered browse, and `historicalSearch` for the historical-only browse. Retain all existing assertions for:

- two-page `LIMIT + 1` behavior;
- no duplicate checklist IDs;
- current revision selection;
- retired-source exclusion;
- deleted-owner display name; and
- deterministic timestamp/UUID ordering.

Add one second matching model to `fixtures[0]` and assert that checklist still appears exactly once:

```go
_, err = testDB.ExecContext(
	context.Background(),
	`INSERT INTO user_pmcs_revision_models
	     (revision_id, display_text, normalized_text)
	 VALUES ($1, $2, $3)`,
	fixtures[0].revisions[0].Input.ID,
	"Task 10 alternate M1165",
	modelSearch+"-alternate",
)
require.NoError(t, err)
```

After the retired assertion, add this deleted-source check:

```go
deletedFixture := newReleasedChecklistFixture(t, 1)
deletedRelease, err := deletedFixture.repository.Release(
	context.Background(),
	deletedFixture.ownerUID,
	deletedFixture.checklist,
	deletedFixture.revisions[0].Input.ID,
	checklistPrecondition(
		deletedFixture.checklist,
		deletedFixture.aggregate.SyncVersion,
	),
)
require.NoError(t, err)
setRevisionModel(
	t,
	deletedFixture.revisions[0].Input.ID,
	storedModel,
	storedModel,
)
_, err = deletedFixture.owned.DeleteChecklist(
	context.Background(),
	deletedFixture.ownerUID,
	deletedFixture.checklist,
	checklistPrecondition(
		deletedFixture.checklist,
		deletedRelease.Aggregate.SyncVersion,
	),
)
require.NoError(t, err)

afterDelete, err := repository.Browse(
	context.Background(),
	shared.CommunityBrowseFilter{
		Limit:           50,
		NormalizedModel: modelSearch,
	},
)
require.NoError(t, err)
require.NotContains(
	t,
	communityChecklistIDs(afterDelete.Items),
	deletedFixture.checklist,
)
```

- [ ] **Step 5: Run focused and package integration verification**

Run serially against the verified test database:

```bash
rtk go test ./tests/user_pmcs \
  -run 'TestCommunityBrowse(StaticKeysetCurrentOnlyAndModelFilter|ModelFilterUsesLiteralCaseAgnosticContains|MovingReleaseAppearsAfterRestart)' \
  -count=1
rtk go test ./api/user_pmcs/community ./tests/user_pmcs \
  -run 'TestCommunityBrowse' -count=1
```

Expected: all selected tests pass, cleanup succeeds, and the test database remains `miltech_ng_test`.

- [ ] **Step 6: Review and commit Task 2**

Run `rtk gofmt -w tests/user_pmcs/community_test.go` and `rtk git diff --check`. Complete independent spec-compliance and code-quality review, resolve all Critical and Important findings, then commit:

```bash
rtk git add tests/user_pmcs/community_test.go
rtk git commit -m "test(user-pmcs): cover community model contains search"
```

---

### Task 3: Preserve the Representative Query-Plan Gate

**Files:**
- Modify: `tests/user_pmcs/performance_test.go:54-63,990-1065,1800-1900,2300-2450`
- Conditional create: `migrations/012_add_user_pmcs_model_search_trigram_index.sql`
- Conditional create: `migrations/012_rollback_user_pmcs_model_search_trigram_index.sql`
- Conditional modify: `tests/user_pmcs/migration_schema_test.go`

**Interfaces:**
- Consumes: Task 1's parameterized `LIKE ... ESCAPE '!'` production SQL and Task 2's verified semantics.
- Produces: captured `contains model browse` performance evidence; accepted index list `user_pmcs_revision_models_pkey`, `user_pmcs_revision_models_lookup_idx`, or conditional `user_pmcs_revision_models_search_trgm_idx`; optional migration `012` when and only when the existing plan gate fails.

- [ ] **Step 1: Change the performance fixture to use a proper substring**

Extend `performanceSubscriptionFixture`:

```go
type performanceSubscriptionFixture struct {
	ownerUID           string
	subscriberUID      string
	checklistIDs       []uuid.UUID
	installedIDs       []uuid.UUID
	currentIDs         []uuid.UUID
	noiseUserUIDs      []string
	normalizedName     string
	containsModelSearch string
	cleanup            performanceSubscriptionFixtureCleanup
}
```

Format alignment with `gofmt`. Initialize both values in
`seedPerformanceSubscriptionsWithNoiseAndAnalyze`:

```go
normalizedName:      "m1165a1-performance-model",
containsModelSearch: "m1165a1",
```

Continue storing `fixture.normalizedName` in the one selective model row, but replace only browse-filter arguments at the current performance scenario and captured-plan setup with:

```go
NormalizedModel: fixture.containsModelSearch,
```

The filtered browse must still return exactly one item.

- [ ] **Step 2: Rename captured evidence and define approved indexes**

Replace every captured-query map key and plan name `exact model browse` with `contains model browse`.

Use this expectation:

```go
{
	name:        "contains model browse",
	observation: captured["contains model browse"],
	expectations: []relationPlanExpectation{
		{
			relation: "user_pmcs_revision_models",
			approvedIndexes: []string{
				"user_pmcs_revision_models_pkey",
				"user_pmcs_revision_models_lookup_idx",
				"user_pmcs_revision_models_search_trgm_idx",
			},
		},
	},
},
```

The helper must continue rejecting `Seq Scan on user_pmcs_revision_models`. Do not weaken `requirePlanUsesApprovedRelationIndex`.

- [ ] **Step 3: Run the representative performance test before adding a migration**

Re-verify `TEST_DATABASE_URL` resolves to `miltech_ng_test`, then run:

```bash
rtk gofmt -w tests/user_pmcs/performance_test.go
rtk go test ./tests/user_pmcs -run '^TestPerformanceScenarios$' -count=1 -v
```

Record:

- exit status;
- filtered and unfiltered latency evidence;
- query count, which must remain `4`;
- the complete `EXPLAIN (ANALYZE, BUFFERS)` block for `contains model browse`;
- whether `user_pmcs_revision_models` uses an approved index; and
- whether a sequential scan appears.

Decision:

- If the test passes and an existing approved index is used, do not create migration `012`; continue to Step 7.
- If the test fails because `user_pmcs_revision_models` is sequentially scanned or no approved index is used, keep the gate intact and execute Steps 4-6.
- If it fails for any other reason, diagnose that failure without creating an index and do not advance until it is resolved.

- [ ] **Step 4: If and only if the index gate fails, create migration `012`**

Create `migrations/012_add_user_pmcs_model_search_trigram_index.sql` exactly as follows:

```sql
-- User PMCS Community Model Contains Search
-- Migration: 012_add_user_pmcs_model_search_trigram_index.sql
--
-- Adds the operator class required to index literal leading-wildcard LIKE
-- searches on normalized revision-level model text.

CREATE EXTENSION IF NOT EXISTS pg_trgm;

CREATE INDEX CONCURRENTLY IF NOT EXISTS
    user_pmcs_revision_models_search_trgm_idx
    ON user_pmcs_revision_models
    USING GIN (normalized_text gin_trgm_ops);
```

Create `migrations/012_rollback_user_pmcs_model_search_trigram_index.sql`:

```sql
-- Rollback: 012_rollback_user_pmcs_model_search_trigram_index.sql
--
-- Removes only the feature-owned index. pg_trgm may be shared, so the
-- extension is intentionally retained.

DROP INDEX CONCURRENTLY IF EXISTS
    user_pmcs_revision_models_search_trgm_idx;
```

Do not wrap either script in `BEGIN`/`COMMIT`; PostgreSQL does not permit concurrent index creation/drop inside a transaction block.

- [ ] **Step 5: If migration `012` exists, add a schema assertion**

Add this test to `tests/user_pmcs/migration_schema_test.go`:

```go
func TestUserPmcsModelContainsSearchIndex(t *testing.T) {
	requireUserPmcsTestDatabase(t, testDB)

	var indexDefinition string
	err := testDB.QueryRow(`
		SELECT pg_get_indexdef(index_class.oid)
		FROM pg_class AS index_class
		JOIN pg_namespace AS index_schema
			ON index_schema.oid = index_class.relnamespace
		WHERE index_schema.nspname = 'public'
			AND index_class.relname =
				'user_pmcs_revision_models_search_trgm_idx'`,
	).Scan(&indexDefinition)
	require.NoError(t, err)
	require.Contains(t, indexDefinition, "USING gin")
	require.Contains(
		t,
		indexDefinition,
		"normalized_text gin_trgm_ops",
	)
}
```

Before applying migration `012`, run the new schema test once:

```bash
rtk go test ./tests/user_pmcs \
  -run '^TestUserPmcsModelContainsSearchIndex$' -count=1
```

Expected: FAIL because
`user_pmcs_revision_models_search_trgm_idx` does not exist yet.

- [ ] **Step 6: If migration `012` exists, rehearse it on non-production databases**

Load `.env` without printing secrets. Before each mutation, query and inspect `current_database()`.

On `miltech_ng_test` only:

```bash
rtk psql "$TEST_DATABASE_URL" -X -Atqc 'SELECT current_database()'
rtk psql "$TEST_DATABASE_URL" -X -Atqc \
  "SELECT EXISTS (SELECT 1 FROM pg_available_extensions WHERE name = 'pg_trgm')"
rtk psql "$TEST_DATABASE_URL" -X -v ON_ERROR_STOP=1 \
  -f migrations/012_add_user_pmcs_model_search_trigram_index.sql
rtk go test ./tests/user_pmcs -run '^TestUserPmcsModelContainsSearchIndex$' \
  -count=1
rtk psql "$TEST_DATABASE_URL" -X -v ON_ERROR_STOP=1 \
  -f migrations/012_rollback_user_pmcs_model_search_trigram_index.sql
rtk psql "$TEST_DATABASE_URL" -X -Atqc \
  "SELECT to_regclass('public.user_pmcs_revision_models_search_trgm_idx') IS NULL"
rtk psql "$TEST_DATABASE_URL" -X -v ON_ERROR_STOP=1 \
  -f migrations/012_add_user_pmcs_model_search_trigram_index.sql
rtk go test ./tests/user_pmcs -run '^TestUserPmcsModelContainsSearchIndex$' \
  -count=1
```

Expected database name: `miltech_ng_test`; expected extension availability and
rollback catalog results: `t`. Stop before the forward migration if the
extension-availability query is not `t`.

For development, construct the connection from the loaded `DB_*` variables without printing the password, verify the target is exactly `miltech_ng`, then apply forward only:

```bash
PGPASSWORD="$DB_PASSWORD" rtk psql -h "$DB_HOST" -p "$DB_PORT" \
  -U "$DB_USERNAME" -d "$DB_NAME" -X -Atqc 'SELECT current_database()'
PGPASSWORD="$DB_PASSWORD" rtk psql -h "$DB_HOST" -p "$DB_PORT" \
  -U "$DB_USERNAME" -d "$DB_NAME" -X -v ON_ERROR_STOP=1 \
  -f migrations/012_add_user_pmcs_model_search_trigram_index.sql
```

Expected database name before the write: exactly `miltech_ng`. Do not run the rollback on development. Do not connect to or modify production.

Rerun the representative performance test and require the trigram index or another approved index with no sequential scan:

```bash
rtk go test ./tests/user_pmcs -run '^TestPerformanceScenarios$' -count=1 -v
```

- [ ] **Step 7: Run focused performance/schema verification**

If no migration was needed:

```bash
rtk go test ./tests/user_pmcs \
  -run 'Test(PerformanceScenarios|CommunityBrowse)' -count=1 -v
```

If migration `012` was needed:

```bash
rtk gofmt -w tests/user_pmcs/migration_schema_test.go
rtk go test ./tests/user_pmcs \
  -run 'Test(UserPmcsModelContainsSearchIndex|PerformanceScenarios|CommunityBrowse)' \
  -count=1 -v
```

Expected: the query count remains `4`, contains browse returns one selective row in the performance fixture, no sequential scan is accepted, and all selected tests pass.

- [ ] **Step 8: Review and commit Task 3**

Run `rtk git diff --check`. Complete independent spec-compliance and database/code-quality review. The reviewer must specifically verify wildcard escaping, the captured SQL arguments, no weakened plan assertion, concurrent-index transaction rules, rollback ownership, and database targeting.

If no migration was required:

```bash
rtk git add tests/user_pmcs/performance_test.go
rtk git commit -m "perf(user-pmcs): validate contains search query plan"
```

If migration `012` was required:

```bash
rtk git add tests/user_pmcs/performance_test.go \
  tests/user_pmcs/migration_schema_test.go \
  migrations/012_add_user_pmcs_model_search_trigram_index.sql \
  migrations/012_rollback_user_pmcs_model_search_trigram_index.sql
rtk git commit -m "perf(user-pmcs): index community model contains search"
```

---

### Task 4: Update Active Contracts and Complete Acceptance

**Files:**
- Modify: `docs/client/2026-07-29-user-pmcs-server-api-contract.md:603-612`
- Modify: `docs/client/2026-07-31-user-pmcs-mobile-api-implementation-guide.md:1163-1190`

**Interfaces:**
- Consumes: Tasks 1-3's final behavior and measured index outcome.
- Produces: active server/mobile documentation aligned with literal case-agnostic contains matching and cursor-reset requirements; complete branch verification evidence.

- [ ] **Step 1: Update the server API contract**

Replace the browse `model` bullet in `docs/client/2026-07-29-user-pmcs-server-api-contract.md` with:

```markdown
- Query: optional opaque `after`; optional `limit` `1..50`, default `20`;
  optional `model`, normalized by the server and matched as a literal,
  case-agnostic substring against revision-level models on each active current
  release. `%`, `_`, and `!` are literal search characters. Start again
  without `after` whenever `model` changes.
```

- [ ] **Step 2: Update the mobile implementation guide**

Change the endpoint description from “optional exact model filtering” to “optional literal, case-agnostic model substring filtering.” Replace the `model` request bullet with:

```markdown
- Query `model`: optional revision-level model text. The server normalizes the
  value and returns active current releases containing that literal normalized
  substring. Matching is case agnostic; `%`, `_`, and `!` have no wildcard
  behavior. Discard `after` and restart from page one whenever the search text
  changes.
```

Add examples showing both of these requests can return a current release whose model is `M1165A1`:

```http
GET /api/v1/user-pmcs/community?limit=20&model=m1165
GET /api/v1/user-pmcs/community?limit=20&model=M1165A1
```

Do not edit historical plans, completed progress ledgers, or the approved design except to correct a discovered contradiction through an explicit review loop.

- [ ] **Step 3: Run formatting and focused acceptance**

Run:

```bash
rtk gofmt -w api/user_pmcs/community/repository_impl.go \
  api/user_pmcs/community/repository_impl_test.go \
  tests/user_pmcs/community_test.go \
  tests/user_pmcs/performance_test.go
rtk go test ./api/user_pmcs/community -count=1
rtk go test ./tests/user_pmcs \
  -run 'Test(CommunityBrowse|PerformanceScenarios)' -count=1 -v
```

If migration `012` exists, include `tests/user_pmcs/migration_schema_test.go` in `gofmt` and add `UserPmcsModelContainsSearchIndex` to the integration-test pattern.

- [ ] **Step 4: Run race verification with the validated toolchain**

Create an isolated cache without deleting any existing cache:

```bash
rtk mkdir -p /private/tmp/miltechserver-user-pmcs-contains-race-gocache
GOTOOLCHAIN=go1.23.4 \
GOCACHE=/private/tmp/miltechserver-user-pmcs-contains-race-gocache \
  rtk go test -race ./api/user_pmcs/... -count=1
```

Report the exact result. If Go 1.23.4 cannot be obtained because of environment/network restrictions, report that as unverified rather than substituting a known-broken Go 1.23.3 race result.

- [ ] **Step 5: Run serialized repository-wide acceptance**

Re-verify `TEST_DATABASE_URL` is `miltech_ng_test`. Ensure no other database-backed Go suite is running, then use an isolated cache:

```bash
rtk mkdir -p /private/tmp/miltechserver-user-pmcs-contains-full-gocache
GOTOOLCHAIN=go1.23.4 \
GOCACHE=/private/tmp/miltechserver-user-pmcs-contains-full-gocache \
  rtk go test -p 1 ./... -count=1
```

Expected: exit `0`. If the baseline has an unrelated failure, capture its exact package/test/error, compare it with a clean baseline using the same command, and do not claim full green.

- [ ] **Step 6: Check documentation, diff, and scope**

Run:

```bash
rtk rg -n -i 'exact normalized|exact model|model.*matched exactly|optional exact model' \
  docs/client/2026-07-29-user-pmcs-server-api-contract.md \
  docs/client/2026-07-31-user-pmcs-mobile-api-implementation-guide.md
rtk git diff --check
rtk git diff --name-status
```

Expected: the active contract files contain no stale exact-search wording; the diff contains only planned files. Historical files are intentionally excluded from this stale-wording scan.

- [ ] **Step 7: Review and commit Task 4**

Complete independent spec-compliance and code-quality review. Resolve all Critical and Important findings, then commit only the active contract files:

```bash
rtk git add docs/client/2026-07-29-user-pmcs-server-api-contract.md \
  docs/client/2026-07-31-user-pmcs-mobile-api-implementation-guide.md
rtk git commit -m "docs(user-pmcs): document contains model search"
```

## Final Whole-Branch Review

After Task 4, review the complete implementation range from the implementation worktree's base commit through `HEAD`.

- Verify every acceptance criterion in the approved design maps to a passing test or explicit verification result.
- Review all production SQL and arguments for injection safety and literal wildcard behavior.
- Verify current-only visibility, keyset ordering, `LIMIT + 1`, response shape, and log redaction were not changed.
- Verify the trigram migration is absent when the existing approved plan passed, or fully rehearsed on test and applied forward-only to development when it was required.
- Resolve the base from the untouched controller branch, then verify the full range:

  ```bash
  user_pmcs_contains_base=$(rtk git merge-base HEAD user_generated_pmcs)
  rtk git diff --check "$user_pmcs_contains_base"..HEAD
  rtk git status --short --branch
  rtk git rev-parse HEAD
  ```
- If review finds a defect, return it to the responsible task's implementer, rerun that task's focused verification and review gates, make a narrow conventional commit, and repeat the whole-branch review.
- Do not push, merge, deploy, or touch production without separate explicit authorization.

## Completion Evidence

The final report must include:

- each task commit and final HEAD;
- focused unit and PostgreSQL integration results;
- the contains-query `EXPLAIN (ANALYZE, BUFFERS)` evidence and approved index used;
- whether migration `012` was required;
- if required, test forward/rollback/forward and development forward-only results with target names;
- race result and exact Go toolchain/cache;
- serialized full-suite result and any independently confirmed baseline failure;
- final whole-branch review verdict; and
- clean/dirty worktree state plus explicit statements that no push, merge, deployment, production migration, or device verification occurred unless separately evidenced.
