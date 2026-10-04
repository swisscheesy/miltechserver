# Shops message sync: measurements

Measured 2026-09-29 for migration `018_add_shop_message_insertion_numbers.sql` and the four sync reader queries in `api/shops/messages/sync_repository.go`.

- Environment: disposable PostgreSQL 14.18 (Homebrew, aarch64 macOS laptop), local Unix socket, default `postgresql.conf` from `initdb`, warm cache, one client at a time. Baseline schema (`tests/testutil/shops_schema_baseline.sql`) plus migrations 016 and 017, then 018.
- These numbers show shape and order of magnitude. They are **not** production numbers: see [What was not measured](#what-was-not-measured).

## Query plans

Reproduce: `scripts/test-shops-isolated.sh ./tests/shops -run 'MessageSyncPlan' -count=1 -v` (skipped by `-short`). The test seeds two shops with 10,000 messages each (trigger-numbered, one second apart), runs `ANALYZE public.shop_messages`, then runs `EXPLAIN (ANALYZE, BUFFERS)` on the reader's SQL with concrete literals and fails on any `Seq Scan on shop_messages`. Default planner settings: no `enable_seqscan` or other planner switch was changed.

Plan nodes and execution time, trimmed (every plan also does `Index Scan using users_pkey` for the author-name join, memoized). Execution times are the range over three consecutive runs of the plan test:

- **initial** (newest 51 of one shop): `Index Scan using idx_shop_messages_shop_created_id`, `Index Cond: (shop_id = 'plan-a')`, no sort node. Execution 0.027 to 0.030 ms.
- **history** (older than a mid-table `(created_at, id)` anchor): `Index Scan using idx_shop_messages_shop_created_id`, `Index Cond: (shop_id = … AND ROW(created_at, id) < ROW(…))`, no sort node. Execution 0.026 to 0.029 ms.
- **catch-up, tail** (`insertion_number > 9950 AND <= 10000`): `Index Scan using shop_messages_shop_insertion_number_key`, both bounds in the `Index Cond`. Execution 0.021 to 0.026 ms.
- **catch-up, first cycle from zero** (`> 0 AND <= 10000`, `LIMIT 101`): `Index Scan using shop_messages_shop_insertion_number_key`, stops after 101 rows. Execution 0.041 to 0.043 ms.
- **reconcile** (100 IDs): see the finding below. Execution 0.656 to 0.687 ms over the three quoted runs; the range recorded for this plan in the test cluster (Task 6 report) is 0.69 to 0.87 ms, so treat 0.87 ms as the worst case observed there.

No plan used `Seq Scan on shop_messages`. `idx_shop_messages_shop_created_id` and `shop_messages_shop_insertion_number_key` from migration 018 are used exactly as intended, so no index change was made to 018.

### Finding: reconcile does not always use `shop_messages_pkey`

With the acceptance seed (two shops x 10,000), the planner chose the older `idx_shop_messages_shop_id` and filtered by ID:

```
Sort  (rows=100)
  Sort Key: m.created_at DESC, m.id DESC
  ->  Index Scan using idx_shop_messages_shop_id on shop_messages m
        Index Cond: (shop_id = 'plan-a')
        Filter: (id = ANY ('{…100 ids…}'))
        Rows Removed by Filter: 9900
Execution Time: 0.656 to 0.687 ms   (three runs)
```

In a scratch cluster (same query, 100 IDs; the other shop fixed at 10,000 rows; a shop of the size below; three runs each) the choice flips with the shop's message count:

- 5,000 messages: `Bitmap Index Scan on idx_shop_messages_shop_id`, 0.374 to 0.440 ms.
- 7,500 messages: `Bitmap Index Scan on idx_shop_messages_shop_id`, 0.574 to 0.647 ms.
- 10,000, 12,000, 15,000, 20,000, 40,000 and 100,000 messages: `Bitmap Index Scan on shop_messages_pkey`, 0.180 to 0.307 ms overall (10,000: 0.185 to 0.212; 100,000: 0.249 to 0.307).
- The 10,000-message case used the pkey in the scratch cluster but `idx_shop_messages_shop_id` in the plan test's cluster (which also differs in how the rows were inserted), so 10,000 sits at the planner's cost crossover and the exact point depends on the data.

Conclusion: this is a planner cost decision between two index-bounded plans, not a Seq Scan. The worst case observed is 0.87 ms per 100-ID chunk (test cluster, `idx_shop_messages_shop_id` plan; 0.69 to 0.87 ms). Other data distributions and shop sizes between the ones above were not tried, so the crossover point itself is not established.

An extra `(shop_id, id)` index was tried in the scratch cluster and not added. At 7,500 messages it took reconcile from 0.556 to 0.615 ms down to 0.169 to 0.209 ms (about 0.4 ms saved per chunk); the index was 1,184 kB at 17,500 table rows and 7,312 kB at 110,000 rows, built in 74 ms at 110,000 rows. It would be an eighth index on every message insert. The gain is small, so it was left out; if reconcile ever shows up in slow-query logs, revisit with a follow-up migration rather than changing 018 (whose checksum is pinned for live application). The plan test therefore accepts either index for reconcile and rejects a Seq Scan.

## Migration 018 cost on 100,000 rows

Method (the scratch scripts are not committed): a cluster built by hand the way `scripts/test-shops-isolated.sh` does it (`initdb`, `CREATE EXTENSION pg_trgm WITH SCHEMA public`, restore the baseline with its `CREATE SCHEMA public;` line removed, apply 016 and 017), one user, 100 shops, then 100,000 rows inserted with `INSERT ... SELECT ... FROM generate_series(1,100000)` (`gen_random_uuid()::text` IDs, `created_at` 20 s apart, a message like `Vehicle 12 needs the PMCS follow-up before Thursday, parts are on order (N)`, about 80 characters), then `ANALYZE` (33 MB table plus indexes). Two probes ran during the migration, started 5 s earlier: `pgbench -n -c 1 -R 100 -T 20 -l` with a script `SELECT count(*) FROM shop_messages WHERE id = (SELECT id FROM shop_messages LIMIT 1)` and another `pgbench` inserting one `'probe'` message per transaction, both at 100 transactions per second. The migration file was then run through `psql` with `\timing on`. The probe inserts add 506 to 518 rows before the migration starts, so the backfill saw 100,506 to 100,518 rows.

Per-statement time, three runs of the current migration (one transaction; `LOCK TABLE public.shops` is the third statement and `LOCK TABLE public.shop_messages`, which takes `ACCESS EXCLUSIVE`, is the fourth):

- Backfill `UPDATE` (window function over all rows): 887 to 899 ms.
- Counter seed `INSERT` (100 shops): 14.7 to 15.1 ms.
- `ALTER TABLE ... ADD CONSTRAINT ... FOREIGN KEY` on the counter table: 0.45 to 0.48 ms.
- `CREATE UNIQUE INDEX shop_messages_shop_insertion_number_key`: 86.6 to 92.8 ms.
- `CREATE INDEX idx_shop_messages_shop_created_id`: 108.5 to 111.8 ms.
- Everything else (NULL check, `ADD COLUMN`, `CREATE TABLE`, function, trigger, commit): each under 2 ms. Raw per-statement backing for "under 2 ms" exists only for the 450,000-row run; the 100,000-row runs recorded the statements listed above and the totals, not these small ones individually.
- **Total lock hold** (from the `shop_messages` lock to `COMMIT`, sum of the statements above): about 1.1 s. Wall time of the whole `psql` process including startup: 1.137 to 1.146 s.

Observed impact on the concurrent probes (three runs): the worst read took 1,106 to 1,110 ms and the worst insert 1,099 to 1,112 ms, against medians of 2.5 to 2.7 ms (reads) and 2.7 to 2.9 ms (inserts). Both were blocked for essentially the whole transaction. The rollback (`018_rollback_…`) on the same data: every statement under 0.5 ms.

At 450,000 rows (154 MB, one run, same method): backfill 5,581 ms, counter seed 80 ms, the two index builds 404 ms and 560 ms, `psql` wall time 6.68 s. Across the two sizes, about 4.5 times the rows took about 6 times the wall time (1.14 s to 6.68 s, one run at the larger size). Two data points do not establish a growth shape, so no extrapolation beyond 450,000 rows is offered.

What this means:

- The migration takes `ACCESS EXCLUSIVE` on `shop_messages`, so **reads and writes of messages block until commit**, in both the legacy and sync endpoints. It also holds `SHARE ROW EXCLUSIVE` on `shops`, which blocks writes to `shops` until commit; that part follows from PostgreSQL's lock rules and was not probed with a writer arriving during the hold.
- `SET LOCAL lock_timeout = '5s'` only bounds the wait to acquire each lock. Once acquired, the lock is held for the whole transaction. This is a short write-and-read-blocking window, not a zero-downtime change.
- The hold time grows with the number of rows in `shop_messages`. At 100,000 rows it is dominated by the backfill (about 0.9 ms per 100 rows) and the two index builds (about 0.2 ms per 100 rows combined). Growth beyond 450,000 rows was not measured; the backfill sort may spill to disk on a much larger table. Check the real production row count first.

### Lock ordering and contention

The 1.1 s hold above was measured **without any contention on `shops`**: no open transactions, no shop updates or deletes. Contended behaviour was probed separately (2026-09-29, same kind of scratch cluster, `deadlock_timeout` at the 1 s default). Three separate problems were found and fixed in review; each has a probe.

**1. Lock order against a cascading shop delete.** The first version locked `shop_messages` (`ACCESS EXCLUSIVE`) and only later, at the counter table's foreign key, asked for `SHARE ROW EXCLUSIVE` on `shops`. A cascading `DELETE FROM shops` locks `shops` first and `shop_messages` second, the reverse order. Probes (`lock_timeout` 3 s so they finish quickly; 100,000-row cluster):

- Deadlock probe. Session A: `BEGIN; UPDATE shops SET name=name WHERE id='shop-2'; SELECT pg_sleep(2); DELETE FROM shops WHERE id='shop-x'; COMMIT;` (`shop-x` has one message). Session B started 1 s after A and ran the migration.
  - Old order: `deadlock detected` after 1.05 s, B aborted, A committed, migration rolled back.
  - Current order (`shops` first): B waited about 1 s for A's table-level `ROW EXCLUSIVE` on `shops` (it conflicts with `SHARE ROW EXCLUSIVE`), then acquired both locks and committed; 2.11 s total.
- Blocked-reader probe. Session A held an open `UPDATE` on `shops` for 8 s; B ran the migration; a `pgbench` reader (100 transactions per second, single-row read of `shop_messages`) ran throughout.
  - Old order: B timed out after 3.04 s **while already holding `ACCESS EXCLUSIVE` on `shop_messages`**; the reader's worst transaction took 2,999 ms (median 6.5 ms). With the real `lock_timeout` of 5 s the wait would block message reads and writes for up to 5 s before the hold even starts.
  - Current order: B timed out after 3.04 s waiting for `shops` without having locked `shop_messages`; the reader's worst transaction took 10.5 ms (median 2.6 ms).

**2. The counter seed's foreign key checks (found by the second review).** Every application mutation first runs `shared.LockShopMutation` (`SELECT ... FROM shops WHERE id = $1 FOR UPDATE`: a `ROW SHARE` table lock and a row lock), then its statement. With the foreign key declared inline on `shop_message_counters`, the seed `INSERT` took `FOR KEY SHARE` on every `shops` row, waiting on the application's row lock, while the application's next statement waited on the migration's table lock: a deadlock. This existed in the original 018 and in the lock-order fix; the lock-order fix did not remove it. The fix: create the counter table without the foreign key, seed it, then `ADD CONSTRAINT ... FOREIGN KEY ... ON DELETE CASCADE` (same definition; validation is one join under the locks already held).

Probe: the application session begins 0.5 s before the migration, runs `SELECT id FROM shops WHERE id='shop-x' FOR UPDATE`, sleeps 1.0 s, then one statement; the migration keeps its real `lock_timeout` of 5 s. (Version "before" is the lock-order fix, commit c1f85ac; "after" is the current file.)

- (1) `INSERT INTO shop_messages ...` (what `CreateShopMessage` does): before, `deadlock detected`, the application transaction was the victim and the migration committed (1.7 s). After, no deadlock: the insert waited for the migration, both committed (migration 1.22 s), and the message was numbered 2 after the backfilled message 1.
- (2) `DELETE FROM shops WHERE id='shop-x'`: before, `deadlock detected` (application victim, migration committed 1.71 s). After: both committed (1.17 s), the shop and its counter and messages were gone.
- (3) `UPDATE shops SET name=... WHERE id='shop-x'`: before, `deadlock detected` (application victim, migration committed 1.71 s). After: both committed (1.17 s), the shop was renamed.
- Variant with the application statement at 1.6 s (after the seed started waiting), pattern (1): before, the migration was the victim (`deadlock detected` at the seed `INSERT`, rolled back, 1.97 s, column absent). After: both committed.
- Large table: 450,001 rows (154 MB), pattern (1), the message post arriving 0.5 s into the migration, during the 5.6 s backfill: no deadlock and no timeout; the migration committed (6.68 s), and the application's insert, blocked behind the migration's `ACCESS EXCLUSIVE` lock, completed when it committed (application session total 7.21 s, so the message post waited roughly 6 s). It was numbered 2.

So which victim the deadlock detector picks depends on who started waiting first; either way the outcome before the fix was an aborted transaction.

**3. Residual, synthetic cycle (not fixable by lock order).** Any application transaction that **touches** `shop_messages` first (even a plain `SELECT`, which holds `ACCESS SHARE` until commit) and then writes `shops` (the reverse of a cascading delete) still deadlocks with the migration: the migration holds `SHARE ROW EXCLUSIVE` on `shops` while waiting for `ACCESS EXCLUSIVE` on `shop_messages`, and the application waits for `ROW EXCLUSIVE` on `shops`. Probe result (with a message write as the first step): `deadlock detected`, the migration was the victim and rolled back, the application transaction committed. The read-first variant follows from the lock-conflict rules and was not separately probed. No such path was found in the application code checked: message create, edit and delete (`api/shops/messages/repository_impl.go`) never write `shops` after touching `shop_messages`; edit and delete read `shop_messages` first (`AuthorizeOwnedMutation`, `api/shops/shared/authorization.go`, about lines 343 to 353) and then lock the shop row. That is still safe because `LockShopMutation`'s `SELECT ... FOR UPDATE` takes only `ROW SHARE` on `shops`, which is compatible with the migration's `SHARE ROW EXCLUSIVE`, and the migration holds no `shops` row locks, so no row-lock wait cycle arises; the second condition is that neither writes `shops` afterwards. The settings, core and members repositories, the other `shops` writers, were checked and do not touch `shop_messages` before writing `shops`. This is a read of those files, not an exhaustive audit of all callers or of ad-hoc SQL. If it ever happens, the migration is the victim, is atomic and can be re-run.

Worst-case window, current file: up to `lock_timeout` (5 s) waiting for the `shops` lock, during which message reads and writes are unaffected; then up to 5 s waiting for the `shop_messages` lock, during which later readers and writers of `shop_messages` queue behind the pending request (standard PostgreSQL lock queueing; not probed); then the hold, about 1.1 s at 100,000 rows and about 6.7 s at 450,000. While the queued `SHARE ROW EXCLUSIVE` request on `shops` waits, later writers of `shops` queue behind it (lock queueing again, not probed). Application transactions that already hold a shops row lock wait for the whole hold and then proceed (probe results above). The migration is one transaction: if a wait times out or it is chosen as a deadlock victim it rolls back atomically (the probes above show the column absent afterwards) and is safe to re-run.

Lock modes observed by querying `pg_locks` inside an uncommitted run (`ROLLBACK` at the end), listing only the two strongest modes:

- Add migration: `shops` `ShareRowExclusiveLock`; `shop_messages` and `shop_message_counters` `AccessExclusiveLock` and `ShareRowExclusiveLock`. The foreign key needed nothing stronger than `SHARE ROW EXCLUSIVE` on `shops`.
- Rollback migration, statements run without the explicit locks: `DROP TABLE public.shop_message_counters` takes `AccessExclusiveLock` on `shops` (it removes the foreign key triggers there), and `DROP COLUMN`/`DROP INDEX` take `AccessExclusiveLock` on `shop_messages`. The rollback therefore locks `shops` in `ACCESS EXCLUSIVE` first. While that queued request waits (up to `lock_timeout`) it queues new readers of `shops` behind it; once granted it blocks reads and writes of `shops` for the few milliseconds the rollback runs. The rollback's contention behaviour was not probed.

## Client request cost per active cycle

Assumptions: the client, on opening a shop that already has loaded messages, runs one catch-up cycle plus reconcile of every loaded row in 100-ID chunks (the reconcile chunk limit). Requests = 1 (catch-up; more if more than 100 messages are new) + ceil(n / 100).

Bytes per row are measured from real response bodies (`SAMPLE` lines from `scripts/test-shops-isolated.sh ./tests/shops -run 'MessageSyncRoutes' -count=1 -v`, 108 rows, average 299.2): 285 to 300 bytes of JSON per row with test messages of 5 to 13 characters, so fixed overhead is about 287 bytes (two UUIDs, two timestamps, user ID, author name, keys) plus the message text. Message length in production is not known. The table shows the measured 300 bytes and a modelled 390 bytes (a 100-character message). An empty catch-up response is 98 bytes; a request body is about 39 bytes per ID (UUID with quotes and comma). All figures are uncompressed JSON; gzip was not measured.

- **50 loaded rows**: 2 requests (1 catch-up + 1 reconcile). Reconcile downloads about 15 KB (19.5 KB at 390 B/row), uploads about 2 KB.
- **500 loaded rows**: 6 requests (1 + 5). Downloads about 150 KB (195 KB), uploads about 20 KB.
- **2,000 loaded rows**: 21 requests (1 + 20). Downloads about 600 KB (780 KB), uploads about 78 KB.

Server-side query time for those requests is small: reconcile executes in 0.18 to 0.87 ms per chunk (see above), so 20 chunks is roughly 4 to 17 ms of query time (20 x the range). HTTP, JSON encoding, authorization and the per-request read-only transaction were not timed.

## What was not measured

- Production hardware, storage, `shared_buffers`, cold-cache behaviour, or the real `shop_messages` row count and message size distribution. Lock-hold time on the live database may differ substantially.
- Concurrent load. The probes were a single reader and a single writer at 100 transactions per second; contention among many clients (queued connections after the lock releases, pool exhaustion while the lock is held) was not tested.
- Migration cost above 450,000 rows (one run at 450,000 only), or with a pre-existing large `shops` table.
- Block on `shops` writes during the migration, and queueing of later `shops` or `shop_messages` requests behind the migration's pending lock requests (from lock semantics only).
- End-to-end request latency, compression, mobile network cost, or the cost of a large first-time backlog for a busy shop.
- Insert throughput impact of the trigger and two extra indexes.
- Table and index bloat and write-ahead log from the backfill. The `UPDATE` rewrites every row, so the table is expected to hold roughly twice its live size (old row versions) until autovacuum or `VACUUM` reclaims it, and to generate WAL of a similar order of size, which affects replicas, replication lag and backups. These are expectations from how PostgreSQL updates rows; neither the size growth nor the WAL volume was measured.
- Contention behaviour of the rollback migration, and the second lock wait (`shop_messages`) under contention with a long-running transaction.

## Separate aggregate measurements

Complete PMCS history capacity, payload, memory, and query plans are recorded in [Shops aggregate PMCS history capacity measurements](shops-aggregate-pmcs-history-measurements.md). Those results measure an aggregate history read and do not establish message sync throughput.

## Remediation interpretation — 2026-10-04

The measurements above describe the original disposable 018 experiments, not
the current fleet or live target. Migration 019 now bridges legacy inserts into
Shop-before-counter locking. Supported external writers must be inventoried and
any counter-first writers fenced before application; local lock regressions do
not prove fleet compatibility or predict live hold times. Numeric restore ABA
remains unresolved without an epoch protocol.

The physical `TestLegacyCursorCurrentWriterCommitOrderLimitation` records that a
current writer can commit behind an observed legacy timestamp cursor. This is
an **OPEN release limitation**, not lossless catch-up or an owner waiver.
Timestamp interpretation remains preserved pending an explicit owner decision.
Actual active invitation population, instance count, per-UID authenticated and
effective edge budgets remain unknown, so C06 is a release blocker. No measured
message-query latency establishes invite guessing safety. See the
[release gate sheet](shops-server-remediation-release.md).
