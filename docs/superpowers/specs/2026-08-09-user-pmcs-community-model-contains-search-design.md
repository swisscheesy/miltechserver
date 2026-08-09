# User PMCS Community Model Contains Search Design

**Date:** 2026-08-09

**Status:** Approved for implementation planning

## Goal

Change the public User PMCS community browse endpoint so its optional `model`
filter performs a literal, case-agnostic substring search instead of an exact
normalized match.

For a current release whose model is `M1165A1`:

- `model=m1165` matches;
- `model=m1165a1` matches;
- `model=M1165A1` matches; and
- a search for `m1165` may also return releases containing models such as
  `M1165` and `M1165A2`.

## Scope

This change applies only to:

```http
GET /api/v1/user-pmcs/community?model=<text>
```

The route, query parameter name, authentication policy, response shape,
sorting, pagination format, caching, gzip behavior, and public rate limits do
not change.

The filter continues to search revision-level checklist models in
`user_pmcs_revision_models`. It does not search section models, checklist
names, descriptions, creator names, or historical releases.

## Current Behavior

The handler forwards `model` to `ServiceImpl.Browse`. The service calls
`shared.NormalizeModel`, which validates UTF-8, collapses Unicode whitespace,
trims surrounding whitespace, and lowercases Unicode characters. The
repository then applies this predicate to an active source's current release:

```sql
model.normalized_text = $n
```

The equality predicate makes the existing search case agnostic but exact after
normalization.

## Search Contract

### Normalization

The implementation must retain `shared.NormalizeModel` as the single
normalization function for both stored revision models and incoming searches.
No second case-folding or punctuation-normalization contract is introduced.

Existing validation remains unchanged:

- invalid UTF-8 returns the existing `validation_failed` response;
- input that normalizes to an empty string returns the existing
  `invalid_request` response; and
- any nonempty normalized term remains valid, including one- and two-character
  terms.

### Matching

Matching is literal substring containment anywhere in a revision model's
normalized text. It is not prefix-only, token-only, fuzzy, stemmed, or
typo-tolerant.

Examples:

| Stored model | Search | Match |
|---|---|---|
| `M1165A1` | `m1165` | yes |
| `M1165A1` | `M1165A1` | yes |
| `M1165A1` | `1165a` | yes |
| `M1165A1` | `m1165a2` | no |
| `XM1165A1-R2` | `m1165a1` | yes |

Characters with meaning in SQL `LIKE` patterns remain literal user text.
Searching for `%`, `_`, or the chosen escape character must match that actual
character and must not widen the search.

When a current release declares multiple revision-level models, a match on any
one model includes the checklist once. The response continues returning the
complete model list for each matched summary, not only the matching model.

## Query Design

Keep the current correlated `EXISTS` boundary so filtering remains limited to
the current released revision:

```sql
AND EXISTS (
    SELECT 1
    FROM user_pmcs_revision_models AS model
    WHERE model.normalized_text LIKE $n ESCAPE '!'
      AND model.revision_id = source.current_release_revision_id
)
```

The query argument is produced by a small repository-local helper:

1. replace `!` with `!!`;
2. replace `%` with `!%`;
3. replace `_` with `!_`; and
4. surround the escaped normalized value with `%`.

For example, normalized input `m1165_1` becomes `%m1165!_1%`.

`LIKE` is sufficient because both operands use the existing lowercase
normalization contract. Adding `ILIKE` would duplicate that responsibility and
make behavior depend more heavily on the database locale.

The value remains a query parameter. User input must never be concatenated
into SQL text.

## Visibility and Pagination

All current visibility rules remain intact:

- only `source.status = 'active'` sources are browsable;
- tombstoned checklists remain excluded;
- only `source.current_release_revision_id` is searched and returned;
- matching a historical or superseded release does not include a checklist;
  and
- retired sources remain excluded.

Keyset ordering remains:

```sql
ORDER BY source.updated_at DESC, source.checklist_id ASC
```

The existing opaque cursor contains only `updated_at` and `checklist_id`; it
does not bind the model filter. Clients must discard the current cursor and
restart from page one whenever the model search changes. The server cursor
format does not change in this refactor.

## Performance and Index Policy

The existing reverse B-tree index
`user_pmcs_revision_models_lookup_idx(normalized_text, revision_id)` was
designed for equality. A leading-wildcard contains predicate cannot rely on
that access path in the same way.

Do not add an index speculatively and do not weaken the existing query-plan
gate. The implementation must capture the real contains-query SQL and run:

```sql
EXPLAIN (ANALYZE, BUFFERS)
```

against the existing representative 500-source performance fixture. The
accepted implementation must preserve:

- no sequential scan of `user_pmcs_revision_models` in the representative
  plan;
- the existing four-query bound for filtered plus unfiltered browse;
- correct `LIMIT + 1` pagination; and
- recorded before/after database and query latency evidence using the same
  fixture. This refactor does not invent a new numeric latency threshold.

The primary key `(revision_id, normalized_text)` may provide an acceptable
current-release lookup before applying the substring filter. A trigram
migration is required if the representative production query performs a
sequential scan of `user_pmcs_revision_models` or fails the existing approved
relation-index assertion. After adding the index, repeat the same plan and
latency checks.

If the trigram migration is required:

- use
  `migrations/012_add_user_pmcs_model_search_trigram_index.sql` and
  `migrations/012_rollback_user_pmcs_model_search_trigram_index.sql`;
- verify `pg_trgm` availability before relying on it;
- create a GIN `gin_trgm_ops` index named
  `user_pmcs_revision_models_search_trgm_idx` on `normalized_text`;
- make rollback remove the feature-owned index but not blindly remove a
  possibly shared extension;
- rehearse forward/rollback/forward only on `miltech_ng_test`;
- apply forward-only to `miltech_ng` after re-verifying the target database;
  and
- do not apply the migration to production as part of implementation.

Short searches remain valid. PostgreSQL trigram indexes may be less selective
for terms shorter than three characters, but introducing a new minimum search
length would be a separate API behavior change and is out of scope.

## Error Handling and Security

No new response code is introduced. Existing model-validation failures retain
their current typed API errors.

The search term remains parameterized and SQL pattern metacharacters are
escaped. Authored model text and query values must remain absent from logs,
including rate-limit and error paths. Existing public rate limiting remains in
force.

## Testing

### Service tests

Retain the existing service test proving case and Unicode-whitespace
normalization. No handler or service interface change is required.

### Query/helper tests

Add focused tests proving that the pattern helper:

- wraps ordinary normalized text for contains matching;
- escapes `%`;
- escapes `_`;
- escapes `!`; and
- handles combinations without changing ordinary characters.

### PostgreSQL integration tests

Extend the existing community browse integration coverage with current
releases containing `M1165`, `M1165A1`, `M1165A2`, unrelated models, and
literal wildcard characters. Prove:

- partial and exact queries return the intended sets;
- mixed-case input returns the same set;
- matches can occur at the beginning, middle, or end;
- `%`, `_`, and `!` are literal;
- a matching historical release does not qualify the current release;
- retired and deleted sources remain excluded;
- multiple matching models do not duplicate a checklist; and
- filtered keyset pages contain no duplicates or skips.

### Performance tests

Rename exact-model performance evidence to contains-model evidence and use a
proper substring of the seeded model. Capture and validate the production SQL
plan rather than substituting a hand-written query.

## Documentation

Update the two active client-facing contracts to describe `model` as a
literal, case-agnostic contains filter and to require pagination restart when
the filter changes:

- `docs/client/2026-07-29-user-pmcs-server-api-contract.md`;
- `docs/client/2026-07-31-user-pmcs-mobile-api-implementation-guide.md`.

Historical implementation plans, progress ledgers, and completed rollout
records retain their original exact-search wording as historical evidence.

## Non-Goals

- fuzzy or typo-tolerant search;
- ranking exact matches ahead of partial matches;
- searching fields other than revision-level models;
- changing public browse ordering or pagination cursors;
- changing rate limits or adding a minimum search length;
- changing response models;
- converting the repository's existing raw SQL to Jet; or
- deploying a database migration to production.

## Acceptance Criteria

The refactor is complete when:

1. `m1165` returns active current releases containing `M1165`, `M1165A1`, and
   `M1165A2` where those fixtures exist;
2. `m1165a1` returns only releases with a model containing `m1165a1`;
3. case and normalized whitespace do not change result membership;
4. SQL pattern metacharacters are treated literally;
5. historical, retired, and deleted content remains excluded;
6. filtered keyset pagination remains correct;
7. the representative query avoids a sequential scan of
   `user_pmcs_revision_models`, retains the existing query-count bound, and
   records comparable latency evidence, with a trigram index added only if the
   existing approved relation-index assertion fails;
8. active client contracts describe the new behavior; and
9. focused, integration, performance, race, and serialized repository-wide
   verification results are reported exactly.
