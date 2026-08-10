-- Rollback: 012_rollback_user_pmcs_model_search_trigram_index.sql
--
-- Removes only the feature-owned index. pg_trgm may be shared, so the
-- extension is intentionally retained.

DROP INDEX CONCURRENTLY IF EXISTS
    user_pmcs_revision_models_search_trgm_idx;
