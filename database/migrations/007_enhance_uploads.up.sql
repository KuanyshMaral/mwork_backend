-- Migration 007: Enhance uploads table for production-ready file system
-- Adds support for image variants, thumbnails, and advanced file metadata

BEGIN;

-- Add new columns to uploads table for enhanced file handling
ALTER TABLE uploads
    ADD COLUMN IF NOT EXISTS original_name TEXT,           -- Original filename from user
    ADD COLUMN IF NOT EXISTS url TEXT,                     -- Public URL for accessing the file
    ADD COLUMN IF NOT EXISTS thumbnail_path TEXT,          -- Path to thumbnail (for images)
    ADD COLUMN IF NOT EXISTS variants JSONB,               -- JSON with different sizes: {"small": "path", "medium": "path", "large": "path"}
    ADD COLUMN IF NOT EXISTS metadata JSONB,               -- Additional metadata (dimensions, duration, etc.)
    ADD COLUMN IF NOT EXISTS storage_provider TEXT DEFAULT 'local', -- 'local', 's3', 'cloudflare_r2'
    ADD COLUMN IF NOT EXISTS expires_at TIMESTAMPTZ,       -- For temporary files
    ADD COLUMN IF NOT EXISTS download_count INTEGER DEFAULT 0, -- Track downloads
    ADD COLUMN IF NOT EXISTS last_accessed_at TIMESTAMPTZ; -- Last access time for cleanup

-- Add index for storage provider queries
CREATE INDEX IF NOT EXISTS idx_uploads_storage_provider ON uploads(storage_provider);

-- Add index for expiration cleanup
CREATE INDEX IF NOT EXISTS idx_uploads_expires_at ON uploads(expires_at) WHERE expires_at IS NOT NULL;

-- Add index for entity lookups
CREATE INDEX IF NOT EXISTS idx_uploads_entity ON uploads(entity_type, entity_id);

-- Add index for usage type
CREATE INDEX IF NOT EXISTS idx_uploads_usage ON uploads(usage);

-- Add check constraint for storage provider
ALTER TABLE uploads
    ADD CONSTRAINT chk_uploads_storage_provider
        CHECK (storage_provider IN ('local', 's3', 'cloudflare_r2'));

-- Add trigger for updated_at
CREATE TRIGGER set_timestamp_uploads
    BEFORE UPDATE ON uploads
    FOR EACH ROW
    EXECUTE PROCEDURE trigger_set_timestamp();

COMMIT;
