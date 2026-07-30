# User-Created PMCS Account Lifecycle and Hardening Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use
> `superpowers:subagent-driven-development`. Use one fresh implementer and one
> independent reviewer per task. Steps use checkbox (`- [ ]`) syntax.

**Goal:** Integrate PMCS retention with account deletion, wire all routes,
enforce rate/observability safeguards, prove concurrency and performance, and
publish the stable contract for the mobile team.

**Architecture:** Account deletion remains one Postgres transaction authorized
only by the verified Firebase UID. Top-level route wiring constructs the four
subdomains with isolated rate-limit buckets. Real-Postgres concurrency and
performance tests are the release gate; deployment remains schema-first and
server-first.

**Tech Stack:** Completed prior plans, Go, Gin, PostgreSQL, Jet, `slog`,
`golang.org/x/time/rate`, gzip, `testify`.

## Global constraints

Inherit the master plan. This plan adds no Firebase Admin deletion and does not
deploy, push, merge, or modify the mobile client.

---

### Task 13: Authenticated account deletion and retained public content

**Files:**

- Create: `api/user_pmcs/persistence/account_cleanup.go`
- Create: `api/user_pmcs/persistence/account_cleanup_test.go`
- Modify: `api/user_general/repository.go`
- Modify: `api/user_general/repository_impl.go`
- Modify: `api/user_general/service.go`
- Modify: `api/user_general/service_impl.go`
- Modify: `api/user_general/route.go`
- Modify: `api/user_general/route_test.go`
- Modify: `api/user_general/service_impl_test.go`
- Create: `tests/user_pmcs/account_deletion_test.go`

**Interfaces:**

```go
type AccountCleaner interface {
	CleanupAccount(ctx context.Context, tx *sql.Tx, uid string) error
}

type Repository interface {
	UpsertUser(user *bootstrap.User, userDTO auth.UserDto) error
	DeleteUser(ctx context.Context, uid string) error
	UpdateUserDisplayName(uid string, displayName string) error
}
```

`user_general.NewRepository` receives the DB and a PMCS `AccountCleaner`.
`DeleteUser` owns the one transaction that cleans PMCS and deletes `users`.

- [ ] **Step 1: Write failing authority and retention tests**

Cover:

- request-body UID cannot delete another user;
- bodyless authenticated deletion derives `bootstrap.User.UserID`;
- private roots/tombstones are removed with the account;
- the deleting user's subscription tombstones are removed;
- unpinned released roots/releases/revisions are removed;
- pinned released roots are tombstoned, owner-null, and retained;
- retained source is retired/current pointer null;
- active subscribers can redownload the pinned revision;
- public browse/detail no longer exposes the deleted source;
- creator attribution becomes `"Deleted user"`; and
- the entire transaction rolls back on an injected cleanup failure.

```go
func TestDeleteUserIgnoresBodyUIDAndUsesAuthenticatedUID(t *testing.T) {
	req := httptest.NewRequest(http.MethodDelete,
		"/api/v1/auth/user/general/delete_user",
		strings.NewReader(`{"uid":"victim"}`))
	resp := httptest.NewRecorder()
	router.ServeHTTP(resp, req)
	require.Equal(t, http.StatusOK, resp.Code)
	require.Equal(t, "caller", repository.deletedUID)
}
```

- [ ] **Step 2: Run and prove failure**

```bash
go test ./api/user_general ./api/user_pmcs/persistence ./tests/user_pmcs \
  -run 'Test(DeleteUser|AccountDeletion)' -count=1
```

- [ ] **Step 3: Implement token-derived deletion**

`deleteUser` requires a valid `*bootstrap.User`, does not bind a deletion body,
and calls:

```go
err := handler.service.DeleteUser(c.Request.Context(), currentUser.UserID)
```

Update service/repository context signatures. Preserve the existing route and
success status so the current mobile workflow remains compatible even if it
still sends a body.

- [ ] **Step 4: Implement PMCS cleanup inside the user transaction**

Use deterministic sorted locks:

1. sync state row;
2. owned checklist UUIDs;
3. related source UUIDs;
4. deleting user's subscription checklist UUIDs;
5. dependent revisions/releases.

Delete the user's subscriptions first after locks are acquired so their pins
are released. For each owned checklist:

- private or never-released: delete revisions and root;
- released with no remaining active pins: clear/delete source, delete release
  rows, delete revisions, delete root;
- released with remaining pins: clear current source pointer, retire source,
  delete unpinned release rows, delete unpinned revisions, tombstone root, and
  set `owner_uid = NULL`.

Then delete sync state and `users` in the same transaction. No Firebase call or
response encoding occurs inside.

- [ ] **Step 5: Verify FK order and rollback safety**

Use restrictive FKs to prove incorrect ordering fails in a test transaction.
Use a fault-injecting cleaner to prove no partial account deletion commits.

- [ ] **Step 6: Verify and commit**

```bash
go test ./api/user_general ./api/user_pmcs/persistence -count=1
go test ./tests/user_pmcs -run 'TestAccountDeletion' -count=1
git add api/user_general api/user_pmcs/persistence \
  tests/user_pmcs/account_deletion_test.go
git commit -m "fix(accounts): delete PMCS data from authenticated identity"
```

---

### Task 14: Top-level routes, compression, rate limits, and safe observability

**Files:**

- Create: `api/user_pmcs/route.go`
- Create: `api/user_pmcs/ratelimit.go`
- Create: `api/user_pmcs/ratelimit_test.go`
- Create: `api/user_pmcs/observability.go`
- Create: `api/user_pmcs/observability_test.go`
- Modify: `api/user_pmcs/shared/config.go`
- Modify: `bootstrap/env.go`
- Modify: `api/route/route.go`
- Modify: `api/route/route_test.go`

**Interfaces:**

```go
type Dependencies struct {
	DB     *sql.DB
	Config shared.Config
}

func RegisterRoutes(
	deps Dependencies,
	publicGroup *gin.RouterGroup,
	authGroup *gin.RouterGroup,
)
```

- [ ] **Step 1: Write failing route-registration tests**

Assert all 17 approved method/route pairs and correct public/auth groups:

```text
GET    /api/v1/auth/user-pmcs/sync
GET    /api/v1/auth/user-pmcs/checklists/:checklist_id
PUT    /api/v1/auth/user-pmcs/checklists/:checklist_id
PUT    /api/v1/auth/user-pmcs/checklists/:checklist_id/drafts/:revision_id
DELETE /api/v1/auth/user-pmcs/checklists/:checklist_id/drafts/:revision_id
PUT    /api/v1/auth/user-pmcs/checklists/:checklist_id/publications/:revision_id
GET    /api/v1/auth/user-pmcs/checklists/:checklist_id/revisions/:revision_id
DELETE /api/v1/auth/user-pmcs/checklists/:checklist_id
PUT    /api/v1/auth/user-pmcs/checklists/:checklist_id/community-releases/:revision_id
DELETE /api/v1/auth/user-pmcs/checklists/:checklist_id/community-source
GET    /api/v1/user-pmcs/community
GET    /api/v1/user-pmcs/community/:checklist_id
PUT    /api/v1/auth/user-pmcs/subscriptions/:checklist_id
DELETE /api/v1/auth/user-pmcs/subscriptions/:checklist_id
GET    /api/v1/auth/user-pmcs/subscriptions/updates
PUT    /api/v1/auth/user-pmcs/subscriptions/:checklist_id/installed-releases/:revision_id
GET    /api/v1/auth/user-pmcs/subscriptions/:checklist_id/installed-releases/:revision_id
```

Every pair above must be asserted.

- [ ] **Step 2: Write failing limiter and log-redaction tests**

Use fake clocks and deterministic limiter factories. Prove separate IP/user
buckets, idle-entry cleanup, 429 envelope, and no authored text/body/email in
captured logs.

- [ ] **Step 3: Implement route wiring**

Construct one shared persistence store and the four repositories/services.
Register public community routes on `v1Route`, every mutation/read requiring
account identity on `authRoutes`, and gzip on delta/community/full-tree GETs.
`route.Setup` passes defaults when `env == nil` so registration tests remain
safe.

- [ ] **Step 4: Implement configurable keyed limits**

Add these defaults:

| Bucket | Rate | Burst |
|---|---:|---:|
| Public community per IP | 2/second | 20 |
| Authenticated reads per user | 10/second | 30 |
| Authenticated mutations per user | 2/second | 10 |
| Community release per user | 12/hour | 3 |
| Community release per IP | 60/hour | 10 |

Add these `shared.Config` fields and environment names:

| Field | Environment name |
|---|---|
| `PublicRequestsPerSecond` | `USER_PMCS_PUBLIC_REQUESTS_PER_SECOND` |
| `PublicRequestBurst` | `USER_PMCS_PUBLIC_REQUEST_BURST` |
| `AuthenticatedReadsPerSecond` | `USER_PMCS_AUTH_READS_PER_SECOND` |
| `AuthenticatedReadBurst` | `USER_PMCS_AUTH_READ_BURST` |
| `AuthenticatedMutationsPerSecond` | `USER_PMCS_AUTH_MUTATIONS_PER_SECOND` |
| `AuthenticatedMutationBurst` | `USER_PMCS_AUTH_MUTATION_BURST` |
| `ReleasesPerUserPerHour` | `USER_PMCS_RELEASES_PER_USER_PER_HOUR` |
| `ReleaseUserBurst` | `USER_PMCS_RELEASE_USER_BURST` |
| `ReleasesPerIPPerHour` | `USER_PMCS_RELEASES_PER_IP_PER_HOUR` |
| `ReleaseIPBurst` | `USER_PMCS_RELEASE_IP_BURST` |
| `LimiterIdleMinutes` | `USER_PMCS_LIMITER_IDLE_MINUTES` |

Use separate maps, `rate.Limiter`, a 15-minute idle TTL, and bounded cleanup
triggered opportunistically no more than once per minute. Keys are UID or
`c.ClientIP()` only; never email/token.

- [ ] **Step 5: Implement structured measurements**

`Observer` records operation, status/code, durations, retry count, node counts,
and request/response bytes. The default emits structured `slog` events that
the existing logging pipeline can aggregate:

```go
type Observation struct {
	Operation      string
	Status         int
	Code           string
	Duration       time.Duration
	DBDuration     time.Duration
	EncodeDuration time.Duration
	RetryCount     int
	NodeCount      int
	RequestBytes   int64
	ResponseBytes  int
}
```

Handlers log UUIDs and authenticated UID only. They never attach DTOs, authored
strings, email, claims, or bodies.

- [ ] **Step 6: Verify and commit**

```bash
go test ./api/user_pmcs ./api/route -count=1
go test -race ./api/user_pmcs -count=1
git add api/user_pmcs api/route/route.go api/route/route_test.go bootstrap/env.go
git commit -m "feat(user-pmcs): wire routes and operational safeguards"
```

---

### Task 15: Real-Postgres concurrency, authorization, and performance gates

**Files:**

- Create: `tests/user_pmcs/concurrency_test.go`
- Create: `tests/user_pmcs/performance_test.go`
- Create: `tests/user_pmcs/http_contract_test.go`
- Modify: `tests/user_pmcs/helpers_test.go`

**Interfaces:**

- Produces repeatable test helpers and evidence only; no production API.

- [ ] **Step 1: Add deterministic concurrency tests**

Use two dedicated `*sql.Conn` instances, channels/barriers, and bounded
contexts. Cover:

- simultaneous saves with one ETag -> exactly one mutation;
- simultaneous publication of the same next number -> no duplicate/skip;
- simultaneous higher/lower release -> no rollback;
- concurrent first mutations -> one sync-state row and ordered versions;
- different users -> no global serialization;
- deadlock SQLSTATE classifier and bounded retry exhaustion; and
- later mutation after delta snapshot -> next page only.

Tests must fail on timeout rather than hang.

- [ ] **Step 2: Add authorization/contract matrix**

For every route, test absent auth, malformed auth context, cross-owner access,
unknown/tombstoned resource, content-type/body/conditional failures, stable
error envelope, and safe 404 hiding.

Also prove:

- owner cannot subscribe to own source;
- publication notice type/completeness service invariant;
- no private draft/superseded-unreleased content appears publicly; and
- current creator display name changes when `users.username` changes.

- [ ] **Step 3: Add representative maximum-tree builder**

Build deterministic client UUID trees at configurable sizes. The full approved
ceiling fixture contains 100 sections, 2,000 items while respecting 500 per
section, 4,000 notices, and 10,000 steps. Keep authored strings small for the
node-count benchmark and create separate 8 MiB body/field-boundary fixtures.

- [ ] **Step 4: Measure required scenarios**

Measure and log p50/p95, DB duration, lock wait, encode time, peak allocated
bytes, gzip/uncompressed bytes, and query count for:

- maximum draft replacement;
- maximum publication;
- 25-root embedded delta near 20 MiB;
- 20 concurrent independent users;
- filtered/unfiltered community browse;
- 500-subscription update discovery; and
- first-sync reconstruction of 50 historical publications with an injected
  interruption and resume.

Use `testing.B`/`testing.AllocsPerRun` for allocation evidence and structured
test logs for latency. Do not encode environment-specific latency thresholds
as correctness assertions. Assert fixed query-count bounds, response byte
ceilings, no aggregate splitting, and no N+1 growth.

- [ ] **Step 5: Capture query plans**

Run and save test logs containing `EXPLAIN (ANALYZE, BUFFERS)` for:

- owner and subscription delta branches;
- batched tree loaders;
- active recent browse;
- exact model browse;
- subscription updates;
- active pin lookup; and
- account-limit counts.

Fail the test if representative plans use an unexpected sequential scan on a
large seeded relation where an approved index should apply.

- [ ] **Step 6: Run focused and race verification**

```bash
go test ./tests/user_pmcs -run 'Test(Concurrency|HTTPContract|Authorization)' -count=1
go test ./tests/user_pmcs -run 'TestPerformanceScenarios' -count=1 -v
go test -race ./api/user_pmcs/... -count=1
```

- [ ] **Step 7: Commit**

```bash
git add tests/user_pmcs
git commit -m "test(user-pmcs): cover concurrency and performance limits"
```

---

### Task 16: Mobile contract, migration rehearsal, and final branch review

**Files:**

- Create: `docs/client/2026-07-29-user-pmcs-server-api-contract.md`
- Modify: `docs/USER_PMCS_SERVER_IMPLEMENTATION_PROGRESS.md`
- Modify only if decisions changed:
  `docs/project_notes/decisions.md`

**Interfaces:**

- Produces the stable mobile-facing JSON/HTTP contract and final verification
  record.

- [ ] **Step 1: Write the client contract**

Document every route with:

- auth requirement;
- path/query/header parameters;
- strict request JSON example;
- standard success/error envelope example;
- ETag/conditional behavior;
- stable error codes;
- pagination cursor rules;
- current draft/publication delta shapes;
- tombstone shapes;
- public creator field;
- offline first-sync sequence; and
- pinned release/update behavior.

Copy exact field names from implemented DTO tags. Do not document aspirational
fields.

- [ ] **Step 2: Rehearse and verify both non-production databases**

Using `TEST_DATABASE_URL`, first confirm the target is `miltech_ng_test`, then
run forward -> schema tests -> rollback -> absence checks -> forward. Next
confirm the standard development connection targets `miltech_ng`, apply the
forward migration there once, and verify the schema exists. Do not roll back
`miltech_ng`, and never target production.

Regenerate Jet from the migrated `miltech_ng` schema. Confirm the regenerated
`.gen` tree is identical to committed generated output and record the exact
result for each database separately in the progress ledger.

- [ ] **Step 3: Run the complete verification matrix**

```bash
go test ./api/user_pmcs/... ./api/user_general ./api/route -count=1
go test ./tests/user_pmcs -count=1
go test -race ./api/user_pmcs/... -count=1
go test ./... -count=1
git diff --check
git status --short
```

Report database/environment failures exactly. Do not claim a full green suite
if integration tests could not connect.

- [ ] **Step 4: Run an independent whole-branch review**

Review the complete diff against both the approved spec and review document.
Audit:

- all route/auth boundaries;
- restrictive FK deletion order;
- account-version lock order;
- conditional request correctness;
- publication/release atomicity;
- tombstone permanence;
- batch-query behavior;
- public privacy;
- log redaction;
- account deletion authority;
- test depth and race coverage; and
- unrelated-file changes.

Return findings to the relevant task implementer, apply fixes, rerun the
affected focused tests, then rerun the complete matrix.

- [ ] **Step 5: Final implementation commit**

```bash
git add docs/client/2026-07-29-user-pmcs-server-api-contract.md \
  docs/USER_PMCS_SERVER_IMPLEMENTATION_PROGRESS.md \
  docs/project_notes/decisions.md
git commit -m "docs(user-pmcs): publish server synchronization contract"
```

Omit `docs/project_notes/decisions.md` from `git add` when it did not change.

## Final handoff

Report:

- final HEAD;
- every task commit;
- exact focused/full/race/migration results;
- separate forward-migration verification for `miltech_ng_test` and
  `miltech_ng`, plus rollback-rehearsal results for `miltech_ng_test`;
- captured performance and query-plan evidence;
- accepted baseline failures;
- exact worktree status;
- deferred v1 limitations;
- production migration not applied; and
- mobile sync implementation not started.

Then use `superpowers:finishing-a-development-branch` to offer integration
choices. Do not push, merge, deploy, or delete the worktree without explicit
user authorization.
