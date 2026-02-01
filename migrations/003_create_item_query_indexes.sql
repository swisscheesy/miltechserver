-- Item Query Performance Optimization - NIIN Index Migration
-- Migration: 003_create_item_query_indexes.sql
--
-- This migration creates indexes on the `niin` column across all tables
-- queried by the detailed item query endpoint. These indexes significantly
-- improve lookup performance for NIIN-based queries.
--
-- All indexes are created CONCURRENTLY to avoid locking production tables.
-- If an index already exists, the IF NOT EXISTS clause prevents errors.

-- AMDF (Army Master Data File) tables
CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_army_master_data_file_niin ON army_master_data_file(niin);
CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_amdf_management_niin ON amdf_management(niin);
CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_amdf_credit_niin ON amdf_credit(niin);
CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_amdf_billing_niin ON amdf_billing(niin);
CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_amdf_matcat_niin ON amdf_matcat(niin);
CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_amdf_phrase_niin ON amdf_phrase(niin);
CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_amdf_i_and_s_niin ON amdf_i_and_s(niin);
CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_amdf_freight_niin ON amdf_freight(niin);

-- FLIS (Federal Logistics Information System) tables
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

-- Army-specific tables
CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_army_management_niin ON army_management(niin);
CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_army_sarsscat_niin ON army_sarsscat(niin);
CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_army_packaging_and_freight_niin ON army_packaging_and_freight(niin);
CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_army_packaging_1_niin ON army_packaging_1(niin);
CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_army_packaging_2_niin ON army_packaging_2(niin);
CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_army_freight_niin ON army_freight(niin);
CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_army_line_item_number_lin ON army_line_item_number(lin);
CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_army_packaging_special_instruct_niin ON army_packaging_special_instruct(niin);
CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_army_pack_supplemental_instruct_niin ON army_pack_supplemental_instruct(niin);

-- Service branch management tables
CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_air_force_management_niin ON air_force_management(niin);
CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_marine_corps_management_niin ON marine_corps_management(niin);
CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_navy_management_niin ON navy_management(niin);
CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_faa_management_niin ON faa_management(niin);

-- Reference and component tables
CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_component_end_item_niin ON component_end_item(niin);
CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_disposition_niin ON disposition(niin);
CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_moe_rule_niin ON moe_rule(niin);
CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_dss_weight_and_cube_niin ON dss_weight_and_cube(niin);

-- CAGE (Commercial and Government Entity) lookup tables
-- These are queried by cage_code, not niin
CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_cage_address_cage_code ON cage_address(cage_code);
CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_cage_status_and_type_cage_code ON cage_status_and_type(cage_code);

-- Part number lookup (for short queries)
CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_part_number_part_number ON part_number(part_number);
CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_part_number_niin ON part_number(niin);

-- Colloquial name lookup (for identification queries)
CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_colloquial_name_inc ON colloquial_name(inc);
