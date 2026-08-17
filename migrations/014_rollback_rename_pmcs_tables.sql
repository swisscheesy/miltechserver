-- Rollback: 014_rollback_rename_pmcs_tables.sql

BEGIN;

SET LOCAL lock_timeout = '2s';
SET LOCAL statement_timeout = '30s';

DO $$
DECLARE
    relation_name TEXT;
BEGIN
    FOREACH relation_name IN ARRAY ARRAY[
        'user_pmcs_inspections',
        'user_pmcs_faults',
        'user_pmcs_inspection_comments'
    ]
    LOOP
        IF NOT EXISTS (
            SELECT 1
            FROM pg_catalog.pg_class AS class
            JOIN pg_catalog.pg_namespace AS namespace
              ON namespace.oid = class.relnamespace
            WHERE namespace.nspname = 'public'
              AND class.relname = relation_name
              AND class.relkind = 'r'
        ) THEN
            RAISE EXCEPTION 'expected ordinary table public.% before migration 014 rollback', relation_name;
        END IF;
    END LOOP;

    FOREACH relation_name IN ARRAY ARRAY[
        'pmcs_sbs_inspections',
        'pmcs_sbs_faults',
        'pmcs_sbs_inspection_comments'
    ]
    LOOP
        IF pg_catalog.to_regclass(format('public.%I', relation_name)) IS NOT NULL THEN
            RAISE EXCEPTION 'legacy relation public.% already exists before migration 014 rollback', relation_name;
        END IF;
    END LOOP;
END
$$;

LOCK TABLE
    public.user_pmcs_inspections,
    public.user_pmcs_faults,
    public.user_pmcs_inspection_comments
    IN ACCESS EXCLUSIVE MODE;

DROP INDEX public.user_pmcs_inspections_performed_by_idx;
DROP INDEX public.user_pmcs_inspection_comments_author_id_idx;

ALTER INDEX public.user_pmcs_inspections_equipment_performed_idx RENAME TO idx_pmcs_sbs_inspections_equipment_performed;
ALTER INDEX public.user_pmcs_inspection_comments_pmcs_id_idx RENAME TO idx_pmcs_sbs_inspection_comments_pmcs_id;

ALTER TABLE public.user_pmcs_inspections RENAME CONSTRAINT user_pmcs_inspections_pkey TO pmcs_sbs_inspections_pkey;
ALTER TABLE public.user_pmcs_inspections RENAME CONSTRAINT fk_user_pmcs_inspections_equipment_id TO fk_pmcs_sbs_inspections_equipment_id;
ALTER TABLE public.user_pmcs_inspections RENAME CONSTRAINT fk_user_pmcs_inspections_performed_by TO fk_pmcs_sbs_inspections_performed_by;
ALTER TABLE public.user_pmcs_inspections RENAME CONSTRAINT user_pmcs_inspections_equipment_id_nonblank_check TO pmcs_sbs_inspections_equipment_id_nonblank_check;
ALTER TABLE public.user_pmcs_inspections RENAME CONSTRAINT user_pmcs_inspections_source_shape_check TO pmcs_sbs_inspections_source_shape_check;
ALTER TABLE public.user_pmcs_inspections RENAME CONSTRAINT user_pmcs_inspections_source_type_check TO pmcs_sbs_inspections_source_type_check;

ALTER TABLE public.user_pmcs_faults RENAME CONSTRAINT user_pmcs_faults_pkey TO pmcs_sbs_faults_pkey;
ALTER TABLE public.user_pmcs_faults RENAME CONSTRAINT fk_user_pmcs_faults_pmcs_id TO fk_pmcs_sbs_faults_pmcs_id;
ALTER TABLE public.user_pmcs_faults RENAME CONSTRAINT user_pmcs_faults_item_index_check TO pmcs_sbs_faults_item_index_check;
ALTER TABLE public.user_pmcs_faults RENAME CONSTRAINT user_pmcs_faults_nonblank_fields_check TO pmcs_sbs_faults_nonblank_fields_check;
ALTER TABLE public.user_pmcs_faults RENAME CONSTRAINT user_pmcs_faults_status_check TO pmcs_sbs_faults_status_check;

ALTER TABLE public.user_pmcs_inspection_comments RENAME CONSTRAINT user_pmcs_inspection_comments_pkey TO pmcs_sbs_inspection_comments_pkey;
ALTER TABLE public.user_pmcs_inspection_comments RENAME CONSTRAINT fk_user_pmcs_inspection_comments_author_id TO fk_pmcs_sbs_inspection_comments_author_id;
ALTER TABLE public.user_pmcs_inspection_comments RENAME CONSTRAINT fk_user_pmcs_inspection_comments_pmcs_id TO fk_pmcs_sbs_inspection_comments_pmcs_id;
ALTER TABLE public.user_pmcs_inspection_comments RENAME CONSTRAINT user_pmcs_inspection_comments_nonblank_check TO pmcs_sbs_inspection_comments_nonblank_check;

ALTER TABLE public.user_pmcs_inspections RENAME TO pmcs_sbs_inspections;
ALTER TABLE public.user_pmcs_faults RENAME TO pmcs_sbs_faults;
ALTER TABLE public.user_pmcs_inspection_comments RENAME TO pmcs_sbs_inspection_comments;

COMMIT;
