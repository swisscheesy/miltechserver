# Shops released-client compatibility evidence

## Status: acceptance blocked (2026-09-26)

No approved release artifacts or captured fixtures were supplied. The current
mobile checkout is not evidence of a deployed release contract.

- Currently deployed Android version/build: **unknown, owner evidence required**.
- Currently deployed iOS version/build: **unknown, owner evidence required**.
- Oldest affected supported Android and iOS version/build: **unknown**.
- Release source commit, artifact identity/checksum and provenance: **unavailable**.
- Actual released parser behavior: **unverified**.
- Captured request/response fixtures: **unavailable**.

For each deployed and oldest affected released client, record platform,
version/build, source commit and signed artifact provenance. Trace its actual
request serialization and response parsing. Supply sanitized fixtures covering
notification updates, equipment updates/usage, audit history (empty and populated),
null versus missing fields, successful no-op updates, and request failures.
Record endpoint, method, status, envelope, parser result and artifact origin for
each fixture; use synthetic identifiers and omit credentials and personal data.

Compatibility acceptance requires running the affected released parsers against
those fixtures. Current-source unit tests cannot close this gate. This document
makes no claim that released clients accept any remediation response changes.

## E3 verification checkpoint — 2026-09-26

Tested server source: `d50a691fe59174f4e3e5d495a83746adafcc7d9e`;
mobile source: `97f2b5317b6d8bfe62f346de90f7c4af6323c2fd`. These are
local remediation commits, not deployed release identities.

- Safe Go unit and race commands over `./api/shops/...`,
  `./api/equipment_services/...`, `./api/response/...`, `./api/middleware/...`,
  and `./tests/testutil/...` with `-count=1` exited 0. Vet over the same packages
  exited 0. Actual service logging direct-file race test exited 0.
- `go test ./... -run '^$' -exec /usr/bin/true` exited 0 **compile only**;
  no test binary or database TestMain executed.
- `scripts/test-shops-isolated.sh ./... -count=1` exited 1 on its package
  allowlist. `scripts/test-shops-isolated.sh ./tests/shops ./tests/equipment_services
  -race -count=1` exited 1 on the missing approved schema/migration boundary.
  Both refused before provisioning; neither is passing integration evidence.
- All commands used `rtk proxy`; Go used
  `GOCACHE=/private/tmp/miltechserver-f1-go-cache`. Five wrapper shell-block
  regressions passed; full disposable provisioning remains unverified.
- Flutter focused acceptance passed 1,222 tests; full suite passed 3,716,
  both exit 0. Full analyzer exited 1 with the 19 unrelated existing diagnostics.
  These current-source host tests are not released-client fixtures.
- `TestContractCapabilitiesDisabled` passes: `typed_errors`,
  `atomic_notification_save`, `service_dates`, `service_reads`, `message_sync`
  remain false. New message production reads explicitly fail closed.

Still **BLOCKED**: no-header released-parser checks for shapes, defaults, date
reads/writes, errors and pagination; old notification separate writes; alternate
old/new writes on the same notification/service/message/list; old server binary
against expanded schema; new-server rollback with capabilities disabled while
retaining data. No server schema expansion was fabricated or applied. Unit SQL
drivers cannot prove physical durability, FK behavior, transaction ordering or
old-writer compatibility. S2/S3 complete service datasets and consumers remain
deferred behind S1 mapping approval.

The mobile report `docs/audits/2026-09-26-shops-remediation-verification.md`
contains the finding-by-finding map, command results, native limits and rollout
checklist. F1 schema approval, released artifacts, owner-coordinated credential
rotation, every-serving-writer evidence and native acceptance remain prerequisites.
No capability activation, live/shared DB access, merge, push or deployment occurred.
