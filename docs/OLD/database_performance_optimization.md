# Database Performance Optimization Analysis

**Analysis Date**: February 8, 2026  
**Database**: PostgreSQL (miltech_ng)  
**Total Tables**: 84  
**Analysis Scope**: Schema design, data types, indexing, constraints, and performance patterns

---

## Executive Summary

This document outlines performance and efficiency opportunities identified in the database schema based on PostgreSQL best practices. The analysis reveals several categories of improvements:

1. **Critical Data Type Issues** - Using deprecated/suboptimal data types
2. **Missing Foreign Key Indexes** - Performance bottlenecks on joins and cascades
3. **Inefficient Hash Indexes** - Suboptimal index type choices
4. **Timestamp Handling** - Inconsistent timezone awareness
5. **String Data Types** - Using VARCHAR instead of TEXT

## Critical Issues

### 1. Data Type Violations

> [!CAUTION]
> **High Priority**: The database uses deprecated and non-compliant PostgreSQL data types that violate best practices outlined in the PostgreSQL skill document.

#### Problem: `timestamp without time zone` Usage

**Affected Tables** (19 tables):
- `users`: `created_at`, `last_login`
- `material_images`: `upload_date`, `created_at`, `updated_at`
- `material_images_votes`: `created_at`, `updated_at`
- `material_images_upload_limits`: `last_upload_time`
- `item_comments`: `created_at`, `updated_at`
- `user_items_serialized`: `save_time`, `last_updated`
- `analytics_event_counters`: `last_seen_at`

**Current**: `timestamp without time zone`  
**Required**: `timestamptz` (timestamp with time zone)

**Impact**:
- Timezone ambiguity in multi-timezone deployments
- Daylight saving time complications
- Data integrity issues when comparing timestamps across regions

**Solution**:
```sql
-- Example migration for users table
ALTER TABLE users 
  ALTER COLUMN created_at TYPE timestamptz USING created_at AT TIME ZONE 'UTC',
  ALTER COLUMN last_login TYPE timestamptz USING last_login AT TIME ZONE 'UTC';
```

**Priority**: HIGH  
**Effort**: Medium (requires migration script for each affected column)

---

#### Problem: `character varying` Instead of `text`

**Affected Tables**: Almost all tables use `varchar` for string columns

**Current**: `character varying`, `character varying(255)`, etc.  
**Required**: `text` with optional `CHECK` constraints

**Impact**:
- Unnecessary length constraints
- Performance overhead from length checking
- Limited flexibility for future data growth

**Solution**:
```sql
-- Example for users table
ALTER TABLE users 
  ALTER COLUMN email TYPE text,
  ALTER COLUMN username TYPE text;

-- If length constraints are needed:
ALTER TABLE users 
  ADD CONSTRAINT check_email_length CHECK (LENGTH(email) <= 255);
```

**Priority**: MEDIUM  
**Effort**: Low (mostly cosmetic, but good hygiene)

---

### 2. Missing Foreign Key Indexes

> [!WARNING]
> **Critical Performance Issue**: PostgreSQL does NOT automatically create indexes on foreign key columns. This causes significant performance degradation on joins, cascades, and parent table operations.

**Affected Foreign Keys** (verified missing indexes):

#### Users Table References
Missing indexes on columns referencing `users.uid`:
- `material_images.user_id` ✓ *Has index*
- `material_images_flags.user_id` ✓ *Has index*
- `material_images_votes.user_id` ✓ *Has index*
- `item_comments.author_id` ✓ *Has index*
- `equipment_services.created_by` ✓ *Has index*
- `shop_invite_codes.created_by` - **Missing index needed**
- `shop_list_items.added_by` - **Missing index needed**
- `shop_lists.created_by` ✓ *Has index*
- `shop_members.user_id` ✓ *Has index*
- `shop_messages.user_id` ✓ *Has index*
- `user_items_serialized.user_id` - **Missing index needed**
- `user_vehicle.user_id` ✓ *Has index*
- `item_comment_flags.flagger_id` ✓ *Has index*
- `user_item_category.user_uid` ✓ *Has index*
- `user_items_categorized.user_id` - **Missing index needed**
- `user_items_quick.user_id` ✓ *Has index*
- `shops.created_by` - **Missing index needed**
- `shop_vehicle.creator_id` - **Missing index needed**
- `shop_vehicle_notification_changes.changed_by` - **Missing index needed**

#### Shop References
Missing indexes on columns referencing `shops.id`:
- `shop_members.shop_id` ✓ *Has index*
- `shop_lists.shop_id` ✓ *Has index*
- `shop_invite_codes.shop_id` ✓ *Has index*
- `equipment_services.shop_id` ✓ *Has index*
- `shop_messages.shop_id` ✓ *Has index*
- `shop_notification_items.shop_id` ✓ *Has index*
- `shop_vehicle.shop_id` ✓ *Has index*
- `shop_vehicle_notification_changes.shop_id` - **Missing index needed**
- `shop_vehicle_notifications.shop_id` ✓ *Has index*

#### NSN/NIIN References
Missing indexes on columns referencing `nsn.niin`:
- `item_comments.comment_niin` - **Missing index needed** (has composite index but not standalone)

#### Other Foreign Keys
- `material_images_flags.image_id` ✓ *Has index*
- `material_images_votes.image_id` - **Missing standalone index** (only in composite PK)
- `item_comment_flags.comment_id` ✓ *Has index*
- `item_comments.parent_id` ✓ *Has index*
- `shop_list_items.list_id` ✓ *Has index*
- `shop_notification_items.notification_id` ✓ *Has index*
- `shop_vehicle_notifications.vehicle_id` ✓ *Has index*
- `shop_vehicle_notification_changes.notification_id` ✓ *Has index*
- `shop_vehicle_notification_changes.vehicle_id` ✓ *Has index*
- `user_notification_items.notification_id` ✓ *Has index*
- `user_vehicle_notifications.vehicle_id` ✓ *Has index*

**Impact**:
- Slow JOIN operations
- Table-level locks during parent DELETE/UPDATE operations
- Sequential scans instead of index scans
- Poor query planner decisions

**Solution**:
```sql
-- Critical missing indexes
CREATE INDEX idx_shop_invite_codes_created_by ON shop_invite_codes(created_by);
CREATE INDEX idx_shop_list_items_added_by ON shop_list_items(added_by);
CREATE INDEX idx_user_items_serialized_user_id ON user_items_serialized(user_id);
CREATE INDEX idx_user_items_categorized_user_id ON user_items_categorized(user_id);
CREATE INDEX idx_shops_created_by ON shops(created_by);
CREATE INDEX idx_shop_vehicle_creator_id ON shop_vehicle(creator_id);
CREATE INDEX idx_shop_vehicle_notification_changes_changed_by ON shop_vehicle_notification_changes(changed_by);
CREATE INDEX idx_shop_vehicle_notification_changes_shop_id ON shop_vehicle_notification_changes(shop_id);
CREATE INDEX idx_item_comments_comment_niin ON item_comments(comment_niin);
CREATE INDEX idx_material_images_votes_image_id ON material_images_votes(image_id);
```

**Priority**: HIGH  
**Effort**: Low (simple index creation, can be done concurrently)

---

### 3. Hash Index Usage

> [!IMPORTANT]
> **Recommendation**: Replace HASH indexes with B-tree indexes for better performance and reliability.

**Affected Tables** (37 hash indexes found):
- `amdf_i_and_s.niin`
- `army_lin_to_niin.lin`
- `army_management.niin`
- `army_pack_supplemental_instruct.niin`
- `faa_management.niin`
- `flis_cancelled_niin.niin`
- ... and 31 more

**Current**: `USING hash`  
**Recommended**: `USING btree`

**Rationale**:
- B-tree indexes support range queries (hash does not)
- B-tree indexes are WAL-logged and crash-safe
- B-tree equality performance is comparable to hash
- B-tree indexes support index-only scans
- Hash indexes can't be used for sorting

**Solution**:
```sql
-- Example: Convert hash indexes to btree
DROP INDEX CONCURRENTLY idx_amdf_i_and_s_niin;
CREATE INDEX CONCURRENTLY idx_amdf_i_and_s_niin ON amdf_i_and_s USING btree (niin);

-- Repeat for all 37 hash indexes
```

**Priority**: MEDIUM  
**Effort**: Medium (requires rebuilding indexes, use `CONCURRENTLY` to avoid locks)

---

### 4. Inconsistent Timezone Handling

**Problem**: Mixed usage of `timestamptz` and `timestamp`

**Tables with `timestamptz`** (correct):
- `shops`: `created_at`, `updated_at`
- `shop_members`: `joined_at`
- `shop_lists`: `created_at`, `updated_at`
- `shop_list_items`: `created_at`, `updated_at`
- `shop_messages`: `created_at`, `updated_at`
- `user_vehicle`: `save_time`, `last_updated`

**Tables with `timestamp`** (incorrect):
- `users`: `created_at`, `last_login`
- `material_images`: `upload_date`, `created_at`, `updated_at`
- `item_comments`: `created_at`, `updated_at`
- ... and 16 more

**Impact**:
- Inconsistent behavior across tables
- Potential data integrity issues in cross-table queries
- Timezone conversion bugs

**Solution**: Standardize all timestamp columns to `timestamptz`

**Priority**: HIGH  
**Effort**: Medium

---

### 5. No PRIMARY KEY on Users Table

**Problem**: `users` table uses `uid` of type `text` as primary key

**Current**:
```sql
uid text PRIMARY KEY
```

**Concern**: Primary key is `text` type instead of preferred `BIGINT GENERATED ALWAYS AS IDENTITY`

**Analysis**:
- `uid` appears to be a Firebase UID (external system identifier)
- Using `text` PK is acceptable when the ID is externally generated
- However, all foreign keys also use `text`, which is less efficient than `bigint`

**Impact**:
- Larger index size (text vs 8-byte bigint)
- Slower joins and lookups
- More storage overhead

**Recommendation**: 
- Keep current design if `uid` must match external system
- Consider adding surrogate `bigint` key if performance becomes critical
- Document this intentional deviation from best practices

**Priority**: LOW (acceptable trade-off for external ID compatibility)  
**Effort**: N/A (design decision)

---

### 6. Potential Missing NOT NULL Constraints

**Problem**: Several columns that should semantically never be NULL lack NOT NULL constraints

**Examples**:
- `material_images.niin` - should always have a value
- `item_comments.comment_niin` - should always reference an item
- `shop_members.shop_id` - should always belong to a shop
- `shop_members.role` - should always have a role

**Impact**:
- Data integrity issues
- NULLs can slip through application validation
- Query performance degradation (NULL handling overhead)

**Solution**: Add NOT NULL constraints where semantically appropriate

**Priority**: MEDIUM  
**Effort**: Medium (requires data validation before adding constraints)

---

## Optimization Opportunities

### 7. Composite Index Optimization

**shop_members Table**:
- Has many composite indexes with overlapping prefixes
- `idx_shop_members_shop_id`
- `idx_shop_members_shop_joined`
- `idx_shop_members_shop_user`
- `idx_shop_members_shop_user_role`

**Analysis**: Some of these may be redundant. Composite index `(shop_id, user_id, role)` can serve queries filtering on:
- `shop_id` alone
- `shop_id, user_id`
- `shop_id, user_id, role`

**Recommendation**: Consolidate indexes where possible to reduce write overhead

**Priority**: LOW  
**Effort**: Low (requires query analysis to confirm usage patterns)

---

### 8. Partial Index Opportunities

**Current Partial Indexes** (good examples):
- `material_images.idx_material_images_flagged WHERE is_flagged = true`
- `material_images.idx_material_images_net_votes WHERE is_active = true`
- `material_images.idx_material_images_niin WHERE is_active = true`

**Additional Opportunities**:
Consider partial indexes for:
- `shop_invite_codes WHERE is_active = true` (if most codes are inactive)
- `equipment_services WHERE is_completed = false` (if most services are completed)
- `shop_vehicle_notifications WHERE completed = false` (if most are completed)

**Priority**: LOW  
**Effort**: Low

---

## Migration Plan

### Phase 1: Critical Fixes (Week 1-2)

1. **Add Missing FK Indexes** (Can run concurrently, no downtime)
   ```sql
   CREATE INDEX CONCURRENTLY idx_shop_invite_codes_created_by ON shop_invite_codes(created_by);
   CREATE INDEX CONCURRENTLY idx_shop_list_items_added_by ON shop_list_items(added_by);
   -- ... etc
   ```

2. **Migrate timestamp to timestamptz** (Requires brief maintenance window)
   - Test migration on staging
   - Run during low-traffic period
   - Validate application compatibility

### Phase 2: Hash Index Replacement (Week 3-4)

1. Convert hash indexes to btree using `CONCURRENTLY`
2. Monitor query performance before/after
3. Validate no regression in lookup times

### Phase 3: Data Type Cleanup (Week 5-6)

1. Convert `varchar` to `text` (low risk)
2. Add CHECK constraints where length limits are needed
3. Document any intentional deviations

### Phase 4: NOT NULL Constraints (Week 7-8)

1. Audit data for NULL values
2. Fix existing NULL data
3. Add NOT NULL constraints
4. Update application to enforce constraints

---

## Verification Queries

### Check for missing FK indexes
```sql
SELECT
    tc.table_name,
    kcu.column_name,
    ccu.table_name AS foreign_table_name,
    ccu.column_name AS foreign_column_name
FROM information_schema.table_constraints AS tc
JOIN information_schema.key_column_usage AS kcu
    ON tc.constraint_name = kcu.constraint_name
JOIN information_schema.constraint_column_usage AS ccu
    ON ccu.constraint_name = tc.constraint_name
WHERE tc.constraint_type = 'FOREIGN KEY'
    AND NOT EXISTS (
        SELECT 1
        FROM pg_indexes
        WHERE tablename = tc.table_name
            AND indexdef LIKE '%' || kcu.column_name || '%'
    );
```

### Find all hash indexes
```sql
SELECT
    schemaname,
    tablename,
    indexname,
    indexdef
FROM pg_indexes
WHERE schemaname = 'public'
    AND indexdef LIKE '%USING hash%'
ORDER BY tablename, indexname;
```

### Find timestamp without time zone columns
```sql
SELECT
    table_name,
    column_name,
    data_type
FROM information_schema.columns
WHERE table_schema = 'public'
    AND data_type = 'timestamp without time zone'
ORDER BY table_name, column_name;
```

---

## Summary of Recommendations

| Issue | Priority | Effort | Impact | Tables Affected |
|-------|----------|--------|--------|----------------|
| Missing FK Indexes | HIGH | Low | High | 10+ tables |
| timestamp → timestamptz | HIGH | Medium | High | 19 tables |
| Hash → Btree Indexes | MEDIUM | Medium | Medium | 37 indexes |
| varchar → text | MEDIUM | Low | Low | All tables |
| Missing NOT NULL | MEDIUM | Medium | Medium | Multiple tables |
| Composite Index Optimization | LOW | Low | Low | 5-10 tables |

---

## Next Steps

1. **Review and approve** this analysis with the development team
2. **Test migrations** on a staging database
3. **Create migration scripts** for each phase
4. **Schedule maintenance windows** for non-concurrent operations
5. **Monitor performance** before and after each phase
6. **Document decisions** and any intentional deviations from best practices
