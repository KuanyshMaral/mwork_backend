-- Rollback migration 007: Remove enhanced upload fields

BEGIN;

-- Drop trigger
DROP TRIGGER IF EXISTS set_timestamp_uploads ON uploads;

-- Drop constraints
ALTER TABLE uploads DROP CONSTRAINT IF EXISTS chk_uploads_storage_provider;

-- Drop indexes
DROP INDEX IF EXISTS idx_uploads_storage_provider;
DROP INDEX IF EXISTS idx_uploads_expires_at;
DROP INDEX IF EXISTS idx_uploads_entity;
DROP INDEX IF EXISTS idx_uploads_usage;

-- Drop columns
ALTER TABLE uploads
DROP COLUMN IF EXISTS original_name,
    DROP COLUMN IF EXISTS url,
    DROP COLUMN IF EXISTS thumbnail_path,
    DROP COLUMN IF EXISTS variants,
    DROP COLUMN IF EXISTS metadata,
    DROP COLUMN IF EXISTS storage_provider,
    DROP COLUMN IF EXISTS expires_at,
    DROP COLUMN IF EXISTS download_count,
    DROP COLUMN IF EXISTS last_accessed_at;

COMMIT;
