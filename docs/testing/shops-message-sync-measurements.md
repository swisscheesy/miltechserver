# Shops message sync: measurements

Measured 2026-09-29 for migration `018_add_shop_message_insertion_numbers.sql` and the four sync reader queries in `api/shops/messages/sync_repository.go`.

- Environment: disposable PostgreSQL 14.18 (Homebrew, aarch64 macOS laptop), local Unix socket, default `postgresql.conf` from `initdb`, warm cache, one client at a time. Baseline schema (`tests/testutil/shops_schema_baseline.sql`) plus migrations 016 and 017, then 018.
- These numbers show shape and order of magnitude. They are **not** production numbers: see [What was not measured](#what-was-not-measured).

## Query plans

Reproduce: `scripts/test-shops-isolated.sh ./tests/shops -run 'MessageSyncPlan' -count=1 -v` (skipped by `-short`). The test seeds two shops with 10,000 messages each (trigger-numbered, one second apart), runs `ANALYZE public.shop_messages`, then runs `EXPLAIN (ANALYZE, BUFFERS)` on the reader's SQL with concrete literals and fails on any `Seq Scan on shop_messages`. Default planner settings: no `enable_seqscan` or other planner switch was changed.

Plan nodes and timing, trimmed (every plan also does `Index Scan using users_pkey` for the author-name join, memoized):

- **initial** (newest 51 of one shop): `Index Scan using idx_shop_messages_shop_created_id`, `Index Cond: (shop_id = 'plan-a')`, no sort node. 51 rows, 7 buffers, execution 0.03 ms.
- **history** (older than a mid-table `(created_at, id)` anchor): `Index Scan using idx_shop_messages_shop_created_id`, `Index Cond: (shop_id = … AND ROW(created_at, id) < ROW(…))`, no sort node. 51 rows, 8 buffers, execution 0.04 ms.
- **catch-up, tail** (`insertion_number > 9950 AND <= 10000`): `Index Scan using shop_messages_shop_insertion_number_key`, both bounds in the `Index Cond`. 50 rows, 6 buffers, execution 0.03 ms.
- **catch-up, first cycle from zero** (`> 0 AND <= 10000`, `LIMIT 101`): `Index Scan using shop_messages_shop_insertion_number_key`, stops after 101 rows. 7 buffers, execution 0.05 ms.
- **reconcile** (100 IDs): see the finding below.

No plan used `Seq Scan on shop_messages`. `idx_shop_messages_shop_created_id` and `shop_messages_shop_insertion_number_key` from migration 018 are used exactly as intended, so no index change was made to 018.

### Finding: reconcile does not always use `shop_messages_pkey`

With the acceptance seed (two shops x 10,000), the planner chose the older `idx_shop_messages_shop_id` and filtered by ID:

```
Sort  (rows=100, actual time=0.677..0.679)
  Sort Key: m.created_at DESC, m.id DESC
  ->  Index Scan using idx_shop_messages_shop_id on shop_messages m
        Index Cond: (shop_id = 'plan-a')
        Filter: (id = ANY ('{…100 ids…}'))
        Rows Removed by Filter: 9900
Execution Time: 0.685 ms   (259 buffers)
```

In a scratch cluster with the same query but different shop sizes (other shop fixed at 10,000 rows; separate 100-ID chunk each time), the choice flips with the shop's message count:

- 5,000 messages in the shop: bitmap scan on `idx_shop_messages_shop_id`, 0.42 ms.
- 10,000 to 100,000 messages (10k, 12k, 15k, 20k, 40k, 100k): `Bitmap Index Scan on shop_messages_pkey`, 0.20 to 0.29 ms.
- The 10,000 case landed on the pkey in the scratch cluster but on `idx_shop_messages_shop_id` in the test's cluster, so it sits right at the planner's cost crossover.

Conclusion: this is a planner cost decision between two index-bounded plans, not a Seq Scan. The worst case observed is 0.69 to 0.87 ms per 100-ID chunk at a 10,000-message shop; in every larger shop size tried (12k to 100k) the planner used the pkey at 0.21 to 0.29 ms. Sizes between 5,000 and 10,000 and other data distributions were not tried, so the crossover point itself is not established. An extra `(shop_id, id)` index was tried in the scratch cluster (0.24 ms at 10,000; 7.3 MB at 100,000 rows) and rejected: it saves roughly 0.5 ms per chunk at the crossover, and adds an eighth index to every message insert plus roughly 100 ms of migration time (an estimate from the two index builds already measured, not a measurement of this index). If reconcile ever shows up in slow-query logs, revisit with a follow-up migration; do not change 018 (its checksum is pinned for live application). The plan test therefore accepts either index for reconcile and rejects a Seq Scan.

## Migration 018 cost on 100,000 rows

Method: a scratch cluster built by hand the way `scripts/test-shops-isolated.sh` does it (`initdb`, `CREATE EXTENSION pg_trgm WITH SCHEMA public`, restore the baseline with its `CREATE SCHEMA public;` line removed, apply 016 and 017), one user, 100 shops, then 100,000 rows inserted with `INSERT ... SELECT ... FROM generate_series(1,100000)` (`gen_random_uuid()::text` IDs, `created_at` 20 s apart, a message like `Vehicle 12 needs the PMCS follow-up before Thursday, parts are on order (N)`), then `ANALYZE`. The scripts are not committed; the probes were `pgbench -n -c 1 -R 100 -T 20 -l` with one script `SELECT count(*) FROM shop_messages WHERE id = (SELECT id FROM shop_messages LIMIT 1)` and one inserting a single `'probe'` message, started 5 s before the migration file was run through `psql` with `\timing on`. Result: 100 shops x about 1,000 messages (33 MB table plus indexes, about 90 bytes of message text per row), all rows numbered by the backfill; both probes ran at 100 transactions per second during the migration. About 500 probe rows were inserted before the migration started, so the backfill saw 100,470 to 100,532 rows.

Per-statement time (four runs; the migration is one transaction, and the `ACCESS EXCLUSIVE` lock is taken by its third statement):

- Backfill `UPDATE` (window function over all rows): 891 to 924 ms.
- Counter seed `INSERT` (100 shops): 15 to 16 ms.
- `CREATE UNIQUE INDEX shop_messages_shop_insertion_number_key`: 89 to 97 ms.
- `CREATE INDEX idx_shop_messages_shop_created_id`: 113 to 119 ms.
- Everything else (NULL check, `ADD COLUMN`, `CREATE TABLE`, function, trigger, commit): under 3 ms combined.
- **Total lock hold** (lock acquisition to `COMMIT`, sum of the statements above): about 1.1 to 1.2 s. Wall time of the whole `psql` process including startup: 1.14 to 1.19 s.

Observed impact on the concurrent probes (max latency of any probe transaction, four runs): reads 1,112 to 1,156 ms, inserts 1,110 to 1,140 ms, against a median of about 2.5 ms. Both were blocked for essentially the whole transaction. The rollback (`018_rollback_…`) on the same data: every statement under 10 ms.

What this means:

- The migration takes `ACCESS EXCLUSIVE` on `shop_messages`, so **reads and writes of messages block until commit**, in both the legacy and sync endpoints. It also holds `SHARE ROW EXCLUSIVE` on `shops` (foreign key of the new counter table), which blocks writes to `shops` until commit; that part is from PostgreSQL's lock rules and was not probed.
- `SET LOCAL lock_timeout = '5s'` only bounds the wait to acquire the lock. Once acquired, the lock is held for the whole transaction. This is a short write-and-read-blocking window, not a zero-downtime change.
- The hold time grows with the number of rows in `shop_messages`. At 100,000 rows it is dominated by the backfill (about 0.9 ms per 100 rows) and the two index builds (about 0.2 ms per 100 rows). Growth beyond 100,000 rows was not measured; the backfill sort may spill to disk on a much larger table. Check the real production row count first.

### Lock ordering and contention

The 1.1 to 1.2 s hold above was measured **without any contention on `shops`**: no open transactions, no shop updates or deletes. Contended behaviour was probed separately (2026-09-29, same scratch cluster, `lock_timeout` set to 3 s instead of 5 s so probes finish sooner, `deadlock_timeout` at the 1 s default).

Review finding, confirmed. The first version of 018 locked `shop_messages` (`ACCESS EXCLUSIVE`) and only later, at the counter table's foreign key, asked for `SHARE ROW EXCLUSIVE` on `shops`. A cascading `DELETE FROM shops` locks `shops` first and `shop_messages` second, the reverse order. Two probes, before (old order) and after (`shops` first, then `shop_messages`, as committed now):

- **Deadlock probe.** Session A: `BEGIN; UPDATE shops SET name=name WHERE id='shop-2'; SELECT pg_sleep(2); DELETE FROM shops WHERE id='shop-x'; COMMIT;` where `shop-x` has one message. Session B started 1 s after A began and ran the migration file.
  - Old order: B locked `shop_messages`, then waited on A's row-level lock on `shops`; when A's delete cascaded into `shop_messages` it waited on B. After 1.05 s PostgreSQL raised `deadlock detected` and aborted B; A committed; the migration rolled back and the column was absent.
  - New order: B waits for the `shops` lock before touching `shop_messages`, A's cascade proceeds, A commits, B then acquires both locks and commits. Total 2.13 s (about 1 s of waiting plus the hold); the shop was deleted and the column was present.
- **Blocked-reader probe.** Session A held `ROW EXCLUSIVE` on `shops` (an open `UPDATE`) for 8 s; B ran the migration with a 3 s timeout; a `pgbench` reader (100 transactions per second, single-row read of `shop_messages`) ran throughout.
  - Old order: B timed out after 3.03 s **while already holding `ACCESS EXCLUSIVE` on `shop_messages`**; the reader's worst transaction took 2,999 ms (median 6 ms). With the real `lock_timeout` of 5 s the same wait would block message reads and writes for up to 5 s before the 1.1 s hold even starts.
  - New order: B timed out after 3.03 s waiting for `shops` and had not locked `shop_messages`; the reader's worst transaction took 10 ms (median 2.3 ms).

Worst-case window, new order: up to `lock_timeout` (5 s) waiting for the `shops` lock, during which messages are unaffected; then up to 5 s more waiting for the `shop_messages` lock, during which readers and writers of `shop_messages` queue behind the pending request in PostgreSQL's lock queue (this second wait was not probed and is unchanged from before); then the hold of about 1.1 s at 100,000 rows. The counter seed (`INSERT ... SELECT` with foreign key checks) can also wait on row locks that open transactions hold on `shops` rows; `lock_timeout` applies to each such wait as well, but this path was not probed. Writes to `shops` are blocked from the first `LOCK TABLE` until `COMMIT` and reads of `shops` continue: that follows from PostgreSQL lock semantics (`SHARE ROW EXCLUSIVE`) and was not probed with a writer arriving while the migration holds the lock (the probes above only covered a writer that held its lock first).

The migration is one transaction. If a lock wait times out or it is chosen as the deadlock victim it rolls back atomically (verified above: column absent, data untouched) and is safe to re-run.

Lock modes observed by querying `pg_locks` inside an uncommitted run (`ROLLBACK` at the end):

- Add migration: `shops` `ShareRowExclusiveLock`, `shop_messages` `AccessExclusiveLock` (both as taken by the explicit `LOCK TABLE`; the foreign key needed no stronger lock on `shops`).
- Rollback migration, statements run without the explicit locks: `DROP TABLE public.shop_message_counters` takes `AccessExclusiveLock` on `shops` (it removes the foreign key triggers there), and `DROP COLUMN`/`DROP INDEX` take `AccessExclusiveLock` on `shop_messages`. The rollback therefore locks `shops` in `ACCESS EXCLUSIVE` first, blocking reads of `shops` for the few milliseconds it runs. The rollback's contention behaviour was not probed.

Uncontended timings were re-measured with the new lock order (two runs): backfill 891 to 924 ms, indexes 85 to 97 ms and 111 to 120 ms, wall 1.14 to 1.19 s, probe max latency about 1.06 to 1.16 s for both reads and inserts, so the reorder did not change the cost.

## Client request cost per active cycle

Assumptions: the client, on opening a shop that already has loaded messages, runs one catch-up cycle plus reconcile of every loaded row in 100-ID chunks (the reconcile chunk limit). Requests = 1 (catch-up; more if more than 100 messages are new) + ceil(n / 100).

Bytes per row are measured from real response bodies (`SAMPLE` lines from `scripts/test-shops-isolated.sh ./tests/shops -run 'MessageSyncRoutes' -count=1 -v`, 108 rows): 285 to 300 bytes of JSON per row with test messages of 5 to 13 characters, so fixed overhead is about 287 bytes (two UUIDs, two timestamps, user ID, author name, keys) plus the message text. Message length in production is not known. The table shows the measured 300 bytes and a modelled 390 bytes (a 100-character message). An empty catch-up response is 98 bytes; a request body is about 39 bytes per ID (UUID with quotes and comma). All figures are uncompressed JSON; gzip was not measured.

- **50 loaded rows**: 2 requests (1 catch-up + 1 reconcile). Reconcile downloads about 15 KB (19.5 KB at 390 B/row), uploads about 2 KB.
- **500 loaded rows**: 6 requests (1 + 5). Downloads about 150 KB (195 KB), uploads about 20 KB.
- **2,000 loaded rows**: 21 requests (1 + 20). Downloads about 600 KB (780 KB), uploads about 78 KB.

Server-side query time for those requests is small: reconcile executes in 0.2 to 0.7 ms per chunk (see above), so 20 chunks is roughly 4 to 14 ms of query time. HTTP, JSON encoding, authorization and the per-request read-only transaction were not timed.

## What was not measured

- Production hardware, storage, `shared_buffers`, cold-cache behaviour, or the real `shop_messages` row count and message size distribution. Lock-hold time on the live database may differ substantially.
- Concurrent load. The probes were a single reader and a single writer at 100 transactions per second; contention among many clients (queued connections after the lock releases, pool exhaustion while the lock is held) was not tested.
- Migration cost above 100,000 rows, or with a pre-existing large `shops` table.
- Block on `shops` writes during the migration (from lock semantics only).
- End-to-end request latency, compression, mobile network cost, or the cost of a large first-time backlog for a busy shop.
- Insert throughput impact of the trigger and two extra indexes.
- Table and index bloat and write-ahead log from the backfill. The `UPDATE` rewrites every row, so the table holds roughly twice its live size in dead tuples until autovacuum (or `VACUUM`) reclaims it, and generates WAL of a similar order of size, which affects replicas, replication lag and backups. Neither the size growth nor the WAL volume was measured.
- Contention behaviour of the rollback migration, and the second lock wait (`shop_messages`) under contention.
