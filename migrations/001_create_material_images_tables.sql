-- Material Images Feature Database Schema
-- Migration: 001_create_material_images_tables.sql

-- 1. material_images table
CREATE TABLE material_images (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    niin VARCHAR(9) NOT NULL,
    user_id TEXT NOT NULL,
    blob_name TEXT NOT NULL UNIQUE,
    blob_url TEXT NOT NULL,
    original_filename TEXT NOT NULL,
    file_size_bytes BIGINT NOT NULL,
    mime_type VARCHAR(100) NOT NULL,
    upload_date TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    is_active BOOLEAN NOT NULL DEFAULT true,
    is_flagged BOOLEAN NOT NULL DEFAULT false,
    flag_count INTEGER NOT NULL DEFAULT 0,
    downvote_count INTEGER NOT NULL DEFAULT 0,
    upvote_count INTEGER NOT NULL DEFAULT 0,
    net_votes INTEGER GENERATED ALWAYS AS (upvote_count - downvote_count) STORED,
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    CONSTRAINT fk_user FOREIGN KEY (user_id) REFERENCES users(uid) ON DELETE CASCADE,
    CONSTRAINT valid_niin CHECK (length(niin) = 9)
);

-- Indexes for performance
CREATE INDEX idx_material_images_niin ON material_images(niin) WHERE is_active = true;
CREATE INDEX idx_material_images_user_id ON material_images(user_id);
CREATE INDEX idx_material_images_upload_date ON material_images(upload_date DESC);
CREATE INDEX idx_material_images_net_votes ON material_images(net_votes DESC) WHERE is_active = true;
CREATE INDEX idx_material_images_flagged ON material_images(is_flagged) WHERE is_flagged = true;

-- 2. material_images_votes table
CREATE TABLE material_images_votes (
    image_id UUID NOT NULL,
    user_id TEXT NOT NULL,
    vote_type VARCHAR(10) NOT NULL CHECK (vote_type IN ('upvote', 'downvote')),
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    PRIMARY KEY (image_id, user_id),
    CONSTRAINT fk_image FOREIGN KEY (image_id) REFERENCES material_images(id) ON DELETE CASCADE,
    CONSTRAINT fk_user FOREIGN KEY (user_id) REFERENCES users(uid) ON DELETE CASCADE
);

-- Index for user vote lookups
CREATE INDEX idx_material_images_votes_user ON material_images_votes(user_id);

-- 3. material_images_flags table
CREATE TABLE material_images_flags (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    image_id UUID NOT NULL,
    user_id TEXT NOT NULL,
    reason VARCHAR(50) NOT NULL CHECK (reason IN ('incorrect_item', 'inappropriate', 'poor_quality', 'duplicate', 'other')),
    description TEXT,
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    CONSTRAINT fk_image FOREIGN KEY (image_id) REFERENCES material_images(id) ON DELETE CASCADE,
    CONSTRAINT fk_user FOREIGN KEY (user_id) REFERENCES users(uid) ON DELETE CASCADE,
    CONSTRAINT unique_user_image_flag UNIQUE (image_id, user_id)
);

-- Indexes for flag management
CREATE INDEX idx_material_images_flags_image ON material_images_flags(image_id);
CREATE INDEX idx_material_images_flags_user ON material_images_flags(user_id);

-- 4. material_images_upload_limits table
CREATE TABLE material_images_upload_limits (
    user_id TEXT NOT NULL,
    niin VARCHAR(9) NOT NULL,
    last_upload_time TIMESTAMP NOT NULL,
    upload_count INTEGER NOT NULL DEFAULT 1,
    PRIMARY KEY (user_id, niin),
    CONSTRAINT fk_user FOREIGN KEY (user_id) REFERENCES users(uid) ON DELETE CASCADE
);

-- Index for cleanup operations
CREATE INDEX idx_upload_limits_time ON material_images_upload_limits(last_upload_time);