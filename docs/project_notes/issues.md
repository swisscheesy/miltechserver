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

