-- 008_add_module_to_uploads.up.sql
-- Добавляем только 'module', так как 'deleted_at' уже существует.
ALTER TABLE uploads
    ADD COLUMN module VARCHAR(50);

CREATE INDEX idx_uploads_module ON uploads (module);