# Key Facts

This file stores project constants, configuration, and frequently-needed **non-sensitive** information for the miltechserver project.

## Security Warning

**NEVER store passwords, API keys, or sensitive credentials in this file.** This file is committed to version control.

**Safe to store:** Database hostnames, ports, project identifiers, API endpoint URLs, service account emails, environment names
**Store secrets in:** `.env` files, Azure Key Vault, environment variables, CI/CD secrets

## Project Stack

**Core Technologies:**
- Language: Go
- Web Framework: Gin
- Database: PostgreSQL
- ORM/Query Builder: Jet (with model generation)
- External Storage: Azure Blob Storage
- Authentication: Firebase Auth

## Database Configuration

**PostgreSQL:**
- Query Builder: Jet
- Model Generation: Jet (auto-generated from schema)
- Connection: Via environment variables

## API Configuration

**Endpoints:**
- Local Development: `http://localhost:8080` (typical Gin default)

## Local Development

**Services:**
- API Server: Gin (default port 8080)
- Database: PostgreSQL

## Project Structure

**Key Directories:**
- `api/` - API layer (controllers, routes, services, repositories)
- `bootstrap/` - Application initialization
- `docs/` - Documentation including project notes

## Important URLs

**Documentation:**
- Gin Framework: https://gin-gonic.com/docs/
- Jet Query Builder: https://github.com/go-jet/jet
- Firebase Auth Go SDK: https://firebase.google.com/docs/auth/admin/verify-id-tokens

<!-- Add more key facts below as you discover them -->

## Shop invitations (2026-10-03)

- Current members may generate unrestricted active invitations; only current administrators may revoke or delete them. Write authorization is rechecked under the persisted Shop lock.
- `max_uses` and `expires_at` must be omitted or JSON null. Every non-null value is rejected with HTTP 400 before a write, including zero and empty string; no expiry or usage-limit schema is implied.
- Codes retain the existing eight uppercase hexadecimal characters. Practical guessing resistance remains a separate measured release gate; this change does not add a guessing budget.
- A duplicate claim preserves the existing membership role and returns the existing already-member error. Revocation/deletion committed before admission prevents a new membership; admission committed first is retained.
- Member removal is not a ban: a removed user can rejoin using a still-active code. Final-member departure, last-admin rules, and explicit Shop deletion policies are tracked separately.

## Generated Jet Models

- Regenerate `.gen/` with `JET_DSN="postgresql://postgres:<password>@<host>:5432/miltech_ng?sslmode=disable" go run ./tools/jetregen` (2026-09-28).
- Never use the plain `jet` CLI: it omits `json:"snake_case"` tags, and API responses marshal these models directly, so every response key would change (`shop_id` -> `ShopID`). It also wipes `.gen/` first, and most of `.gen/` is gitignored, so git cannot restore it.
- After regenerating, `git status .gen` must show no changes to the 32 tracked `user_pmcs_*` files.



## Message Sync (2026-09-29)

- Regeneration is mandatory after authorized migration/data-repair steps. The explicit nine-field message DTO and legacy SQL projections preserve keys/nullability both before and after 018; never hand-edit generated fields or suppress `insertion_number` tags. The trigger remains the sole insertion-number allocator.
- Startup and `tools/jetregen` share `internal/jetgen`. CLI requires `JET_DSN`; optional `JET_OUTPUT_DIR`/`JET_SCHEMA` default to `.gen`/`public`. Output is always `<output>/miltech_ng/<schema>` while its manifest preserves the real database/address/port/role and catalog digest. Startup uses the application's actual database pool and requires the compiled `public` schema.
- Output needs write access. A cross-process lock serializes staging/publication; publication preserves a complete backup for rollback but its two directory renames are not one atomic exchange. Do not run unlocked builds concurrently with publication.
- Runtime regeneration cannot rebuild a binary. Legacy message schema compatibility is checked; broader migration/function/grant/build readiness remains a separate release gate. The disposable baseline lacks `LookupLinNiinMat`, preventing a full application build from only that generated fixture; the isolated generation probe covers the message DTO and generated model/table/view packages.
- Flag: `SHOPS_MESSAGE_SYNC_ENABLED` (strict `true`/`false`). Apply migration 018 before turning it on; enable only on a uniform new-binary fleet.
- Live application: `scripts/apply-shops-message-sync-migration.sh` (pinned SHA-256 of 018; pre-018 schema pins are `UNPINNED` until the operator records them in `docs/testing/shops-database.md`).
- Sync response timestamps carry the DB session UTC offset, not necessarily `Z`.

## Current remediation release status (2026-10-04)

- Actual application role comes from `DB_USERNAME`; separate SELECT/INSERT/UPDATE
  checks must all pass under that effective role, including a real legacy write.
- New guarded forward runner: `scripts/apply-shops-remediation-migrations.sh`. Both
  target pin records are UNPINNED; named-target contact/application is unexecuted.
- Task 26 mandatory 020/023 startup validation precedes automatic tagged generation.
  Generic generation supports intermediate stages; the synthetic empty disposable
  TMDE ABI does not establish the unknown live view definition.
- Source/disposable verification is not deployment, rotation, device, container
  runtime or fleet approval. [Gate sheet](../testing/shops-server-remediation-release.md).
