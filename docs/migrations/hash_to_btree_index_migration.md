# Hash to B-tree Index Migration

**Migration Date**: TBD  
**Purpose**: Convert all hash indexes to btree for better performance and reliability  
**Total Indexes**: 25

---

## Overview

This document contains the SQL statements needed to convert all hash indexes to btree indexes as identified in issue #3 of the [database performance optimization analysis](./database_performance_optimization.md).

**Why this migration is needed**:
- B-tree indexes support range queries (hash does not)
- B-tree indexes are WAL-logged and crash-safe
- B-tree equality performance is comparable to hash
- B-tree indexes support index-only scans
- Hash indexes can't be used for sorting

---

## Migration Instructions

1. **Run during low-traffic period** (recommended: maintenance window)
2. **Use `CONCURRENTLY` option** to avoid blocking reads/writes
3. **Monitor disk space** - indexes will temporarily exist in both forms
4. **Verify each step** before proceeding to the next
5. **Test queries** after migration to ensure performance is maintained

---

## Migration Scripts

### Table: amdf_i_and_s

```sql
-- Drop hash index and create btree replacement
DROP INDEX CONCURRENTLY idx_amdf_i_and_s_niin;
CREATE INDEX CONCURRENTLY idx_amdf_i_and_s_niin ON amdf_i_and_s USING btree (niin);
```

### Table: army_lin_to_niin

```sql
DROP INDEX CONCURRENTLY idx_army_lin_to_niin;
CREATE INDEX CONCURRENTLY idx_army_lin_to_niin ON army_lin_to_niin USING btree (lin);
```

### Table: army_management

```sql
DROP INDEX CONCURRENTLY idx_army_management_niin;
CREATE INDEX CONCURRENTLY idx_army_management_niin ON army_management USING btree (niin);
```

### Table: army_pack_supplemental_instruct

```sql
DROP INDEX CONCURRENTLY idx_army_pack_suppl_niin;
CREATE INDEX CONCURRENTLY idx_army_pack_suppl_niin ON army_pack_supplemental_instruct USING btree (niin);
```

### Table: colloquial_name

```sql
DROP INDEX CONCURRENTLY idx_colloquial_name_inc;
CREATE INDEX CONCURRENTLY idx_colloquial_name_inc ON colloquial_name USING btree (inc);
```

### Table: component_end_item

```sql
DROP INDEX CONCURRENTLY idx_component_end_item;
CREATE INDEX CONCURRENTLY idx_component_end_item ON component_end_item USING btree (niin);
```

### Table: faa_management

```sql
DROP INDEX CONCURRENTLY idx_faa_management_niin;
CREATE INDEX CONCURRENTLY idx_faa_management_niin ON faa_management USING btree (niin);
```

### Table: flis_cancelled_niin

```sql
DROP INDEX CONCURRENTLY idx_cancelled_niin;
CREATE INDEX CONCURRENTLY idx_cancelled_niin ON flis_cancelled_niin USING btree (niin);
```

### Table: flis_item_characteristics

```sql
DROP INDEX CONCURRENTLY idx_flis_item_characteristics_niin;
CREATE INDEX CONCURRENTLY idx_flis_item_characteristics_niin ON flis_item_characteristics USING btree (niin);
```

### Table: flis_management

```sql
DROP INDEX CONCURRENTLY idx_flis_management;
CREATE INDEX CONCURRENTLY idx_flis_management ON flis_management USING btree (niin);
```

### Table: flis_packaging_1

```sql
DROP INDEX CONCURRENTLY idx_flis_packaging_1_niin;
CREATE INDEX CONCURRENTLY idx_flis_packaging_1_niin ON flis_packaging_1 USING btree (niin);
```

### Table: flis_packaging_2

```sql
DROP INDEX CONCURRENTLY idx_flis_packaging_2_niin;
CREATE INDEX CONCURRENTLY idx_flis_packaging_2_niin ON flis_packaging_2 USING btree (niin);
```

### Table: flis_phrase

```sql
DROP INDEX CONCURRENTLY idx_flis_phrase_niin;
CREATE INDEX CONCURRENTLY idx_flis_phrase_niin ON flis_phrase USING btree (niin);
```

### Table: flis_reference (3 indexes)

```sql
-- Index 1: cage_code
DROP INDEX CONCURRENTLY idx_flis_reference_cage_code;
CREATE INDEX CONCURRENTLY idx_flis_reference_cage_code ON flis_reference USING btree (cage_code);

-- Index 2: niin
DROP INDEX CONCURRENTLY idx_flis_reference_niin;
CREATE INDEX CONCURRENTLY idx_flis_reference_niin ON flis_reference USING btree (niin);

-- Index 3: part_number
DROP INDEX CONCURRENTLY idx_flis_reference_part_number;
CREATE INDEX CONCURRENTLY idx_flis_reference_part_number ON flis_reference USING btree (part_number);
```

### Table: flis_standardization

```sql
DROP INDEX CONCURRENTLY idx_standardization_niin;
CREATE INDEX CONCURRENTLY idx_standardization_niin ON flis_standardization USING btree (niin);
```

### Table: lookup_uoc (2 indexes)

```sql
-- Index 1: model
DROP INDEX CONCURRENTLY idx_usable_on_codes_model;
CREATE INDEX CONCURRENTLY idx_usable_on_codes_model ON lookup_uoc USING btree (model);

-- Index 2: uoc
DROP INDEX CONCURRENTLY idx_usable_on_codes_uoc;
CREATE INDEX CONCURRENTLY idx_usable_on_codes_uoc ON lookup_uoc USING btree (uoc);
```

### Table: marine_corps_management

```sql
DROP INDEX CONCURRENTLY idx_marine_corps_management;
CREATE INDEX CONCURRENTLY idx_marine_corps_management ON marine_corps_management USING btree (niin);
```

### Table: marines_sl_6_2_item_id

```sql
DROP INDEX CONCURRENTLY idx_marine_sl_62_item_id_niin;
CREATE INDEX CONCURRENTLY idx_marine_sl_62_item_id_niin ON marines_sl_6_2_item_id USING btree (niin);
```

### Table: marines_sl_6_2_item_supp

```sql
DROP INDEX CONCURRENTLY idx_marine_sl_62_item_supp;
CREATE INDEX CONCURRENTLY idx_marine_sl_62_item_supp ON marines_sl_6_2_item_supp USING btree (niin);
```

### Table: moe_rule

```sql
DROP INDEX CONCURRENTLY idx_moe_rule;
CREATE INDEX CONCURRENTLY idx_moe_rule ON moe_rule USING btree (niin);
```

### Table: part_number (2 indexes)

```sql
-- Index 1: niin
DROP INDEX CONCURRENTLY idx_part_number_niin;
CREATE INDEX CONCURRENTLY idx_part_number_niin ON part_number USING btree (niin);

-- Index 2: part_number (note: quoted identifier)
DROP INDEX CONCURRENTLY "idx_part_number_partNumber";
CREATE INDEX CONCURRENTLY "idx_part_number_partNumber" ON part_number USING btree (part_number);
```

---

## Consolidated Script (Run All at Once)

> [!WARNING]
> Running all migrations at once will consume significant I/O and disk space. Consider running in batches during multiple maintenance windows.

```sql
-- Batch 1: amdf and army tables
DROP INDEX CONCURRENTLY idx_amdf_i_and_s_niin;
CREATE INDEX CONCURRENTLY idx_amdf_i_and_s_niin ON amdf_i_and_s USING btree (niin);

DROP INDEX CONCURRENTLY idx_army_lin_to_niin;
CREATE INDEX CONCURRENTLY idx_army_lin_to_niin ON army_lin_to_niin USING btree (lin);

DROP INDEX CONCURRENTLY idx_army_management_niin;
CREATE INDEX CONCURRENTLY idx_army_management_niin ON army_management USING btree (niin);

DROP INDEX CONCURRENTLY idx_army_pack_suppl_niin;
CREATE INDEX CONCURRENTLY idx_army_pack_suppl_niin ON army_pack_supplemental_instruct USING btree (niin);

-- Batch 2: colloquial and component tables
DROP INDEX CONCURRENTLY idx_colloquial_name_inc;
CREATE INDEX CONCURRENTLY idx_colloquial_name_inc ON colloquial_name USING btree (inc);

DROP INDEX CONCURRENTLY idx_component_end_item;
CREATE INDEX CONCURRENTLY idx_component_end_item ON component_end_item USING btree (niin);

-- Batch 3: faa and flis tables
DROP INDEX CONCURRENTLY idx_faa_management_niin;
CREATE INDEX CONCURRENTLY idx_faa_management_niin ON faa_management USING btree (niin);

DROP INDEX CONCURRENTLY idx_cancelled_niin;
CREATE INDEX CONCURRENTLY idx_cancelled_niin ON flis_cancelled_niin USING btree (niin);

DROP INDEX CONCURRENTLY idx_flis_item_characteristics_niin;
CREATE INDEX CONCURRENTLY idx_flis_item_characteristics_niin ON flis_item_characteristics USING btree (niin);

DROP INDEX CONCURRENTLY idx_flis_management;
CREATE INDEX CONCURRENTLY idx_flis_management ON flis_management USING btree (niin);

DROP INDEX CONCURRENTLY idx_flis_packaging_1_niin;
CREATE INDEX CONCURRENTLY idx_flis_packaging_1_niin ON flis_packaging_1 USING btree (niin);

DROP INDEX CONCURRENTLY idx_flis_packaging_2_niin;
CREATE INDEX CONCURRENTLY idx_flis_packaging_2_niin ON flis_packaging_2 USING btree (niin);

DROP INDEX CONCURRENTLY idx_flis_phrase_niin;
CREATE INDEX CONCURRENTLY idx_flis_phrase_niin ON flis_phrase USING btree (niin);

-- Batch 4: flis_reference table (3 indexes)
DROP INDEX CONCURRENTLY idx_flis_reference_cage_code;
CREATE INDEX CONCURRENTLY idx_flis_reference_cage_code ON flis_reference USING btree (cage_code);

DROP INDEX CONCURRENTLY idx_flis_reference_niin;
CREATE INDEX CONCURRENTLY idx_flis_reference_niin ON flis_reference USING btree (niin);

DROP INDEX CONCURRENTLY idx_flis_reference_part_number;
CREATE INDEX CONCURRENTLY idx_flis_reference_part_number ON flis_reference USING btree (part_number);

DROP INDEX CONCURRENTLY idx_standardization_niin;
CREATE INDEX CONCURRENTLY idx_standardization_niin ON flis_standardization USING btree (niin);

-- Batch 5: lookup and marine tables
DROP INDEX CONCURRENTLY idx_usable_on_codes_model;
CREATE INDEX CONCURRENTLY idx_usable_on_codes_model ON lookup_uoc USING btree (model);

DROP INDEX CONCURRENTLY idx_usable_on_codes_uoc;
CREATE INDEX CONCURRENTLY idx_usable_on_codes_uoc ON lookup_uoc USING btree (uoc);

DROP INDEX CONCURRENTLY idx_marine_corps_management;
CREATE INDEX CONCURRENTLY idx_marine_corps_management ON marine_corps_management USING btree (niin);

DROP INDEX CONCURRENTLY idx_marine_sl_62_item_id_niin;
CREATE INDEX CONCURRENTLY idx_marine_sl_62_item_id_niin ON marines_sl_6_2_item_id USING btree (niin);

DROP INDEX CONCURRENTLY idx_marine_sl_62_item_supp;
CREATE INDEX CONCURRENTLY idx_marine_sl_62_item_supp ON marines_sl_6_2_item_supp USING btree (niin);

-- Batch 6: moe and part_number tables
DROP INDEX CONCURRENTLY idx_moe_rule;
CREATE INDEX CONCURRENTLY idx_moe_rule ON moe_rule USING btree (niin);

DROP INDEX CONCURRENTLY idx_part_number_niin;
CREATE INDEX CONCURRENTLY idx_part_number_niin ON part_number USING btree (niin);

DROP INDEX CONCURRENTLY "idx_part_number_partNumber";
CREATE INDEX CONCURRENTLY "idx_part_number_partNumber" ON part_number USING btree (part_number);
```

---

## Verification Queries

### Confirm no hash indexes remain

```sql
SELECT
    tablename,
    indexname,
    indexdef
FROM pg_indexes
WHERE schemaname = 'public'
    AND indexdef LIKE '%USING hash%'
ORDER BY tablename, indexname;
```

**Expected result**: 0 rows

### Verify all indexes were recreated

```sql
SELECT
    tablename,
    indexname,
    indexdef
FROM pg_indexes
WHERE schemaname = 'public'
    AND indexname IN (
        'idx_amdf_i_and_s_niin',
        'idx_army_lin_to_niin',
        'idx_army_management_niin',
        'idx_army_pack_suppl_niin',
        'idx_colloquial_name_inc',
        'idx_component_end_item',
        'idx_faa_management_niin',
        'idx_cancelled_niin',
        'idx_flis_item_characteristics_niin',
        'idx_flis_management',
        'idx_flis_packaging_1_niin',
        'idx_flis_packaging_2_niin',
        'idx_flis_phrase_niin',
        'idx_flis_reference_cage_code',
        'idx_flis_reference_niin',
        'idx_flis_reference_part_number',
        'idx_standardization_niin',
        'idx_usable_on_codes_model',
        'idx_usable_on_codes_uoc',
        'idx_marine_corps_management',
        'idx_marine_sl_62_item_id_niin',
        'idx_marine_sl_62_item_supp',
        'idx_moe_rule',
        'idx_part_number_niin',
        'idx_part_number_partNumber'
    )
ORDER BY tablename, indexname;
```

**Expected result**: 25 rows, all should show `USING btree`

### Check index sizes

```sql
SELECT
    schemaname,
    tablename,
    indexname,
    pg_size_pretty(pg_relation_size(indexname::regclass)) AS index_size
FROM pg_indexes
WHERE schemaname = 'public'
    AND indexname LIKE 'idx_%'
ORDER BY pg_relation_size(indexname::regclass) DESC
LIMIT 50;
```

---

## Performance Testing

After migration, run these queries to ensure performance is maintained or improved:

```sql
-- Test NIIN lookups (most common pattern)
EXPLAIN ANALYZE
SELECT * FROM flis_management WHERE niin = '012345678';

-- Test part number lookups
EXPLAIN ANALYZE
SELECT * FROM part_number WHERE part_number = 'ABC123';

-- Test cage code lookups
EXPLAIN ANALYZE
SELECT * FROM flis_reference WHERE cage_code = '12345';
```

**Expected**: Should see "Index Scan using <index_name>" with similar or better execution times

---

## Rollback Plan

If issues arise, hash indexes can be recreated (not recommended):

```sql
-- Example rollback for one index
DROP INDEX CONCURRENTLY idx_amdf_i_and_s_niin;
CREATE INDEX CONCURRENTLY idx_amdf_i_and_s_niin ON amdf_i_and_s USING hash (niin);
```

However, **rolling back is NOT recommended** unless critical issues occur, as btree indexes are superior in nearly all scenarios.

---

## Estimated Timeline

- **Preparation**: 1 hour (staging environment testing)
- **Migration execution**: 2-4 hours (depends on table sizes and system load)
- **Verification**: 1 hour (testing and monitoring)
- **Total**: 4-6 hours

---

## Post-Migration Checklist

- [ ] Run verification queries to confirm all hash indexes are gone
- [ ] Verify all 25 btree indexes were created successfully
- [ ] Test critical application queries for performance
- [ ] Monitor database logs for any index-related errors
- [ ] Check disk space usage (should be similar to before)
- [ ] Monitor query performance for 24-48 hours
- [ ] Update documentation to reflect index changes
- [ ] Mark issue #3 as complete in performance optimization document
