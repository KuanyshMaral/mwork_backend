-- 010_add_soft_delete_to_uploads.up.sql
ALTER TABLE uploads
    ADD COLUMN deleted_at TIMESTAMPTZ;

-- Добавление индекса для быстрого поиска не удаленных записей
CREATE INDEX idx_uploads_deleted_at ON uploads (deleted_at);