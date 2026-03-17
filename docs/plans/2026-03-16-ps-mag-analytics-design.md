# PS Magazine Download Analytics Design

**Date**: 2026-03-16
**Status**: Approved
**Feature**: Track PS Magazine issue downloads in `analytics_event_counters`

---

## Summary

Extend the existing analytics system to record a counter each time a PS Magazine issue SAS URL is generated (i.e., a user downloads an issue). Follows the same pattern used for PMCS manual downloads.

---

## Event Shape

| Field | Value | Example |
|---|---|---|
| `event_type` | `ps_mag_download` | `ps_mag_download` |
| `entity_key` | Uppercased raw filename | `PS_MAGAZINE_ISSUE_004_SEPTEMBER_1951.PDF` |
| `entity_label` | Cleaned, uppercased display label | `ISSUE 004 SEPTEMBER 1951` |

**Label derivation** from filename `PS_Magazine_Issue_004_September_1951.pdf`:
1. Strip `PS_Magazine_` prefix → `Issue_004_September_1951.pdf`
2. Strip file extension → `Issue_004_September_1951`
3. Replace `_` with space → `Issue 004 September 1951`
4. `normalizeAnalyticsKey` (uppercase + trim) → `ISSUE 004 SEPTEMBER 1951`

**Key derivation**: `normalizeAnalyticsKey(filename)` → `PS_MAGAZINE_ISSUE_004_SEPTEMBER_1951.PDF`

---

## Architecture

No schema changes required. `analytics_event_counters` already supports arbitrary `event_type` values.

```
GET /library/ps-mag/download
  → ps_mag.Handler.generateDownloadURL
  → ps_mag.ServiceImpl.GenerateDownloadURL
      [blob exists + SAS URL generated successfully]
  → ps_mag.ServiceImpl.trackPSMagDownload(blobPath)
  → analytics.ServiceImpl.IncrementPSMagDownload(filename)
  → analytics.RepositoryImpl.IncrementCounter(...)
  → analytics_event_counters (upsert)
```

Analytics errors are logged as warnings and never block the response.

---

## Files Changed

### `api/analytics/service.go`
- Add `IncrementPSMagDownload(filename string) error` to `Service` interface.

### `api/analytics/service_impl.go`
- Add constant `analyticsEventPSMagDownload = "ps_mag_download"`.
- Add `formatPSMagLabel(filename string) string` helper.
- Add `IncrementPSMagDownload` method on `ServiceImpl`.

### `api/library/ps_mag/service_impl.go`
- Add `analytics analytics.Service` field to `ServiceImpl`.
- Update `NewService` signature: add `analyticsService analytics.Service` parameter.
- Add `trackPSMagDownload(blobPath string) error` helper.
- Call `trackPSMagDownload` in `GenerateDownloadURL` after SAS URL is obtained.

### `api/library/ps_mag/route.go`
- Update `RegisterHandlers` signature: add `analyticsService analytics.Service` parameter.
- Pass `analyticsService` through to `NewService`.

### `api/library/route.go`
- Pass `deps.Analytics` as the new argument to `ps_mag.RegisterHandlers`.

---

## Tests

### `api/analytics/service_impl_test.go` (new)
- `TestFormatPSMagLabel` — table-driven: standard filename, missing prefix, empty string.
- `TestIncrementPSMagDownload_CallsRepo` — verifies correct event_type, key, label.
- `TestIncrementPSMagDownload_EmptyFilename` — verifies no-op on empty input.

### `api/analytics/service_impl_test.go` additions to `api/library/ps_mag/service_impl_test.go`
- Update `TestGenerateDownloadURLValidation`: `NewService(nil, nil)` → `NewService(nil, nil, nil)`.
- Add `analyticsStub` satisfying `analytics.Service`.
- `TestTrackPSMagDownload_CallsAnalytics` — verifies filename passed to analytics stub.
- `TestTrackPSMagDownload_NilAnalytics` — verifies no panic when analytics is nil.
- `TestGenerateDownloadURL_AnalyticsError_DoesNotFail` — analytics error does not propagate.

---

## Decisions

1. Label and key are both uppercased via `normalizeAnalyticsKey` for consistency with PMCS.
2. Analytics errors are non-fatal (warn log only) — tracking never blocks downloads.
3. No DB migration required — `analytics_event_counters` supports arbitrary event types.
4. Tracking fires after SAS URL generation succeeds, not on failed requests.
