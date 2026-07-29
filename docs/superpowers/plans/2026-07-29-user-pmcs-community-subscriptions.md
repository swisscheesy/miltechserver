# User-Created PMCS Community and Subscriptions Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use
> `superpowers:subagent-driven-development`. Use one fresh implementer and one
> independent reviewer per task. Steps use checkbox (`- [ ]`) syntax.

**Goal:** Add explicit public releases, anonymous recent/model discovery, and
authenticated linked subscriptions with pinned immutable revisions and
explicit update acceptance.

**Architecture:** Community source state is separate from checklist ownership.
Release mutations advance the owner checklist/account versions but never fan
out subscriber writes. Subscription roots have their own ETag and account
change version; update discovery is a lightweight paginated comparison.

**Tech Stack:** Completed foundation and owned-sync plans, Go, Gin, PostgreSQL,
Jet, gzip middleware, `testify`.

## Global constraints

Inherit the master plan. Community release numbers are strictly increasing,
rollback is rejected, current release selection is explicit, and moderation is
absent. Public DTOs never contain UID or email.

---

### Task 9: Revision-specific release, retirement, and retained pins

**Files:**

- Create: `api/user_pmcs/community/repository.go`
- Create: `api/user_pmcs/community/repository_impl.go`
- Create: `api/user_pmcs/community/service.go`
- Create: `api/user_pmcs/community/service_impl.go`
- Create: `api/user_pmcs/community/handler.go`
- Create: `api/user_pmcs/community/route.go`
- Create: `api/user_pmcs/community/handler_test.go`
- Create: `api/user_pmcs/community/service_impl_test.go`
- Create: `tests/user_pmcs/community_test.go`

**Interfaces:**

```go
type Repository interface {
	Release(ctx context.Context, ownerUID string, checklistID,
		revisionID uuid.UUID, precondition shared.Precondition) (
		*ReleaseMutationResult, error)
	Retire(ctx context.Context, ownerUID string, checklistID uuid.UUID,
		precondition shared.Precondition) (*ReleaseMutationResult, error)
}

type ReleaseMutationResult struct {
	Aggregate  shared.ChecklistAggregate
	Idempotent bool
}
```

- [ ] **Step 1: Write failing release/retirement tests**

Cover owner-only access, parent-checklist ETag, never-published revision,
incomplete historical revision revalidation, first release, higher release,
same-release retry, lower/equal rollback, voluntary retirement, retired
reactivation only through a higher release, tombstone non-reactivation, and
zero subscriber fanout.

```go
func TestReleaseRejectsRollback(t *testing.T) {
	checklist := publishedChecklist(t, owner, 3)
	releaseRevision(t, owner, checklist, 3)
	_, err := service.Release(ctx, owner, checklist.ID.String(),
		checklist.Revisions[1].ID.String(), checklist.ETag)
	requireAPIError(t, err, http.StatusConflict, "invalid_transition")
}
```

- [ ] **Step 2: Run and prove failure**

```bash
go test ./api/user_pmcs/community ./tests/user_pmcs \
  -run 'Test(Release|Retire)' -count=1
```

- [ ] **Step 3: Implement release validation and transaction**

Before the transaction, parse IDs and conditional headers. Inside the
transaction lock:

1. owner account sync row;
2. checklist root;
3. community source root, when present; and
4. target immutable revision.

Verify owner, checklist ETag, non-tombstone, target belongs to checklist and is
published/superseded, and full tree still passes publication validation.

For first release, insert release row then source row. For later release,
require `revision_number > latest_release_revision_number`, insert release
row, then update source current pointer/status/timestamps. Advance checklist
and owner account versions once.

The source composite FK proves current release ownership. Repeating the exact
current release is idempotent and does not advance versions.

- [ ] **Step 4: Implement retirement**

Retirement clears `current_release_revision_id` before setting retired state
and preserves `latest_release_revision_number`. Voluntary retirement may be
followed only by a strictly higher release. A checklist tombstone permanently
blocks reactivation.

- [ ] **Step 5: Prove no fanout**

Create 100 subscriptions in the integration test, release the next revision,
and assert no subscription `sync_version`, `account_change_version`, or
`updated_at` changed.

- [ ] **Step 6: Verify and commit**

```bash
go test ./api/user_pmcs/community -count=1
go test ./tests/user_pmcs -run 'Test(Release|Retire)' -count=1
git add api/user_pmcs/community tests/user_pmcs/community_test.go
git commit -m "feat(user-pmcs): release community checklist revisions"
```

---

### Task 10: Anonymous recent browse and current public detail

**Files:**

- Modify: `api/user_pmcs/community/repository.go`
- Modify: `api/user_pmcs/community/repository_impl.go`
- Modify: `api/user_pmcs/community/service.go`
- Modify: `api/user_pmcs/community/service_impl.go`
- Modify: `api/user_pmcs/community/handler.go`
- Modify: `api/user_pmcs/community/route.go`
- Modify: `api/user_pmcs/shared/domain.go`
- Modify: `api/user_pmcs/community/handler_test.go`
- Modify: `tests/user_pmcs/community_test.go`

**Interfaces:**

Add:

```go
type CommunityBrowseFilter struct {
	After           *CommunityCursor
	Limit           int
	NormalizedModel string
}

type PublicCommunitySummary struct {
	ChecklistID        uuid.UUID    `json:"checklist_id"`
	RevisionID         uuid.UUID    `json:"revision_id"`
	RevisionNumber     int32        `json:"revision_number"`
	Name               string       `json:"name"`
	Description        string       `json:"description"`
	Models             []ModelValue `json:"models"`
	CreatorDisplayName string       `json:"creator_display_name"`
	ReleasedAt         time.Time    `json:"released_at"`
	UpdatedAt          time.Time    `json:"updated_at"`
}

type CommunityPage struct {
	NextCursor *string                  `json:"next_cursor,omitempty"`
	HasMore    bool                     `json:"has_more"`
	Items      []PublicCommunitySummary `json:"items"`
}

type PublicChecklistRelease struct {
	ChecklistID        uuid.UUID `json:"checklist_id"`
	CreatorDisplayName string    `json:"creator_display_name"`
	ReleasedAt         time.Time `json:"released_at"`
	Revision           Revision  `json:"revision"`
}

Browse(ctx context.Context, filter shared.CommunityBrowseFilter) (
	*shared.CommunityPage, error)
GetCurrentRelease(ctx context.Context, checklistID uuid.UUID) (
	*shared.PublicChecklistRelease, error)
```

- [ ] **Step 1: Write failing browse/detail tests**

Cover anonymous access, active-only visibility, current release only, default
and max limits, malformed cursor, exact normalized-model filter, public summary
shape, current `users.username`, no UID/email, deleted-account label, 404 for
retired/deleted/never released, static keyset no duplicates/skips, and current
detail conditional GET.

- [ ] **Step 2: Run and prove failure**

```bash
go test ./api/user_pmcs/community ./tests/user_pmcs \
  -run 'TestCommunity(Browse|Detail)' -count=1
```

- [ ] **Step 3: Implement recent keyset browse**

The unfiltered predicate after a cursor is:

```sql
WHERE source.status = 'active'
  AND (
      source.updated_at < $cursor_updated_at
      OR (
          source.updated_at = $cursor_updated_at
          AND source.checklist_id > $cursor_checklist_id
      )
  )
ORDER BY source.updated_at DESC, source.checklist_id ASC
LIMIT $limit_plus_one
```

Use ascending checklist UUID consistently as the timestamp tie-breaker in both
ORDER BY and the cursor predicate. For model filtering, add `EXISTS` against current release
`user_pmcs_revision_models(normalized_text, revision_id)`. Normalize the query
value server-side.

The response contains summary metadata and creator display name from a
`LEFT JOIN users`; null owner/username maps to `"Deleted user"`.

- [ ] **Step 4: Implement current public detail**

Load only an active source's current release and complete immutable tree.
Derive a strong ETag from checklist/revision/content hash and set:

```http
Cache-Control: public, no-cache
Vary: Accept-Encoding
```

Use gzip for browse and detail. No public query may select private drafts,
superseded unreleased revisions, owner UID, or email.

- [ ] **Step 5: Document and test mutable-feed behavior**

The no-duplicate/no-skip assertion applies to a static dataset. Add a test
showing that a concurrent new release can move a source ahead of the current
cursor and is seen after the client restarts at page one. This is accepted v1
recent-feed behavior, not a snapshot guarantee.

- [ ] **Step 6: Verify query plans and commit**

Run `EXPLAIN (ANALYZE, BUFFERS)` for filtered and unfiltered browse and prove
the partial recent index and reverse model index are used.

```bash
go test ./api/user_pmcs/community -count=1
go test ./tests/user_pmcs -run 'TestCommunity(Browse|Detail)' -count=1
git add api/user_pmcs/community tests/user_pmcs/community_test.go
git commit -m "feat(user-pmcs): browse active community checklists"
```

---

### Task 11: Install, unsubscribe, resubscribe, and pinned release reads

**Files:**

- Create: `api/user_pmcs/subscriptions/repository.go`
- Create: `api/user_pmcs/subscriptions/repository_impl.go`
- Create: `api/user_pmcs/subscriptions/service.go`
- Create: `api/user_pmcs/subscriptions/service_impl.go`
- Create: `api/user_pmcs/subscriptions/handler.go`
- Create: `api/user_pmcs/subscriptions/route.go`
- Create: `api/user_pmcs/subscriptions/handler_test.go`
- Create: `api/user_pmcs/subscriptions/service_impl_test.go`
- Create: `tests/user_pmcs/subscriptions_test.go`

**Interfaces:**

```go
type Repository interface {
	Install(ctx context.Context, subscriberUID string, checklistID uuid.UUID,
		precondition shared.Precondition) (*MutationResult, error)
	Unsubscribe(ctx context.Context, subscriberUID string,
		checklistID uuid.UUID, precondition shared.Precondition) (
		*MutationResult, error)
	GetInstalledRelease(ctx context.Context, subscriberUID string,
		checklistID, revisionID uuid.UUID) (
		*shared.InstalledChecklistRelease, error)
}

type MutationResult struct {
	Subscription shared.Subscription
	Installed    *shared.InstalledChecklistRelease
	Created      bool
	Idempotent   bool
}
```

- [ ] **Step 1: Write failing subscription tests**

Cover active public source only, owner self-install rejection, 500-active
ceiling, new `If-None-Match: *`, tombstoned resubscribe requiring current
subscription `If-Match`, exact current release install, same-install retry,
unsubscribe tombstone, stale reinstall rejection, repeated unsubscribe,
retired source resubscribe rejection, pinned redownload after retirement and
owner deletion, and denial of any non-installed revision.

- [ ] **Step 2: Run and prove failure**

```bash
go test ./api/user_pmcs/subscriptions ./tests/user_pmcs \
  -run 'TestSubscription(Install|Unsubscribe|Resubscribe|Pinned)' -count=1
```

- [ ] **Step 3: Implement install/resubscribe transaction**

Lock subscriber sync state, source/checklist root, then subscription row.
Verify account initialization, source active, current release present, and
subscriber is not the checklist owner.

For a new composite identity, require `If-None-Match: *`, enforce the active
subscription ceiling under the account lock, insert at sync version 1, and pin
the source's current release.

For a tombstone, require current subscription `If-Match`, clear `deleted_at`,
install the then-current release, increment subscription sync version, and
advance account version. An active identical subscription may return an
idempotent 200; an incompatible active state returns 412.

- [ ] **Step 4: Implement unsubscribe**

Lock account then subscription, verify ETag, clear installed revision before
setting `deleted_at`, advance versions once, and return a lightweight
tombstone. Clearing the FK releases the content pin. Repeated unsubscribe may
return the same tombstone without a new version.

- [ ] **Step 5: Implement pinned immutable read**

Authorize only the active subscription's exact installed revision. The source
may be active, retired, or owner-deleted. Return the retained immutable tree
with private immutable caching and conditional GET:

```http
Cache-Control: private, max-age=31536000, immutable
```

- [ ] **Step 6: Verify and commit**

```bash
go test ./api/user_pmcs/subscriptions -count=1
go test ./tests/user_pmcs \
  -run 'TestSubscription(Install|Unsubscribe|Resubscribe|Pinned)' -count=1
git add api/user_pmcs/subscriptions tests/user_pmcs/subscriptions_test.go
git commit -m "feat(user-pmcs): add linked checklist subscriptions"
```

---

### Task 12: Paginated update discovery and explicit acceptance

**Files:**

- Modify: `api/user_pmcs/subscriptions/repository.go`
- Modify: `api/user_pmcs/subscriptions/repository_impl.go`
- Modify: `api/user_pmcs/subscriptions/service.go`
- Modify: `api/user_pmcs/subscriptions/service_impl.go`
- Modify: `api/user_pmcs/subscriptions/handler.go`
- Modify: `api/user_pmcs/subscriptions/route.go`
- Modify: `api/user_pmcs/shared/domain.go`
- Modify: `api/user_pmcs/subscriptions/handler_test.go`
- Modify: `tests/user_pmcs/subscriptions_test.go`

**Interfaces:**

Add:

```go
type SubscriptionUpdate struct {
	ChecklistID              uuid.UUID  `json:"checklist_id"`
	SourceStatus             string     `json:"source_status"`
	InstalledRevisionID      uuid.UUID  `json:"installed_revision_id"`
	InstalledRevisionNumber  int32      `json:"installed_revision_number"`
	CurrentReleaseRevisionID *uuid.UUID `json:"current_release_revision_id,omitempty"`
	CurrentReleaseNumber     *int32     `json:"current_release_revision_number,omitempty"`
	UpdateAvailable          bool       `json:"update_available"`
}

type SubscriptionUpdatePage struct {
	NextCursor *string              `json:"next_cursor,omitempty"`
	HasMore    bool                 `json:"has_more"`
	Items      []SubscriptionUpdate `json:"items"`
}

ListUpdates(ctx context.Context, subscriberUID string,
	after *uuid.UUID, limit int) (*shared.SubscriptionUpdatePage, error)
AcceptUpdate(ctx context.Context, subscriberUID string, checklistID,
	revisionID uuid.UUID, precondition shared.Precondition) (
	*MutationResult, error)
```

- [ ] **Step 1: Write failing update tests**

Cover default 50/max 100, stable checklist-UUID cursor, malformed cursor,
active/no-update/newer-update/retired summaries, lightweight payload with no
tree, no mutation during discovery, exact current higher release acceptance,
stale target rejection, rollback rejection, subscription ETag, idempotent
same-target retry, and account-delta reflection after acceptance.

- [ ] **Step 2: Run and prove failure**

```bash
go test ./api/user_pmcs/subscriptions ./tests/user_pmcs \
  -run 'TestSubscription(Update|Accept)' -count=1
```

- [ ] **Step 3: Implement stable paginated discovery**

Query active subscriptions for the authenticated subscriber:

```sql
WHERE subscription.subscriber_uid = $1
  AND subscription.deleted_at IS NULL
  AND ($after::uuid IS NULL OR subscription.checklist_id > $after)
ORDER BY subscription.checklist_id
LIMIT $limit_plus_one
```

Join source/current release and installed revision numbers. Return source
status, installed identity/number, current identity/number when active, and
`update_available`. Do not load authored trees and do not write rows.

- [ ] **Step 4: Implement explicit acceptance**

Lock subscriber account state, source, and subscription in approved order.
Verify subscription ETag, source active, target equals the source's current
release, and target revision number is strictly greater than installed.
Update the installed FK, advance subscription/account versions once, and
return the complete newly installed immutable aggregate.

An exact already-installed target is idempotent. A different old/non-current
target returns 409 without changing the pin.

- [ ] **Step 5: Verify plans and commit**

Run `EXPLAIN (ANALYZE, BUFFERS)` for update discovery at 500 active
subscriptions and verify the partial composite subscription index is used.

```bash
go test ./api/user_pmcs/subscriptions -count=1
go test ./tests/user_pmcs \
  -run 'TestSubscription(Update|Accept)' -count=1
git add api/user_pmcs/subscriptions tests/user_pmcs/subscriptions_test.go
git commit -m "feat(user-pmcs): discover and accept subscription updates"
```

## Community/subscription completion gate

```bash
go test ./api/user_pmcs/community ./api/user_pmcs/subscriptions -count=1
go test ./tests/user_pmcs \
  -run 'Test(Release|Retire|Community|Subscription)' -count=1
go test -race ./api/user_pmcs/community ./api/user_pmcs/subscriptions -count=1
go test ./... -run '^$'
```

Record exact results, query-plan evidence, HEAD, and worktree state.
