-- Item Query Performance Optimization - NIIN Index Rollback
-- Migration: 003_rollback_item_query_indexes.sql
--
-- Rollback script for the NIIN indexes. Use with caution -
-- removing these indexes will significantly degrade query performance.

-- AMDF tables
DROP INDEX CONCURRENTLY IF EXISTS idx_army_master_data_file_niin;
DROP INDEX CONCURRENTLY IF EXISTS idx_amdf_management_niin;
DROP INDEX CONCURRENTLY IF EXISTS idx_amdf_credit_niin;
DROP INDEX CONCURRENTLY IF EXISTS idx_amdf_billing_niin;
DROP INDEX CONCURRENTLY IF EXISTS idx_amdf_matcat_niin;
DROP INDEX CONCURRENTLY IF EXISTS idx_amdf_phrase_niin;
DROP INDEX CONCURRENTLY IF EXISTS idx_amdf_i_and_s_niin;
DROP INDEX CONCURRENTLY IF EXISTS idx_amdf_freight_niin;

-- FLIS tables
DROP INDEX CONCURRENTLY IF EXISTS idx_flis_management_niin;
DROP INDEX CONCURRENTLY IF EXISTS idx_flis_management_id_niin;
DROP INDEX CONCURRENTLY IF EXISTS idx_flis_standardization_niin;
DROP INDEX CONCURRENTLY IF EXISTS idx_flis_cancelled_niin_niin;
DROP INDEX CONCURRENTLY IF EXISTS idx_flis_phrase_niin;
DROP INDEX CONCURRENTLY IF EXISTS idx_flis_identification_niin;
DROP INDEX CONCURRENTLY IF EXISTS idx_flis_reference_niin;
DROP INDEX CONCURRENTLY IF EXISTS idx_flis_freight_niin;
DROP INDEX CONCURRENTLY IF EXISTS idx_flis_packaging_1_niin;
DROP INDEX CONCURRENTLY IF EXISTS idx_flis_packaging_2_niin;
DROP INDEX CONCURRENTLY IF EXISTS idx_flis_item_characteristics_niin;

-- Army-specific tables
DROP INDEX CONCURRENTLY IF EXISTS idx_army_management_niin;
DROP INDEX CONCURRENTLY IF EXISTS idx_army_sarsscat_niin;
DROP INDEX CONCURRENTLY IF EXISTS idx_army_packaging_and_freight_niin;
DROP INDEX CONCURRENTLY IF EXISTS idx_army_packaging_1_niin;
DROP INDEX CONCURRENTLY IF EXISTS idx_army_packaging_2_niin;
DROP INDEX CONCURRENTLY IF EXISTS idx_army_freight_niin;
DROP INDEX CONCURRENTLY IF EXISTS idx_army_line_item_number_lin;
DROP INDEX CONCURRENTLY IF EXISTS idx_army_packaging_special_instruct_niin;
DROP INDEX CONCURRENTLY IF EXISTS idx_army_pack_supplemental_instruct_niin;

-- Service branch management tables
DROP INDEX CONCURRENTLY IF EXISTS idx_air_force_management_niin;
DROP INDEX CONCURRENTLY IF EXISTS idx_marine_corps_management_niin;
DROP INDEX CONCURRENTLY IF EXISTS idx_navy_management_niin;
DROP INDEX CONCURRENTLY IF EXISTS idx_faa_management_niin;

-- Reference and component tables
DROP INDEX CONCURRENTLY IF EXISTS idx_component_end_item_niin;
DROP INDEX CONCURRENTLY IF EXISTS idx_disposition_niin;
DROP INDEX CONCURRENTLY IF EXISTS idx_moe_rule_niin;
DROP INDEX CONCURRENTLY IF EXISTS idx_dss_weight_and_cube_niin;

-- CAGE lookup tables
DROP INDEX CONCURRENTLY IF EXISTS idx_cage_address_cage_code;
DROP INDEX CONCURRENTLY IF EXISTS idx_cage_status_and_type_cage_code;

-- Part number lookup
DROP INDEX CONCURRENTLY IF EXISTS idx_part_number_part_number;
DROP INDEX CONCURRENTLY IF EXISTS idx_part_number_niin;

-- Colloquial name lookup
DROP INDEX CONCURRENTLY IF EXISTS idx_colloquial_name_inc;
