# Shop message parent ownership (021)

Migration `021_enforce_shop_message_parent_ownership.sql` replaces the verified
single-column `shop_message_parent_id_fkey` with
`shop_message_parent_shop_fkey (parent_id, shop_id) REFERENCES
shop_messages(id, shop_id) ON DELETE CASCADE`. It reuses migration 020's
`shop_messages_id_shop_unique`; rollback 021 retains that key and the managed
asset reference dependency that already uses it.

## Preconditions and refusal

The approved physical fixture has **ON DELETE CASCADE**, despite historical
migration 005 documenting SET NULL. Neither 005 nor 015–018 is changed.
Under an exclusive message-table lock, 021 checks actual catalog identity:
text identity columns, required ID/Shop nullability, nullable parent, the exact
validated nondeferrable composite unique key, one validated nondeferrable parent
foreign key, expected source/target columns and table, MATCH SIMPLE,
ON UPDATE NO ACTION, and ON DELETE CASCADE. Repeated forward/reverse operations
refuse rather than silently replacing drifted relationships.

Foreign-Shop and missing historical parents cause refusal before DDL changes.
No row is repaired, reparented, orphaned or deleted by 021. A different actual
FK name, shape, type or action requires a separately reviewed conversion
manifest and explicit authorization. There is no flag that bypasses the gate.
A manifest must identify the exact target and current catalog definition,
explain any intended change in deletion behavior, inventory affected message
and asset identities, define authorized data repair and recovery, and establish
validation evidence. Migration source history alone is insufficient evidence
of the current target's behavior.

Existing databases `miltech_ng_test` then `miltech_ng` remain separate execution
gates requiring explicit authorization and same-session identity/schema checks.
No existing target was inspected or changed for this implementation. Schedule
the exclusive-lock migration for a reviewed low-traffic window; prepare a
backup/recovery procedure and inspect waiting transactions before application.
Generate tagged Jet models after every authorized forward/reverse/refusal or
repair rehearsal and verify all 32 tracked `user_pmcs_*` files are unchanged.
Never run plain Jet or hand-edit generated output.

## Application and cleanup behavior

Create locks the current Shop and verifies current membership, then resolves
and key-share-locks the parent with both ID and persisted Shop ID in the same
transaction. Foreign and missing parents receive the same error, including
when the actor belongs to both Shops. This happens before insertion, allocator
side effects, asset references or cleanup candidates. The parent lock lasts
through commit; concurrent deletion must serialize or cause rejection, and
request cancellation reaches the query. Same-Shop replies preserve the legacy
wire shape and nullable parent key. Omitted and unbounded reads are unchanged.

Parent deletion cascades through same-Shop descendants. Migration 020's
reference-deletion hook retains immutable registry targets and durable cleanup
jobs; migration 021 does not alter that hook. Cleanup rechecks zero references
and current target proof before cloud deletion. Queue Finish acknowledges only
the queue; the next Reconcile normally advances deleting to deleted using
persisted success proof. Other Shops' references, jobs and cloud targets remain
outside the parent's deletion scope. Cloud execution was tested with fakes.

## Disposable verification

Run only the marker-protected loopback wrapper:

```sh
rtk proxy env -u TEST_DATABASE_URL -u TEST_DATABASE_MARKER -u TEST_DB_URL \
  bash scripts/test-shops-isolated.sh \
  --verify-notification-migration --verify-message-sync-migration \
  --verify-remediation-migrations -v -count=1 \
  ./tests/shops ./tests/equipment_services
```

`TestMessageParentSameShop`, `TestMessageParentDatabaseOwnership`,
`TestMessageParentConcurrentDeletion`, and `TestMessageParentCascadeAssets`
cover repository rejection before side effects, database enforcement, delete
commit/rollback and request cancellation, and durable cascade cleanup without
another Shop's targets. Existing parent response tests cover legacy reads.

`TestMessageParentMigrationRefusesForeignRows` is an asserted **shell verifier
case**, not a Go-selectable test. The verifier also checks missing parents,
unapproved SET NULL action, deferrability/shape and type drift, populated
forward/reverse/reapply, repeated-operation refusal, interrupted reverse and
reverse-shape refusal. Every refusal compares persisted message/counter/asset/
reference/job data and physical schema metadata before and after, then invokes
fresh tagged generation and compiles generated packages. Reverse checks the
same 020 unique-constraint OID and surviving asset FK dependency. The complete
available candidate is built afterward. A focused pass does not imply the
broader suite, live migration, deployed role, Azure or release gates passed.
