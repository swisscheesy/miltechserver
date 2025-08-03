# PostgreSQL Database Schema Optimization Plan

## Overview
This document outlines a comprehensive plan to optimize the database schema by implementing proper primary keys for all tables. The analysis reveals that many tables use artificial ID fields as primary keys when natural, meaningful primary keys exist.

## Key Findings

### Tables with Proper Natural Primary Keys (No Changes Needed)
These tables already use appropriate primary keys:
- **NIIN-based tables**: air_force_management, amdf_billing, amdf_credit, amdf_freight, amdf_i_and_s, amdf_management, amdf_matcat, amdf_phrase, army_lin_to_niin, disposition, flis_identification, flis_management, flis_management_id, nsn
- **CAGE-based tables**: cage_address, cage_status_and_type
- **User/Shop tables**: users (uid), shops (id - appears to be a UUID)

### Tables Requiring Primary Key Optimization

#### 1. **army_management** (Currently uses artificial ID)
- **Current**: id (auto-increment)
- **Issue**: Contains 1,521,056 records with only 3 duplicate NIINs
- **Proposed Primary Key**: `niin`
- **Action**: 
  - Remove duplicate entries (only 3 cases)
  - Drop ID column
  - Make NIIN primary key

#### 2. **part_number** (Currently uses artificial ID)
- **Current**: id (auto-increment)
- **Issue**: No duplicates found for (niin, part_number, cage_code) combination
- **Proposed Primary Key**: Composite key `(niin, part_number, cage_code)`
- **Action**: 
  - Drop ID column
  - Create composite primary key
  - This properly represents the relationship: a part number from a specific manufacturer (CAGE) for a specific item (NIIN)

#### 3. **flis_reference** (Currently uses artificial ID)
- **Current**: id (auto-increment)
- **Issue**: No duplicates found for (niin, part_number, cage_code) combination
- **Proposed Primary Key**: Composite key `(niin, part_number, cage_code)`
- **Action**: 
  - Drop ID column
  - Create composite primary key
  - Similar to part_number table - represents cross-reference data

#### 4. **component_end_item** (Currently uses artificial ID)
- **Current**: id (auto-increment)
- **Issue**: Contains duplicates for (niin, wpn_sys_id) - up to 4 duplicates per combination
- **Proposed Primary Key**: Composite key `(niin, wpn_sys_id, wpn_sys_svc, wpn_sys_ind)`
- **Action**: 
  - Analyze duplicate records to ensure all 4 fields create unique combinations
  - Drop ID column
  - Create composite primary key

#### 5. **marines_sl_6_2_item_id** (Currently uses artificial ID)
- **Current**: id (auto-increment)
- **Issue**: High duplicate count for (niin, idn) - up to 17 duplicates
- **Analysis Needed**: Need to understand why so many duplicates exist
- **Proposed Options**:
  - Option A: Add additional fields to create uniqueness (e.g., exit_date, cec)
  - Option B: Keep ID but add unique constraint on meaningful combination
  - **Recommended**: Investigate data to determine business rules

#### 6. **colloquial_name** (Currently uses artificial ID)
- **Current**: id (auto-increment)
- **Issue**: Some duplicates for (inc, colloquial_name)
- **Proposed Primary Key**: Composite key `(inc, colloquial_name, related_inc)`
- **Action**: 
  - Verify this combination is unique
  - Drop ID column
  - Create composite primary key

#### 7. **shop_members** (Has ID but already has unique constraint)
- **Current**: id (UUID) with unique constraint on (shop_id, user_id)
- **Proposed Primary Key**: `(shop_id, user_id)`
- **Action**: 
  - Drop ID column
  - Convert unique constraint to primary key
  - This is a classic many-to-many relationship table

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

##### 10. **army_pack_supplemental_instruct** (Currently uses artificial ID)
- **Current**: id (auto-increment)
- **Analysis**: 252,833 records, all with unique NIINs
- **Proposed Primary Key**: `niin`
- **Action**: 
  - Drop ID column
  - Make NIIN primary key
  - No duplicates exist, so this is straightforward

##### 11. **moe_rule** (Currently uses artificial ID)
- **Current**: id (auto-increment)
- **Analysis**: 18M+ records with many NIINs having multiple MOE rules
- **Issue**: Each NIIN can have multiple MOE rules (up to 1,128 different rules)
- **Proposed Primary Key**: Composite key `(niin, moe_rl)`
- **Action**: 
  - Verify no duplicates exist for (niin, moe_rl) combination
  - Drop ID column
  - Create composite primary key

##### 12. **marines_sl_6_2_item_supp** (Currently uses artificial ID)
- **Current**: id (auto-increment)
- **Analysis**: 783,518 records with 776,747 unique (niin, idn) combinations
- **Issue**: Some duplicates exist (about 6,771 duplicate combinations)
- **Proposed Options**:
  - Option A: Add additional fields like qty3, qty4, or ptrf to create uniqueness
  - Option B: Keep ID but add unique constraint on meaningful combination
- **Recommended Action**: Investigate the duplicate records to understand business rules

##### 13. **quick_list_wheel_tires** (Currently uses artificial ID)
- **Current**: id (auto-increment)
- **Analysis**: No duplicates found for (vehicle, assembly_nsn, tire_nsn)
- **Proposed Primary Key**: Composite key `(vehicle, assembly_nsn, tire_nsn)`
- **Action**: 
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

## Implementation Strategy

### Phase 1: Low-Risk Changes (Tables with clear natural keys)
1. army_management → NIIN as primary key
2. shop_members → (shop_id, user_id) as primary key
3. army_pack_supplemental_instruct → NIIN as primary key

### Phase 2: Composite Key Implementation
1. part_number → (niin, part_number, cage_code)
2. flis_reference → (niin, part_number, cage_code)
3. colloquial_name → (inc, colloquial_name, related_inc)
4. moe_rule → (niin, moe_rl)
5. quick_list_wheel_tires → (vehicle, assembly_nsn, tire_nsn)

### Phase 3: Complex Cases Requiring Analysis
1. component_end_item → Analyze duplicate patterns
2. marines_sl_6_2_item_id → Understand business rules (17 duplicates per combination)
3. marines_sl_6_2_item_supp → Understand business rules (6,771 duplicate combinations)
4. shop_list_items → Determine uniqueness requirements
5. user_items_categorized → Simplify composite key

### Summary of Tables Already Optimized
The analysis revealed that many tables already have proper primary keys:
- 23 tables use NIIN as primary key
- 3 tables use other natural keys (LIN, CAGE_CODE, NSN)
- 5 tables use appropriate UUIDs (shops, users, shop_lists)
- Only 13 tables need optimization (down from initial estimate)

## Benefits of This Optimization

1. **Data Integrity**: Natural keys enforce business rules at the database level
2. **Performance**: Eliminating unnecessary ID lookups and joins
3. **Storage**: Removing redundant ID columns saves space
4. **Clarity**: Table relationships become self-documenting
5. **Maintenance**: Fewer indexes to maintain

## Next Steps

1. Create backup of current database
2. Analyze tables marked for "further analysis"
3. Create migration scripts for each phase
4. Test migrations in development environment
5. Plan maintenance window for production implementation
6. Execute migrations in phases with rollback capability

## Important Considerations

- Foreign key relationships will need to be updated when primary keys change
- Application code may need updates to handle composite keys
- Existing queries and stored procedures will need review
- Performance testing should be conducted after each phase