-- User Suggestions Feature Database Schema
-- Migration: 004_create_user_suggestions_tables.sql

-- User suggestions table
CREATE TABLE user_suggestions (
    id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id     TEXT NOT NULL,
    title       TEXT NOT NULL CHECK (length(title) BETWEEN 1 AND 200),
    description TEXT NOT NULL CHECK (length(description) BETWEEN 1 AND 2000),
    status      TEXT NOT NULL DEFAULT 'Submitted',
    created_at  TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at  TIMESTAMPTZ
);

CREATE INDEX idx_user_suggestions_user_id ON user_suggestions (user_id);
CREATE INDEX idx_user_suggestions_created_at ON user_suggestions (created_at);

-- User suggestion votes table
CREATE TABLE user_suggestion_votes (
    suggestion_id UUID NOT NULL REFERENCES user_suggestions(id) ON DELETE CASCADE,
    voter_id      TEXT NOT NULL,
    direction     SMALLINT NOT NULL CHECK (direction IN (-1, 1)),
    created_at    TIMESTAMPTZ NOT NULL DEFAULT now(),
    PRIMARY KEY (suggestion_id, voter_id)
);
