# Bug Log

This file logs bugs and their solutions for the miltechserver project. Keep entries brief and chronological.

## Format

Each bug entry should include:
- Date (YYYY-MM-DD)
- Brief description of the bug/issue
- Solution or fix applied
- Any prevention notes (optional)

Use bullet lists for simplicity. Older entries can be manually removed when they become irrelevant.

## Entries

<!-- Add new entries below this line, most recent first -->
- 2026-10-03: Invite admission previously trusted preflight reads and upserted membership roles, allowing revoked-code admission and delayed-duplicate demotion. Admission and invite writers now share the persisted Shop lock, recheck invite/current membership or admin authority inside the write transaction, and insert without role updates. Physical PostgreSQL interleavings and cancellation regressions live in `tests/shops/shops_invite_admission_race_test.go`. Departure/last-admin policy remains a separate remediation task.
- 2026-01-31: user_general unauthorized logging format placeholder caused malformed log output; replaced with plain message and refactored handler into new module.

- 2026-10-04: F29 operational privilege gates used PostgreSQL comma-list ANY
  semantics. Replaced with independent AND checks under the actual DB_USERNAME
  plus legacy-write proof; SELECT-only/INSERT-only/UPDATE-only disposable roles
  reproduce the false positive. F30 migration 017 instructions now require
  test-first order and tagged Jet, preserving all migration bytes. Target
  application/rotation/deployment remain unexecuted; see the release gate sheet.

- 2026-10-04: Task 28 F22 acceptance found that equipment-service username
  repositories returned cancellation errors but nine service operations swallowed
  them as fallback success. Service methods and the shared response mapper now
  propagate resolver errors with their causes; cache hits honor cancellation.
  The real occupied-pool regressions in
  `tests/equipment_services/service_username_cancellation_test.go` cover all nine
  boundaries. Create/update/complete may already have committed before enrichment
  fails: the saved mutation remains, no compensation runs, and a failed legacy
  response must not cause a blind retry. Missing/empty-name fallback is retained.

- 2026-10-04 final review fixes: legacy notifications-with-items now authorizes/reads in one RR snapshot; notification/bootstrap selected IDs bind as arrays (67000 notification parents/items verified); cleanup DB phases/lease acknowledgment are bounded; reverse020 locks all lifecycle proof before checks. Vehicle audit gaps use the common sanitized event, equipment deletion logs only after commit, obsolete unlocked membership helper removed, and sampler cleanup is idempotent. See `docs/testing/shops-server-remediation-acceptance.md` for exact final source/evidence/open gates.
