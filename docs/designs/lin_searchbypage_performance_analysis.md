# LIN SearchByPage Performance Analysis

**Analysis Date**: February 15, 2026
**Reported Issue**: Users complaining of slow query speeds from mobile app when loading the LIN page
**Affected Endpoint**: `SearchByPage` in `api/item_lookup/lin/repository_impl.go`
**Database**: PostgreSQL 14.18

---

## Executive Summary

The `SearchByPage` method has **three compounding performance problems** that together create a poor mobile experience:

1. **The COUNT query is the dominant bottleneck** - taking 114ms to perform a full nested-loop join across 18,204 rows on every single page request, even though the count rarely changes.
2. **OFFSET-based pagination degrades linearly** - page 1 takes 0.28ms, but the last page takes 271ms (968x slower).
3. **Two separate database roundtrips** per request double the network latency impact for mobile users on high-latency connections.

Combined worst case: a user on page 900 on mobile experiences **~270ms query time + 114ms count + 2x network RTT**, easily exceeding 500ms+ total response time.

---

## Architecture Overview

### Current Data Flow

```
Mobile App → API (SearchByPage) → 2 SQL Queries → View (lookup_lin_niin) → JOIN(nsn, army_lin_to_niin)
```

### View Definition

```sql
-- public.lookup_lin_niin
SELECT nsn.niin, nsn.item_name, army_lin_to_niin.lin
FROM nsn, army_lin_to_niin
WHERE nsn.niin::text = army_lin_to_niin.niin::text;
```

This is an **implicit inner join** between two tables:

| Table | Rows | Size | Primary Key |
|-------|------|------|-------------|
| `nsn` | 7,372,398 | 698 MB | `niin` (btree) |
| `army_lin_to_niin` | 18,204 | 928 KB | `niin` (btree) |

The view produces **18,204 rows** (one per `army_lin_to_niin` row that matches an `nsn` record).

### Go Code (repository_impl.go)

```go
func (repo *RepositoryImpl) SearchByPage(page int) (response.LINPageResponse, error) {
    // Query 1: Fetch page of data
    stmt := SELECT(view.LookupLinNiin.AllColumns).
        FROM(view.LookupLinNiin).
        LIMIT(20).OFFSET(offset)

    // Query 2: Count total rows (runs EVERY request)
    countStmt := SELECT(COUNT(view.LookupLinNiin.Lin)).
        FROM(view.LookupLinNiin)
}
```

**Problems identified in the code:**
- No `ORDER BY` clause - pagination results are **non-deterministic**
- Count query runs on every request despite data being near-static
- Two separate DB roundtrips

---

## Detailed Findings

### Finding 1: COUNT Query is the Primary Bottleneck

**EXPLAIN ANALYZE** for `SELECT COUNT(lin) FROM lookup_lin_niin`:

```
Aggregate  (cost=63675.35..63675.36 rows=1 width=8) (actual time=114.452..114.454 rows=1 loops=1)
  Buffers: shared hit=56906
  ->  Nested Loop  (cost=0.43..63629.84 rows=18204 width=7) (actual time=0.039..113.049 rows=18204 loops=1)
        Buffers: shared hit=56906
        ->  Seq Scan on army_lin_to_niin  (cost=0.00..298.04 rows=18204 width=17)
              (actual time=0.029..1.271 rows=18204 loops=1)
              Buffers: shared hit=116
        ->  Index Only Scan using nsn_pkey on nsn  (cost=0.43..3.48 rows=1 width=10)
              (actual time=0.006..0.006 rows=1 loops=18204)
              Heap Fetches: 0
              Buffers: shared hit=56790
  Execution Time: 114.472 ms
```

**Analysis:**
- The COUNT forces PostgreSQL to **join all 18,204 rows** to count them
- It performs 18,204 index lookups on the 698 MB `nsn` table
- Reads **56,906 buffer pages** (each 8 KB = ~444 MB of buffer access)
- This runs on **every single page request** even though the count is always ~18,204
- The count value changes only when the underlying `army_lin_to_niin` data is updated (very rarely - this is reference data)

**Impact**: 114ms of pure waste on every request.

---

### Finding 2: OFFSET Pagination Degrades Linearly

| Page | OFFSET | Execution Time | Buffer Reads | Notes |
|------|--------|---------------|--------------|-------|
| 1 | 0 | 0.28ms | 81 | Fast - reads only 20 rows |
| 45 | 900 | 19ms | 3,686 | 68x slower |
| 900 | 18,000 | 271ms | 72,195 | 968x slower, triggers JIT |

**Why OFFSET is slow:**
PostgreSQL must scan and discard all rows before the offset. For `OFFSET 18000`, the database:
1. Sequential scans all 18,204 rows from `army_lin_to_niin`
2. Performs 18,020 index lookups on `nsn` (698 MB table)
3. Reads 72,195 buffer pages (disk I/O)
4. Discards 18,000 joined rows
5. Returns only 20 rows

At high offsets, PostgreSQL also triggers **JIT compilation** (6.5ms overhead), adding further latency.

With `DefaultPageSize = 20` and 18,204 total rows, there are **911 possible pages**. Users navigating to later pages will experience significantly degraded performance.

---

### Finding 3: Missing ORDER BY Creates Non-Deterministic Pagination

The paginated query has **no ORDER BY clause**:

```sql
SELECT niin, item_name, lin FROM lookup_lin_niin LIMIT 20 OFFSET 0
```

Without ORDER BY, PostgreSQL returns rows in **arbitrary order** that can change between executions due to:
- Concurrent vacuuming
- Table updates
- Buffer cache state
- Parallel query execution

**Impact**: Users may see duplicate items or miss items entirely when paginating. This is a correctness bug, not just a performance issue.

---

### Finding 4: No Foreign Key Constraint Between Tables

There is **no foreign key** from `army_lin_to_niin.niin` to `nsn.niin`:

```
army_lin_to_niin:
  - PK: niin (btree)
  - Index: lin (btree)
  - NO FK to nsn

nsn:
  - PK: niin (btree)
  - NO FK references from army_lin_to_niin
```

**Impact on query planner**: Without FK metadata, PostgreSQL cannot make assumptions about join selectivity or use certain join optimizations. The planner cannot know that every `army_lin_to_niin.niin` has a matching `nsn.niin`.

---

### Finding 5: Database Configuration Amplifies the Problem

| Setting | Value | Concern |
|---------|-------|---------|
| `shared_buffers` | 160 MB | `nsn` table alone is 698 MB - most lookups hit OS cache or disk |
| `work_mem` | 4 MB | Adequate for this join size |
| `effective_cache_size` | 5 GB | Reasonable |
| Cache hit ratio | 87% | Below the recommended 99%+ for production |

The `nsn` table (698 MB) plus its index cannot fit in `shared_buffers` (160 MB). This means higher-offset queries that touch many `nsn` index pages are more likely to incur disk I/O, especially under concurrent load.

---

### Finding 6: varchar Comparison in Join Condition

The view join uses an implicit cast:

```sql
WHERE (nsn.niin)::text = (army_lin_to_niin.niin)::text
```

Both columns are `character varying`. The `::text` cast is generated by PostgreSQL for the comparison. While the btree index on `nsn.niin` **does work** for this comparison (confirmed by EXPLAIN showing Index Scan), this is worth noting as a minor inefficiency compared to matching types.

---

## Performance Summary

### Per-Request Cost Breakdown (Page 1)

| Component | Time | % of Total |
|-----------|------|-----------|
| Data query (LIMIT 20 OFFSET 0) | 0.28ms | 0.2% |
| COUNT query | 114.5ms | 99.8% |
| **Total DB time** | **114.8ms** | |
| Network RTT (mobile, 2 roundtrips) | ~100-300ms | Variable |
| **Total perceived latency** | **~215-415ms** | |

### Per-Request Cost Breakdown (Last Page)

| Component | Time | % of Total |
|-----------|------|-----------|
| Data query (LIMIT 20 OFFSET 18000) | 271.6ms | 70.3% |
| COUNT query | 114.5ms | 29.7% |
| **Total DB time** | **386.1ms** | |
| Network RTT (mobile, 2 roundtrips) | ~100-300ms | Variable |
| **Total perceived latency** | **~486-686ms** | |

---

## Proposed Fixes (Ranked by Impact)

### Fix 1: Replace View with Materialized View [HIGH IMPACT, LOW EFFORT]

**Problem solved**: COUNT query bottleneck + JOIN cost on every request

Create a materialized view that pre-computes the join result. This eliminates the nested-loop join from every query and allows direct indexing on the materialized data.

```sql
CREATE MATERIALIZED VIEW lookup_lin_niin_mat AS
SELECT nsn.niin, nsn.item_name, army_lin_to_niin.lin
FROM nsn
INNER JOIN army_lin_to_niin ON nsn.niin = army_lin_to_niin.niin
ORDER BY army_lin_to_niin.lin, nsn.niin;

-- Add indexes for pagination and search
CREATE UNIQUE INDEX idx_lookup_lin_niin_mat_niin ON lookup_lin_niin_mat (niin);
CREATE INDEX idx_lookup_lin_niin_mat_lin ON lookup_lin_niin_mat (lin);
```

**Benefits:**
- Data query becomes a simple sequential scan on a ~1 MB table (18,204 rows x ~37 bytes)
- COUNT becomes `SELECT COUNT(*) FROM lookup_lin_niin_mat` - sequential scan on small table (~4ms vs 114ms)
- Eliminates the 698 MB `nsn` table join from hot path
- Can be refreshed periodically: `REFRESH MATERIALIZED VIEW CONCURRENTLY lookup_lin_niin_mat`

**Trade-offs:**
- Data staleness between refreshes (acceptable - this is reference data that changes infrequently)
- Requires a refresh mechanism (cron job, trigger, or manual)
- Need `CONCURRENTLY` option to avoid locking during refresh (requires unique index)

**Estimated improvement**: COUNT query 114ms -> ~4ms (28x faster)

---

### Fix 2: Cache the Total Count [HIGH IMPACT, LOW EFFORT]

**Problem solved**: COUNT query running on every page request

Since the total count (18,204) changes very rarely, cache it at the application level.

**Option A: In-memory cache with TTL**
```go
// Cache count for 5 minutes
var cachedCount int
var countCacheTime time.Time
const countCacheTTL = 5 * time.Minute

func (repo *RepositoryImpl) getCachedCount() (int, error) {
    if time.Since(countCacheTime) < countCacheTTL && cachedCount > 0 {
        return cachedCount, nil
    }
    // ... fetch from DB and update cache
}
```

**Option B: Use `reltuples` estimate (instant, no query needed)**
```sql
SELECT reltuples::bigint FROM pg_class WHERE relname = 'army_lin_to_niin';
-- Returns: 18204 (approximate, updated by ANALYZE/VACUUM)
```

**Estimated improvement**: Eliminates 114ms COUNT on cache hits

---

### Fix 3: Add ORDER BY for Deterministic Pagination [CRITICAL, LOW EFFORT]

**Problem solved**: Non-deterministic pagination (correctness bug)

The query **must** have an ORDER BY for pagination to be meaningful:

```go
stmt := SELECT(view.LookupLinNiin.AllColumns).
    FROM(view.LookupLinNiin).
    ORDER_BY(view.LookupLinNiin.Lin.ASC(), view.LookupLinNiin.Niin.ASC()).
    LIMIT(shared.DefaultPageSize).
    OFFSET(offset)
```

**Note**: With a materialized view (Fix 1), this can leverage the index for efficient sorted access. Without it, adding ORDER BY to the current view will add a sort step but is still necessary for correctness.

---

### Fix 4: Keyset (Cursor) Pagination Instead of OFFSET [HIGH IMPACT, MEDIUM EFFORT]

**Problem solved**: OFFSET degradation at high page numbers

Replace OFFSET-based pagination with keyset pagination using a WHERE clause:

```sql
-- Instead of: LIMIT 20 OFFSET 18000
-- Use:
SELECT niin, item_name, lin
FROM lookup_lin_niin_mat
WHERE (lin, niin) > ('last_seen_lin', 'last_seen_niin')
ORDER BY lin, niin
LIMIT 20
```

**Benefits:**
- Constant O(1) performance regardless of page number
- Uses index seeks instead of scanning/discarding rows
- Last page is just as fast as the first page

**Trade-offs:**
- Cannot jump to arbitrary page numbers (must navigate sequentially)
- Requires changing the API contract (cursor-based instead of page numbers)
- Mobile app would need to pass `last_seen` values instead of page numbers

**API change required:**
```json
// Current request
{"page": 45}

// New request format
{"after_lin": "L12345", "after_niin": "001234567", "limit": 20}
```

**Estimated improvement**: Last page 271ms -> ~0.3ms (900x faster)

---

### Fix 5: Window Function to Combine Queries [MEDIUM IMPACT, LOW EFFORT]

**Problem solved**: Two separate DB roundtrips

Combine the data and count into a single query:

```sql
SELECT niin, item_name, lin, COUNT(*) OVER() AS total_count
FROM lookup_lin_niin
ORDER BY lin, niin
LIMIT 20 OFFSET 0
```

**Benefits:**
- Single roundtrip (saves 50-150ms of mobile network latency)
- PostgreSQL can optimize the combined execution plan

**Trade-offs:**
- `COUNT(*) OVER()` still needs to compute the full count internally
- Adds the count column to every row (minor overhead)
- Best combined with Fix 2 (caching) to avoid the count cost entirely

---

### Fix 6: Increase shared_buffers [LOW IMPACT ON THIS QUERY, MEDIUM EFFORT]

**Problem solved**: Low cache hit ratio (87%) causing disk I/O

Current `shared_buffers` is 160 MB but the `nsn` table alone is 698 MB. Increasing to 512 MB - 1 GB would improve cache hit ratio across all queries.

```
# postgresql.conf
shared_buffers = 512MB   # or up to 25% of total RAM
```

**Trade-offs:**
- Requires PostgreSQL restart
- Must coordinate with available system RAM
- Benefits all queries, not just this one

---

## Recommended Implementation Plan

### Step 1: Add ORDER BY (Correctness Fix)
- **Impact**: Fixes non-deterministic pagination bug
- **Risk**: Low
- **Code change**: Add `.ORDER_BY()` to Jet query in `repository_impl.go`

### Step 2: Create Materialized View + Cache Count
- **Impact**: Reduces combined query time from ~115-386ms to ~4-8ms
- **Risk**: Low (can keep original view as fallback)
- **Changes**:
  - SQL migration to create materialized view with indexes
  - Update Jet generated code or query to use materialized view
  - Add count caching with 5-minute TTL
  - Add refresh mechanism (cron job or on data update)

### Step 3: Combine Into Single Query
- **Impact**: Eliminates one network roundtrip (~50-150ms on mobile)
- **Risk**: Low
- **Code change**: Use window function or cached count in single query

### Step 4 (Optional): Keyset Pagination
- **Impact**: Eliminates OFFSET degradation entirely
- **Risk**: Medium (API contract change, mobile app update needed)
- **Changes**:
  - New API endpoint or parameter format
  - Mobile app pagination logic update
  - Repository method signature change

---

## Appendix: Raw EXPLAIN ANALYZE Output

### Data Query (Page 1, OFFSET 0)
```
Limit  (cost=0.43..142.95 rows=20 width=37) (actual time=0.046..0.256 rows=20 loops=1)
  Buffers: shared hit=81
  ->  Nested Loop  (cost=0.43..129717.84 rows=18204 width=37) (actual time=0.045..0.253 rows=20 loops=1)
        Buffers: shared hit=81
        ->  Seq Scan on army_lin_to_niin  (cost=0.00..298.04 rows=18204 width=17) (actual time=0.030..0.032 rows=20 loops=1)
              Buffers: shared hit=1
        ->  Index Scan using nsn_pkey on nsn  (cost=0.43..7.11 rows=1 width=30) (actual time=0.010..0.010 rows=1 loops=20)
              Index Cond: ((niin)::text = (army_lin_to_niin.niin)::text)
              Buffers: shared hit=80
Planning Time: 27.325 ms
Execution Time: 0.281 ms
```

### Data Query (Last Page, OFFSET 18000)
```
Limit  (cost=128264.18..128406.70 rows=20 width=37) (actual time=270.648..270.821 rows=20 loops=1)
  Buffers: shared hit=70080 read=2115
  ->  Nested Loop  (cost=0.43..129717.84 rows=18204 width=37) (actual time=0.087..264.278 rows=18020 loops=1)
        Buffers: shared hit=70080 read=2115
        ->  Seq Scan on army_lin_to_niin  (cost=0.00..298.04 rows=18204 width=17) (actual time=0.064..1.372 rows=18020 loops=1)
              Buffers: shared hit=115
        ->  Index Scan using nsn_pkey on nsn  (cost=0.43..7.11 rows=1 width=30) (actual time=0.014..0.014 rows=1 loops=18020)
              Index Cond: ((niin)::text = (army_lin_to_niin.niin)::text)
              Buffers: shared hit=69965 read=2115
Planning Time: 0.639 ms
JIT:
  Functions: 8
  Options: Inlining false, Optimization false, Expressions true, Deforming true
  Timing: Generation 0.694 ms (Total 6.528 ms)
Execution Time: 271.606 ms
```

### COUNT Query
```
Aggregate  (cost=63675.35..63675.36 rows=1 width=8) (actual time=114.452..114.454 rows=1 loops=1)
  Buffers: shared hit=56906
  ->  Nested Loop  (cost=0.43..63629.84 rows=18204 width=7) (actual time=0.039..113.049 rows=18204 loops=1)
        Buffers: shared hit=56906
        ->  Seq Scan on army_lin_to_niin  (cost=0.00..298.04 rows=18204 width=17) (actual time=0.029..1.271 rows=18204 loops=1)
              Buffers: shared hit=116
        ->  Index Only Scan using nsn_pkey on nsn  (cost=0.43..3.48 rows=1 width=10) (actual time=0.006..0.006 rows=1 loops=18204)
              Heap Fetches: 0
              Buffers: shared hit=56790
Planning Time: 0.283 ms
Execution Time: 114.472 ms
```

### COUNT on army_lin_to_niin directly (baseline comparison)
```
Aggregate  (cost=343.55..343.56 rows=1 width=8) (actual time=4.545..4.546 rows=1 loops=1)
  Buffers: shared hit=116
  ->  Seq Scan on army_lin_to_niin  (cost=0.00..298.04 rows=18204 width=10) (actual time=0.047..2.119 rows=18204 loops=1)
        Buffers: shared hit=116
Execution Time: 4.565 ms
```

### Table Sizes

| Table | Total Size (with indexes) | Table Only |
|-------|--------------------------|------------|
| `nsn` | 920 MB | 698 MB |
| `army_lin_to_niin` | 1,776 KB | 928 KB |

### Database Configuration

| Setting | Value |
|---------|-------|
| PostgreSQL Version | 14.18 |
| shared_buffers | 160 MB |
| work_mem | 4 MB |
| effective_cache_size | 5 GB |
| max_connections | 100 |
| Cache hit ratio | 87% |
