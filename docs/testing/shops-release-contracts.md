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
