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

### 2026-09-28 - Notification item nickname and unit of measure
- **Status**: Completed (isolated integration suite not yet run)
- **Description**: Migration 017 adds nullable `nickname` / `unit_of_measure` to `shop_notification_items`. Atomic save, legacy item add and the maintenance snapshot read and write them; an omitted field means "unchanged" so released clients stay compatible. Recreated `tools/jetregen` (json-tagged jet generation).
- **Notes**: Manual SQL in `docs/migrations/shop_notification_item_nickname_uom_migration.md`. Deploy before the matching mobile release. Applied to the 192.168.20.70 `miltech_ng` dev database on 2026-09-28.

### 2025-01-30 - Project Memory System Initialized
- **Status**: Completed
- **Description**: Set up project memory infrastructure in docs/project_notes/
- **Notes**: Created bugs.md, decisions.md, key_facts.md, and issues.md

