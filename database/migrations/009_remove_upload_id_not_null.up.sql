-- 009_remove_upload_id_not_null.up.sql
ALTER TABLE portfolio_items
    ALTER COLUMN upload_id DROP NOT NULL;