-- Rollback Equipment Services Feature Database Schema
-- Rollback Migration: 002_rollback_equipment_services_table.sql

-- Drop indexes first
DROP INDEX IF EXISTS idx_equipment_services_shop_completed;
DROP INDEX IF EXISTS idx_equipment_services_equipment_date;
DROP INDEX IF EXISTS idx_equipment_services_shop_date;
DROP INDEX IF EXISTS idx_equipment_services_is_completed;
DROP INDEX IF EXISTS idx_equipment_services_service_type;
DROP INDEX IF EXISTS idx_equipment_services_created_by;
DROP INDEX IF EXISTS idx_equipment_services_service_date;
DROP INDEX IF EXISTS idx_equipment_services_equipment_id;
DROP INDEX IF EXISTS idx_equipment_services_shop_id;

-- Drop the table
DROP TABLE IF EXISTS equipment_services;