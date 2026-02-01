# Item Query Performance Optimization Design

## Overview

This document outlines performance improvements for the `item_query` feature, specifically the detailed item query endpoint (`GET /api/v1/queries/items/detailed`).

## Current State Analysis

### Architecture Summary

The item_query feature consists of two endpoints:

| Endpoint | Path | Purpose |
|----------|------|---------|
| Short Query | `/queries/items/initial` | Quick lookup by NIIN or part number |
| Detailed Query | `/queries/items/detailed` | Comprehensive item data retrieval |

### Critical Performance Issue: Sequential Database Queries

The detailed query endpoint executes **~45 sequential database round-trips** for a single NIIN lookup.

#### Query Distribution by Function

| Function | Location | Sequential Queries |
|----------|----------|-------------------|
| `GetAmdfData` | [queries/amdf.go](../../api/item_query/detailed/queries/amdf.go) | 8 (7 fixed + 1 conditional) |
| `GetArmyPackagingAndFreight` | [queries/packaging.go](../../api/item_query/detailed/queries/packaging.go) | 6 |
| `GetSarsscat` | [queries/sarsscat.go](../../api/item_query/detailed/queries/sarsscat.go) | 3 |
| `GetIdentification` | [queries/identification.go](../../api/item_query/detailed/queries/identification.go) | 4 (3 fixed + 1 conditional) |
| `GetManagement` | [queries/management.go](../../api/item_query/detailed/queries/management.go) | 8 |
| `GetReference` | [queries/reference.go](../../api/item_query/detailed/queries/reference.go) | 4 (2 fixed + 2 conditional) |
| `GetFreight` | [queries/freight.go](../../api/item_query/detailed/queries/freight.go) | 1 |
| `GetPackaging` | [queries/packaging.go](../../api/item_query/detailed/queries/packaging.go) | 4 (3 fixed + 1 conditional) |
| `GetCharacteristics` | [queries/characteristics.go](../../api/item_query/detailed/queries/characteristics.go) | 1 |
| `GetDisposition` | [queries/disposition.go](../../api/item_query/detailed/queries/disposition.go) | 1 |
| **TOTAL** | | **~40-45 sequential queries** |

#### Current Execution Flow

```
Request → Handler → Service → Repository
                                   ↓
                    GetAmdfData (8 queries, sequential)
                                   ↓
                    GetArmyPackagingAndFreight (6 queries, sequential)
                                   ↓
                    GetSarsscat (3 queries, sequential)
                                   ↓
                    ... 7 more functions, all sequential ...
                                   ↓
                              Response
```

### Additional Issues Identified

1. **No Context Propagation**: Queries cannot be cancelled when clients disconnect
2. **Default Connection Pool**: Using Go's default `MaxIdleConns = 2`, insufficient for parallel queries
3. **No Caching**: Every request hits the database, even for identical NIINs
4. **Synchronous Analytics**: Analytics tracking blocks request completion
5. **Missing Database Indexes**: Potential missing indexes on frequently queried columns

---

## Proposed Optimizations

### Priority 1: Connection Pool Tuning (15 minutes)

**Impact**: High | **Effort**: Low | **Risk**: Low

**Problem**: Default Go connection pool is configured for low concurrency.

**Solution**: Configure pool for parallel query workload.

**File**: [bootstrap/database.go](../../bootstrap/database.go)

```go
// Configure connection pool for parallel queries
db.SetMaxOpenConns(50)              // Allow up to 50 concurrent connections
db.SetMaxIdleConns(25)              // Keep 25 connections warm
db.SetConnMaxLifetime(5 * time.Minute)  // Recycle connections periodically
db.SetConnMaxIdleTime(1 * time.Minute)  // Close idle connections after 1 min
```

**Recommendation**: Make configurable via environment variables:

```go
type Env struct {
    // ... existing fields
    DBMaxOpenConns int `env:"DB_MAX_OPEN_CONNS" envDefault:"50"`
    DBMaxIdleConns int `env:"DB_MAX_IDLE_CONNS" envDefault:"25"`
}
```

---

### Priority 2: Top-Level Query Parallelization (1 hour)

**Impact**: Very High | **Effort**: Medium | **Risk**: Medium

**Problem**: 10 independent query functions execute sequentially.

**Solution**: Use `errgroup` to execute all 10 query functions in parallel.

**File**: [detailed/repository_impl.go](../../api/item_query/detailed/repository_impl.go)

**Current Code** (lines 19-63):
```go
func (repo *RepositoryImpl) GetDetailedItemData(niin string) (response.DetailedResponse, error) {
    amdfData, err := queries.GetAmdfData(repo.Db, niin)           // Waits
    armyPackData, err := queries.GetArmyPackagingAndFreight(...)  // Waits
    sarsscatData, err := queries.GetSarsscat(...)                 // Waits
    // ... 7 more sequential calls
}
```

**Proposed Code**:
```go
import "golang.org/x/sync/errgroup"

func (repo *RepositoryImpl) GetDetailedItemData(ctx context.Context, niin string) (response.DetailedResponse, error) {
    var result response.DetailedResponse
    g, ctx := errgroup.WithContext(ctx)

    g.Go(func() error {
        data, err := queries.GetAmdfData(ctx, repo.Db, niin)
        if err != nil {
            repo.logQueryError("amdf", niin, err)
            return nil // Continue with partial data
        }
        result.Amdf = data
        return nil
    })

    g.Go(func() error {
        data, err := queries.GetArmyPackagingAndFreight(ctx, repo.Db, niin)
        // ... similar pattern
    })

    // ... 8 more parallel goroutines

    if err := g.Wait(); err != nil {
        return response.DetailedResponse{}, err
    }
    return result, nil
}
```

**Expected Improvement**: Response time reduced from ~45 sequential round-trips to ~8 parallel round-trips (limited by the slowest query function).

---

### Priority 3: Context Propagation (2 hours)

**Impact**: High | **Effort**: Low | **Risk**: Low

**Problem**: No `context.Context` usage means:
- Requests cannot be cancelled when clients disconnect
- No request timeouts
- Database connections held unnecessarily

**Solution**: Thread context through entire call chain.

**Files to Update**:
- [detailed/route.go](../../api/item_query/detailed/route.go)
- [detailed/service.go](../../api/item_query/detailed/service.go)
- [detailed/service_impl.go](../../api/item_query/detailed/service_impl.go)
- [detailed/repository.go](../../api/item_query/detailed/repository.go)
- [detailed/repository_impl.go](../../api/item_query/detailed/repository_impl.go)
- All files in [detailed/queries/](../../api/item_query/detailed/queries/)

**Handler Change** ([detailed/route.go](../../api/item_query/detailed/route.go)):
```go
func (handler *Handler) findDetailed(c *gin.Context) {
    ctx := c.Request.Context() // Extract context from Gin
    niin := c.Query("niin")
    itemData, err := handler.service.FindDetailedItem(ctx, niin)
    // ...
}
```

**Query Change** (all query files):
```go
// Before
err := stmt.Query(db, &result)

// After
err := stmt.QueryContext(ctx, db, &result)
```

---

### Priority 4: Inner Query Parallelization (3 hours)

**Impact**: High | **Effort**: Medium | **Risk**: Medium

**Problem**: Within each query function, multiple independent queries run sequentially.

**Example**: `GetManagement` makes 8 sequential queries to different tables, all filtering by the same NIIN.

**Solution**: Apply the same `errgroup` pattern within each query function.

**File**: [queries/management.go](../../api/item_query/detailed/queries/management.go)

**Current Code**:
```go
func GetManagement(db *sql.DB, niin string) (details.Management, error) {
    // Query 1
    flisManagementStmt := SELECT(...).FROM(table.FlisManagement).WHERE(...)
    err := flisManagementStmt.Query(db, &management.FLisManagement)

    // Query 2 - waits for Query 1
    flisPhraseStmt := SELECT(...).FROM(table.FlisPhrase).WHERE(...)
    err = flisPhraseStmt.Query(db, &management.FlisPhrase)

    // ... 6 more sequential queries
}
```

**Proposed Code**:
```go
func GetManagement(ctx context.Context, db *sql.DB, niin string) (details.Management, error) {
    management := details.Management{}
    g, ctx := errgroup.WithContext(ctx)

    g.Go(func() error {
        stmt := SELECT(table.FlisManagement.AllColumns).
            FROM(table.FlisManagement).
            WHERE(table.FlisManagement.Niin.EQ(String(niin)))
        return stmt.QueryContext(ctx, db, &management.FLisManagement)
    })

    g.Go(func() error {
        stmt := SELECT(table.FlisPhrase.AllColumns).
            FROM(table.FlisPhrase).
            WHERE(table.FlisPhrase.Niin.EQ(String(niin)))
        return stmt.QueryContext(ctx, db, &management.FlisPhrase)
    })

    // ... 6 more parallel goroutines

    return management, g.Wait()
}
```

**Expected Improvement**: Combined with Priority 2, total response time drops from ~45 sequential queries to the latency of 1-2 round-trips. **Potential 20-40x improvement**.

---

### Priority 5: In-Memory Caching (2 hours)

**Impact**: High | **Effort**: Medium | **Risk**: Low

**Problem**: Item data is read-heavy and changes infrequently, but every request hits the database.

**Solution**: Implement a TTL-based in-memory cache at the service layer.

**New File**: `api/item_query/detailed/cache.go`

```go
package detailed

import (
    "sync"
    "time"

    "miltechserver/api/response"
)

type cacheEntry struct {
    data      response.DetailedResponse
    expiresAt time.Time
}

type Cache struct {
    mu      sync.RWMutex
    entries map[string]cacheEntry
    ttl     time.Duration
}

func NewCache(ttl time.Duration) *Cache {
    c := &Cache{
        entries: make(map[string]cacheEntry),
        ttl:     ttl,
    }
    go c.cleanup()
    return c
}

func (c *Cache) Get(niin string) (response.DetailedResponse, bool) {
    c.mu.RLock()
    defer c.mu.RUnlock()

    entry, ok := c.entries[niin]
    if !ok || time.Now().After(entry.expiresAt) {
        return response.DetailedResponse{}, false
    }
    return entry.data, true
}

func (c *Cache) Set(niin string, data response.DetailedResponse) {
    c.mu.Lock()
    defer c.mu.Unlock()

    c.entries[niin] = cacheEntry{
        data:      data,
        expiresAt: time.Now().Add(c.ttl),
    }
}

func (c *Cache) cleanup() {
    ticker := time.NewTicker(c.ttl / 2)
    for range ticker.C {
        c.mu.Lock()
        now := time.Now()
        for k, v := range c.entries {
            if now.After(v.expiresAt) {
                delete(c.entries, k)
            }
        }
        c.mu.Unlock()
    }
}
```

**Service Integration** ([detailed/service_impl.go](../../api/item_query/detailed/service_impl.go)):

```go
type ServiceImpl struct {
    repo  Repository
    cache *Cache
}

func NewService(repo Repository) *ServiceImpl {
    return &ServiceImpl{
        repo:  repo,
        cache: NewCache(5 * time.Minute), // 5-minute TTL
    }
}

func (service *ServiceImpl) FindDetailedItem(ctx context.Context, niin string) (response.DetailedResponse, error) {
    // Check cache first
    if cached, ok := service.cache.Get(niin); ok {
        return cached, nil
    }

    // Cache miss - fetch from database
    data, err := service.repo.GetDetailedItemData(ctx, niin)
    if err != nil {
        return response.DetailedResponse{}, err
    }

    // Store in cache
    service.cache.Set(niin, data)
    return data, nil
}
```

**Consideration**: If running multiple server instances, consider Redis instead of in-memory caching for cache consistency.

---

### Priority 6: Async Analytics (30 minutes)

**Impact**: Low | **Effort**: Low | **Risk**: Low

**Problem**: Analytics tracking in [short/service_impl.go](../../api/item_query/short/service_impl.go) blocks the response.

**Current Code** (line 73-79):
```go
func (service *ServiceImpl) trackItemSearchSuccess(niin string, nomenclature string) {
    if service.analytics == nil || niin == "" {
        return
    }
    if err := service.analytics.IncrementItemSearchSuccess(niin, nomenclature); err != nil {
        slog.Warn("Failed to increment analytics...", ...)
    }
}
```

**Solution**: Use buffered channel for fire-and-forget analytics.

```go
type ServiceImpl struct {
    repo       Repository
    analytics  shared.AnalyticsTracker
    analyticsQ chan analyticsEvent
}

type analyticsEvent struct {
    niin         string
    nomenclature string
}

func NewService(repo Repository, analytics shared.AnalyticsTracker) *ServiceImpl {
    s := &ServiceImpl{
        repo:       repo,
        analytics:  analytics,
        analyticsQ: make(chan analyticsEvent, 100),
    }
    go s.processAnalytics()
    return s
}

func (s *ServiceImpl) processAnalytics() {
    for event := range s.analyticsQ {
        if err := s.analytics.IncrementItemSearchSuccess(event.niin, event.nomenclature); err != nil {
            slog.Warn("Failed to increment analytics", "niin", event.niin, "error", err)
        }
    }
}

func (service *ServiceImpl) trackItemSearchSuccess(niin string, nomenclature string) {
    if service.analytics == nil || niin == "" {
        return
    }
    select {
    case service.analyticsQ <- analyticsEvent{niin: niin, nomenclature: nomenclature}:
    default:
        slog.Warn("Analytics queue full, dropping event", "niin", niin)
    }
}
```

---

### Priority 7: Database Index Verification (30 minutes)

**Impact**: High | **Effort**: Low | **Risk**: Low

**Problem**: Potential missing indexes on `niin` columns across tables.

**Solution**: Verify and create indexes on all tables queried by NIIN.

**SQL to Check Existing Indexes**:
```sql
SELECT tablename, indexname, indexdef
FROM pg_indexes
WHERE indexdef LIKE '%niin%';
```

**Tables to Verify** (all tables queried in the detailed endpoint):

```sql
-- Create missing indexes (run CONCURRENTLY to avoid locking)
CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_army_master_data_file_niin ON army_master_data_file(niin);
CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_amdf_management_niin ON amdf_management(niin);
CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_amdf_credit_niin ON amdf_credit(niin);
CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_amdf_billing_niin ON amdf_billing(niin);
CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_amdf_matcat_niin ON amdf_matcat(niin);
CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_amdf_phrase_niin ON amdf_phrase(niin);
CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_amdf_i_and_s_niin ON amdf_i_and_s(niin);
CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_flis_management_niin ON flis_management(niin);
CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_flis_management_id_niin ON flis_management_id(niin);
CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_flis_standardization_niin ON flis_standardization(niin);
CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_flis_cancelled_niin_niin ON flis_cancelled_niin(niin);
CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_flis_phrase_niin ON flis_phrase(niin);
CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_flis_identification_niin ON flis_identification(niin);
CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_flis_reference_niin ON flis_reference(niin);
CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_flis_freight_niin ON flis_freight(niin);
CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_flis_packaging_1_niin ON flis_packaging_1(niin);
CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_flis_packaging_2_niin ON flis_packaging_2(niin);
CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_flis_item_characteristics_niin ON flis_item_characteristics(niin);
CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_army_management_niin ON army_management(niin);
CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_army_sarsscat_niin ON army_sarsscat(niin);
CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_army_packaging_and_freight_niin ON army_packaging_and_freight(niin);
CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_army_packaging_1_niin ON army_packaging_1(niin);
CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_army_packaging_2_niin ON army_packaging_2(niin);
CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_army_freight_niin ON army_freight(niin);
CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_air_force_management_niin ON air_force_management(niin);
CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_marine_corps_management_niin ON marine_corps_management(niin);
CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_navy_management_niin ON navy_management(niin);
CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_faa_management_niin ON faa_management(niin);
CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_component_end_item_niin ON component_end_item(niin);
CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_disposition_niin ON disposition(niin);
CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_moe_rule_niin ON moe_rule(niin);
CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_amdf_freight_niin ON amdf_freight(niin);
CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_dss_weight_and_cube_niin ON dss_weight_and_cube(niin);

-- Also verify index on part_number table for short queries
CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_part_number_part_number ON part_number(part_number);
CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_part_number_niin ON part_number(niin);
```

---

## Implementation Summary

| Priority | Change | Effort | Impact | Files Affected |
|----------|--------|--------|--------|----------------|
| 1 | Connection Pool Tuning | 15 min | High | `bootstrap/database.go` |
| 2 | Top-Level Parallelization | 1 hr | Very High | `detailed/repository_impl.go` |
| 3 | Context Propagation | 2 hr | High | All detailed query files |
| 4 | Inner Query Parallelization | 3 hr | High | All `queries/*.go` files |
| 5 | In-Memory Caching | 2 hr | High | New `cache.go`, `service_impl.go` |
| 6 | Async Analytics | 30 min | Low | `short/service_impl.go` |
| 7 | Database Indexes | 30 min | High | Database migration |

**Total Estimated Effort**: ~9.5 hours

---

## Expected Performance Improvement

| Metric | Current | After Optimization |
|--------|---------|-------------------|
| DB Round-trips per request | ~45 sequential | ~1-2 parallel |
| Response time (estimated) | ~450ms (at 10ms/query) | ~20-50ms |
| Cache hit response time | N/A | <1ms |
| Connection pool utilization | Poor | Optimal |

**Estimated improvement**: **10-40x faster response times** for detailed queries.

---

## Questions for Implementation

1. **Error Handling Strategy**: Should a failure in one query section fail the entire request, or return partial data? Current code logs errors but continues.

2. **Cache Invalidation**: How often does item data change? Is there a data import process that should trigger cache invalidation?

3. **Multi-Instance Deployment**: Running multiple server instances? If yes, consider Redis cache instead of in-memory.

4. **Performance Baseline**: Current latency metrics for detailed query endpoint would help measure improvement.

5. **Database Connection Limits**: PostgreSQL `max_connections` setting affects how aggressive we can be with `SetMaxOpenConns`.

---

## Future Considerations

### Database View or Stored Function

For maximum performance, consider creating a PostgreSQL function that returns all detailed data in a single call:

```sql
CREATE OR REPLACE FUNCTION get_detailed_item(p_niin TEXT)
RETURNS JSON AS $$
    SELECT json_build_object(
        'amdf', (SELECT row_to_json(t) FROM army_master_data_file t WHERE niin = p_niin),
        'management', (SELECT json_agg(row_to_json(t)) FROM flis_management t WHERE niin = p_niin),
        -- ... other tables
    );
$$ LANGUAGE sql STABLE;
```

This would reduce the entire detailed query to **1 database call**.

### Redis Caching

If scaling to multiple server instances:

```go
import "github.com/redis/go-redis/v9"

type RedisCache struct {
    client *redis.Client
    ttl    time.Duration
}

func (c *RedisCache) Get(ctx context.Context, niin string) (response.DetailedResponse, bool) {
    val, err := c.client.Get(ctx, "detailed:"+niin).Bytes()
    if err != nil {
        return response.DetailedResponse{}, false
    }
    var data response.DetailedResponse
    json.Unmarshal(val, &data)
    return data, true
}
```

---

## References

- [Go errgroup documentation](https://pkg.go.dev/golang.org/x/sync/errgroup)
- [go-jet QueryContext](https://github.com/go-jet/jet)
- [PostgreSQL Connection Pooling Best Practices](https://www.postgresql.org/docs/current/runtime-config-connection.html)
