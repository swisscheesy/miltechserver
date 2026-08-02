# User PMCS Account-Delta ETags Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Return the exact opaque current-root ETag with every User PMCS account-delta change and provide the mobile team with a verified continuation contract.

**Architecture:** Extend the existing `AccountChange` wire DTO with a required `etag` string. Populate it inside the repeatable-read delta snapshot with the same checklist and subscription ETag helpers used by mutation preconditions, so active resources and tombstones carry validators for the exact root state returned in that change.

**Tech Stack:** Go, Gin, PostgreSQL, `testing`, `testify/require`, Markdown

## Global Constraints

- Do not add a database migration, route, or generated Jet artifact.
- The ETag is a required JSON property on every checklist and subscription change.
- Checklist changes use the current checklist root ETag; subscription changes use the current subscription root ETag.
- Active resources and tombstones use the same root-specific ETag contract.
- Never substitute `account_change_version`, an installed-release validator, or client-side ETag derivation.
- Preserve complete-root pagination and account-delta byte-limit behavior.
- Add a standalone mobile handoff rather than relying only on edits to existing reference documentation.

---

### Task 1: Add account-delta root ETags and mobile continuation documentation

**Files:**
- Modify: `api/user_pmcs/sync/repository.go`
- Modify: `api/user_pmcs/sync/repository_impl.go`
- Modify: `api/user_pmcs/sync/handler_test.go`
- Modify: `tests/user_pmcs/sync_test.go`
- Modify: `docs/client/2026-07-29-user-pmcs-server-api-contract.md`
- Modify: `docs/client/2026-07-31-user-pmcs-mobile-api-implementation-guide.md`
- Create: `docs/client/2026-08-02-user-pmcs-account-delta-etag-mobile-handoff.md`

**Interfaces:**
- Consumes: `shared.MakeChecklistETag(id uuid.UUID, version int64) string` and `shared.MakeSubscriptionETag(checklistID uuid.UUID, version int64) string`.
- Produces: `AccountChange.ETag string` serialized as required JSON property `etag`.

- [ ] **Step 1: Write the failing repository contract assertions**

Extend `TestAccountDeltaMergesCompleteCurrentAggregatesAndTombstones` with literal expectations derived from the fixture identities and root sync versions:

```go
require.Equal(t, shared.MakeChecklistETag(checklistID, 2), owned.ETag)
require.Equal(
    t,
    shared.MakeSubscriptionETag(sourceChecklistID, 3),
    activeSubscription.ETag,
)
require.Equal(
    t,
    shared.MakeChecklistETag(deletedChecklistID, 4),
    checklistTombstone.ETag,
)
require.Equal(
    t,
    shared.MakeSubscriptionETag(unsubscribedChecklistID, 5),
    subscriptionTombstone.ETag,
)
```

- [ ] **Step 2: Write the failing HTTP wire assertion**

Make `TestAccountDeltaPassesRawPaginationToService` return one change with an explicit quoted validator and assert the decoded response preserves it:

```go
ETag: `"opaque-current-root"`,
```

The existing whole-response equality assertion must fail until `AccountChange` exposes the `etag` JSON field.

- [ ] **Step 3: Run the focused tests and verify RED**

Run:

```bash
rtk go test ./api/user_pmcs/sync
rtk go test ./tests/user_pmcs -run TestAccountDeltaMergesCompleteCurrentAggregatesAndTombstones -count=1
```

Expected: compilation or assertion failure because `AccountChange` has no `ETag` field and the delta loader does not populate validators.

- [ ] **Step 4: Implement the minimal wire and loader change**

Add the required field:

```go
ETag string `json:"etag"`
```

When building a checklist change, set:

```go
change.ETag = shared.MakeChecklistETag(
    loaded.aggregate.ID,
    loaded.aggregate.SyncVersion,
)
```

When building a subscription change, set:

```go
change.ETag = shared.MakeSubscriptionETag(
    loaded.subscription.ChecklistID,
    loaded.subscription.SyncVersion,
)
```

Populate the validator after confirming the root exists and before appending the change. Do not branch on `DeletedAt`; tombstones require the same validator construction.

- [ ] **Step 5: Format and verify GREEN**

Run:

```bash
rtk gofmt -w api/user_pmcs/sync/repository.go api/user_pmcs/sync/repository_impl.go api/user_pmcs/sync/handler_test.go tests/user_pmcs/sync_test.go
rtk go test ./api/user_pmcs/sync
rtk go test ./tests/user_pmcs -run TestAccountDeltaMergesCompleteCurrentAggregatesAndTombstones -count=1
```

Expected: both commands exit zero.

- [ ] **Step 6: Update the two existing contract references**

Add `etag` to checklist and subscription account-delta examples in both documents. State that:

- it is the exact quoted, opaque current root validator;
- it is present for active roots and tombstones;
- it must be stored atomically with the root and `through_cursor`;
- clients must replay it unchanged as `If-Match`; and
- clients must not derive it from `sync_version`, use `account_change_version`, or substitute an installed-release ETag.

- [ ] **Step 7: Create the standalone mobile-team handoff**

Document the server commit baseline, changed wire shape, active and tombstone examples, decoding/storage rules, mutation usage, atomic delta application, Plan 3 unblock checklist, compatibility notes, and server verification evidence. Make clear that no route, request, cursor, database schema, or mutation wrapper changed.

- [ ] **Step 8: Run final verification**

Run:

```bash
rtk go test ./api/user_pmcs/...
rtk go test ./tests/user_pmcs -count=1
rtk go test ./...
rtk go test -race ./api/user_pmcs/... ./tests/user_pmcs
rtk git diff --check
rtk git status --short
```

Record any environment-dependent integration-test limitation exactly rather than describing partial verification as full green.

- [ ] **Step 9: Review and commit**

Review the complete diff against this plan, resolve all Critical and Important findings, then create one conventional commit:

```bash
rtk git add api/user_pmcs/sync/repository.go api/user_pmcs/sync/repository_impl.go api/user_pmcs/sync/handler_test.go tests/user_pmcs/sync_test.go docs/client/2026-07-29-user-pmcs-server-api-contract.md docs/client/2026-07-31-user-pmcs-mobile-api-implementation-guide.md docs/client/2026-08-02-user-pmcs-account-delta-etag-mobile-handoff.md docs/superpowers/plans/2026-08-02-user-pmcs-account-delta-etags.md
rtk git commit -m "feat(user-pmcs): include root etags in account delta"
```
