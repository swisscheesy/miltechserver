-- User PMCS Community Model Contains Search
-- Migration: 012_add_user_pmcs_model_search_trigram_index.sql
--
-- Adds the operator class required to index literal leading-wildcard LIKE
-- searches on normalized revision-level model text.

CREATE EXTENSION IF NOT EXISTS pg_trgm;

CREATE INDEX CONCURRENTLY IF NOT EXISTS
    user_pmcs_revision_models_search_trgm_idx
    ON user_pmcs_revision_models
    USING GIN (normalized_text gin_trgm_ops);
