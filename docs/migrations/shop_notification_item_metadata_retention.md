# Notification item metadata retention (023)

Forward: `migrations/023_create_shop_notification_item_metadata.sql`.
Reverse: `migrations/023_rollback_shop_notification_item_metadata.sql`.
This is server-only and follows 019–022; 015–018 are unchanged.

## Data and locking

The key is `(notification_id, niin)` using the exact stored strings. The table
retains nullable nickname and raw unit, `resolved`/`ambiguous` state, JSON raw
candidate values with source physical UUIDs and observation versions, and a
monotonic logical-key version. `resolution_version` records the logical version
of the latest accepted explicit metadata operation establishing agreement; it
starts at zero for backfill and may never exceed the logical version. Ordinary
retention, deletion, quantity-only writes and evidence delivery advance only the
logical version. The table has no quantity or active-item identity.
Only notification deletion cascades away this record. Existing active-item
constraints are preserved; 023 neither merges rows nor adds logical uniqueness.

Migration takes notification then item table locks, verifies the existing item
column shape (017 uses `text` nickname and `varchar(50)` unit), and refuses
conflicting raw nullable candidate pairs. Null differs from empty/default.
Agreeing duplicates preserve every source ID in the backfill; group values are
identical by preflight, so no physical-ID winner is selected. Metadata does not
cross whitespace/case variants of NIIN or notification boundaries.

Inventory conflicting exact keys before rollout. A conflict requires an explicit
owner-approved resolution manifest and separately authorized repair; this pair
does not implement repair, normalize NIIN/units, or aggregate quantities.
A populated reverse refuses loss, including records whose active rows have
already been removed. Empty reverse is a disposable rehearsal, not deployment
authorization. Notification deletion in the disposable fixture proves cascade;
it is not a recommended way to empty real retained data for rollback.

## Writer protocol

All item mutations take existing current-resource authority and notification
coordinating locks. The shared item helper uses the supplied transaction and
context throughout. Existing physical rows resolve omission from themselves;
new UUIDs can fall back only on unambiguous logical metadata. Explicit empty
nickname clears and explicit valid raw unit replaces. The original request's
presence must reach resolution before any default conversion.

Deletion captures candidates before removing physical rows. Ambiguity remains
sticky after one or all conflicting rows disappear. An explicit metadata edit
may resolve only when resulting active candidates agree. A new UUID must supply
both fields to deliberately resolve an ambiguous retained key; no field may be
borrowed from ambiguous history. Default-only atomic no-op behavior is preserved.
A sequence that deletes and re-adds creates a new physical UUID and uses the new
requested quantity, never the removed quantity.

Atomic requests are fingerprinted before lookup/resolution; replay returns its
original receipt before any metadata work. Mandatory audit and receipt records
remain in the same business transaction. Legacy audits remain best effort.

On fallback conflict the business transaction rolls back. A separate integrity
transaction, bounded by the request context and a two-second timeout, rechecks
current notification authority and locks. It may retain only raw committed
candidates captured at the first read of that logical key. Evidence is superseded
only when the stored explicit `resolution_version` is newer than that read's
durable logical version. Ordinary version advances cannot discard previously
observed committed disagreement, even if an external conflicting row disappeared
before a supported deletion or quantity-only save. Each retry gets fresh
observation state. Rolled-back prospective edits and new UUIDs never become
evidence. A newer accepted explicit resolution advances the separate fence,
preventing late evidence from undoing that resolution.
Cancellation or inability to deliver evidence keeps the original safe conflict
and emits `Notification metadata ambiguity evidence unavailable` with actor,
notification and the `integrity_evidence_unavailable` failure category; candidate
values are never logged. Monitor that event before considering rollout healthy.

Legacy failures retain the established safe 500 envelope; negotiated contract 2
returns 409 / `notification_item_metadata_conflict`. This does not alter valid
legacy success envelopes or general failure mappings.

## Verification and rollout

Use only `scripts/test-shops-isolated.sh` with all external test DB selectors
unset. `--verify-notification-migration --verify-message-sync-migration
--verify-remediation-migrations` applies ordered pairs and verifies populated
023 backfill, conflicting/null-empty refusal, incompatible column refusal,
repeated forward/reverse refusal, populated reverse refusal, interrupted reverse,
empty reverse/reapply, raw null/default/unknown values, zero backfill resolution provenance, negative/future
resolution-version constraint refusal and notification cascade.
The intentionally incompatible column type and actual refusal occur in one
rolled-back transaction. Tagged regeneration follows every DDL rehearsal;
generated packages compile at each stage and the final full candidate compiles.
No guard is bypassed and no intermediate generated prefix is published.

The collision fixture uses differing historical `shop_id` labels because the
existing unique key is `(niin, shop_id, notification_id)`. It demonstrates why
physical UUID/current uniqueness does not establish one notification/NIIN value,
without changing any production uniqueness policy.

Canonical publication uses `--publish-candidate-generated`, with exact linked
worktree/branch checks, disposable identity, candidate source path set and
SHA-256 comparison, symlink refusal, and all 32 tracked PMCS files equal to HEAD.
Generated files are never edited manually. Normal automatic startup generation
is unchanged. The generation fixture's documented materialized-view compile ABI
remains synthetic and does not certify a real database or deployment.

Existing named databases, explicit repair manifests, fleet write fence/uniform
writers, actual grants, deployment flags, signed binaries and client/device
acceptance require separate owner authorization/evidence. Old writers can bypass
retention; additive schema alone does not authorize a mixed-writer release.
