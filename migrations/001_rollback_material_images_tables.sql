-- Rollback script for material_images tables
-- Migration: 001_rollback_material_images_tables.sql

-- Drop tables in reverse order of creation (due to foreign key constraints)
DROP TABLE IF EXISTS material_images_upload_limits CASCADE;
DROP TABLE IF EXISTS material_images_flags CASCADE;
DROP TABLE IF EXISTS material_images_votes CASCADE;
DROP TABLE IF EXISTS material_images CASCADE;