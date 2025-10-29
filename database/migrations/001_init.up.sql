-- Начинаем транзакцию
BEGIN;

-- 1. УСТАНОВКА РАСШИРЕНИЙ И СХЕМ
CREATE EXTENSION IF NOT EXISTS "uuid-ossp";
CREATE SCHEMA IF NOT EXISTS chat;

-- 2. СОЗДАНИЕ ПЕРЕЧИСЛЕНИЙ (ENUM TYPES)
CREATE TYPE user_status AS ENUM (
    'pending',
    'active',
    'suspended',
    'banned'
);

CREATE TYPE user_role AS ENUM (
    'model',
    'employer',
    'admin'
);

CREATE TYPE casting_status AS ENUM (
    'draft',
    'active',
    'closed',
    'cancelled'
);

CREATE TYPE response_status AS ENUM (
    'pending',
    'accepted',
    'rejected',
    'withdrawn'
);

CREATE TYPE subscription_status AS ENUM (
    'active',
    'expired',
    'cancelled'
);

CREATE TYPE payment_status AS ENUM (
    'pending',
    'paid',
    'failed',
    'refunded'
);

-- На основе review.go (const)
CREATE TYPE review_status AS ENUM (
    'pending',
    'approved',
    'rejected'
);

-- 3. ТРИГГЕРНАЯ ФУНКЦИЯ ДЛЯ 'updated_at'
-- Эта функция будет автоматически обновлять поле 'updated_at' при любом изменении строки
CREATE OR REPLACE FUNCTION trigger_set_timestamp()
RETURNS TRIGGER AS $$
BEGIN
  NEW.updated_at = NOW();
RETURN NEW;
END;
$$ LANGUAGE plpgsql;


-- 4. СОЗДАНИЕ ТАБЛИЦ (в порядке зависимостей)

-- Таблица 'users' (из response.go)
CREATE TABLE IF NOT EXISTS users (
                                     id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    created_at TIMESTAMPTZ DEFAULT now(),
    updated_at TIMESTAMPTZ DEFAULT now(),

    name VARCHAR(100) NOT NULL,
    email TEXT NOT NULL,
    password_hash TEXT NOT NULL,
    role user_role NOT NULL,
    status user_status DEFAULT 'pending',
    is_verified BOOLEAN DEFAULT false,
    verification_token TEXT,
    reset_token TEXT,
    reset_token_exp TIMESTAMPTZ,

    CONSTRAINT uq_users_email UNIQUE(email)
    );
CREATE TRIGGER set_timestamp
    BEFORE UPDATE ON users
    FOR EACH ROW
    EXECUTE PROCEDURE trigger_set_timestamp();

-- Таблица 'subscription_plans' (из response.go)
CREATE TABLE IF NOT EXISTS subscription_plans (
                                                  id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    created_at TIMESTAMPTZ DEFAULT now(),
    updated_at TIMESTAMPTZ DEFAULT now(),

    name TEXT NOT NULL,
    price NUMERIC(10, 2) NOT NULL,
    currency VARCHAR(10) DEFAULT 'KZT',
    duration TEXT NOT NULL,
    features JSONB,
    limits JSONB,
    is_active BOOLEAN DEFAULT true,
    payment_status TEXT
    );
CREATE TRIGGER set_timestamp
    BEFORE UPDATE ON subscription_plans
    FOR EACH ROW
    EXECUTE PROCEDURE trigger_set_timestamp();

-- Таблица 'uploads' (из portfolio.go)
CREATE TABLE IF NOT EXISTS uploads (
                                       id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    created_at TIMESTAMPTZ DEFAULT now(),
    updated_at TIMESTAMPTZ DEFAULT now(),

    user_id UUID NOT NULL,
    entity_type TEXT,
    entity_id TEXT,
    file_type TEXT,
    usage TEXT,
    path TEXT NOT NULL,
    mime_type TEXT,
    size BIGINT,
    is_public BOOLEAN DEFAULT true,

    FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE
    );
CREATE INDEX IF NOT EXISTS idx_uploads_user_id ON uploads(user_id);
CREATE TRIGGER set_timestamp
    BEFORE UPDATE ON uploads
    FOR EACH ROW
    EXECUTE PROCEDURE trigger_set_timestamp();

-- Таблица 'model_profiles' (из profile.go)
CREATE TABLE IF NOT EXISTS model_profiles (
                                              id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    created_at TIMESTAMPTZ DEFAULT now(),
    updated_at TIMESTAMPTZ DEFAULT now(),

    user_id UUID NOT NULL,
    name TEXT NOT NULL,
    age INTEGER NOT NULL,
    height INTEGER NOT NULL,
    weight INTEGER NOT NULL,
    gender TEXT NOT NULL,
    experience INTEGER,
    hourly_rate NUMERIC(10, 2),
    description TEXT,
    clothing_size TEXT,
    shoe_size TEXT,
    city TEXT NOT NULL,
    languages JSONB,
    categories JSONB,
    barter_accepted BOOLEAN DEFAULT false,
    profile_views INTEGER DEFAULT 0,
    rating NUMERIC(3, 1) DEFAULT 0, -- (e.g., 4.5)
    is_public BOOLEAN DEFAULT true,

    CONSTRAINT uq_model_profiles_user_id UNIQUE(user_id),
    FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE
    );
CREATE TRIGGER set_timestamp
    BEFORE UPDATE ON model_profiles
    FOR EACH ROW
    EXECUTE PROCEDURE trigger_set_timestamp();

-- Таблица 'employer_profiles' (из profile.go)
CREATE TABLE IF NOT EXISTS employer_profiles (
                                                 id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    created_at TIMESTAMPTZ DEFAULT now(),
    updated_at TIMESTAMPTZ DEFAULT now(),

    user_id UUID NOT NULL,
    company_name TEXT NOT NULL,
    contact_person TEXT,
    phone TEXT,
    website TEXT,
    city TEXT,
    company_type TEXT,
    description TEXT,
    is_verified BOOLEAN DEFAULT false,
    rating NUMERIC(3, 1) DEFAULT 0,

    CONSTRAINT uq_employer_profiles_user_id UNIQUE(user_id),
    FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE
    );
CREATE TRIGGER set_timestamp
    BEFORE UPDATE ON employer_profiles
    FOR EACH ROW
    EXECUTE PROCEDURE trigger_set_timestamp();

-- Таблица 'portfolio_items' (из portfolio.go)
CREATE TABLE IF NOT EXISTS portfolio_items (
                                               id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    created_at TIMESTAMPTZ DEFAULT now(),
    updated_at TIMESTAMPTZ DEFAULT now(),

    model_id UUID NOT NULL,
    upload_id UUID NOT NULL,
    title TEXT,
    description TEXT,
    order_index INTEGER DEFAULT 0,

    FOREIGN KEY (model_id) REFERENCES model_profiles(id) ON DELETE CASCADE,
    FOREIGN KEY (upload_id) REFERENCES uploads(id) ON DELETE CASCADE
    );
CREATE INDEX IF NOT EXISTS idx_portfolio_items_model_id ON portfolio_items(model_id);
CREATE INDEX IF NOT EXISTS idx_portfolio_items_upload_id ON portfolio_items(upload_id);
CREATE TRIGGER set_timestamp
    BEFORE UPDATE ON portfolio_items
    FOR EACH ROW
    EXECUTE PROCEDURE trigger_set_timestamp();

-- Таблица 'castings' (из casting.go)
CREATE TABLE IF NOT EXISTS castings (
                                        id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    created_at TIMESTAMPTZ DEFAULT now(),
    updated_at TIMESTAMPTZ DEFAULT now(),

    employer_id UUID NOT NULL,
    title TEXT NOT NULL,
    description TEXT,
    payment_min NUMERIC(10, 2),
    payment_max NUMERIC(10, 2),
    casting_date TIMESTAMPTZ,
    casting_time TEXT,
    address TEXT,
    city TEXT NOT NULL,
    categories JSONB,
    gender TEXT,
    age_min INTEGER,
    age_max INTEGER,
    height_min NUMERIC(5, 2),
    height_max NUMERIC(5, 2),
    weight_min NUMERIC(5, 2),
    weight_max NUMERIC(5, 2),
    clothing_size TEXT,
    shoe_size TEXT,
    experience_level TEXT,
    languages JSONB,
    job_type TEXT,
    status casting_status DEFAULT 'draft',
    views INTEGER DEFAULT 0,

    FOREIGN KEY (employer_id) REFERENCES employer_profiles(id) ON DELETE CASCADE
    );
CREATE INDEX IF NOT EXISTS idx_castings_employer_id ON castings(employer_id);
CREATE TRIGGER set_timestamp
    BEFORE UPDATE ON castings
    FOR EACH ROW
    EXECUTE PROCEDURE trigger_set_timestamp();

-- Таблица 'reviews' (из review.go)
CREATE TABLE IF NOT EXISTS reviews (
                                       id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    created_at TIMESTAMPTZ DEFAULT now(),
    updated_at TIMESTAMPTZ DEFAULT now(),

    model_id UUID NOT NULL,
    employer_id UUID NOT NULL,
    casting_id UUID,
    rating INTEGER NOT NULL CHECK (rating >= 1 AND rating <= 5),
    review_text TEXT,
    status review_status DEFAULT 'pending',

    FOREIGN KEY (model_id) REFERENCES model_profiles(id) ON DELETE CASCADE,
    FOREIGN KEY (employer_id) REFERENCES employer_profiles(id) ON DELETE SET NULL,
    FOREIGN KEY (casting_id) REFERENCES castings(id) ON DELETE SET NULL
    );
CREATE INDEX IF NOT EXISTS idx_reviews_model_id ON reviews(model_id);
CREATE INDEX IF NOT EXISTS idx_reviews_employer_id ON reviews(employer_id);
CREATE INDEX IF NOT EXISTS idx_reviews_casting_id ON reviews(casting_id);
CREATE TRIGGER set_timestamp
    BEFORE UPDATE ON reviews
    FOR EACH ROW
    EXECUTE PROCEDURE trigger_set_timestamp();

-- Таблица 'casting_responses' (из casting.go)
CREATE TABLE IF NOT EXISTS casting_responses (
                                                 id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    created_at TIMESTAMPTZ DEFAULT now(),
    updated_at TIMESTAMPTZ DEFAULT now(),

    casting_id UUID NOT NULL,
    model_id UUID NOT NULL,
    message TEXT,
    status response_status DEFAULT 'pending',

    FOREIGN KEY (casting_id) REFERENCES castings(id) ON DELETE CASCADE,
    FOREIGN KEY (model_id) REFERENCES model_profiles(id) ON DELETE CASCADE
    );
CREATE INDEX IF NOT EXISTS idx_casting_responses_casting_id ON casting_responses(casting_id);
CREATE INDEX IF NOT EXISTS idx_casting_responses_model_id ON casting_responses(model_id);
CREATE TRIGGER set_timestamp
    BEFORE UPDATE ON casting_responses
    FOR EACH ROW
    EXECUTE PROCEDURE trigger_set_timestamp();

-- Таблица 'refresh_tokens' (из response.go)
CREATE TABLE IF NOT EXISTS refresh_tokens (
                                              id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    created_at TIMESTAMPTZ DEFAULT now(),
    updated_at TIMESTAMPTZ DEFAULT now(),

    user_id UUID NOT NULL,
    token TEXT NOT NULL,
    expires_at TIMESTAMPTZ NOT NULL,

    CONSTRAINT uq_refresh_tokens_token UNIQUE(token),
    FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE
    );
CREATE INDEX IF NOT EXISTS idx_refresh_tokens_user_id ON refresh_tokens(user_id);
CREATE TRIGGER set_timestamp
    BEFORE UPDATE ON refresh_tokens
    FOR EACH ROW
    EXECUTE PROCEDURE trigger_set_timestamp();

-- Таблица 'user_subscriptions' (из response.go)
CREATE TABLE IF NOT EXISTS user_subscriptions (
                                                  id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    created_at TIMESTAMPTZ DEFAULT now(),
    updated_at TIMESTAMPTZ DEFAULT now(),

    user_id UUID NOT NULL,
    plan_id UUID NOT NULL,
    status subscription_status DEFAULT 'active',
    inv_id TEXT,
    current_usage JSONB,
    start_date TIMESTAMPTZ,
    end_date TIMESTAMPTZ,
    auto_renew BOOLEAN DEFAULT true,
    cancelled_at TIMESTAMPTZ,

    CONSTRAINT uq_user_subscriptions_inv_id UNIQUE(inv_id),
    FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE,
    FOREIGN KEY (plan_id) REFERENCES subscription_plans(id) ON DELETE RESTRICT
    );
CREATE INDEX IF NOT EXISTS idx_user_subscriptions_user_id ON user_subscriptions(user_id);
CREATE INDEX IF NOT EXISTS idx_user_subscriptions_plan_id ON user_subscriptions(plan_id);
CREATE TRIGGER set_timestamp
    BEFORE UPDATE ON user_subscriptions
    FOR EACH ROW
    EXECUTE PROCEDURE trigger_set_timestamp();

-- Таблица 'payment_transactions' (из response.go)
CREATE TABLE IF NOT EXISTS payment_transactions (
                                                    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    created_at TIMESTAMPTZ DEFAULT now(),
    updated_at TIMESTAMPTZ DEFAULT now(),

    user_id UUID NOT NULL,
    subscription_id UUID NOT NULL,
    amount NUMERIC(10, 2),
    status payment_status DEFAULT 'pending',
    inv_id TEXT,
    paid_at TIMESTAMPTZ,

    CONSTRAINT uq_payment_transactions_inv_id UNIQUE(inv_id),
    FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE,
    FOREIGN KEY (subscription_id) REFERENCES user_subscriptions(id) ON DELETE CASCADE
    );
CREATE INDEX IF NOT EXISTS idx_payment_transactions_user_id ON payment_transactions(user_id);
CREATE INDEX IF NOT EXISTS idx_payment_transactions_subscription_id ON payment_transactions(subscription_id);
CREATE TRIGGER set_timestamp
    BEFORE UPDATE ON payment_transactions
    FOR EACH ROW
    EXECUTE PROCEDURE trigger_set_timestamp();

-- Таблица 'notifications' (из notification.go)
CREATE TABLE IF NOT EXISTS notifications (
                                             id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    created_at TIMESTAMPTZ DEFAULT now(),
    updated_at TIMESTAMPTZ DEFAULT now(),

    user_id UUID NOT NULL,
    type VARCHAR(50) NOT NULL,
    title TEXT NOT NULL,
    message TEXT,
    data JSONB,
    is_read BOOLEAN DEFAULT false,
    read_at TIMESTAMPTZ,

    FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE
    );
CREATE INDEX IF NOT EXISTS idx_notifications_user_id ON notifications(user_id);
CREATE TRIGGER set_timestamp
    BEFORE UPDATE ON notifications
    FOR EACH ROW
    EXECUTE PROCEDURE trigger_set_timestamp();

-- Таблица 'usage_tracking' (из response.go)
CREATE TABLE IF NOT EXISTS usage_tracking (
                                              id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    user_id UUID,
    event_type VARCHAR(100) NOT NULL,
    metadata JSONB,
    created_at TIMESTAMPTZ DEFAULT CURRENT_TIMESTAMP,

    FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE SET NULL
    );
CREATE INDEX IF NOT EXISTS idx_usage_tracking_user_id ON usage_tracking(user_id);
CREATE INDEX IF NOT EXISTS idx_usage_tracking_event_type ON usage_tracking(event_type);
CREATE INDEX IF NOT EXISTS idx_usage_tracking_created_at ON usage_tracking(created_at);

-- 5. ТАБЛИЦЫ СХЕМЫ 'chat'

-- Таблица 'chat.dialogs' (из response.go)
-- Создаем без last_message_id из-за циклической зависимости
CREATE TABLE IF NOT EXISTS chat.dialogs (
                                            id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    is_group BOOLEAN DEFAULT false,
    title TEXT,
    image_url TEXT,
    casting_id UUID,
    -- last_message_id UUID, (будет добавлен позже)
    created_at TIMESTAMPTZ DEFAULT now(),
    updated_at TIMESTAMPTZ DEFAULT now(),

    FOREIGN KEY (casting_id) REFERENCES public.castings(id) ON DELETE SET NULL
    );
CREATE INDEX IF NOT EXISTS idx_chat_dialogs_casting_id ON chat.dialogs(casting_id);
CREATE TRIGGER set_timestamp
    BEFORE UPDATE ON chat.dialogs
    FOR EACH ROW
    EXECUTE PROCEDURE trigger_set_timestamp();

-- Таблица 'chat.messages' (из response.go)
CREATE TABLE IF NOT EXISTS chat.messages (
                                             id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    dialog_id UUID NOT NULL,
    sender_id UUID NOT NULL,
    type VARCHAR(20) DEFAULT 'text',
    content TEXT,
    attachment_url TEXT,
    attachment_name TEXT,
    forward_from_id UUID,
    reply_to_id UUID,
    status VARCHAR(20) DEFAULT 'sent',
    deleted_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ DEFAULT now(),

    FOREIGN KEY (dialog_id) REFERENCES chat.dialogs(id) ON DELETE CASCADE,
    FOREIGN KEY (sender_id) REFERENCES public.users(id) ON DELETE SET NULL,
    FOREIGN KEY (forward_from_id) REFERENCES chat.messages(id) ON DELETE SET NULL,
    FOREIGN KEY (reply_to_id) REFERENCES chat.messages(id) ON DELETE SET NULL
    );
CREATE INDEX IF NOT EXISTS idx_chat_messages_dialog_id ON chat.messages(dialog_id);
CREATE INDEX IF NOT EXISTS idx_chat_messages_sender_id ON chat.messages(sender_id);
CREATE INDEX IF NOT EXISTS idx_chat_messages_forward_from_id ON chat.messages(forward_from_id);
CREATE INDEX IF NOT EXISTS idx_chat_messages_reply_to_id ON chat.messages(reply_to_id);

-- Теперь, когда 'chat.messages' существует, добавляем 'last_message_id' в 'chat.dialogs'
ALTER TABLE chat.dialogs
    ADD COLUMN IF NOT EXISTS last_message_id UUID;

-- Добавляем внешний ключ
-- (Обернуто в DO-блок, чтобы избежать ошибки, если связь уже существует)
DO $$
BEGIN
    IF NOT EXISTS (
        SELECT 1 FROM pg_constraint
        WHERE conname = 'fk_chat_dialogs_last_message'
    ) THEN
ALTER TABLE chat.dialogs
    ADD CONSTRAINT fk_chat_dialogs_last_message
        FOREIGN KEY (last_message_id)
            REFERENCES chat.messages(id) ON DELETE SET NULL;
END IF;
END;
$$;
CREATE INDEX IF NOT EXISTS idx_chat_dialogs_last_message_id ON chat.dialogs(last_message_id);


-- Таблица 'chat.dialog_participants' (из response.go)
CREATE TABLE IF NOT EXISTS chat.dialog_participants (
                                                        id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    dialog_id UUID NOT NULL,
    user_id UUID NOT NULL,
    role VARCHAR(20) DEFAULT 'member',
    joined_at TIMESTAMPTZ,
    last_seen_at TIMESTAMPTZ,
    is_muted BOOLEAN DEFAULT false,
    typing_until TIMESTAMPTZ,
    left_at TIMESTAMPTZ,

    FOREIGN KEY (dialog_id) REFERENCES chat.dialogs(id) ON DELETE CASCADE,
    FOREIGN KEY (user_id) REFERENCES public.users(id) ON DELETE CASCADE
    );
CREATE INDEX IF NOT EXISTS idx_chat_dialog_participants_dialog_id ON chat.dialog_participants(dialog_id);
CREATE INDEX IF NOT EXISTS idx_chat_dialog_participants_user_id ON chat.dialog_participants(user_id);

-- Таблица 'chat.message_attachments' (из response.go)
CREATE TABLE IF NOT EXISTS chat.message_attachments (
                                                        id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    message_id UUID NOT NULL,
    uploader_id UUID,
    file_type VARCHAR(20),
    mime_type VARCHAR(100),
    file_name TEXT,
    url TEXT,
    size BIGINT,
    created_at TIMESTAMPTZ DEFAULT now(),

    FOREIGN KEY (message_id) REFERENCES chat.messages(id) ON DELETE CASCADE,
    FOREIGN KEY (uploader_id) REFERENCES public.users(id) ON DELETE SET NULL
    );
CREATE INDEX IF NOT EXISTS idx_chat_message_attachments_message_id ON chat.message_attachments(message_id);
CREATE INDEX IF NOT EXISTS idx_chat_message_attachments_uploader_id ON chat.message_attachments(uploader_id);

-- Таблица 'chat.message_reactions' (из response.go)
CREATE TABLE IF NOT EXISTS chat.message_reactions (
                                                      id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    message_id UUID NOT NULL,
    user_id UUID NOT NULL,
    emoji VARCHAR(10) NOT NULL,
    created_at TIMESTAMPTZ DEFAULT now(),

    FOREIGN KEY (message_id) REFERENCES chat.messages(id) ON DELETE CASCADE,
    FOREIGN KEY (user_id) REFERENCES public.users(id) ON DELETE CASCADE
    );
CREATE INDEX IF NOT EXISTS idx_chat_message_reactions_message_id ON chat.message_reactions(message_id);
CREATE INDEX IF NOT EXISTS idx_chat_message_reactions_user_id ON chat.message_reactions(user_id);

-- Таблица 'chat.message_read_receipts' (из response.go)
CREATE TABLE IF NOT EXISTS chat.message_read_receipts (
                                                          id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    message_id UUID NOT NULL,
    user_id UUID NOT NULL,
    read_at TIMESTAMPTZ,

    FOREIGN KEY (message_id) REFERENCES chat.messages(id) ON DELETE CASCADE,
    FOREIGN KEY (user_id) REFERENCES public.users(id) ON DELETE CASCADE
    );
CREATE INDEX IF NOT EXISTS idx_chat_message_read_receipts_message_id ON chat.message_read_receipts(message_id);
CREATE INDEX IF NOT EXISTS idx_chat_message_read_receipts_user_id ON chat.message_read_receipts(user_id);


-- Завершаем транзакцию
COMMIT;