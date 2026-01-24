# Analytics Search + PMCS Download Tracking Design Document

**Date**: 2026-01-19  
**Author**: System Design  
**Status**: Design Phase  
**Feature**: Analytics counters for item searches and PMCS manual downloads

---

## Table of Contents
1. [Executive Summary](#executive-summary)
2. [Current Implementation Analysis](#current-implementation-analysis)
3. [Requirements](#requirements)
4. [Proposed Solution](#proposed-solution)
5. [Data Model and Migration](#data-model-and-migration)
6. [Application Flow](#application-flow)
7. [Implementation Plan](#implementation-plan)
8. [Testing Strategy](#testing-strategy)
9. [Risks and Tradeoffs](#risks-and-tradeoffs)
10. [Decisions](#decisions)

---

## Executive Summary

Introduce a single analytics counter table that tracks:
1. Successful short item searches by NIIN or part.
2. Successful PMCS manual downloads by file name without extension (key) with formatted equipment name as label.

Counts are incremented atomically via `INSERT ... ON CONFLICT DO UPDATE` to avoid race conditions. No new endpoints are required at this time. The tracking is invoked from existing services after successful results are returned.

---

## Current Implementation Analysis

### Item Search
- Short item lookups are implemented via:
  - `api/controller/item_query_controller.go`
  - `api/service/item_short_service_impl.go`
  - `api/repository/item_query_repository_impl.go`
- Short item search returns `model.NiinLookup`.

### PMCS Download
- PMCS document listings and downloads are implemented via:
  - `api/controller/library_controller.go`
  - `api/service/library_service_impl.go`
- Downloads are performed through `GenerateDownloadURL`, which returns a SAS URL for a blob path.

### Database Access
- Database access uses Jet with generated types in `.gen/miltech_ng/public`.
- Migrations live in `migrations/` with plain SQL files.

---

## Requirements

### Functional
1. When a user performs a successful short item search by NIIN, increment the count for that NIIN.
2. When a user performs a successful short item search by part, increment the count for each NIIN returned.
3. When a user successfully requests a PMCS manual download, increment the count for the file name without extension and store the formatted equipment name as the label.
4. Support adding new analytics counters without additional tables.

### Non-Functional
- Updates must be atomic and concurrency-safe.
- Tracking should never block the main request path (fail-safe behavior).
- Counts are global (not user- or org-scoped).
- Counts are a rolling tally with no resets.
- No new endpoints or API responses are required for Phase 1.

---

## Proposed Solution

### Overview
Add a single counter table (`analytics_event_counters`) and an `AnalyticsService` that exposes:
- `IncrementItemSearchSuccess(niin string)`
- `IncrementPMCSManualDownload(formattedEquipmentName string)`
- `IncrementCounter(eventType string, entityKey string, entityLabel string)`

### Architecture
```
Controller -> Service -> Repository
                  |
                  +-> AnalyticsService -> AnalyticsRepository -> DB
```

Calls are added only after successful queries and before responses are returned.

---

## Data Model and Migration

### Table: analytics_event_counters

```sql
CREATE TABLE analytics_event_counters (
    id VARCHAR(36) PRIMARY KEY, -- UUID, consistent with existing pattern
    event_type TEXT NOT NULL, -- e.g., "item_search_success", "pmcs_manual_download"
    entity_key TEXT NOT NULL, -- NIIN or formatted equipment name
    entity_label TEXT NULL, -- optional display label
    count BIGINT NOT NULL DEFAULT 1,
    last_seen_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    CONSTRAINT chk_count_non_negative CHECK (count >= 0)
);

CREATE UNIQUE INDEX uq_analytics_event_key
    ON analytics_event_counters (event_type, entity_key);

CREATE INDEX idx_analytics_event_type
    ON analytics_event_counters (event_type);

CREATE INDEX idx_analytics_event_count
    ON analytics_event_counters (event_type, count DESC);
```

### Notes
- `event_type` enables reuse for future counters without schema changes.
- `entity_key` stores the canonical lookup key used for aggregation.
-- `entity_label` can store a human-friendly label if different from `entity_key`.

### Upsert Pattern (Jet / SQL)
```
INSERT INTO analytics_event_counters (id, event_type, entity_key, entity_label, count)
VALUES ($1, $2, $3, $4, 1)
ON CONFLICT (event_type, entity_key)
DO UPDATE SET
  count = analytics_event_counters.count + 1,
  last_seen_at = CURRENT_TIMESTAMP,
  entity_label = COALESCE(EXCLUDED.entity_label, analytics_event_counters.entity_label);
```

---

## Application Flow

### 1. Item Search Success (NIIN)
Trigger points (after successful lookup):
- `ItemShortServiceImpl.FindShortByNiin` when `model.NiinLookup` is returned.
- `ItemShortServiceImpl.FindShortByPart` when results contain NIINs.

Normalization:
- Trim whitespace, ensure upper-case.
- Use the NIIN returned from the database rather than user input when available.
- For part search results, increment once per unique NIIN per request.

Event:
- `event_type = "item_search_success"`
- `entity_key = "<NIIN>"`
- `entity_label = "<NIIN>"`

### 2. PMCS Manual Download
Trigger point:
- `LibraryServiceImpl.GenerateDownloadURL` after SAS URL generation succeeds.

Deriving equipment name:
- For PMCS paths: `pmcs/<vehicle_name>/<file>.pdf`
- Use `<file>` without extension as the key, with underscores removed and the words "CHECKLIST" and "PACKET" stripped if present.
- Use `formatDisplayName(<vehicle_name>)` as the label.

Event:
- `event_type = "pmcs_manual_download"`
- `entity_key = "<display_name>"`
- `entity_label = "<display_name>"`

---

## Implementation Plan

1. Database migration has been applied and Jet models generated.
2. Add `AnalyticsRepository` with `IncrementCounter` using Jet upsert.
3. Add `AnalyticsService` to wrap repository and provide semantic methods.
4. Wire `AnalyticsService` into:
   - `ItemShortServiceImpl` (NIIN + part searches)
   - `LibraryServiceImpl` (PMCS downloads)
5. Ensure analytics errors are logged but do not fail requests.
6. Add unit tests for repository upsert logic and service calls.

---

## Testing Strategy

- **Repository tests**: verify upsert increments count and updates timestamps.
- **Service tests**: ensure correct event_type/entity_key mapping.
- **Integration tests**: simulate successful NIIN/part search and PMCS download flows and confirm counters increment.

---

## Risks and Tradeoffs

- **Race conditions**: mitigated with `ON CONFLICT DO UPDATE`.
- **Incorrect key normalization**: needs consistent handling to avoid duplicates.
- **Partial tracking**: if a search fails, no event is recorded (intended).

---

## Decisions

1. Track only successful short searches by NIIN and part.
2. Counts are global.
3. PMCS equipment name uses formatted display name.
4. PMCS downloads only (no BII tracking).
5. Counts are a rolling tally with no resets.
