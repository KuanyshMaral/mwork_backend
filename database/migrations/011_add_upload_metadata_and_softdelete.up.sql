-- 011_add_upload_metadata_and_softdelete.up.sql
ALTER TABLE uploads
    -- 1. Добавляем столбец 'module', который вызывает ошибку
    ADD COLUMN module VARCHAR(50),

    -- 2. Добавляем 'deleted_at' для Soft Delete (от предыдущей ошибки)
    ADD COLUMN deleted_at TIMESTAMPTZ;

-- Добавление индексов
CREATE INDEX idx_uploads_module ON uploads (module);
CREATE INDEX idx_uploads_deleted_at ON uploads (deleted_at);