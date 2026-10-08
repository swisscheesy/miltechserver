# Codebase Modernization Refactor — Design

**Date:** 2026-08-31
**Status:** Proposed
**Scope:** Planning only. No code changes are included in or authorized by this document.

## Context

Two Explore-agent audits surveyed the `api/` tree (24 domains, ~41.8k lines),
`bootstrap/`, `api/middleware/`, `api/route/`, `api/response/`,
`api/request/`, `tests/`, `migrations/`, and the project's own decision
record (`docs/project_notes/decisions.md`, 21 ADRs) to answer: what is
slowing changes down, and what's actually fine?

The headline finding: the router-wiring migration documented in ADR-005
through ADR-011 (colocated `route.go`/`handler.go`/`service.go`/
`repository.go` per domain, narrow per-domain `Dependencies{}` structs
instead of the full `bootstrap.Application`) **succeeded** and is followed
consistently across all ~20 domains. That migration is not touched by this
plan.

What ADR-005–011 didn't cover — response envelopes, transaction handling,
pagination parsing, test harness setup — was left to each domain's
discretion, and those areas diverged independently over time. This plan is
mostly about finishing that unfinished consolidation, plus deleting code
that's already dead, plus splitting a handful of files that outgrew their
original shape. It does not propose any new architecture.

`user_pmcs`, the most recently and actively developed domain (ADR-021,
2026-08-16), has drifted furthest — it has its own response envelope
(ETags, strict JSON, precondition handling), its own rate limiter, and its
own transaction helper. The audits found this system is **more capable**
than what the rest of the codebase uses, not worse — so this plan explicitly
excludes `user_pmcs` from response-envelope consolidation rather than
downgrading it to match everyone else.

## Goals

- Delete confirmed dead code.
- Collapse duplicated boilerplate (pagination parsing, response
  construction, transaction handling, test harness setup) into shared
  helpers, adopted incrementally per domain.
- Split the handful of files that mix unrelated concerns, along seams the
  audits already identified.
- Thread `context.Context` through the service layer where the repository
  layer underneath already accepts it.
- Preserve all existing behavior and public API contracts. No pass in this
  plan changes an HTTP response shape, status code, or database schema
  unless explicitly called out as out of scope for a future migration task.

## Non-goals

- No framework, dependency, or major version upgrades.
- No database schema changes.
- No changes to `user_pmcs`'s response/rate-limit/transaction systems —
  they're excluded, not deprecated.
- No new features or behavior changes of any kind.

## Validation philosophy

ADR-021 rehearsed a schema rename against `miltech_ng_test`, compared row
fingerprints and catalog definitions before and after, and applied to
`miltech_ng` only once rehearsal succeeded. This plan applies the same
discipline to code-level changes: every pass states current behavior, the
structural change, and a concrete check that proves behavior didn't move.
Where a pass changes response bodies (Pass 5), a golden-fixture snapshot is
captured *before* the change so the diff has something real to compare
against, not memory.

Global validation available to every pass: `go build ./...`, `go vet
./...`, and `go test -p 1 ./...` (the `-p 1` requirement is pre-existing —
`tests/shops` and `tests/pmcs_sbs_progress` share a mutable test database
and race under parallel package execution).

## Passes

Each pass is scoped to be one reviewable PR (Pass 5 and Pass 7 are
explicitly multiple PRs — see notes). Ordered lowest-risk-first; passes 1–3
have no dependencies on each other or on later passes.

### Pass 1 — Delete confirmed dead code

**Correction (2026-08-31, found during execution):** This pass originally
also named `api/middleware/error_handler.go` and `api/middleware/
rate_limiter.go` as dead. Both are confirmed live on re-verification:
`error_handler.go`'s `ErrorHandler` is registered globally at `api/route/
route.go:111` (`v1Route.Use(middleware.ErrorHandler)`), and `rate_limiter.go`'s
`RateLimiter()` is called directly in 5 route registrations
(`api/docs_equipment/route.go` ×2, `api/library/route.go`, `api/library/
ps_mag/route.go`, `api/library/pmcs_sbs/route.go` ×2), returning HTTP 429
when tripped. The original audit correctly found `RateLimiter` wasn't wired
into the *global* middleware chain in `Setup()`, but missed the 5 direct
per-route call sites, and took `error_handler.go`'s self-doubting TODO
comment as evidence of inactivity without confirming whether it was
registered. This pass is corrected below to delete only the file that is
genuinely dead. Neither `error_handler.go` nor `rate_limiter.go` is part of
this pass, or any other pass in this document, going forward.

**Current behavior:** `api/service/auth_service.go` compiles and is part of
the binary but has zero callers anywhere in the repo (`AuthService`/
`NewAuthService` — confirmed via repo-wide grep); its `Login` method is
entirely commented out and its other seven methods are empty bodies. Real
authentication runs through Firebase in `api/middleware/authentication.go`.

**Structural improvement:** Delete `api/service/auth_service.go`. Remove any
now-dangling imports.

**Validation:** `go build ./...` succeeds; `go vet ./...` clean; `go test
-p 1 ./...` passes with identical results to the pre-change baseline; grep
confirms zero remaining references to `AuthService` anywhere in the repo.

**Public API impact:** None. Nothing reachable through HTTP touches this
file today.

---

### Pass 2 — Fix decisions.md numbering and the `.gen` gitignore mismatch

**Current behavior:** `docs/project_notes/decisions.md` has two entries both
titled `### ADR-011` ("Complete Shops HTTP Adapter Migration to Bounded
Subdomains", 2026-02-18, and "Shops Performance Optimization Refactor",
2026-02-01). `.gitignore` contains `*_gen.go` and `miltech_ng/`, neither of
which matches the actual generated-code path `.gen/miltech_ng/...` — the
32 files under `.gen/miltech_ng/public/{model,table,view}/` are tracked in
git, last touched by commit `930c92e`.

**Structural improvement:** Renumber the earlier-dated ADR-011 entry
("Shops Performance Optimization Refactor", 2026-02-01) forward in sequence
so numbering is chronological and unique — insert it as ADR-011 and bump
"Complete Shops HTTP Adapter Migration" to the next free number, or append
both at the end with a numbering note; exact renumbering mechanics are a
small editorial decision made at implementation time, not upfront in this
spec.

This pass does **not** touch `.gitignore` or `.gen/`. The mismatched
`*_gen.go`/`miltech_ng/` patterns are left exactly as they are here — fixing
them is coupled to the tracked-vs-generated-at-build decision, which is a
separate, explicit out-of-scope item below. Editing the gitignore pattern
without that decision first would either silently start ignoring files
still meant to be tracked, or leave a no-op edit that implies a decision was
made when it wasn't.

**Validation:** `grep -c "^### ADR-0"` on each number shows no duplicates;
document renders/reads correctly; `git diff` for this pass touches only
`docs/project_notes/decisions.md`.

**Public API impact:** None — documentation only.

---

### Pass 3 — Remove the dead DSN read in `tests/item_lookup/main_test.go`

**Current behavior:** `tests/item_lookup/main_test.go:17-25` reads
`TEST_DATABASE_URL` from the environment into `dsn`, fatally errors if
unset, and then **ignores `dsn` entirely**, opening a hardcoded
`postgres://postgres:potato123@192.168.20.70/miltech_ng_test?sslmode=disable`
connection string instead. Per confirmation, `192.168.20.70` is the correct
and intentional test database host for both `miltech_ng` and
`miltech_ng_test` — this isn't a misconfiguration, the dead code is the
unused env-var read.

**Structural improvement:** Delete the `os.Getenv("TEST_DATABASE_URL")`
read and its fatal-check, keep the hardcoded connection string as-is,
matching every other `tests/<domain>/main_test.go` file's existing
convention.

**Validation:** `go test -p 1 ./tests/item_lookup/...` passes against the
real test database with output identical to the pre-change run.

**Public API impact:** None — test-only code.

---

### Pass 4 — Shared pagination helper, adopted per call site

**Current behavior:** The same page/pageSize parse-and-validate block
(`c.DefaultQuery("page", "1")` → `strconv.Atoi` → 400 on error or `page <
1`) is hand-copied in 15+ places, worst offender
`api/sb_700_20/handlers_apps.go` (9 times in one file). A working helper
already exists at `api/item_lookup/shared/pagination.go` but isn't reused
outside that domain.

**Structural improvement:** Promote the existing helper (or a thin
new package if its current signature is too `item_lookup`-specific to
generalize cleanly — determined during implementation, not this spec) to a
shared location reachable by all domains. Replace the copy-pasted blocks in
`sb_700_20`, `docs_equipment`, `eic`, `tmde`, and `library/ps_mag` with
calls to it, one domain per commit within this pass so a bad extraction is
easy to bisect.

**Validation:** A new table-driven test on the shared helper itself covers
valid, missing, zero, negative, and non-numeric page params. For each
touched domain, existing handler tests must produce identical status codes
and response bodies before and after — this is provable per domain via
`tests/<domain>/` where it exists, or via the colocated `route_test.go`
where it doesn't.

**Public API impact:** None if done correctly — this pass replaces
equivalent logic with equivalent logic. Any behavior change (different
error message text, different default page size) is a bug in this pass,
not an intended outcome.

---

### Pass 5 — Consolidate response construction, per domain

This is the largest pass and the one most likely to visibly change
something if done carelessly, since it touches every response body. **Split
into one PR per domain**, per your direction — recommended order starts
with `material_images` (highest ratio of raw `gin.H{}` usage: 10 raw calls,
zero use of the existing `StandardResponse` type) and proceeds through
`pmcs_sbs_progress`, `shops`, `tmde`, `item_lookup`, `eic`, and the
remaining domains that use any response pattern other than a clean,
consistent `StandardResponse`.

**`user_pmcs` is explicitly excluded from this pass, permanently, not just
deferred.** Its response system (`api/user_pmcs/shared/http.go`) — typed
`apiErrorResponse{Status, Message, Data, Error{Code, Details}}`, ETag/
precondition handling, strict JSON decoding that rejects unknown fields —
is materially more rigorous than `StandardResponse` and was a deliberate
build-out for that domain's concurrency and sync requirements. Pulling it
down to match the simpler envelope would be a functional regression
disguised as cleanup.

**Current behavior (per domain, to be confirmed at the start of each
domain's PR):** A mix of `response.StandardResponse{...}` struct literals,
raw `gin.H{}` literals, and in some domains (`material_images`) entirely
domain-specific response types (`response.ImageFlagResponse`,
`response.PaginatedImagesResponse`, `response.ImageVoteResponse`), with no
domain applying one convention consistently even within a single file
(`pmcs_sbs_progress/route.go` uses `StandardResponse` on some lines and
`gin.H{"message": ...}` on others).

**Structural improvement:** First collapse the three structurally-identical
response types in `api/response/` (`StandardResponse`, `NoItemFoundResponse`,
`ErrorResponse`) down to `StandardResponse` alone, since the other two add
no distinct shape. Add `response.OK(c *gin.Context, data any)` and
`response.Error(c *gin.Context, err error)` (or equivalent) helper
functions — today there is no function that actually writes a response,
every call site does `c.JSON(...)` by hand, which is a root cause of the
drift. Then, per domain, replace raw `gin.H{}` and inline struct literals
with calls to the new helpers, preserving each domain's current field
names, status codes, and error messages exactly.

**Validation:** Before each domain's PR, capture a golden fixture — actual
response bodies (status + JSON) for that domain's representative endpoints,
pulled from the existing integration tests in `tests/<domain>/` or by
running the handler directly. After the change, diff new output against
the golden fixture byte-for-byte on both success and error paths. A domain
is not done until this diff is empty (or every difference is an explicitly
approved, pre-agreed field-name fix — see "Recommended docs/specs" below
for how to handle domains whose current response shape has bugs worth
fixing along the way).

**Public API impact:** None intended — if a domain's current response
shape genuinely needs to change (e.g., a domain returning inconsistent
field names that a client already depends on), that's flagged as a
separate, explicit API-contract change proposed to you before that
domain's PR proceeds, not silently bundled into the consolidation.

---

### Pass 6 — Shared transaction helper

**Current behavior:** Manual `Begin()` / wrap-error / `defer Rollback()` /
`Commit()` boilerplate is repeated in 18 files. `api/user_pmcs/owned/
repository_impl.go` already has a working `withTx`-style callback pattern
(`func(tx *sql.Tx) (*MutationResult, error)`) plus a dedicated
`rollbackReadTransaction` helper.

**Structural improvement:** Lift the `user_pmcs/owned` pattern into a small
shared package (e.g. alongside `bootstrap/database.go`, or a new
`api/shared/db` — exact location decided at implementation time).
Migrate the other 17 files to use it, one domain per commit.

**Validation:** For each migrated method, the existing integration test
covering its commit path must still pass, and at least one test must force
an error mid-transaction and assert the resulting rollback leaves no
partial writes — this is the one place a shared helper could silently
change behavior if its callback signature doesn't preserve the original
per-call error-wrapping text that some existing tests may assert against,
so those assertions are checked explicitly, not just "does it still
compile."

**Public API impact:** None — internal repository implementation detail,
invisible above the service layer.

---

### Pass 7 — Split oversized files along existing seams

One file per PR; these are large diffs by line count even with zero
behavior change, so reviewability comes from keeping each PR to exactly one
file's reorganization.

**Current behavior:**
- `api/shops/aggregates/repository_impl.go` (1233 lines) mixes 5 unrelated
  aggregation concerns — vehicle notifications, recent changes, service
  history, shop snapshot/bootstrap, PMCS history — plus generic null-scan
  helpers (`timePtr`, `nullTimePtr`, `nullStringPtr`, `nullBoolPtr`,
  `nullInt32Ptr`, `errorsIsNoRows`, `placeholders`) that belong in a shared
  package, not this file.
- `api/user_pmcs/community/repository_impl.go` (909 lines) mixes voting
  logic and browse/discovery logic.
- `api/pmcs_sbs_progress/service_impl.go` (652 lines) mixes ~11 public
  service methods, ~11 validation helpers, and ~5 DTO-mapping helpers with
  no file separation, despite the domain already having a `types.go`
  precedent for splitting concerns out of the main file.
- `api/user_pmcs/owned/repository_impl.go` (1011 lines) has fewer distinct
  concerns but individual methods run 100–224 lines each with manual
  transaction/tree logic inlined — this file is addressed by Pass 6's
  transaction-helper extraction first; whether further splitting is needed
  is reassessed after Pass 6 lands, not decided now.

**Structural improvement:** Split each file along the seams the audit
already identified by function name and line range: `shops/aggregates` →
`repository_notifications.go`, `repository_snapshot.go`,
`repository_services.go`, `repository_pmcs_history.go`, plus moving the
generic scan helpers to `api/shops/shared/`; `pmcs_sbs_progress/
service_impl.go` → extract `validation.go` and `mapping.go` alongside the
existing `types.go`; `user_pmcs/community/repository_impl.go` → split
voting into its own file from browse/discovery. Same package, same
exported symbols — this is file reorganization, not a redesign.

**Validation:** `go build ./...` (a pure split cannot compile if a
reference was missed); full existing test suite for the touched domain
passes with no changes; `git diff --stat` reviewed manually to confirm the
diff is function bodies moving between files, with no logic edited
mid-move — any "while I'm in here" improvement belongs in a later, clearly
labeled pass, not folded into the split.

**Public API impact:** None — package-internal reorganization only.

---

### Pass 8 — Thread `context.Context` through the remaining service layer

**Current behavior:** Only 7 of 54 `service_impl.go` files accept `ctx
context.Context` in their public methods. The other 47 — including the
large `pmcs_sbs_progress/service_impl.go` — take `*bootstrap.User` and
other params with no context, breaking cancellation and tracing
propagation even in cases where the repository layer directly underneath
already accepts `ctx` (e.g. `shops/aggregates/repository_impl.go` does,
but the service calling it doesn't pass one through).

**Structural improvement:** Add `ctx context.Context` as the first
parameter on public service methods in the 47 files, threading from the
handler's `c.Request.Context()` down to whichever repository calls already
accept it. This pass does **not** extend context-acceptance into
repository methods that don't have it yet — that's explicitly deferred
(see "Out of scope").

**Validation:** Signature-only change, so `go build ./...` surfaces every
call site that needs updating. Full test suite must pass unchanged. This
pass can additionally *add* a genuinely new regression check per domain —
a test asserting a canceled context actually aborts an in-flight query —
which wasn't previously possible to test since `ctx` wasn't threaded.

**Public API impact:** None at the HTTP boundary — this only changes
internal Go function signatures between the handler and service layers.

---

### Pass 9 — Shared test harness

**Current behavior:** 12 `tests/<domain>/main_test.go` files hardcode the
identical plaintext DSN (`postgres://postgres:potato123@192.168.20.70/
miltech_ng_test?sslmode=disable`). Each domain also reimplements its own
fake-auth test middleware (e.g. `tests/shops/helpers_test.go`'s
`testUserMiddleware()` reading `X-User-ID`/`X-User-Name`/`X-User-Email`
headers directly) rather than sharing one `testutil` package.

**Structural improvement:** Extract a shared `testutil` package: one place
for the DSN (still hardcoded to the confirmed-correct
`192.168.20.70` host — this pass is about removing duplication, not
changing the connection target), one shared fake-auth middleware, one
shared router-builder helper. Migrate domains to use it incrementally.

**Validation:** `go test -p 1 ./...` passes with identical pass/fail
results before and after. This pass touches only test code, so production
behavior is unaffected by construction — the only risk is breaking test
setup itself, which the full suite run catches directly.

**Public API impact:** None — test-only code.

---

### Pass 10 — Remove the vestigial `shops_route.go` indirection

**Current behavior:** `api/route/shops_route.go`'s `NewShopsRouter()` wraps
`shops.RegisterRoutes()` with no added behavior — every other domain calls
`RegisterRoutes` directly from `route.go`. This is a one-hop indirection
left over from before `shops` adopted the same convention as everything
else.

**Structural improvement:** Inline the call in `route.go`; delete
`shops_route.go`.

**Validation:** `go build ./...`; `tests/shops/...` suite passes unchanged;
diff Gin's registered route list (`router.Routes()`) before and after to
confirm route registration order and paths are byte-identical.

**Public API impact:** None — internal wiring only.

---

## Out of scope — separate migration tasks

These items came up in the audit but are explicitly **not** part of this
plan, because each either changes behavior, requires a team decision this
document can't make unilaterally, or has blast radius beyond a mechanical
refactor:

- **Standardizing on `user_pmcs`'s response system repo-wide (or vice
  versa).** An architecture decision with real API-contract implications
  (ETags, strict JSON, structured error codes) for existing clients. Needs
  its own ADR before any code moves, following this project's existing ADR
  process.
- **Collapsing the two rate-limiting systems** (`api/middleware/rate_limiter.go`
  is confirmed live — wired into 5+ route registrations — and `user_pmcs`
  also has its own bespoke limiter; the two were never consolidated) into
  one. Touches request-handling behavior under load — outside this plan's
  "preserve behavior" constraint.
- **Reconciling raw-SQL vs. Jet query-builder usage.** Looks like a
  deliberate escape hatch for complex CTE/aggregation queries in several
  domains, not an oversight. Worth an ADR documenting it as sanctioned, or
  a real migration if the actual intent is to push everything through Jet
  — that's a product/team decision, not something this plan infers.
- **Whether `.gen/miltech_ng/` should be tracked in git or regenerated at
  build time.** If the decision is "stop tracking it," untracking 32 files
  and wiring generation into CI/build is a build-pipeline change with its
  own risk profile, not a code refactor.
- **Any dependency or framework version bumps.** None were flagged as
  urgent by the audits, but versions weren't explicitly checked against
  latest upstream. If wanted, that's a separate audit and a separate PR
  per dependency, given Go's minimal-version-selection semantics.
- **Extending `context.Context` into repository methods that don't accept
  it today.** Pass 8 only threads context where the repository layer
  already accepts it. Going further touches query-cancellation semantics
  under real load and deserves its own performance-validated pass, not a
  bundled signature change.

## Recommended docs/specs before implementation

- **A `CONTEXT.md` at the repo root.** `docs/agents/domain.md` expects one
  (and a `docs/adr/` directory) but neither exists — decisions currently
  live in `docs/project_notes/decisions.md` instead. Not a blocker (the
  domain-docs skill is explicitly designed to proceed silently when these
  are absent), but a lightweight domain glossary (`Shop`, `Checklist`,
  `Aggregate`, etc.) would let future passes cite shared vocabulary instead
  of re-deriving it each time.
- **One short ADR per consolidation decision** (response envelope,
  transaction helper, pagination helper) in `docs/project_notes/
  decisions.md`, following the same format ADR-021 already models —
  context, decision, alternatives considered, consequences.
- **A golden-fixture snapshot per domain touched in Pass 5**, captured
  before that domain's PR starts, as described in that pass's validation
  section. This is the concrete artifact that makes "response bodies
  didn't change" provable rather than asserted.

## Sequencing recommendation

1. CodeGraph initialization (`codegraph init -i`) — confirmed, run first,
   independent of all passes, speeds up every subsequent pass's
   file-location and call-site work.
2. Passes 1–3 (dead code, doc fix, test DSN cleanup) — no dependencies,
   lowest risk, can land same week.
3. Pass 4 (pagination helper) — independent, unblocks nothing else but is
   good practice groundwork before Pass 5 touches the same handler files.
4. Pass 5 (response consolidation), one domain PR at a time — the longest
   pass calendar-wise given the golden-fixture discipline per domain.
5. Pass 6 (transaction helper) — independent of Pass 5, can run in
   parallel with it.
6. Pass 7 (file splits) — do `pmcs_sbs_progress` and `user_pmcs/community`
   any time; hold `user_pmcs/owned` and `shops/aggregates` until after
   Pass 6 lands, since Pass 6 touches the same files.
7. Pass 8 (context threading) — do last among the code passes, since it's
   the widest-reaching signature change and benefits from the codebase
   already being tidied by 1–7.
8. Pass 9 (test harness) and Pass 10 (shops_route.go) — small, independent,
   fit in wherever convenient.

## Open decisions for the team, not resolved by this document

- `.gen/miltech_ng/` tracked-vs-generated-at-build (referenced above).
- Whether any Pass-5 domain's *current* response shape has a bug worth
  fixing as an explicit, separate API change rather than preserving as-is.
