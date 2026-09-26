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
