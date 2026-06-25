# PMCS SBS Feature Audit

Date: 2026-06-25

## Scope

This audit covers the current PMCS SBS server feature in this repository:

- Public PMCS SBS library endpoints under `api/library/pmcs_sbs`.
- Authenticated PMCS SBS fault endpoints under `api/pmcs_sbs_progress`.
- The live Postgres `pmcs_sbs_faults`, `shop_vehicle`, and `shop_members` schema shape.
- Current PMCS SBS API docs and focused tests.

This is an implementation review, not a mobile-client implementation plan.

## Current Implementation Summary

The public library API remains blob-backed and exposes:

- `GET /api/v1/library/pmcs-sbs/folders`
- `GET /api/v1/library/pmcs-sbs/:folder/files`
- `GET /api/v1/library/pmcs-sbs/content?blob_path=...`

The authenticated persistence API is now faults-only and exposes:

- `GET /api/v1/auth/pmcs-sbs/equipment/:equipment_id/faults?guide_manual=...`
- `PUT /api/v1/auth/pmcs-sbs/equipment/:equipment_id/faults`
- `DELETE /api/v1/auth/pmcs-sbs/equipment/:equipment_id/faults`

The route still says `equipment`, but the value is `shop_vehicle.id`. PMCS SBS no longer owns equipment records, completions, or sync batches on the server.

The live `pmcs_sbs_faults` table currently has this key shape:

- `equipment_id text NOT NULL`
- `guide_manual text NOT NULL`
- `section_id text NOT NULL`
- `item_index integer NOT NULL`
- primary key on `(equipment_id, guide_manual, section_id, item_index)`
- foreign key from `equipment_id` to `shop_vehicle(id)`
- no `deleted_at` column

This shape is aligned with the current generated Jet model and fixes the prior cross-manual fault leak by making `guide_manual` part of fault identity.

## Positive Findings

### Authentication And Authorization

The authenticated fault routes are registered under `/api/v1/auth`, behind `AuthenticationMiddleware`. Each repository operation then checks that the authenticated user is a member of the shop that owns the target `shop_vehicle`.

Unauthorized and missing vehicles intentionally collapse to `404 {"message":"pmcs sbs equipment not found"}`. That is a reasonable security posture because it avoids revealing whether another shop's vehicle exists.

### Guide/Manual Isolation

Fault listing filters by `equipment_id` and `guide_manual`, and save/delete identity includes `guide_manual`. The composite primary key supports:

- independent fault sets for the same vehicle across different PMCS SBS manuals;
- deterministic upsert conflict handling;
- efficient list and delete lookups using the leftmost primary-key prefix.

### Input Validation

The service trims string fields and rejects blank required fields, negative `item_index`, invalid status values, non-PMCS guide paths, non-JSON guide paths, backslashes, and path-cleaning mismatches.

The `guide_manual` path validation mirrors the library API's directory traversal posture: client-provided paths must stay under `pmcs_sbs/` and must be clean.

### Database Constraints

The live table has server-side checks for nonblank fields, allowed status, nonnegative item index, and guide path format. This is good defense in depth; the API validates before insert, and Postgres still protects the table from bad writes outside the API.

### Focused Test Coverage

Current tests cover:

- route registration under auth;
- missing/invalid auth context;
- request validation;
- status normalization;
- non-member denial;
- missing vehicle behavior;
- guide/manual isolation;
- idempotent delete;
- `shop_vehicle` delete cascade.

These tests cover the highest-risk behavior changes from the faults-only and guide/manual-scoped refactors.

## Findings And Recommendations

### 1. Stale PMCS SBS API Doc Conflicts With Current Contract

Severity: Medium

`docs/api/pmcs-sbs-progress-sync.md` still documents `GET /pmcs-sbs/equipment/:equipment_id/faults` without the required `guide_manual` query parameter and save/delete bodies without `guide_manual`.

Current code requires `guide_manual` for list, save, and delete. The newer docs `docs/api/pmcs_sbs_faults_mobile_refactor.md` and `docs/api/pmcs_sbs_faults_guide_manual_mobile.md` are aligned with the implementation.

Risk:

- Mobile or future backend work can accidentally integrate against the stale contract and receive `400 {"message":"invalid guide manual"}`.
- The repo now has multiple PMCS SBS API docs with overlapping names and different contracts.

Recommendation:

- Mark `docs/api/pmcs-sbs-progress-sync.md` as superseded or replace it with a pointer to the current guide/manual-scoped docs.
- Keep one canonical mobile-facing fault contract.
- If historical docs are kept, move old docs under an archive path or add an obvious superseded banner at the top.

### 2. Schema Source Of Truth Is Not Obvious From Migrations

Severity: Medium

The live database and generated Jet files include the correct `guide_manual` primary-key shape. The repository's `migrations/` directory does not contain a PMCS SBS table migration, and the schema recreation SQL is not obvious from the active migration set.

Risk:

- A fresh developer database or CI database may not be reproducible from checked-in migrations alone.
- Jet regeneration depends on an already-correct database, so schema drift can be hard to diagnose.

Recommendation:

- Add an explicit schema artifact for the current `pmcs_sbs_faults` table. If this project is not using forward migrations for pre-production PMCS SBS work, document the exact table DDL in the current design or API doc.
- Include the composite primary key, foreign key, check constraints, and cascade behavior.
- Regenerate Jet only from that canonical schema.

### 3. Request Context Is Not Propagated Into PMCS SBS Database Work

Severity: Medium

Handlers call service methods without passing `c.Request.Context()`, and repository methods execute Jet queries through `repo.db` instead of a request-scoped context.

Risk:

- If a mobile client disconnects or the server times out the HTTP request, the database query may continue until it finishes or the database cancels it independently.
- This is low impact for the current small PMCS SBS queries, but it becomes more important under load or if future endpoints add larger reads.

Recommendation:

- Add `context.Context` to the PMCS SBS `Service` and `Repository` methods.
- Pass `c.Request.Context()` from handlers.
- Use Jet/database execution paths that honor context where available.
- Apply the same pattern to Firebase auth middleware over time; it currently uses `context.Background()` for token verification and user lookup.

### 4. Fault API Does Not Verify Guide Or Item Existence

Severity: Medium

The fault API validates that `guide_manual` is a clean PMCS SBS JSON path, but it does not verify that the blob exists or that `section_id` and `item_index` are valid for that guide.

Risk:

- A valid shop member can create fault rows for nonexistent guide paths or nonexistent section/item identities.
- Bad client state or manual path bugs can persist polluted fault data.

Tradeoff:

- The current approach keeps the fault API independent from Azure Blob reads and avoids adding blob latency to every fault save.

Recommendation:

- If strict data integrity is required, introduce guide metadata validation or a cached guide-manifest table.
- If decoupling is intentional, document that the client is the source of truth for section/item identity and add a periodic cleanup/reporting path for orphaned guide identifiers.

### 5. Authorization Logic Is Correct But Duplicated

Severity: Low to Medium

`api/pmcs_sbs_progress.RepositoryImpl.requireVehicleAccess` performs its own `shop_vehicle -> shop_members` membership query. The Shops package already has shared authorization helpers and per-request cached authorization support.

Risk:

- PMCS SBS authorization can drift from Shops authorization semantics over time.
- Future PMCS SBS operations that make multiple authorization checks in one request will not benefit from the existing request-level cache.

Recommendation:

- Keep the current query for now if the goal is minimal PMCS SBS churn; it is simple and efficient for one check per request.
- For future cleanup, route PMCS SBS through a shared authorization helper or add a narrow `RequireVehicleShopMember` helper in the Shops shared package.
- Preserve the current security contract: any shop member can manage faults, and missing/unauthorized vehicles return the same 404.

### 6. DELETE Is Idempotent And Does Not Report Missing Faults

Severity: Low

After vehicle access passes, `DeleteFault` executes a delete and ignores affected row count. Deleting a missing fault returns success.

Risk:

- This is fine for mobile sync ergonomics, but the server cannot tell clients whether a row actually existed.
- It also does not create an obvious audit point for repeated deletes of nonexistent fault identities.

Recommendation:

- Keep idempotent delete if the mobile contract depends on it.
- Document it clearly as an intentional behavior.
- If auditing becomes important, inspect affected row count and log a structured debug/info event without changing the response contract.

### 7. Fault List Is Unpaginated

Severity: Low

`ListFaults` returns all faults for one vehicle and one guide/manual. The composite primary key supports the query efficiently, and expected cardinality is probably bounded by the number of PMCS items in a manual.

Risk:

- If clients can create many faults over repeated guide revisions or if manual data grows unexpectedly, response size grows without a server-side limit.

Recommendation:

- Keep unpaginated list if the accepted product bound is one row per faulted PMCS item for one loaded manual.
- Document that bound.
- Revisit pagination only if measured response sizes or p95 latency justify the added client complexity.

### 8. Duplicate Shops Indexes Are Present

Severity: Low

Database health analysis reports redundant indexes around `shop_members(shop_id, user_id)` and `shop_vehicle(shop_id)`. This is not a PMCS SBS correctness bug, but PMCS SBS authorization uses these tables on every fault operation.

Relevant examples:

- `idx_shop_members_shop_user` is covered by the unique `shop_members_shop_id_user_id_key`.
- `idx_shop_members_shop_id` is covered by a wider shop/user index.
- `idx_shop_vehicle_shop_id` is covered by `idx_shop_vehicle_shop_save_time` for leftmost-prefix lookups.

Risk:

- Extra indexes increase write overhead and maintenance cost.

Recommendation:

- Do not remove indexes as part of the PMCS SBS feature without a broader Shops/database review.
- Schedule a separate index cleanup after checking production query patterns and constraint dependencies.

## Security Review

Current security posture is mostly sound for the PMCS SBS fault surface:

- Fault endpoints require Firebase auth.
- User membership is checked before reading or mutating fault rows.
- Missing and unauthorized vehicles return the same 404.
- Internal database errors are logged server-side and returned as generic 500 responses.
- `guide_manual` path validation blocks obvious traversal patterns.
- The public library content route already has rate limiting and JSON validation per ADR-014.

Security recommendations:

- Add request-context propagation to avoid runaway backend work after client cancellation.
- Decide whether PMCS SBS faults should be member-editable forever or whether fault mutation should eventually respect Shops roles/admin settings.
- Consider a data-integrity validation layer for guide/manual and section/item identities if clients are not trusted to preserve valid manual state.

## Performance Review

The current database shape is efficient for the implemented access patterns:

- Fault list query filters on `(equipment_id, guide_manual)` and orders by `(section_id, item_index)`, which matches the composite primary key order.
- Fault upsert conflicts on the full primary key.
- Fault delete filters on the full primary key.
- Vehicle-access checks join `shop_vehicle.id` to `shop_members.shop_id/user_id`; supporting indexes exist.
- Live EXPLAIN for the fault list query uses an index scan on `pmcs_sbs_faults`.

Performance recommendations:

- Keep the current `pmcs_sbs_faults` primary key order.
- Do not add a separate `pmcs_sbs_faults(equipment_id)` index; the primary key already supports that leftmost-prefix lookup and the FK cascade path.
- Avoid speculative indexes until query volume or EXPLAIN evidence justifies them.
- Clean duplicate Shops indexes in a separate database maintenance task, not inside this PMCS SBS feature audit.

## Error Handling Review

The PMCS SBS fault API has a clear error map:

- invalid auth context: 401
- invalid id/manual/body/status: 400
- missing or unauthorized vehicle: 404
- unexpected errors: generic 500

Recommendations:

- Keep the generic 500 response to avoid leaking internals.
- Consider using the same response envelope for DELETE success as other PMCS SBS responses if mobile consistency matters. Current docs and code return only `{"message":"Fault deleted"}`.
- Add tests that assert the exact stale-doc-sensitive behavior: missing `guide_manual` on GET returns 400, and missing `guide_manual` in PUT/DELETE bodies returns 400.

## Documentation Recommendations

The PMCS SBS docs should be simplified:

1. Keep `docs/api/pmcs_sbs_faults_guide_manual_mobile.md` or `docs/api/pmcs_sbs_faults_mobile_refactor.md` as the canonical mobile contract.
2. Supersede or remove `docs/api/pmcs-sbs-progress-sync.md`.
3. Add a short backend schema note with the current `pmcs_sbs_faults` DDL or link to the canonical migration/schema artifact.
4. Add an ADR amendment noting that guide/manual scoping is now part of server fault identity.

## Recommended Next Steps

1. Fix PMCS SBS documentation drift by superseding the stale progress-sync doc.
2. Add or document canonical `pmcs_sbs_faults` DDL for fresh database reproducibility.
3. Add request-context propagation to PMCS SBS service and repository calls.
4. Add focused tests for missing `guide_manual` at route level.
5. Decide whether guide/manual existence and section/item identity should be server-validated or intentionally client-owned.
6. Schedule redundant Shops index cleanup as a separate database optimization task.

## Verification Performed

- Inspected PMCS SBS library and authenticated fault code.
- Inspected focused PMCS SBS service, route, and repository tests.
- Checked live Postgres object details for `pmcs_sbs_faults`, `shop_vehicle`, and `shop_members`.
- Checked EXPLAIN output for the PMCS SBS fault list query and vehicle-access query.
- Ran database index health analysis and reviewed duplicate-index findings relevant to Shops authorization tables.
