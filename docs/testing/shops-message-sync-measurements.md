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

Conclusion: this is a planner cost decision between two index-bounded plans, not a Seq Scan. Worst case observed is 0.69 ms per 100-ID chunk at a 10,000-message shop; cost grows only with the shop's message count on the `shop_id` path, and the planner moves to the pkey path before that becomes larger. An extra `(shop_id, id)` index was tried in the scratch cluster (0.24 ms at 10,000; 7.3 MB at 100,000 rows) and rejected: it saves roughly 0.5 ms per chunk at the crossover, and adds an eighth index to every message insert plus about 100 ms of migration time. If reconcile ever shows up in slow-query logs, revisit with a follow-up migration; do not change 018 (its checksum is pinned for live application). The plan test therefore accepts either index for reconcile and rejects a Seq Scan.

## Migration 018 cost on 100,000 rows

Method: scratch cluster (not the repo's scripts), 100 shops x about 1,000 messages (33 MB table plus indexes, about 90 bytes of message text per row), all rows numbered by the backfill. `psql` with `\timing on` applied the file. Two `pgbench` probes ran at 100 transactions per second during the migration: one single-row read, one single-row insert into `shop_messages`. About 500 probe rows were inserted before the migration started, so the backfill saw 100,470 to 100,532 rows.

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
