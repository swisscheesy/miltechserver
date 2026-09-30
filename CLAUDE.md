# miltechserver: Agent Instructions

## Project

- **Language / framework**: Go 1.23, Gin
- **Database**: Postgres, queried and modelled with Jet (generated models in `.gen/miltech_ng/`)
- **External storage**: Azure Blob Storage
- **Authentication**: Firebase Auth

Follow the current application architecture. New code should fit existing functionality and project standards.
Use the golang-pro skill when working on Go code.
When working with the Postgres database, use the Database Optimizer and Supabase Postgres Best Practices skills.

## Commands

```bash
go run .                        # start API on :8080; .env is only loaded when DEBUG=true
go test ./api/...               # unit tests (no database)
scripts/test-shops-isolated.sh -v ./tests/shops ./tests/equipment_services   # integration tests against a throwaway Postgres
JET_DSN="postgresql://postgres:<password>@<host>:5432/miltech_ng?sslmode=disable" go run ./tools/jetregen   # regenerate .gen/
```

- Integration tests need a disposable database: `TEST_DATABASE_URL` (loopback host, database named `miltech_test_*`) plus a 32-byte hex `TEST_DATABASE_MARKER`, both checked by `tests/testutil`. The isolation script provides both, but only accepts `./tests/shops` and `./tests/equipment_services`.
- `tests/user_pmcs` and `tests/sb_700_20` fall back to loading a dotenv file for their database.
- Running several `tests/` packages in one command requires `go test -p 1`: packages TRUNCATE the same tables and race otherwise.

## Architecture

- `main.go`: builds the Gin engine, regenerates `.gen/` (see Gotchas), wires routes via `route.Setup`
- `bootstrap/`: env loading and `Application{Db, FireAuth, BlobClient}`
- `api/<feature>/`: one package per feature, layered `route.go` → `handler.go` → `service.go`/`service_impl.go` → `repository.go`/`repository_impl.go` (Jet queries). Larger features such as `api/shops/` and `api/user_pmcs/` nest sub-packages with a `shared/` package.
- `api/middleware/`: Firebase authentication, optional auth, rate limiting, error handling, logging
- `api/route/route.go`: central route registration
- `tests/<feature>/`: integration tests as separate packages; `tests/testutil` holds the disposable-DB check and fake auth middleware
- `migrations/`: numbered `NNN_create_*.sql` / `NNN_rollback_*.sql` pairs; migration write-ups live in `docs/migrations/`

## Gotchas

- **Never run the plain `jet` CLI.** It omits `json:"snake_case"` tags, and API responses marshal `.gen` models directly, so every response key would change (`shop_id` → `ShopID`). It also wipes `.gen/`, most of which is gitignored and cannot be restored from git. Use `go run ./tools/jetregen`.
- After regenerating, `git status .gen` must show no changes to the tracked `user_pmcs_*` files.
- `go run .` also regenerates `.gen/` (`generateSchema` in `main.go`) from whatever database the env points at, on port 5432. Starting the server against a different schema rewrites the models.

## Project Memory

Check these before acting; add to them when you learn something new.

- `docs/project_notes/decisions.md`: existing architectural decisions. Don't contradict one silently.
- `docs/project_notes/bugs.md`: known bugs and fixes. Log new bugs and their solutions.
- `docs/project_notes/key_facts.md`: non-secret configuration, ports, URLs
- `docs/project_notes/issues.md`: work log
- Issue tracker: GitHub Issues for swisscheesy/miltechserver via `gh`. See `docs/agents/issue-tracker.md`.
- Domain docs: see `docs/agents/domain.md`. `CONTEXT.md` and `docs/adr/` do not exist yet; they are created when terms or decisions are resolved.

## Research and Tools

Tool names below are Claude Code's; other agents use their equivalent.

- **Documentation**: before using an unfamiliar or version-sensitive API, search `mcp__ref__ref_search_documentation` and read the results with `mcp__ref__ref_read_url`. Codebase > Docs > Training data, in order of truth.
- **Sequential thinking** (`mcp__sequential-thinking__sequentialthinking`): for feature design, non-trivial debugging, architecture, performance, security, and refactoring plans.
- **Git history**: before modifying a file, check its recent history (`mcp__git__git_log` with a file path, `mcp__git__git_show` for a commit). Search code with CodeGraph or grep.

## Questions Before Implementation

Ask when a decision is genuinely the user's to make: product behaviour, defaults, failure handling, migration of existing data, or scope. Don't ask about facts you can verify in the code; check them. When a sensible convention exists, use it and say so. Number questions so they can be answered in order.

Things worth clarifying:
- **Features**: expected user experience, configurability and defaults, integration with existing features
- **Errors**: what happens on failure, retries, logging that would help debugging
- **Performance**: constraints, data volume
- **Security**: authorization and data-exposure implications
- **Maintenance**: migration path for existing data, how it will be tested, which docs need updating

## Code Quality

Write code as if the person maintaining it is a violent psychopath who knows where you live:

- **NO CLEVER TRICKS**: clear, obvious code only
- **DESCRIPTIVE NAMING**: `processTextNodes()` not `ptn()` or `handleStuff()`
- **COMMENT THE WHY**: only explain why, never what
- **SINGLE RESPONSIBILITY**: each function does one thing
- **EXPLICIT ERROR HANDLING**: no silent failures
- **MEASURE THEN OPTIMIZE**: no premature optimization
- **SIMPLICITY FIRST**: remove everything non-essential
- **YAGNI**: apply to speculative requirements and premature abstraction, not to correctness, security, testing, maintainability, or explicitly requested product quality.

### Naming (Go)

- MixedCaps only; capitalisation decides export (`MaxRetries` exported, `maxRetries` not). No `SCREAMING_SNAKE_CASE`, including constants.
- Boolean prefixes: `is`, `has`, `can`, `should`
- No `I` prefix on interfaces. Interfaces describe contracts (`Service`, `Repository`); implementations live in `*_impl.go`.
- Files are `snake_case.go`; tests are `*_test.go`.

### Honest Technical Assessment

- If code has problems, explain the specific issues
- If an approach has limitations, quantify them
- If there are security risks, detail them clearly
- If performance will degrade, provide metrics
- If implementation is complex, justify why
- If you chose a suboptimal solution, explain the tradeoffs
- If you're uncertain, say so explicitly

Examples:
- "This will work for 1000 users but will break at 10,000 due to database bottleneck"
- "This fix addresses the symptom but not the root cause - we'll see this bug again"
- "I'm not certain this handles all edge cases - particularly around concurrent access"

### Context and Documentation

Preserve technical context; never delete important information. Keep code examples with line numbers, performance measurements, rationale for decisions, explanations of non-obvious patterns, and cross-references to issues. Remove only decoration, marketing language, redundant information, and clearly obsolete content.

## Security

1. NEVER store secrets in code or commits (`.env` and `fire_auth_key.json` are gitignored)
2. ALWAYS validate and sanitize ALL inputs; assume all user input is hostile
3. NO dynamic code execution with user data; build SQL only through Jet
4. IMPLEMENT rate limiting where appropriate (`api/middleware/rate_limiter.go`)
5. VALIDATE permissions on every request: Firebase auth in middleware, then resource-level authorization (e.g. shop membership) in the service
6. ENCRYPT sensitive data at rest and in transit
7. LOG security events for monitoring
8. FAIL securely: errors must not leak internal information

## Performance

1. **Measure first**: `pprof`, benchmarks, or `EXPLAIN ANALYZE` for queries
2. **Analyze**: use sequential thinking to understand the issue
3. **Implement**: follow established patterns
4. **Verify**: measure again to confirm the improvement
5. **Document**: record the optimization and its impact

## Workflow

Plan in concrete steps, not timeframes ("Step 1: add repository method", not "Week 1").

1. **Understand**: read the request fully, identify the task type (feature/bug/refactor/debug) and constraints.
2. **Research current state**: recent commits, related code, existing similar implementations, project memory.
3. **Verify understanding**: ask the questions above where needed, confirm scope, identify edge cases.
4. **Research best practices**: read relevant documentation; note security and performance considerations.
5. **Plan**: break the work into concrete steps.
6. **Execute**: follow the plan; test the edge cases identified earlier.
7. **Validate**: build and run relevant tests; verify every requirement; check for unintended side effects.
8. **Complete**: summarize what was done and any follow-ups. Commit only when asked.

## Commits

Follow Conventional Commits v1.0.0:

```
<type>[optional scope]: <description>

[optional body]

[optional footer(s)]
```

**Types**: `feat` (MINOR), `fix` (PATCH), `refactor`, `perf`, `docs`, `test`, `build`, `ci`, `chore`, `style`

**Breaking changes**: `feat(api)!: remove deprecated endpoints`, or a `BREAKING CHANGE:` footer

**Example**:
```
fix(auth): prevent race condition in token refresh

Add mutex to ensure only one token refresh happens at a time.
Previous implementation could cause multiple simultaneous refreshes
under high load.

Fixes: #123
```

**Requirements**:
- One logical change per commit
- Run tests before committing
- Include context for future developers
- Reference issue numbers when applicable
- Never mention "Claude", "AI", "ANTHROPIC", "DEVELOPER TOOLS" in commits

## Core Principles

1. Research before coding; the codebase is the source of truth
2. Clarify genuine ambiguity instead of assuming
3. Write clear, obvious code without clever tricks
4. Provide honest assessment of technical decisions
5. Preserve context; don't delete valuable information
6. Make atomic commits with clear messages
7. Document why decisions were made, not just what was done
8. Test thoroughly before declaring completion
9. Handle all errors explicitly
10. Treat user data as sacred
