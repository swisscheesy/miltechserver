# Shop message asset lifecycle (020)

The forward/reverse pair is `020_create_shop_message_asset_lifecycle.sql` and
`020_rollback_shop_message_asset_lifecycle.sql`. It follows the reviewed 019
allocator bridge; migrations 015–018 remain unchanged. Deployment, named database
application, cloud operations and historical ownership repair are separate gates.

## Data and authority

`shop_message_uploads` holds server-minted UUIDs and operation UUIDs, immutable
original Shop/uploader IDs, canonical account/container/key/URL/extension,
operation lease, state, timestamps and classified failure. Original IDs have no
cascading parent FK: deleting a message, account or Shop cannot remove ownership
proof or terminal registry rows. `(account,container,blob_key)` is unique.

`shop_message_asset_references` uses a message/upload primary key and composite
FKs to `(id,shop_id)` on messages and uploads. Only message deletion cascades;
active references prevent deleting the upload row. The new message composite
unique constraint is also the planned 021 reply-integrity prerequisite.

`shop_message_blob_cleanup_jobs` stores independent, immutable target snapshots.
It has no FK that can erase queued work. Upload state transitions are enforced
by an immutable-target trigger: uploading → ready → cleanup_pending → deleting →
deleted, plus uploading → cleanup_pending. Job states are pending, leased,
retryable, completed and manual-review. Due/lease/asset indexes support bounded
claim and recovery. The cursor field is reserved for Task14's durable prefix
paging; no prefix deletion executes in Task12.

Every deleted managed reference inserts a new cleanup candidate in the same
transaction, including FK cascades. Candidates deliberately are not deduplicated
against leased jobs, and do not constitute deletion authority. The Task14 worker
must recheck registry/zero references under lock, freeze the target, verify
configured account/container against the actual SDK endpoint, then make a bounded
cloud call outside SQL locks. It must wait for a live upload lease even if failure
compensation or Shop deletion has changed uploading to cleanup_pending. Retry
must retain deleting state; finalization cannot reactivate a frozen target.

The AFTER DELETE trigger is SECURITY INVOKER and takes no explicit asset/Shop
lock. Every deletion role needs SELECT on the registry and INSERT on the jobs
in addition to its deletion rights. Missing rights fail and roll back deletion.
Disposable verification covers PostgreSQL owner and a deliberately restricted
NOLOGIN deletion role both without and with queue privileges. Target deployment
must inventory the real application and account-deletion roles before approval.

## Message and discard behavior

All marker occurrences are parsed. HTTPS account host, container and exactly
once-decoded blob key select same-Shop registry rows; URL queries/fragments do
not hide a reference. No suffix comparison or path cleaning grants ownership.
Unknown/external/foreign markers retain their original display text and never
register a managed reference. Known targets in cleanup/deleting/deleted states
cannot attach. Sorted affected-asset locks serialize attachment/discard decisions
inside the current-authority Shop/message transaction.

Discard accepts only a ready, registered same-Shop upload with zero references,
from its uploader or a current admin who is still a member. Upload UUIDs are
independent of message-row IDs. Published assets stay protected, including assets
shared by two messages and repeated markers in one message. Ready unattached
uploads have no automatic expiry policy.

No historical uploader is inferred from authors or URLs. Historical registration
requires an explicit owner-approved mapping (Task27). Whole-Shop registry enqueue
is an internal transaction API; Task14 wires it into both authorized Shop deletion
paths. Unknown historical individual targets remain protected.
Migration 024 is that owner-approved mapping for canonical same-Shop markers; see
`shop_message_legacy_image_registration.md` and ADR-023.

## Reversal and verification

Reverse first takes ACCESS EXCLUSIVE locks on uploads/assets, references, then
cleanup jobs, in that order, before any emptiness check. `SET LOCAL lock_timeout
= '5s'` bounds each acquisition wait; it does not bound total migration runtime.
In-flight writers must finish before the checks or the reverse transaction
refuses on timeout/deadlock. Committed reservation/job proof then causes refusal;
no empty-check/drop window remains. Fence all lifecycle writers operationally.
Reverse refuses if any registry, reference or job row exists, including terminal
proof. It does not discard pending work or tombstones. Empty reverse drops only
020 objects; subsequent migrations depending on the composite message key must
be reversed first. Reapplying after empty reverse is supported. A repeated forward
fails transactionally rather than silently accepting schema drift.

Use the marker-protected loopback wrapper, never an inherited DSN:

```sh
rtk proxy env -u TEST_DATABASE_URL -u TEST_DATABASE_MARKER -u TEST_DB_URL \
  bash scripts/test-shops-isolated.sh \
  --verify-notification-migration --verify-message-sync-migration \
  --verify-remediation-migrations -v -count=1 \
  ./tests/shops ./tests/equipment_services
```

The helper proves historical text preservation/no inferred ownership, forward,
reverse/reapply, repeated-forward refusal, populated-reverse refusal and trigger
privileges. Each schema step generates fresh canonical tagged Jet outputs,
verifies private-socket/TCP marker identity, compares all 32 tracked PMCS files,
and compiles generated packages. Final available candidate source gets a full
build. The synthetic empty TMDE compile fixture is not live/runtime certification.

The controller-approved `--publish-candidate-generated` option publishes only
that final untouched full available candidate to the fixed isolated remediation
worktree. It rejects another worktree/branch, symlinks, incomplete or changed
tracked PMCS references, intermediate target identity and source changes since
the candidate build. Publication is serialized and stages a full tree before
rename, retaining previous inputs on failure. Generation, source and PMCS
reference manifests accompany the output. This option does not permit arbitrary
output roots or primary-checkout generation.


## Durable cleanup operations (Task 14)

The feature worker starts after tagged startup generation and route construction,
with one worker per process, batch 10, a 5-second poll, 30-second cloud deadline,
and 2-minute claims. Failed cloud attempts back off from 5 seconds, doubling to a
5-minute ceiling; the tenth failed attempt or a target/account mismatch becomes
manual-review. Successful prefix pages persist the continuation cursor; retries
repeat an incomplete page idempotently. Listing returns at most 100 keys per page.
Cancellation stops cloud work, joins the worker, and then closes the database.

Claims commit before asset preparation. Preparation proves current zero
references, honors retained uploading leases (including cleanup_pending), freezes
the asset and commits before cloud I/O. Finish locks only its owned queue row.
A subsequent reconciliation pass applies deleting-to-deleted using a committed,
successful job with the identical immutable target. A crash between those steps
leaves a recoverable frozen asset, never an attachable one. Reconciliation also
re-enqueues terminal targets after the 5-minute horizon, because a canceled PUT
can arrive after its first successful delete. No ready-draft expiration exists.

Monitor pending/retryable backlog count and oldest due age, leased jobs past their
deadline, manual-review count, and attempts distribution. The worker emits
`shop_image_cleanup_lease_recovered`, `shop_image_cleanup_manual_review` and a
sanitized `shop_image_cleanup_iteration_failed` event. Alert on any manual-review
job, increasing overdue backlog across polls, or repeated lease recovery. Keep
terminal registry and queue proof; pruning it requires a separately reviewed
retention policy that preserves late-PUT recovery. Reconciliation is bounded per
poll, and retained historical targets continue to consume reconciliation work.

For account/container configuration changes, restore the authorized target or
resolve the target-specific manual-review job through an audited operational
procedure; never rewrite immutable target identity to the current account.
Historical individual uploader ownership remains an explicit inventory/mapping
approval gate. Exact persisted Shop UUID prefix proof authorizes whole-Shop
cleanup and survives parent deletion; it does not establish individual uploader
ownership. No inventory mapping, live Azure acceptance, deployment role grant,
named database migration, or feature activation is established by local tests.

The final review repair changes only the new 020 reverse file, SHA-256
`d8003e69aae701e8df15b0ee4ab9a7aa2b7c9ebcf2f2a91b96e35bf7039a1f6a`.
The guarded runner SOURCE_PINS and its lock-order/checksum regression match.
015–018 and every other reviewed forward/reverse body remain frozen. Physical
`TestCleanup020ReverseSerializesWriters` uses separate PostgreSQL connections
and observes the actual blocking PID before committing reservations/jobs; the
exact lock/check prefix must refuse and retain their proof. The migration matrix
separately verifies full empty reverse/reapply and populated refusal with tagged
generation after each action. This is disposable evidence, never named-target
or fleet approval.
