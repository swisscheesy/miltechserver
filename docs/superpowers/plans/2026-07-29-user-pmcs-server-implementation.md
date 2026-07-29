# User-Created PMCS Server Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use
> `superpowers:subagent-driven-development` to implement this plan
> task-by-task. Use one fresh implementer and one independent reviewer for
> every task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Add production-ready Postgres persistence and Gin endpoints for
private user-created PMCS synchronization, immutable publication history,
public community releases, and linked subscriptions without storing checklist
execution progress.

**Architecture:** Implement one bounded `api/user_pmcs` module with shared
domain/persistence packages and four vertical subdomains: owned checklists,
account delta, community distribution, and subscriptions. A transactionally
locked per-user sync row orders account changes; checklist and subscription
roots provide ETag concurrency; normalized revision trees remain relational
and are loaded/written in batches.

**Tech Stack:** Go 1.23, Gin 1.10.1, PostgreSQL, Jet 2.13.0,
`database/sql`, `github.com/google/uuid`, `github.com/clipperhouse/uax29/v2`
v2.4.0, `testify/require`, Firebase-authenticated `bootstrap.User`.

## Authoritative inputs

- Approved specification:
  `docs/superpowers/specs/2026-07-28-user-pmcs-server-schema-api-design.md`
- Independent review:
  `docs/superpowers/specs/2026-07-28-user-pmcs-server-schema-api-design-REVIEW.md`
- Mobile-origin schema proposal:
  `docs/client/2026-07-27-user-pmcs-server-sync-schema-design.md`
- Current planning baseline: branch `user_generated_pmcs`, HEAD
  `726161c918ce150926c2a6db158249db2a8aa8c4`

If the branch advances before execution, re-read all three documents, recheck
the migration number, and record the new baseline in the progress ledger
before changing code.

## Global constraints

- Do not store user-PMCS inspection execution, progress, completions, faults,
  notes, equipment choices, exports, or history.
- Do not add moderation, suspended sources, rollback releases, direct sharing,
  editable forks, JSONB revision documents, Postgres RLS, Firebase identity
  deletion, or a generic idempotency-key table.
- Existing device-local account-free checklists remain local-only.
- All authored and structural UUIDs are client-generated PostgreSQL `UUID`
  values with no server default.
- Checklist and subscription mutation identity always comes from the
  authenticated `bootstrap.User`; request bodies never supply owner or
  subscriber UID.
- Authenticated Firebase users without a `users` row receive
  `409 account_not_initialized`; PMCS endpoints never auto-provision accounts.
- Existing resources require `If-Match`; first creation of a client-known
  identity requires `If-None-Match: *`; missing preconditions return 428 and
  stale preconditions return 412.
- Every draft, publication, and community-release nested mutation targets the
  parent checklist ETag. Revisions do not have independent mutation ETags.
- Deletion wins over stale offline work. Checklist and subscription tombstones
  remain while the owning/subscribing account exists.
- Account delta embeds complete changed aggregates, never splits one aggregate,
  defaults to 10 roots, caps at 25 roots, and stops before 20 MiB uncompressed
  canonical JSON except that one valid aggregate may occupy a page alone.
- Mutation bodies are strict, uncompressed `application/json`, capped at
  8 MiB, reject unknown fields, reject trailing JSON, and validate UTF-8.
- Short authored fields allow 200 Unicode extended grapheme clusters and
  8 KiB; long fields allow 4,000 graphemes and 64 KiB.
- Use `github.com/clipperhouse/uax29/v2/graphemes` v2.4.0 because it uses
  Unicode 16 data, matching the mobile client's locked `characters` 1.4.1.
- Initial per-revision limits are: 100 checklist models, 100 sections, 100
  section models per section, 1,000 section models total, 500 items per
  section, 2,000 items total, 100 notices per item, 4,000 notices total, 250
  steps per item, and 10,000 steps total.
- Initial account limits are configurable defaults of 250 active owned
  checklists and 500 active subscriptions. Tombstones do not count.
- Field grapheme/byte violations return `422 validation_failed`; node,
  account-root, and whole-body ceilings return `413 content_too_large`.
- Community browse remains ordered by current source `updated_at DESC`,
  checklist UUID. V1 accepts that a new release can restore recency and a
  concurrent update can move an entry ahead of an in-progress cursor; clients
  restart from the first page to refresh the feed.
- Subscription updates paginate by stable checklist UUID with default 50 and
  maximum 100 entries.
- Community browse defaults to 20 entries and caps at 50 entries.
- Write transactions make at most 3 attempts total for retryable SQLSTATE
  40P01/40001 failures.
- Community release rows and retained revisions use restrictive foreign keys.
  Deletion code clears current-release references, deletes unpinned release
  rows, then deletes unpinned revisions in that order.
- No periodic release garbage collector is added in v1. Unpinned historical
  releases are reclaimed only during checklist or account deletion.
- Keep transactions short: decode, normalize, count, validate, and hash before
  opening the transaction; acquire locks in the approved deterministic order;
  encode responses after commit.
- Never log authored text, email, tokens, request bodies, or Firebase claims.
- Do not modify the unrelated shops changes introduced at current HEAD unless
  a fresh user request explicitly changes scope.
- Never hand-edit `.gen`; regenerate Jet output from the migrated schema.

## Review-finding resolution matrix

| Finding | Implementation resolution |
|---|---|
| H1 | Add composite FK from community source `(checklist_id, current_release_revision_id)` to community releases |
| H2 | Use `ON DELETE RESTRICT`; explicitly delete release rows before revision rows |
| M1 | Preserve approved recent-release ordering; document v1 gaming limitation and add release-specific throttling |
| M2 | Field grapheme/byte = 422; node/account/body ceilings = 413 |
| M3 | Add 250/500 configurable account ceilings and paginate subscription updates |
| M4 | Store server-computed SHA-256 canonical revision hashes; compare hashes and identity metadata under lock |
| M5 | Add 500-items-per-section default |
| L1 | Test static keyset correctness and explicitly accept concurrent mutable-feed movement |
| L2 | No background GC; deletion transactions reclaim unpinned releases |
| L3 | Add focused integration tests for every service-only invariant |
| L4 | Keep defensive `40001` classification with `40P01`; test classifier independently |
| L5 | Add first-sync N-publication latency and resumability verification |
| L6 | Pin Unicode-16 grapheme implementation and shared fixture corpus |
| L7 | Parent-checklist ETag requirement is explicit in handlers and tests |

## Planned package structure

```text
api/user_pmcs/
├── route.go                         # dependency construction and route wiring
├── shared/
│   ├── config.go                    # configurable limits and limiter settings
│   ├── domain.go                    # aggregate/request/response value types
│   ├── errors.go                    # stable typed domain errors
│   ├── http.go                      # strict JSON, envelopes, ETags, conditionals
│   ├── normalize.go                 # model normalization
│   ├── validation.go                # draft/publication validation and ceilings
│   ├── content_hash.go              # canonical SHA-256 tree hashing
│   └── cursor.go                    # opaque signed-free structural cursors
├── persistence/
│   ├── store.go                     # DB handle and transaction entry points
│   ├── retry.go                     # bounded 40P01/40001 transaction retry
│   ├── sync_state.go                # account row provisioning and locking
│   ├── tree_reader.go               # batched relational tree loading
│   ├── tree_writer.go               # batched draft replacement
│   └── account_cleanup.go           # PMCS part of account deletion transaction
├── owned/
│   ├── handler.go
│   ├── route.go
│   ├── service.go
│   ├── service_impl.go
│   ├── repository.go
│   └── repository_impl.go
├── sync/
│   ├── handler.go
│   ├── route.go
│   ├── service.go
│   ├── service_impl.go
│   ├── repository.go
│   └── repository_impl.go
├── community/
│   ├── handler.go
│   ├── route.go
│   ├── service.go
│   ├── service_impl.go
│   ├── repository.go
│   └── repository_impl.go
└── subscriptions/
    ├── handler.go
    ├── route.go
    ├── service.go
    ├── service_impl.go
    ├── repository.go
    └── repository_impl.go

tests/user_pmcs/
├── main_test.go
├── helpers_test.go
├── migration_schema_test.go
├── owned_test.go
├── publication_test.go
├── deletion_test.go
├── sync_test.go
├── community_test.go
├── subscriptions_test.go
├── account_deletion_test.go
├── concurrency_test.go
└── performance_test.go
```

Production files stay below roughly 350 lines. Split a file by the
responsibility shown above before allowing a large mixed-concern file.

## Plan sequence

Execute these plans strictly in order:

1. [Foundation plan](2026-07-29-user-pmcs-foundation.md)
2. [Owned synchronization plan](2026-07-29-user-pmcs-owned-sync.md)
3. [Community and subscriptions plan](2026-07-29-user-pmcs-community-subscriptions.md)
4. [Account lifecycle and hardening plan](2026-07-29-user-pmcs-hardening-rollout.md)

Each linked plan is independently testable. Do not begin a later plan until
all tasks in the prior plan pass their focused tests and their independent
review gate.

## Progress ledger

Create `docs/USER_PMCS_SERVER_IMPLEMENTATION_PROGRESS.md` when execution
starts. After each task, record:

- task number and name;
- implementer and reviewer run identifiers;
- commit hash;
- focused test command and exact pass/fail result;
- accepted baseline failures, if any;
- reviewer findings and resolution commit;
- worktree status; and
- next task.

Do not claim a full green suite if only focused packages passed. Do not hide
database availability failures behind skipped tests.

## Final branch gate

After all four plans:

1. run every focused unit and integration package;
2. run `go test ./...` once against the configured test database;
3. run `go test -race` for user-PMCS unit packages;
4. run migration forward/rollback/forward verification in a disposable
   database;
5. regenerate Jet and prove a clean generated diff;
6. run the performance scenarios at approved ceilings;
7. inspect every route registration and authentication boundary;
8. perform a whole-branch review by an independent reviewer;
9. resolve every finding with the original task implementer when possible;
10. record final HEAD and exact worktree status in the ledger.

No mobile remote-sync implementation, production migration application, push,
merge, or deployment is authorized by these plan documents.
