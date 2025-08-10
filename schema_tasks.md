# PostgreSQL Database Schema Optimization Plan

## Overview
This document outlines a comprehensive plan to optimize the database schema by implementing proper primary keys for all tables. The analysis reveals that many tables use artificial ID fields as primary keys when natural, meaningful primary keys exist.

## Last Updated: 2025-08-09
- Verified all completed optimizations in production database
- Analyzed additional tables for optimization opportunities
- Updated recommendations for remaining tables

## Verification Status (2025-08-09)

### Successfully Completed Optimizations ✓
The following tables have been successfully migrated to use natural primary keys:

1. **part_number** - Composite key (niin, part_number, cage_code) ✓
2. **flis_reference** - Composite key (niin, part_number, cage_code) ✓
3. **component_end_item** - Composite key (niin, wpn_sys_id, wpn_sys_svc) ✓
4. **colloquial_name** - Composite key (inc, colloquial_name, related_inc) ✓
5. **army_pack_supplemental_instruct** - NIIN as primary key ✓
6. **moe_rule** - Composite key (niin, moe_rl) ✓

## Key Findings

### Tables with Proper Natural Primary Keys (No Changes Needed)
These tables already use appropriate primary keys:
- **NIIN-based tables**: air_force_management, amdf_billing, amdf_credit, amdf_freight, amdf_i_and_s, amdf_management, amdf_matcat, amdf_phrase, army_lin_to_niin, disposition, flis_identification, flis_management, flis_management_id, nsn, coast_guard_management, army_packaging_and_freight, army_packaging_special_instruct, army_sarsscat, flis_freight
- **CAGE-based tables**: cage_address, cage_status_and_type
- **Composite key tables**: lookup_uoc (uoc, model)
- **User/Shop tables**: users (uid), shops (id - appears to be a UUID)

### Tables Requiring Primary Key Optimization

#### NEW FINDINGS - Additional Tables Identified for Optimization (2025-08-09)

##### High Priority Optimizations (Clear Natural Keys)

1. **army_substitute_lin** (249 records, NO duplicates)
   - **Current**: id (auto-increment)
   - **Proposed Primary Key**: Composite key `(lin, substitute_lin)`
   - **Justification**: Perfect uniqueness, represents substitution relationship
   - **Action**: Drop ID, create composite primary key

2. **flis_cancelled_niin** (890,272 records, NO duplicates)
   - **Current**: id (auto-increment)
   - **Proposed Primary Key**: Composite key `(niin, cancelled_niin)`
   - **Justification**: Perfect uniqueness, represents cancellation relationship
   - **Action**: Drop ID, create composite primary key

3. **flis_standardization** (389,251 records, NO duplicates)
   - **Current**: id (auto-increment)
   - **Proposed Primary Key**: Composite key `(niin, related_nsn)`
   - **Justification**: Perfect uniqueness, represents standardization relationship
   - **Action**: Drop ID, create composite primary key

##### Tables with Minor Duplicates Requiring Cleanup

4. **faa_management** (26,262 records, 57 duplicate NIINs)
   - **Current**: id (auto-increment)
   - **Issue**: Small number of duplicates (57 out of 26,262)
   - **Proposed Primary Key**: `niin`
   - **Action**: 
     - Investigate and remove/merge duplicate entries
     - Drop ID column
     - Make NIIN primary key

5. **flis_packaging_1** (3,970,593 records, 6,231 duplicate NIINs)
   - **Current**: id (auto-increment)
   - **Issue**: Small percentage of duplicates (0.16%)
   - **Analysis Needed**: Determine if duplicates are data errors or valid variations
   - **Proposed Options**:
     - Option A: Clean duplicates and use NIIN as primary key
     - Option B: Find additional field to create composite key (e.g., pica_sica, pkg_cat)

6. **flis_packaging_2** (3,970,593 records, 6,231 duplicate NIINs - identical to packaging_1)
   - **Current**: id (auto-increment)
   - **Issue**: Same duplicate pattern as flis_packaging_1
   - **Note**: These tables appear to be related and should be handled together

##### Complex Tables Requiring Business Logic Analysis

7. **flis_item_characteristics** (44.5M records, 1.5M duplicates for niin+mrc)
   - **Current**: id (auto-increment)
   - **Issue**: Significant duplicates (3.4% of records)
   - **Analysis**: Need to understand why same NIIN+MRC has multiple characteristics
   - **Proposed**: Keep ID but add indexes for performance

#### Previously Identified Tables Still Requiring Optimization

#### 1. **army_management** (Currently uses artificial ID)
- **Current**: id (auto-increment)
- **Issue**: Contains 1,521,056 records with only 3 duplicate NIINs (verified 2025-08-09)
- **Proposed Primary Key**: `niin`
- **Action**: 
  - Remove duplicate entries (only 3 cases)
  - Drop ID column
  - Make NIIN primary key

#### 2. **part_number** (COMPLETED ✓)
- **Previous**: id (auto-increment)
- **Issue**: Invalid NIINs starting with 'LLH' were removed; null cage_code values were cleaned up
- **Implemented Primary Key**: Composite key `(niin, part_number, cage_code)`
- **Status**: COMPLETED
  - ID column dropped
  - Invalid LLH NIINs removed
  - Composite primary key successfully created
  - All three columns (niin, part_number, cage_code) are now NOT NULL
  - This properly represents the relationship: a part number from a specific manufacturer (CAGE) for a specific item (NIIN)

#### 3. **flis_reference** (COMPLETED ✓)
- **Previous**: id (auto-increment)
- **Implemented Primary Key**: Composite key `(niin, part_number, cage_code)`
- **Status**: COMPLETED
  - ID column dropped
  - Composite primary key successfully created
  - All three columns (niin, part_number, cage_code) are now NOT NULL
  - Similar to part_number table - represents cross-reference data

#### 4. **component_end_item** (COMPLETED ✓)
- **Previous**: id (auto-increment)
- **Issue**: Contains duplicates for (niin, wpn_sys_id) - resolved by including wpn_sys_svc
- **Implemented Primary Key**: Composite key `(niin, wpn_sys_id, wpn_sys_svc)`
- **Status**: COMPLETED
  - ID column dropped
  - Composite primary key successfully created with 3 columns (not 4 as originally proposed)
  - All three key columns (niin, wpn_sys_id, wpn_sys_svc) are now NOT NULL
  - wpn_sys_ind remains nullable and is not part of the primary key

#### 5. **marines_sl_6_2_item_id** (Currently uses artificial ID)
- **Current**: id (auto-increment)
- **Issue**: High duplicate count for (niin, idn) - up to 17 duplicates
- **Analysis Needed**: Need to understand why so many duplicates exist
- **Proposed Options**:
  - Option A: Add additional fields to create uniqueness (e.g., exit_date, cec)
  - Option B: Keep ID but add unique constraint on meaningful combination
  - **Recommended**: Investigate data to determine business rules

#### 6. **colloquial_name** (COMPLETED ✓)
- **Previous**: id (auto-increment)
- **Issue**: Some duplicates for (inc, colloquial_name) - resolved by including related_inc
- **Implemented Primary Key**: Composite key `(inc, colloquial_name, related_inc)`
- **Status**: COMPLETED
  - ID column dropped
  - Composite primary key successfully created
  - All three columns (inc, colloquial_name, related_inc) are now NOT NULL
  - This properly represents colloquial name relationships

#### 7. **shop_members** (Has ID but already has unique constraint) - VERIFIED 2025-08-09
- **Current**: id (TEXT/UUID) with unique constraint on (shop_id, user_id)
- **Confirmed**: Unique constraint exists as `shop_members_shop_id_user_id_key`
- **Proposed Primary Key**: `(shop_id, user_id)`
- **Action**: 
  - Drop ID column
  - Drop existing primary key constraint
  - Convert unique constraint to primary key
  - This is a classic many-to-many relationship table
- **Note**: This is user-generated data but the optimization is safe as the unique constraint already enforces the business rule

#### 8. **shop_list_items** (Currently uses artificial ID)
- **Current**: id (UUID)
- **Analysis**: Need to determine if multiple instances of same NIIN can exist in a list
- **Proposed Options**:
  - If one NIIN per list: Primary key `(list_id, niin)`
  - If multiple allowed: Keep ID or add sequence number

#### 9. **user_items_categorized** (Complex composite key)
- **Current**: Composite key (niin, category_id, id)
- **Issue**: Having 'id' in a composite key defeats the purpose
- **Proposed**: Analyze if (user_id, niin, category_id) is unique
- **Action**: 
  - If unique, use as primary key
  - If not, determine what makes records unique

### Tables Requiring Further Analysis - COMPLETED

After detailed analysis, these tables were found to already have appropriate primary keys:

#### Tables Already Using NIIN as Primary Key (No Changes Needed):
- **army_freight**: Already uses NIIN as primary key ✓
- **army_line_item_number**: Already uses LIN as primary key ✓
- **army_master_data_file**: Already uses NIIN as primary key ✓
- **army_packaging_1**: Already uses NIIN as primary key ✓
- **army_packaging_2**: Already uses NIIN as primary key ✓
- **army_related_nsn**: Already uses NIIN as primary key ✓
- **dss_weight_and_cube**: Already uses NIIN as primary key ✓
- **marines_mhif**: Already uses NIIN as primary key ✓
- **marines_sl_6_1**: Already uses NIIN as primary key ✓

#### Tables Already Using Other Natural Keys (No Changes Needed):
- **quick_list_battery**: Already uses NSN as primary key ✓
- **quick_list_clothing**: Already uses NSN as primary key ✓

#### Tables Still Requiring Optimization:

##### 10. **army_pack_supplemental_instruct** (COMPLETED ✓)
- **Previous**: id (auto-increment)
- **Analysis**: 252,833 records, all with unique NIINs
- **Implemented Primary Key**: `niin`
- **Status**: COMPLETED
  - ID column dropped
  - NIIN is now the primary key
  - NIIN column is now NOT NULL
  - No duplicates existed, so conversion was straightforward

##### 11. **moe_rule** (COMPLETED ✓)
- **Previous**: id (auto-increment)
- **Analysis**: 18M+ records with many NIINs having multiple MOE rules
- **Issue**: Each NIIN can have multiple MOE rules (up to 1,128 different rules)
- **Implemented Primary Key**: Composite key `(niin, moe_rl)`
- **Status**: COMPLETED
  - ID column dropped
  - Composite primary key successfully created
  - Both columns (niin, moe_rl) are now NOT NULL

##### 12. **marines_sl_6_2_item_supp** (Currently uses artificial ID)
- **Current**: id (auto-increment)
- **Analysis**: 783,518 records with 776,747 unique (niin, idn) combinations
- **Issue**: Some duplicates exist (about 6,771 duplicate combinations)
- **Proposed Options**:
  - Option A: Add additional fields like qty3, qty4, or ptrf to create uniqueness
  - Option B: Keep ID but add unique constraint on meaningful combination
- **Recommended Action**: Investigate the duplicate records to understand business rules

##### 13. **quick_list_wheel_tires** (Currently uses artificial ID) - VERIFIED 2025-08-09
- **Current**: id (auto-increment)
- **Confirmed**: Table structure verified, nullable fields present
- **Issue**: All key fields (vehicle, assembly_nsn, tire_nsn) are currently nullable
- **Proposed Primary Key**: Composite key `(vehicle, assembly_nsn, tire_nsn)`
- **Action**: 
  - First clean any NULL values in key fields
  - Set key fields to NOT NULL
  - Drop ID column
  - Create composite primary key
  - This properly represents the relationship between vehicle and tire specifications

### User and Shop Related Tables

These modern application tables generally have appropriate keys but some could be optimized:
- **shop_invite_codes**: Verify if code is unique
- **shop_messages**: Keep ID (messages need unique identifiers)
- **shop_notification_items**: Analyze uniqueness requirements
- **user_item_comments**: Keep ID (comments need unique identifiers)
- **user_vehicle_comments**: Keep ID (comments need unique identifiers)

## Implementation Strategy (Updated 2025-08-09)

### Phase 1: Low-Risk Changes (Tables with NO duplicates and clear natural keys)
**NEW ADDITIONS:**
1. army_substitute_lin → (lin, substitute_lin) as primary key - 249 records, NO duplicates
2. flis_cancelled_niin → (niin, cancelled_niin) as primary key - 890K records, NO duplicates
3. flis_standardization → (niin, related_nsn) as primary key - 389K records, NO duplicates

**Previously Identified:**
4. shop_members → (shop_id, user_id) as primary key (has unique constraint already)
5. quick_list_wheel_tires → (vehicle, assembly_nsn, tire_nsn) after NULL cleanup

### Phase 2: Tables Requiring Minor Data Cleanup
1. army_management → NIIN as primary key (remove 3 duplicates)
2. faa_management → NIIN as primary key (investigate 57 duplicates)
3. flis_packaging_1 → Investigate 6,231 duplicates (0.16% of data)
4. flis_packaging_2 → Investigate 6,231 duplicates (same as packaging_1)

### Phase 3: Complex Cases Requiring Analysis
1. ~~component_end_item → Analyze duplicate patterns~~ **COMPLETED ✓** (moved to Phase 2 with (niin, wpn_sys_id, wpn_sys_svc))
2. marines_sl_6_2_item_id → Understand business rules (17 duplicates per combination)
3. marines_sl_6_2_item_supp → Understand business rules (6,771 duplicate combinations)
4. shop_list_items → Determine uniqueness requirements
5. user_items_categorized → Simplify composite key

### Summary of Tables Already Optimized (Updated 2025-08-09)
The comprehensive analysis revealed:
- **27 tables** already use NIIN as primary key (4 more identified)
- **4 tables** use other natural keys (LIN, CAGE_CODE, NSN, composite keys)
- **5 tables** use appropriate UUIDs (shops, users, shop_lists)
- **6 tables** successfully optimized (part_number, flis_reference, component_end_item, colloquial_name, army_pack_supplemental_instruct, moe_rule)
- **20+ tables** need optimization (7 new tables identified today)

## Benefits of This Optimization

1. **Data Integrity**: Natural keys enforce business rules at the database level
2. **Performance**: Eliminating unnecessary ID lookups and joins
3. **Storage**: Removing redundant ID columns saves space
4. **Clarity**: Table relationships become self-documenting
5. **Maintenance**: Fewer indexes to maintain

## Next Steps (Priority Order)

### Immediate Actions (Phase 1 - No Data Loss Risk)
1. **Create backup of current database**
2. **Implement Phase 1 optimizations** (tables with NO duplicates):
   - army_substitute_lin
   - flis_cancelled_niin
   - flis_standardization
3. **Clean NULL values and optimize**:
   - quick_list_wheel_tires (check for NULLs first)
   - shop_members (already has unique constraint)

### Short-term Actions (Phase 2 - Minor Cleanup Required)
4. **Investigate and clean duplicates**:
   - army_management (only 3 duplicates)
   - faa_management (57 duplicates - investigate cause)
5. **Analyze packaging tables together**:
   - flis_packaging_1 and flis_packaging_2 (identical duplicate patterns)

### Long-term Actions (Phase 3 - Complex Analysis)
6. **Deep analysis required**:
   - marines_sl_6_2_item_id (high duplicate count)
   - marines_sl_6_2_item_supp (moderate duplicates)
   - flis_item_characteristics (1.5M duplicates - may need to keep ID)
7. **Test migrations in development environment**
8. **Plan maintenance window for production implementation**
9. **Execute migrations in phases with rollback capability**

## Important Considerations

- Foreign key relationships will need to be updated when primary keys change
- Application code may need updates to handle composite keys
- Existing queries and stored procedures will need review
- Performance testing should be conducted after each phase
- Shop and user tables are excluded from optimization due to constant flux and user-generated nature

## Migration Script Templates

### Template for Tables with NO Duplicates (Phase 1)
```sql
-- Example: army_substitute_lin
BEGIN;
ALTER TABLE army_substitute_lin DROP CONSTRAINT IF EXISTS army_substitute_lin_pkey CASCADE;
ALTER TABLE army_substitute_lin ADD PRIMARY KEY (lin, substitute_lin);
ALTER TABLE army_substitute_lin DROP COLUMN id;
COMMIT;
```

### Template for Tables Requiring Duplicate Cleanup (Phase 2)
```sql
-- Example: army_management
BEGIN;
-- First identify duplicates
WITH duplicates AS (
  SELECT niin, COUNT(*) as cnt
  FROM army_management
  GROUP BY niin
  HAVING COUNT(*) > 1
)
SELECT * FROM army_management WHERE niin IN (SELECT niin FROM duplicates);
-- Manual review and cleanup required here
-- Then proceed with PK change
ALTER TABLE army_management DROP CONSTRAINT IF EXISTS army_management_pkey CASCADE;
ALTER TABLE army_management ADD PRIMARY KEY (niin);
ALTER TABLE army_management DROP COLUMN id;
COMMIT;
```

## Performance Impact Analysis
- **Positive**: Direct NIIN lookups will be faster (no ID indirection)
- **Positive**: JOIN operations on natural keys will be more efficient
- **Neutral**: Composite keys may slightly increase index size but improve query selectivity
- **Monitor**: Write performance on tables with composite keys