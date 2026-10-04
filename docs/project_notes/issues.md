# Issues/Work Log

This file logs work completed on tickets for the miltechserver project. Keep it simple - just enough to remember what was done.

## Format

Each entry should include:
- Date (YYYY-MM-DD)
- Ticket ID (if applicable)
- Brief description (1-2 lines)
- URL to ticket (if available)
- Status (optional: completed, in-progress, blocked)

## Recent Work

<!-- Add new entries below this line, most recent first -->

### 2026-09-29 - Shops message sync activation (server)
- **Status**: Code complete on branch, not merged or deployed; live application and flag activation not authorised
- **Ticket**: none; branch `feature/shops-message-sync` (no URL)
- **Description**: Migration 018 (per-shop `insertion_number`, counter table, sole-allocator trigger, indexes), reader wired behind `SHOPS_MESSAGE_SYNC_ENABLED`, capability = flag AND readiness probe, real-Postgres tests, query-plan and lock measurements, pinned runner `scripts/apply-shops-message-sync-migration.sh`.
- **Notes**: Runner refuses until the operator pins pre-018 schema checksums (`UNPINNED` sentinel). Preflight query results and the app-role privilege check are still outstanding. Known unrelated failures: `TestAtomicNotificationItemFieldsSurviveReleasedClientSaves`, `scripts/test-shops-isolated_test.py`. See ADR-022 and `docs/testing/shops-database.md`.

### 2026-09-28 - Notification item nickname and unit of measure
- **Status**: Completed (isolated integration suite not yet run)
- **Description**: Migration 017 adds nullable `nickname` / `unit_of_measure` to `shop_notification_items`. Atomic save, legacy item add and the maintenance snapshot read and write them; an omitted field means "unchanged" so released clients stay compatible. Recreated `tools/jetregen` (json-tagged jet generation).
- **Notes**: Manual SQL in `docs/migrations/shop_notification_item_nickname_uom_migration.md`. Deploy before the matching mobile release. Applied to the 192.168.20.70 `miltech_ng` dev database on 2026-09-28.

### 2025-01-30 - Project Memory System Initialized
- **Status**: Completed
- **Description**: Set up project memory infrastructure in docs/project_notes/
- **Notes**: Created bugs.md, decisions.md, key_facts.md, and issues.md


### 2026-10-04 - Shops server remediation Task 27

- **Status**: Local runner/runbooks implemented; release BLOCKED on owner gates.
- **Description**: One-action guarded 019–023 runner, pinned source/refusal tests,
  actual-role privilege intersections and real legacy insert proof; corrected
  017 test-first/tagged-generation instructions and explicit generation checkpoints.
- **Boundaries**: Neither named target contacted; no rotations, config changes,
  commits, push, deployment or activation. F09 and unrelated Task 18 incident
  credential-owner confirmations remain separate. C06 budgets, actual Docker
  runtime, live TMDE, date mapping, fleet/external writers and released device
  evidence remain unknown/unexecuted. Task 22 late commit and numeric restore
  ABA limitations remain OPEN. Historical entries above retain their dates;
  the current gate sheet supersedes them for release decisions.

### 2026-10-04 - Shops integrated acceptance (Task 28)
- **Status**: Local verification and acceptance record; independent task/branch
  review and owner release gates remain separate. No commits, merge, push, named
  target migration, credential rotation, deployment or activation performed.
- **Description**: Added percentile/query/pool/lock/payload/allocation measurements
  at the approved overview and full PMCS workloads, plus concurrent message
  writes. Fixed the confirmed F22 username-cancellation propagation gap and
  verified retained post-commit state. Exact results, provenance, inherited skips
  and remaining gates: `docs/testing/shops-server-remediation-acceptance.md`.


- 2026-10-04: Consolidated final review fix wave I1–I6/M1–M5 implemented in the isolated server worktree; no commit/integration/deployment. Final source `843b5f28f2b970d4254a8c4a30f83cbf5b53f93932722be37ff42fef50511f49`; full protected matrix 303/1002 PASS, host 619/1308 PASS (3 inherited skips), physical race 39/169 PASS. Only 020 reverse/pin changed among reviewed migrations; 32 PMCS files remain HEAD-identical. [Independent scoped local re-review completed](../../.superpowers/sdd/2026-10-03-shops-server-release-remediation/final-fix-scoped-review.md): all I1–I6/M1–M5 addressed, no new Critical/Important breakage confirmed; all owner/live/Docker/parser/device/credential gates remain. Current record: `docs/testing/shops-server-remediation-acceptance.md`.
