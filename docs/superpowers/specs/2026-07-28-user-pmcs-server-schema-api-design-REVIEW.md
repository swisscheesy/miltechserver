# Review Findings: User-Created PMCS Server Schema and API Design

**Reviewed document:** `2026-07-28-user-pmcs-server-schema-api-design.md`
**Review date:** 2026-07-28
**Review type:** Manual line-by-line consistency, integrity, and efficiency
audit. No changes were made to the source document.

This document is unusually rigorous for a design spec — most of the edge
cases an implementer would normally discover the hard way (idempotency,
lock ordering, tombstone permanence, cursor collapsing) are already worked
out. The findings below are the gaps and rough edges that survived a careful
read, not a wholesale critique. Nothing here blocks moving forward; most
items are clarifications the implementation plan should resolve explicitly
rather than leaving to an implementer's judgment call.

Severity is about "how much implementer discretion / bug risk this leaves
open," not "how broken the design is."

---

## High severity

### H1. `current_release_revision_id` lacks a composite FK, unlike every other cross-checklist reference in the doc

**Location:** §6.11 `user_pmcs_community_sources`

The doc is otherwise disciplined about this exact problem: §6.3 defines
`(checklist_id, id)` unique on revisions specifically "for composite
ownership references," and §6.12 uses it — `user_pmcs_community_releases`
has an explicit composite FK verifying "the revision belongs to the
checklist." `user_pmcs_community_sources.current_release_revision_id`
is the same class of reference (a source pointing at one specific revision
that must belong to *this* checklist) but the doc only states the
"must belong to the same checklist" rule in prose, with no composite FK
called out.

**Why it matters:** Without `(checklist_id, current_release_revision_id)
REFERENCES user_pmcs_community_releases(checklist_id, revision_id)`, that
invariant is enforceable only in application code. Given the doc's own
pattern (composite FKs wherever cross-table checklist ownership must be
guaranteed), this is very likely an oversight rather than an intentional
choice to rely on app logic here specifically.

**Recommendation:** Add the composite FK explicitly in the schema section,
consistent with the pattern already used for `user_pmcs_community_releases`.

---

### H2. `user_pmcs_community_releases` FK delete behavior is never specified, and the deletion sections are asymmetric about who removes it

**Location:** §6.12, §17.2, §17.4

§6.12 lists constraints for `user_pmcs_community_releases` but never states
`ON DELETE` behavior for either `revision_id → user_pmcs_revisions(id)` or
`checklist_id → user_pmcs_checklists(id)`. This matters because two
different deletion flows both need to remove "unpinned" revisions while a
release row may still reference them:

- §17.2 (single released-checklist deletion) says it "removes the draft and
  every unpinned revision" — but says nothing about removing the
  corresponding release rows first.
- §17.4 (account deletion) *does* explicitly call this out as its own step:
  "removes unpinned releases and content" (step 8), separate from revision
  removal.

If the FK isn't `ON DELETE CASCADE` (and given revisions are otherwise
immutable/append-only, CASCADE here seems risky — you'd be relying on a
cascade to enforce a business rule about pinning), then §17.2's revision
removal will violate referential integrity unless it also deletes the
release row first, which it never says.

**Recommendation:** Either (a) state the FK is `RESTRICT` and make §17.2
explicitly delete the release row before the revision row, matching §17.4's
explicit step-8 phrasing, or (b) state the FK is `CASCADE` and note that the
"unpinned" check that gates revision removal *is* the safety mechanism (i.e.
cascade is safe here specifically because removal is already gated on
"unpinned"). Either is fine — the doc just needs to pick one and say so,
since right now the two deletion paths read as if they follow different
rules for the same table.

---

## Medium severity

### M1. Discovery sort order (`updated_at DESC`) is a free, repeatable ranking-gaming vector

**Location:** §12.1

Community browse sorts strictly by "current source `updated_at DESC`."
Per §6.11/§6.12, *any* new release — including a functionally meaningless
one (e.g., a whitespace fix in a description) — bumps `updated_at` and
therefore jumps the source back to the top of the feed. This isn't blocked
by the idempotency rules in §10.2, because those only no-op *identical*
retries of the *same* revision; a genuinely new (if trivial) revision is a
real mutation every time.

**Why it matters:** Since ranking/popularity signals are explicitly deferred
(§24) and "recency" is the *only* discovery ordering, an owner can keep a
checklist permanently pinned to the top of community browse by re-releasing
every few minutes. §19's generic per-user/per-IP rate limits would slow this
down but aren't described as tuned to this specific case, and rate-limiting
the *release* action heavily enough to stop gaming would also cap legitimate
rapid iteration.

**Recommendation:** Worth a product decision now rather than after launch —
e.g. sort by `first_released_at` for ranking purposes while still surfacing
"updated" state separately, or note this as a known, accepted limitation of
the v1 discovery model. Given how much care the rest of the doc puts into
abuse/resource-ceiling thinking (§8, §19), this specific gap stands out.

---

### M2. Ambiguous status code for grapheme/byte/node-count ceiling violations

**Location:** §7, §8, §9.1, §15.3/§15.4

§9.1 groups "grapheme limits," "byte limits," and "node counts" together
with "UUID syntax" and "body size" as pre-transaction pure validation. But
the error-code table only cleanly covers the extremes:

- `content_too_large` / 413 — clearly the *body size* / 20 MiB aggregate
  ceiling.
- `validation_failed` / 422 — clearly *publication completeness* rules
  (§16's list: nonblank fields, "at least one X," etc.).

Grapheme-count, byte-count, and node-count ceiling violations fall between
these. They're not JSON/UUID syntax problems (ruling out clean `400`), they
apply to *drafts* too and aren't publication-specific (ruling out a clean
reading of `422` per its stated definition), and they're per-field/per-node
rather than whole-body (not quite `413` either).

**Why it matters:** Client error-handling and retry logic (e.g., "should I
let the user keep typing vs. hard-block the request") differs meaningfully
by status code. Two implementers reading this doc could reasonably pick
different codes for the same violation.

**Recommendation:** Add one explicit sentence mapping ceiling-type
violations to a status code (a plausible resolution: field-level grapheme/
byte overruns → `422 validation_failed` since they're semantic per-field
rules, not syntax; node-count-per-parent overruns → `413`, since they're
closer to "your request is asking for too much of one thing").

---

### M3. No ceiling on total owned checklists or total subscriptions per account; `subscriptions/updates` has no pagination

**Location:** §8, §13

§8 is thorough about per-revision content ceilings (sections, items,
notices, steps) but there is no cap anywhere on how many *checklists* a
user may own or how many *subscriptions* a user may hold. The account-delta
endpoint (§11) has cursor pagination that bounds each page, but nothing
bounds the total number of roots a full resync (or the `subscriptions/
updates` endpoint, §13) must scan and return in one shot — `subscriptions/
updates` has no `after`/`limit` parameters at all, unlike every other
listing endpoint in the doc.

**Why it matters:** This is the same class of resource-abuse concern §8
already takes seriously for content nodes, just at the account level
instead of the revision level. A user (or buggy client) that creates a very
large number of checklists, or subscribes to a very large number of
community sources, produces an unbounded-size response from
`subscriptions/updates` and a very long initial full-sync tail from
`/user-pmcs/sync`.

**Recommendation:** Add a configurable per-account ceiling (owned
checklists, active subscriptions) alongside the §8 table, and give
`subscriptions/updates` the same `after`/`limit` keyset shape used
elsewhere.

---

### M4. Idempotent-retry tree comparison happens under lock, and "identical representation" is never defined

**Location:** §10.2, §10.1

§10.2 requires proving "the exact requested target state is already
present" before treating a failed precondition as a no-op success — and for
the checklist-creation endpoint (§14.1: create root *and* initial draft in
one call), that means comparing a full submitted tree (up to the §8
ceilings — thousands of nodes, up to 8 MiB) against current stored state.
§10.1 states preconditions are evaluated "while holding the root lock," so
this comparison happens inside the transaction, not during the pre-lock
pure-validation pass.

**Why it matters:** Two open questions with real implementation
consequences: (1) what exactly counts as "identical" — byte-exact text,
UUID-for-UUID structural match, or something that tolerates position
renumbering? — and (2) doing that comparison while holding the checklist
lock means lock-hold time scales with tree size specifically on the retry
path, which is the path most likely to fire under poor mobile connectivity
(exactly when you'd want locks held *briefly*).

**Recommendation:** Define tree-equality precisely (even just "same UUID
set with byte-identical text and positions") and consider whether the
comparison can be structured to bail out fast on the first mismatch rather
than materializing both trees fully before comparing.

---

### M5. Items have no per-parent ceiling; every sibling node type but this one does

**Location:** §8

The ceiling table gives every other node type both a per-parent and a
total cap (section models: 100/section *and* 1,000 total; notices: 100/item
*and* 4,000 total; steps: 250/item *and* 10,000 total). Items only get a
total cap (2,000) with no "items per section" limit. A single section could
legally hold all 2,000 items.

**Why it matters:** Probably an oversight rather than a deliberate choice —
nothing in the doc explains why items are exempt from the per-parent
pattern applied everywhere else, and an unbounded single section undermines
the intent of ceilings as an abuse-protection mechanism.

**Recommendation:** Add an explicit "items per section" ceiling, or state
the reason it's intentionally omitted.

---

## Low severity / observations

### L1. Keyset pagination on a mutable sort key (`updated_at`) can skip entries mid-pagination

**Location:** §12.1

Standard, well-known trade-off: if a source already returned on page 1 is
updated again while a client is mid-pagination, it jumps ahead of the
cursor and will never appear on a later page of that same pagination pass
(it isn't duplicated, it's silently skipped until the client restarts from
the top). This is likely acceptable for a "recent" feed, but the doc's
correctness bar elsewhere (§22.6 explicitly tests "keyset pagination has no
duplicates or skips") suggests this specific, expected class of skip should
be called out as accepted behavior rather than left for the concurrency
test to "discover" as a surprise.

### L2. No garbage-collection path for unpinned-but-superseded releases outside deletion events

**Location:** §6.12

"An unpinned historical release may be removed" is stated as a
*possibility*, but the only places the doc actually removes release rows
are checklist deletion (§17.2) and account deletion (§17.4). A checklist
that's actively maintained for years, releasing frequently, with all
subscribers eventually updating off old revisions, has no described path to
reclaim those now-unpinned release rows (and their associated immutable
revision trees) short of the owner deleting the whole checklist. This is a
storage-growth concern, not a correctness one — worth a decision (accept
unbounded retention for v1, or add a periodic sweep) rather than leaving it
implicit.

### L3. Several invariants are service-layer-only with no DB backstop

**Location:** §6.13 ("an owner cannot subscribe to their own source"),
§6.8/§16 (published notices must have non-null type — the CHECK constraint
in §6.8 only fires when type is non-null, it can't itself distinguish draft
vs. published rows)

Both are reasonable relational-modeling trade-offs (cross-table and
cross-state invariants are genuinely awkward to express as plain CHECK
constraints), and §18.4 already acknowledges the broader version of this
trade-off ("Postgres RLS is not introduced... proven with integration
tests"). Flagging only so the implementation plan treats these as
single-point-of-failure invariants worth deliberate test coverage, not
because the design choice itself is wrong.

### L4. `40001` (serialization failure) retry is likely unreachable on the described write path

**Location:** §9.4

§9 states writes use `READ COMMITTED` with explicit row locks, and the
read-only delta endpoint (§11.2) uses `REPEATABLE READ` but is read-only.
`40001` is characteristic of `REPEATABLE READ`/`SERIALIZABLE` write-write
conflicts; under `READ COMMITTED` with explicit locking, conflicts should
normally surface as lock waits or `40P01` deadlocks, not `40001`. Not
harmful to retry for it defensively, but worth confirming it's not a signal
that some write path was intended to run at a stricter isolation level than
described.

### L5. First-sync reconstruction of long local publish histories is O(2n) sequential round trips, and isn't in the performance-verification list

**Location:** §14.2, §22.7

The protocol is well-designed for correctness and resumability (explicitly
avoids an oversized first upload, supports resuming after interruption), but
for a checklist with many historical local publications, first sync
requires *n* draft-upload + *n* publish round trips before any other device
sees the full history. §22.7's performance scenarios list maximum-size
single operations and concurrent users, but not this specific
many-small-sequential-requests case, which has a different performance
profile (dominated by round-trip latency, not payload size or lock
contention).

**Recommendation:** Add "first-sync reconstruction of N historical
publications" to §22.7's verification list.

### L6. Unicode grapheme-cluster parity between Flutter and Go is a real dependency risk (already tracked, but worth naming explicitly)

**Location:** §7.1, §22.2

§22.2 already requires "Flutter/Go Unicode normalization fixtures match,"
which is the right mitigation. Worth naming the actual risk explicitly in
the doc: Go has no standard-library equivalent of Flutter's `characters`
package (UAX #29 grapheme segmentation), so this will depend on a
third-party Go library, and the two implementations must be checked against
the *same* Unicode version — segmentation rules do change between Unicode
versions, so a client/server library pinned to different Unicode data
versions could accept or reject the same string inconsistently even with
correct code on both sides.

### L7. Precondition target for nested resources (drafts/publications/community-releases) is implied, not stated

**Location:** §10.1, §14.1

ETags only exist for checklists and subscriptions (§5.1) — revisions have
none of their own. §6.3 confirms publication "verifies the ETag" of the
locked checklist. But §10.1's general precondition rules don't explicitly
say that for every nested mutation endpoint (drafts, publications,
community-releases under a `checklist_id`), `If-Match`/`If-None-Match`
always targets the *parent checklist's* ETag, never something per-revision.
It's inferable from context, but a single explicit sentence would remove
any doubt for an implementer wiring up the middleware generically.

---

## Summary

| # | Finding | Severity |
|---|---|---|
| H1 | `current_release_revision_id` missing composite FK to releases | High |
| H2 | `user_pmcs_community_releases` FK delete behavior unspecified; §17.2/§17.4 asymmetric | High |
| M1 | Discovery sort order is a free, repeatable ranking-gaming vector | Medium |
| M2 | Ambiguous status code for grapheme/byte/node ceiling violations | Medium |
| M3 | No account-level ceiling on checklists/subscriptions; `subscriptions/updates` unpaginated | Medium |
| M4 | Idempotent-retry tree comparison under lock; equality undefined | Medium |
| M5 | Items have no per-section ceiling, unlike every sibling node type | Medium |
| L1 | Keyset pagination on mutable `updated_at` can skip entries | Low |
| L2 | No GC path for unpinned releases outside deletion events | Low |
| L3 | Service-layer-only invariants with no DB backstop | Low |
| L4 | `40001` retry likely unreachable under stated isolation level | Low |
| L5 | First-sync history reconstruction missing from perf verification | Low |
| L6 | Flutter/Go Unicode version-skew risk (mitigation already tracked) | Low |
| L7 | Nested-resource ETag target is implied, not stated | Low |

No findings contradict the document's stated product decisions (§3) or
scope boundaries (§2, §24) — everything above is either a schema-integrity
gap, an ambiguity an implementer would have to guess at, or a
performance/abuse edge case the doc's own rigor elsewhere suggests it would
want to address explicitly.
