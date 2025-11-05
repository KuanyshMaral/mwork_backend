BEGIN;

-- 1. Исправляем недостающие столбцы в subscription_plans
-- (На основе ошибок "столбец ... не существует")
ALTER TABLE public.subscription_plans ADD COLUMN IF NOT EXISTS duration VARCHAR(50);
ALTER TABLE public.subscription_plans ADD COLUMN IF NOT EXISTS payment_status VARCHAR(100);

-- 2. Добавляем недостающую таблицу notification_templates
-- (На основе ошибки "отношение "notification_templates" не существует")
CREATE TABLE IF NOT EXISTS public.notification_templates (
                                                             id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    created_at TIMESTAMPTZ DEFAULT now(),
    updated_at TIMESTAMPTZ DEFAULT now(),
    deleted_at TIMESTAMPTZ,
    type VARCHAR(100) NOT NULL UNIQUE,
    title TEXT,
    message TEXT,
    variables JSONB,
    is_active BOOLEAN DEFAULT true
    );

-- (Добавляем триггер и индексы для новой таблицы)
CREATE TRIGGER set_timestamp_notification_templates
    BEFORE UPDATE ON public.notification_templates
    FOR EACH ROW
    EXECUTE PROCEDURE trigger_set_timestamp();

CREATE INDEX IF NOT EXISTS idx_notification_templates_deleted_at ON public.notification_templates(deleted_at);
CREATE INDEX IF NOT EXISTS idx_notification_templates_type ON public.notification_templates(type);

-- 3. Добавляем все недостающие индексы GORM Soft Delete
-- (На основе 000_complete_schema.sql и запроса на "add, index")
CREATE INDEX IF NOT EXISTS idx_users_deleted_at ON public.users(deleted_at);
CREATE INDEX IF NOT EXISTS idx_model_profiles_deleted_at ON public.model_profiles(deleted_at);
CREATE INDEX IF NOT EXISTS idx_employer_profiles_deleted_at ON public.employer_profiles(deleted_at);
CREATE INDEX IF NOT EXISTS idx_subscription_plans_deleted_at ON public.subscription_plans(deleted_at);
CREATE INDEX IF NOT EXISTS idx_user_subscriptions_deleted_at ON public.user_subscriptions(deleted_at);
CREATE INDEX IF NOT EXISTS idx_payment_transactions_deleted_at ON public.payment_transactions(deleted_at);
CREATE INDEX IF NOT EXISTS idx_uploads_deleted_at ON public.uploads(deleted_at);
CREATE INDEX IF NOT EXISTS idx_portfolio_items_deleted_at ON public.portfolio_items(deleted_at);
CREATE INDEX IF NOT EXISTS idx_castings_deleted_at ON public.castings(deleted_at);
CREATE INDEX IF NOT EXISTS idx_casting_responses_deleted_at ON public.casting_responses(deleted_at);
CREATE INDEX IF NOT EXISTS idx_reviews_deleted_at ON public.reviews(deleted_at);
CREATE INDEX IF NOT EXISTS idx_notifications_deleted_at ON public.notifications(deleted_at);
CREATE INDEX IF NOT EXISTS idx_usage_tracking_deleted_at ON public.usage_tracking(deleted_at);
CREATE INDEX IF NOT EXISTS idx_refresh_tokens_deleted_at ON public.refresh_tokens(deleted_at);

-- 4. Добавляем недостающие индексы GORM Soft Delete для схемы CHAT
CREATE INDEX IF NOT EXISTS idx_chat_dialogs_deleted_at ON chat.dialogs(deleted_at);
CREATE INDEX IF NOT EXISTS idx_chat_messages_deleted_at ON chat.messages(deleted_at);
CREATE INDEX IF NOT EXISTS idx_chat_dialog_participants_deleted_at ON chat.dialog_participants(deleted_at);
CREATE INDEX IF NOT EXISTS idx_chat_message_attachments_deleted_at ON chat.message_attachments(deleted_at);
CREATE INDEX IF NOT EXISTS idx_chat_message_reactions_deleted_at ON chat.message_reactions(deleted_at);
CREATE INDEX IF NOT EXISTS idx_chat_message_read_receipts_deleted_at ON chat.message_read_receipts(deleted_at);

-- 5. Добавляем другие важные недостающие индексы
-- (На основе 000_complete_schema.sql, важно для upload_service)
CREATE INDEX IF NOT EXISTS idx_uploads_module ON public.uploads(module);

COMMIT;