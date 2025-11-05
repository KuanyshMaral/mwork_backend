CREATE TABLE IF NOT EXISTS notification_templates (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    created_at TIMESTAMPTZ DEFAULT now(),
    updated_at TIMESTAMPTZ DEFAULT now(),
    deleted_at TIMESTAMPTZ, -- (Для GORM Soft Delete)

    type VARCHAR(100) NOT NULL UNIQUE,
    title TEXT,
    message TEXT,
    variables JSONB,
    is_active BOOLEAN DEFAULT true
    );

CREATE TRIGGER set_timestamp_notification_templates
    BEFORE UPDATE ON notification_templates
    FOR EACH ROW
    EXECUTE PROCEDURE trigger_set_timestamp();

CREATE INDEX idx_notification_templates_deleted_at ON notification_templates(deleted_at);
CREATE INDEX idx_notification_templates_type ON notification_templates(type);