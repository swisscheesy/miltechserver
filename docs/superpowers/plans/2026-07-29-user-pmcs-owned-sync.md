# User-Created PMCS Owned Synchronization Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use
> `superpowers:subagent-driven-development`. Use one fresh implementer and one
> independent reviewer per task. Steps use checkbox (`- [ ]`) syntax.

**Goal:** Deliver authenticated creation, draft synchronization, publication
history, permanent deletion, and complete account-delta synchronization for
owned user-created PMCS checklists.

**Architecture:** Each operation is a vertical handler/service/repository
slice. Services validate and hash complete inputs before a transaction;
repositories lock account state then checklist root, enforce the parent ETag,
mutate the normalized tree, and advance both resource and account versions.

**Tech Stack:** Foundation plan types and persistence primitives, Go, Gin,
PostgreSQL, Jet, `database/sql`, `testify`.

## Global constraints

Complete the foundation plan first and inherit the master plan constraints.
All routes in this plan live under `/api/v1/auth/user-pmcs`. Every nested
draft/publication mutation checks the parent checklist ETag.

---

### Task 5: Owned create, current read, and draft lifecycle

**Files:**

- Create: `api/user_pmcs/owned/repository.go`
- Create: `api/user_pmcs/owned/repository_impl.go`
- Create: `api/user_pmcs/owned/service.go`
- Create: `api/user_pmcs/owned/service_impl.go`
- Create: `api/user_pmcs/owned/handler.go`
- Create: `api/user_pmcs/owned/route.go`
- Create: `api/user_pmcs/owned/handler_test.go`
- Create: `api/user_pmcs/owned/service_impl_test.go`
- Create: `tests/user_pmcs/owned_test.go`

**Interfaces:**

```go
type Repository interface {
	Get(ctx context.Context, ownerUID string, checklistID uuid.UUID) (
		*shared.ChecklistAggregate, error)
	Create(ctx context.Context, ownerUID string, checklistID uuid.UUID,
		draft shared.PreparedRevision,
		precondition shared.Precondition) (*MutationResult, error)
	PutDraft(ctx context.Context, ownerUID string, checklistID uuid.UUID,
		draft shared.PreparedRevision,
		precondition shared.Precondition) (*MutationResult, error)
	DeleteDraft(ctx context.Context, ownerUID string, checklistID,
		revisionID uuid.UUID, precondition shared.Precondition) (
		*MutationResult, error)
}

type MutationResult struct {
	Aggregate  shared.ChecklistAggregate
	Created    bool
	Idempotent bool
}
```

The service exposes the same operations with `*bootstrap.User`, string path
IDs, and raw conditional-header values. It returns canonical strong ETags
derived from the returned checklist `sync_version`.

- [ ] **Step 1: Write failing handler and service tests**

Cover:

- invalid/missing auth context;
- invalid checklist/revision UUID;
- wrong media type, compressed body, unknown field, trailing JSON, and >8 MiB;
- create without `If-None-Match: *` -> 428;
- existing mutation without `If-Match` -> 428;
- malformed conditional -> 400;
- stale conditional -> 412;
- missing Postgres account -> 409 `account_not_initialized`;
- cross-owner read -> 404;
- current GET with matching `If-None-Match` -> bodyless 304;
- create -> 201, idempotent retry -> 200;
- draft replacement -> 200 and one account-version increment; and
- draft deletion without a publication -> 409 `invalid_transition`.

```go
func TestPutDraftRequiresParentChecklistETag(t *testing.T) {
	req := httptest.NewRequest(http.MethodPut,
		"/api/v1/auth/user-pmcs/checklists/"+checklistID+
			"/drafts/"+revisionID,
		strings.NewReader(validDraftJSON))
	req.Header.Set("Content-Type", "application/json")
	resp := httptest.NewRecorder()
	router.ServeHTTP(resp, req)
	require.Equal(t, http.StatusPreconditionRequired, resp.Code)
	requireErrorCode(t, resp.Body.Bytes(), "precondition_required")
}
```

- [ ] **Step 2: Run tests and verify failure**

```bash
go test ./api/user_pmcs/owned ./tests/user_pmcs \
  -run 'Test(Create|GetOwned|PutDraft|DeleteDraft)' -count=1
```

- [ ] **Step 3: Implement service validation**

Create requires an initial draft whose body ID equals the path revision ID
inside the body. The server enforces a null revision number for drafts,
then calls `shared.PrepareDraft` before the repository.

`PutDraft` permits replacing a current draft with a different client UUID only
when the supplied parent ETag is current. `DeleteDraft` succeeds only when the
path ID is the current draft and a current publication exists.

- [ ] **Step 4: Implement create transaction**

Lock order:

1. verify `users(uid)` and lock/create `user_pmcs_sync_state`;
2. inspect checklist ID;
3. if absent, enforce active-owned-checklist ceiling;
4. advance account version;
5. insert root at sync version 1;
6. insert draft row and complete tree; and
7. commit.

If the checklist ID already exists:

- a different owner returns hidden 404;
- a tombstone returns 412 and no authored data;
- exact same draft UUID, byte-exact metadata, and canonical content hash may
  return an idempotent 200 without a version increment; and
- any other state returns 412.

- [ ] **Step 5: Implement current read and draft transactions**

Current GET loads root, draft, publication, and community summary through the
batched tree reader. It returns an owner-visible tombstone for the owning
account but no authored content.

Draft replacement locks account then checklist, verifies current ETag, rejects
tombstones, writes the complete tree, increments checklist sync version once,
advances account version once, and returns the committed aggregate after
commit.

- [ ] **Step 6: Implement HTTP responses**

Use the shared standard/error envelopes. Set ETag on every 200/201/304
resource response. Never include owner UID in response DTOs. Use
`Cache-Control: private, no-cache` for owned mutable aggregates.

- [ ] **Step 7: Run focused verification**

```bash
go test ./api/user_pmcs/owned -count=1
go test ./tests/user_pmcs -run 'Test(Create|GetOwned|PutDraft|DeleteDraft)' -count=1
```

Inspect the active-owned count query with `EXPLAIN (ANALYZE, BUFFERS)`.

- [ ] **Step 8: Commit**

```bash
git add api/user_pmcs/owned tests/user_pmcs/owned_test.go
git commit -m "feat(user-pmcs): sync owned checklist drafts"
```

---

### Task 6: Publication, immutable history, and offline replay

**Files:**

- Modify: `api/user_pmcs/owned/repository.go`
- Modify: `api/user_pmcs/owned/repository_impl.go`
- Modify: `api/user_pmcs/owned/service.go`
- Modify: `api/user_pmcs/owned/service_impl.go`
- Modify: `api/user_pmcs/owned/handler.go`
- Modify: `api/user_pmcs/owned/route.go`
- Modify: `api/user_pmcs/owned/handler_test.go`
- Modify: `api/user_pmcs/owned/service_impl_test.go`
- Create: `tests/user_pmcs/publication_test.go`

**Interfaces:**

Add:

```go
Publish(ctx context.Context, ownerUID string, checklistID uuid.UUID,
	revision shared.PreparedRevision,
	precondition shared.Precondition) (*MutationResult, error)
GetRevision(ctx context.Context, ownerUID string, checklistID,
	revisionID uuid.UUID) (*shared.Revision, error)
```

- [ ] **Step 1: Write failing tests**

Cover publication completeness, exact next number, concurrent next-number
attempts, immutable old content, current-to-superseded transition, publication
without prior draft upload, idempotent retry after the revision later becomes
superseded, historical owner-only GET, immutable GET 304, and interrupted
offline reconstruction.

```go
func TestPublishRejectsRevisionNumberGap(t *testing.T) {
	created := createChecklistWithDraft(t, owner, 1)
	req := validPublication(created.Draft.ID, 3)
	_, err := service.Publish(ctx, owner, created.ID.String(), req,
		created.ETag)
	requireAPIError(t, err, http.StatusConflict, "invalid_transition")
}
```

- [ ] **Step 2: Run and prove failure**

```bash
go test ./api/user_pmcs/owned ./tests/user_pmcs \
  -run 'Test(Publish|Historical|OfflineReplay)' -count=1
```

- [ ] **Step 3: Implement publication validation**

Require `RevisionInput.ID` to match the route revision UUID and a non-null
positive revision number. Call `shared.PreparePublication` before opening the
transaction.

- [ ] **Step 4: Implement the publication transaction**

After account/checklist locks and ETag validation:

1. reject a tombstoned checklist;
2. compute `COALESCE(MAX(revision_number), 0) + 1` under the checklist lock;
3. require the submitted exact next number;
4. if the revision UUID is the current draft, replace its tree only when the
   submitted hash differs, then promote it;
5. if it is not the current draft, replace/remove any current mutable draft,
   insert the submitted revision tree as the draft, then promote it;
6. transition the prior current publication to `superseded`;
7. set the submitted revision to `published`, timestamps, and number;
8. increment checklist and account versions once; and
9. commit.

Published content and metadata never pass through mutable replacement again.

For an already published/superseded same revision UUID, same checklist,
revision number, metadata, and content hash, return an idempotent result
without mutation. A mismatched hash/number returns 412 or 409 and does not
load or compare full trees while holding the lock.

- [ ] **Step 5: Implement immutable revision reads**

Owner-only historical GET returns only `published` or `superseded` revisions
with complete trees. Derive an immutable ETag from checklist UUID, revision
UUID, and stored content hash. Set:

```http
Cache-Control: private, max-age=31536000, immutable
```

- [ ] **Step 6: Test offline history reconstruction**

Reconstruct revisions 1..N with:

1. create root + revision 1 draft;
2. publish revision 1;
3. put revision N draft;
4. publish exact N; and
5. resume from current server state after an injected failure.

Prove server UUIDs/numbers match local inputs exactly and no operation silently
renumbers.

- [ ] **Step 7: Verify and commit**

```bash
go test ./api/user_pmcs/owned -count=1
go test ./tests/user_pmcs -run 'Test(Publish|Historical|OfflineReplay)' -count=1
git add api/user_pmcs/owned tests/user_pmcs/publication_test.go
git commit -m "feat(user-pmcs): publish immutable checklist revisions"
```

---

### Task 7: Checklist tombstones and deletion retention

**Files:**

- Modify: `api/user_pmcs/owned/repository.go`
- Modify: `api/user_pmcs/owned/repository_impl.go`
- Modify: `api/user_pmcs/owned/service.go`
- Modify: `api/user_pmcs/owned/service_impl.go`
- Modify: `api/user_pmcs/owned/handler.go`
- Modify: `api/user_pmcs/owned/route.go`
- Create: `tests/user_pmcs/deletion_test.go`

**Interfaces:**

Add:

```go
DeleteChecklist(ctx context.Context, ownerUID string,
	checklistID uuid.UUID, precondition shared.Precondition) (
	*MutationResult, error)
```

- [ ] **Step 1: Write failing deletion tests**

Cover private deletion, released deletion with zero pins, released deletion
with active subscriber pins, stale create after deletion, stale draft after
deletion, repeated delete, cross-owner deletion, and release-FK deletion order.

- [ ] **Step 2: Run and prove failure**

```bash
go test ./tests/user_pmcs -run 'TestDeleteChecklist' -count=1
```

- [ ] **Step 3: Implement private deletion**

Lock account then checklist, verify parent ETag, delete revisions (cascading
content), set `deleted_at`, preserve root/owner, increment versions once, and
return a lightweight tombstone.

- [ ] **Step 4: Implement released deletion**

Use this exact order while locks are held:

1. set source retired and clear `current_release_revision_id`;
2. identify revisions pinned by active subscriptions;
3. delete unpinned rows from `user_pmcs_community_releases`;
4. delete the draft and all unpinned revisions;
5. retain pinned release rows and immutable revision trees;
6. tombstone the checklist root; and
7. advance versions once.

Never use FK failure as pin detection. Query active pins explicitly first.

- [ ] **Step 5: Implement idempotent tombstone behavior**

A repeated authenticated owner delete returns the current tombstone and ETag
without another account-version increment. Any create/draft/publication using
the tombstoned UUID fails without authored content.

- [ ] **Step 6: Verify and commit**

```bash
go test ./tests/user_pmcs -run 'TestDeleteChecklist' -count=1
go test ./api/user_pmcs/owned -count=1
git add api/user_pmcs/owned tests/user_pmcs/deletion_test.go
git commit -m "feat(user-pmcs): retain permanent checklist tombstones"
```

---

### Task 8: Embedded owner-plus-subscription account delta

**Files:**

- Create: `api/user_pmcs/sync/repository.go`
- Create: `api/user_pmcs/sync/repository_impl.go`
- Create: `api/user_pmcs/sync/service.go`
- Create: `api/user_pmcs/sync/service_impl.go`
- Create: `api/user_pmcs/sync/handler.go`
- Create: `api/user_pmcs/sync/route.go`
- Create: `api/user_pmcs/sync/handler_test.go`
- Create: `tests/user_pmcs/sync_test.go`

**Interfaces:**

```go
type Repository interface {
	GetDelta(ctx context.Context, userUID string, after int64,
		limit int, byteLimit int) (*shared.AccountDelta, error)
}

type AccountDelta struct {
	FromCursor    int64           `json:"from_cursor"`
	ThroughCursor int64           `json:"through_cursor"`
	AccountVersion int64          `json:"account_version"`
	HasMore       bool            `json:"has_more"`
	Changes       []AccountChange `json:"changes"`
}

type AccountChange struct {
	AccountChangeVersion int64                      `json:"account_change_version"`
	Kind                 string                     `json:"kind"`
	Checklist            *ChecklistAggregate        `json:"checklist,omitempty"`
	Subscription         *Subscription              `json:"subscription,omitempty"`
	Installed            *InstalledChecklistRelease `json:"installed,omitempty"`
}
```

`kind` is exactly `checklist` or `subscription`. A checklist change has only
`checklist`; an active subscription has `subscription` plus `installed`; an
unsubscribe tombstone has only `subscription`.

- [ ] **Step 1: Write failing delta tests**

Cover invalid cursor/limit, account-not-initialized, initialized account with no
sync row, ordered merge of owner and subscription roots, collapsed multiple
root changes, embedded current draft/publication only, subscription installed
tree, tombstones, entry limit, byte limit, one oversize-but-valid aggregate,
repeatable snapshot, and no aggregate splitting.

- [ ] **Step 2: Run and prove failure**

```bash
go test ./api/user_pmcs/sync ./tests/user_pmcs \
  -run 'TestAccountDelta' -count=1
```

- [ ] **Step 3: Implement the repeatable-read root page**

Open one read-only `REPEATABLE READ` transaction. Verify the `users` row, read
sync current version (zero when no sync row), then `UNION ALL`:

```sql
SELECT account_change_version, 'checklist' AS kind, id::text AS identity
FROM user_pmcs_checklists
WHERE owner_uid = $1 AND account_change_version > $2
UNION ALL
SELECT account_change_version, 'subscription' AS kind,
       checklist_id::text AS identity
FROM user_pmcs_subscriptions
WHERE subscriber_uid = $1 AND account_change_version > $2
ORDER BY account_change_version
LIMIT $3;
```

Fetch `limit + 1` roots to determine `has_more`. Load all chosen aggregates in
batches from the same transaction snapshot.

- [ ] **Step 4: Enforce response byte boundaries without splitting**

Canonical-encode each complete change independently after the transaction
loads it, accumulate entries until the next would exceed 20 MiB, and allow the
first valid change alone. `through_cursor` is the last included version or
`after` for an empty page. If byte truncation occurs, `has_more` is true.

Encode the final standard response envelope after commit. Apply gzip middleware
and `Vary: Accept-Encoding`.

- [ ] **Step 5: Verify snapshot behavior**

Use two database connections and barriers to mutate after the repeatable-read
snapshot starts. Prove the later change is absent from the current page,
appears after the returned cursor on the next pull, and no raw sequence is
used.

- [ ] **Step 6: Verify and commit**

```bash
go test ./api/user_pmcs/sync -count=1
go test ./tests/user_pmcs -run 'TestAccountDelta' -count=1
git add api/user_pmcs/sync tests/user_pmcs/sync_test.go
git commit -m "feat(user-pmcs): add embedded account delta sync"
```

## Owned-sync completion gate

```bash
go test ./api/user_pmcs/shared ./api/user_pmcs/persistence \
  ./api/user_pmcs/owned ./api/user_pmcs/sync -count=1
go test ./tests/user_pmcs \
  -run 'Test(Create|GetOwned|PutDraft|DeleteDraft|Publish|Historical|OfflineReplay|DeleteChecklist|AccountDelta)' \
  -count=1
go test -race ./api/user_pmcs/... -count=1
go test ./... -run '^$'
```

Record the exact test counts, HEAD, and worktree state before community work.
