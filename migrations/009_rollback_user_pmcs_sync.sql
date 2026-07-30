BEGIN;
DROP TABLE IF EXISTS user_pmcs_subscriptions;
ALTER TABLE IF EXISTS user_pmcs_community_sources
    DROP CONSTRAINT IF EXISTS fk_user_pmcs_community_sources_current_release;
DROP TABLE IF EXISTS user_pmcs_community_releases;
DROP TABLE IF EXISTS user_pmcs_community_sources;
DROP TABLE IF EXISTS user_pmcs_procedure_steps;
DROP TABLE IF EXISTS user_pmcs_notices;
DROP TABLE IF EXISTS user_pmcs_items;
DROP TABLE IF EXISTS user_pmcs_section_models;
DROP TABLE IF EXISTS user_pmcs_sections;
DROP TABLE IF EXISTS user_pmcs_revision_models;
DROP TABLE IF EXISTS user_pmcs_revisions;
DROP TABLE IF EXISTS user_pmcs_checklists;
DROP TABLE IF EXISTS user_pmcs_sync_state;
COMMIT;
