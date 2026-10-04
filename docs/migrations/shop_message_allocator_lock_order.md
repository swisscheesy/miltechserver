# Shop message allocator lock order (019)

Migration `019_fix_shop_message_allocator_lock_order.sql` fixes the mixed-writer
deadlock in the unchanged 018 allocator. The trigger takes the persisted Shop
row `FOR KEY SHARE` before touching `shop_message_counters`. Current writers keep
`shared.LockShopMutation` and its stronger Shop `FOR UPDATE` authorization lock.
The trigger still solely allocates insertion numbers, overwrites supplied values,
and retains the counter lock until transaction commit or rollback.

## Evidence and supported writers

`TestMessageAllocatorMixedWriters` first ran against original 018. With an existing
counter, a current writer held Shop `FOR UPDATE`; a legacy-shaped INSERT held the
counter and waited for its Shop foreign key. The current INSERT then waited for
the counter. PostgreSQL returned **40P01 deadlock detected**, with both physical
backend PIDs in the reciprocal transaction-lock detail. The test polls
`pg_blocking_pids` and `pg_stat_activity` for those exact sessions before starting
the second operation; a timeout is never accepted as deadlock evidence.

The 019 tests cover both writer arrival orders, same-Shop numbers in commit order,
different Shops, rollback without gaps, deletion waiting for an insert, and an
insert waiting for deletion then failing with foreign-key SQLSTATE 23503. Existing
50-writer and numbering tests remain in the focused suite.

| Writer | Ordering / supported boundary |
| --- | --- |
| Current server `messages.RepositoryImpl.CreateShopMessage` | `sharedb.WithTxContext` -> `LockShopMutation` (Shop, membership) -> message INSERT -> trigger counter; caller-context transaction |
| Legacy-shaped direct INSERT of the original five writable fields | Trigger acquires Shop before the counter; omitted insertion number remains supported |
| Multiple legacy INSERTs within one Shop | Compatible Shop key-share locks, then counter serialization through commit |
| Shop deletion | Shop deletion waits for key-share; committed deletion cascades messages/counters; a later insert refuses missing Shop |
| Arbitrary external counter/message-first transactions, lock upgrades or multi-Shop transactions in inconsistent order | **Not certified.** Inventory and fence them at cutover, or separately review whole-transaction deadlock retry and consistent ordering |

Repository search found one production message INSERT in
`api/shops/messages/repository_impl.go`. This inventory is source evidence, not
proof of installed fleet binaries, operational jobs, external SQL, or clients.
No blanket claim of deadlock freedom is made.

## Preconditions, atomicity and reverse

Both files lock `shops`, `shop_messages`, then `shop_message_counters` before
verification/replacement. Each lock wait is limited to five seconds; this does not
bound lock hold time. Fence unsupported writers before starting; the locks do not
make external inversions safe. Failed guards, lock timeouts and deadlocks roll
back atomically. Review the failure before retrying the whole migration.

Forward accepts only the exact original 018 function body (MD5
`5b2f595523c9653cf80ec52148537550`) with PL/pgSQL, trigger return type, volatile,
invoker execution, and no function-local settings. It also verifies the enabled
BEFORE ROW INSERT trigger and numbering/counter data. NULL/nonpositive/duplicate
numbers, missing or trailing counters, and orphan/invalid counters refuse without
repair. Counters above remaining message maxima are legitimate after deletions
and must stay unchanged. New Shops without messages may lack a counter.

The body digest pins known source; it is not a security signature or a replacement
for migration-file checksums and full schema/role readiness. Existing 015–018 SQL
and checksum pins are unchanged. No rows, columns, indexes, grants, function owner,
or trigger bindings are rewritten by 019; CREATE OR REPLACE retains the function
identity and ACL. Repeated direct forward/reverse execution refuses the wrong
source version; migration runners must use their reviewed applied-state protocol.

`019_rollback_fix_shop_message_allocator_lock_order.sql` accepts only the 019 body
(MD5 `42f7105c5298267751a040de0fe30012`) and restores pinned 018 exactly. This
**reintroduces the mixed-writer deadlock**. Use reverse only in the guarded
disposable rehearsal or a separately approved deployment with all relevant
writers fenced. It preserves numbers/counters and is followed by tagged generation.

## Effective writer privileges

The function remains SECURITY INVOKER. Migration adds no grants. In addition to
normal schema usage and message INSERT/readback privileges, the allocator needs:

| Object | Effective privileges for the statements exercised |
| --- | --- |
| `public.shops` | SELECT on `id`; UPDATE on at least one column to use FOR KEY SHARE (the fixture grants only UPDATE on `name`) |
| `public.shop_message_counters` | INSERT on `shop_id,last_number`; SELECT on `shop_id,last_number`; UPDATE on `last_number` |
| `public.shop_messages` | The fixture uses INSERT on `id,shop_id,user_id,message,created_at` and SELECT on `insertion_number` for RETURNING |

Shop SELECT alone does not authorize the row lock. [PostgreSQL SELECT locking
privileges](https://www.postgresql.org/docs/17/sql-select.html) require UPDATE on
at least one selected-table column; [row-lock conflict rules](https://www.postgresql.org/docs/17/explicit-locking.html)
explain why KEY SHARE waits for FOR UPDATE and deletion.

Nine restricted-role cases execute real statements with distinct NOLOGIN,
NOSUPERUSER, NOBYPASSRLS, NOINHERIT effective roles using SET LOCAL ROLE on reserved
physical connections. Missing Shop SELECT, Shop row-lock rights, and every missing
counter privilege fail explicitly with SQLSTATE 42501 and the expected relation.
The authorized role succeeds with minimal column grants on both fresh and existing
counter paths. Failed writes leave neither message nor counter behind. Roles and
grants are removed at cleanup. This tests effective invoker permissions, not a real
application login, authentication, RLS policy, or named target's privileges.

Actual `DB_USERNAME` application-role proof and catalog/function/trigger/index/
constraint readiness are separate deployment gates (Task 27). A legacy-shaped
send must succeed under that role before production, even if message sync is off:
the trigger runs independently of capability flags. Operator ownership and DDL
rights must also be checked; the disposable migration role is postgres.

## Disposable rehearsal

Run from the approved worktree (unset inherited target selectors):

```sh
rtk proxy env -u TEST_DATABASE_URL -u TEST_DATABASE_MARKER -u TEST_DB_URL \
  bash scripts/test-shops-isolated.sh \
  --verify-notification-migration --verify-message-sync-migration \
  --verify-remediation-migrations -v -count=1 \
  ./tests/shops ./tests/equipment_services
```

The wrapper applies the empty forward path; the sourced verification helper adds
a marked `miltech_test_allocator_bridge` database. It checks populated forward,
repeated refusal, interrupted reverse rollback, exact reverse, source/security/
trigger/data drift refusals, explicit fixture repairs, and reapply. JSON row
fingerprints prove refused operations and successful replacement retain messages
and counters. A deleted number stays reserved: the next insert is 3, even when
only message 1 remains, and a caller-supplied 999 is overwritten.

Every schema or fixture repair stage regenerates from the verified loopback
identity using the tagged generator into a fresh isolated output; the 32 tracked
PMCS references must match and generated packages compile at each stage. The full
available candidate is built before serialized supported integration packages.
The LIN fixture is documented supplemental schema and TMDE is an explicitly
synthetic empty compile dependency; neither certifies an actual target schema.

Existing named databases, deployment, fleet cutover, startup artifacts, and
capability activation require their later explicit gates. This migration does not
activate a capability or certify a release.
