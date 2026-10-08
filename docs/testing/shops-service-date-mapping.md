# Service calendar date mapping gate

Date: 2026-09-26. Status: **BLOCKED; no mapping approved or implemented.**

S1 currently adds only independent strict Gregorian date parsers (years 0001–9999) and repairs mobile completion timestamp clearing. Neither parser is connected to legacy serialization or storage. Completion and audit timestamps remain instants. `service_dates` remains false.

## Evidence required from the deployment owner

- Deployed database revision and actual `equipment_services.service_date` column type, nullability and defaults, plus relevant triggers and constraints.
- Database, connection and writer session `TimeZone` values, including connection pool overrides and every active server version.
- Sanitized representative stored values and exact legacy request/response strings from supported released clients and active server versions. Include null, midnight, non-midnight, positive/negative offsets, UTC strings, leap days and dates around daylight-saving changes.
- Provenance linking each sample to its writer and schema/session settings. Do not infer the creator timezone from a viewer timezone.
- Verified local Drift representation and historical migration behavior for each supported local schema before selecting any local backfill.
- Owner-approved date-to-legacy encoding for new calendar dates, including how old clients parse and subsequently edit them.

Repository migration declarations and generated models are historical source evidence only. They do not establish deployed type, session settings or actual data interpretation. No live database was inspected for this task.

## Decisions to record after evidence arrives

For each verified legacy shape, record its original value, intended calendar components, exact legacy-to-date rule, reverse encoding, null handling and ambiguity policy. Identify which original timestamps must remain byte-for-byte unchanged on unrelated edits. Obtain an explicit owner decision for values whose intended calendar date cannot be established; never silently shift or guess them.

Both directions remain **undecided**. SQL casts, UTC truncation, viewer conversion, creator-zone assumptions and local backfills are not authorized substitutes for this evidence.

## Gates before implementation and activation

1. Approve the evidence and reversible mapping, including ambiguous-row reporting and rollback.
2. Establish F1 disposable baseline and supported released-client fixtures.
3. Implement additive storage, versioned adapters and transactional dual writes; allocate migrations only against that verified baseline. Decide verified compatibility trigger versus disabled capability while old writers remain.
4. Verify original legacy timestamps on unrelated edits, old/new cross-writes, old binary writes against expanded schema, nulls, leap/DST cases, migration preservation and malformed input rejection.
5. Generate Jet/Drift artifacts through configured tools, run historical upgrades and hidden PMCS/FK checks, and prove every serving writer supports the mapping before capability activation.

Until these gates pass, no scheduled-date column, backfill, database migration, versioned endpoint, legacy adapter change or service-date capability activation is part of S1 delivery.

## E3 evidence refresh — 2026-09-26

At server source `d50a691fe59174f4e3e5d495a83746adafcc7d9e` and mobile
source `97f2b5317b6d8bfe62f346de90f7c4af6323c2fd`, both mapping directions
remain undecided. No additional owner schema/session/writer samples or released
artifacts were supplied. No scheduled-date storage, backfill, versioned date
adapter, server migration, trigger or index was added during E3.

Fresh safe Go unit/race checks include the standalone parser and pass; fresh
Flutter acceptance/full suites include `shop_service_date_test.dart` and
`local completion persists, omitted copy retains, explicit null reopens` against
real Drift and pass. That proves parser and completion-null behavior only.
`service_dates` and `service_reads` remain false. S2 independent calendar/alerts/
history reads and S3 resolver/UI consumers remain deferred and unverified.

The isolated server race wrapper exited 1 before provisioning because the approved
baseline/migration boundary is absent. Physical fresh/populated upgrades,
legacy-to-date/date-to-legacy preservation, ambiguous-row handling, old/new
cross-writes and old-server expanded-schema rollback remain BLOCKED. The full
E3 coverage map is in the mobile remediation verification report.

## Server remediation checkpoint — 2026-10-04

Mapping remains **BLOCKED** with both capabilities false. This server-only task
adds no date conversion, backfill, or client work. Historical E3 fixture/setup
failures above describe that checkpoint; the current protected wrapper now
rehearses 019–023 with tagged generation, which does not supply any live mapping
or released-parser/device evidence. Owner target/session/writer samples and both
directions of the mapping still require explicit approval. See the
[release gate sheet](shops-server-remediation-release.md).
