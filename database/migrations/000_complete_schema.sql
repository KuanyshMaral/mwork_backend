-- ============================================================
-- MWORK PLATFORM - COMPLETE DATABASE SCHEMA MIGRATION
-- This is a consolidated migration that creates the entire schema from scratch
-- ============================================================

BEGIN;

-- ============================================================
-- 1. EXTENSIONS & SCHEMA SETUP
-- ============================================================

CREATE EXTENSION IF NOT EXISTS "uuid-ossp";
CREATE EXTENSION IF NOT EXISTS "pgcrypto";
CREATE SCHEMA IF NOT EXISTS chat;

-- ============================================================
-- 2. ENUM TYPES
-- ============================================================

CREATE TYPE user_status AS ENUM ('pending', 'active', 'suspended', 'banned');
CREATE TYPE user_role AS ENUM ('model', 'employer', 'admin');
CREATE TYPE casting_status AS ENUM ('draft', 'active', 'closed', 'cancelled');
CREATE TYPE response_status AS ENUM ('pending', 'accepted', 'rejected', 'withdrawn');
CREATE TYPE subscription_status AS ENUM ('active', 'expired', 'cancelled');
CREATE TYPE payment_status AS ENUM ('pending', 'paid', 'failed', 'refunded');
CREATE TYPE review_status AS ENUM ('pending', 'approved', 'rejected');
CREATE TYPE message_type AS ENUM ('text', 'image', 'video', 'file', 'voice');
CREATE TYPE message_status AS ENUM ('sent', 'delivered', 'read');

-- ============================================================
-- 3. TRIGGER FUNCTIONS
-- ============================================================

CREATE OR REPLACE FUNCTION trigger_set_timestamp()
RETURNS TRIGGER AS $$
BEGIN
  NEW.updated_at = NOW();
  RETURN NEW;
END;
$$ LANGUAGE plpgsql;

-- ============================================================
-- 4. CORE TABLES (users, profiles, subscriptions)
-- ============================================================

-- Users table
CREATE TABLE IF NOT EXISTS users (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    created_at TIMESTAMPTZ DEFAULT now(),
    updated_at TIMESTAMPTZ DEFAULT now(),
    
    name VARCHAR(100) NOT NULL,
    email TEXT NOT NULL UNIQUE,
    password_hash TEXT NOT NULL,
    phone TEXT,
    
    role user_role NOT NULL,
    status user_status DEFAULT 'pending',
    
    is_verified BOOLEAN DEFAULT false,
    verification_token TEXT,
    
    reset_token TEXT,
    reset_token_exp TIMESTAMPTZ,
    
    last_login_at TIMESTAMPTZ,
    last_login_ip TEXT,
    
    two_factor_enabled BOOLEAN DEFAULT false,
    two_factor_secret TEXT
);

CREATE TRIGGER set_timestamp_users
    BEFORE UPDATE ON users
    FOR EACH ROW
    EXECUTE PROCEDURE trigger_set_timestamp();

CREATE INDEX idx_users_email ON users(email);
CREATE INDEX idx_users_status ON users(status);
CREATE INDEX idx_users_role ON users(role);
CREATE INDEX idx_users_created_at ON users(created_at DESC);

-- Model Profiles
CREATE TABLE IF NOT EXISTS model_profiles (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    created_at TIMESTAMPTZ DEFAULT now(),
    updated_at TIMESTAMPTZ DEFAULT now(),
    
    user_id UUID NOT NULL UNIQUE,
    
    bio TEXT,
    description TEXT,
    
    -- Physical attributes
    age INTEGER CHECK (age >= 18 AND age <= 100),
    height DECIMAL(5,2) CHECK (height > 0),
    weight DECIMAL(5,2) CHECK (weight > 0),
    gender VARCHAR(20),
    
    -- Sizes
    clothing_size VARCHAR(10),
    shoe_size VARCHAR(10),
    
    -- Professional info
    experience_years INTEGER DEFAULT 0,
    hourly_rate DECIMAL(10,2),
    
    -- Location
    city VARCHAR(100),
    country VARCHAR(100),
    
    -- Skills & categories
    languages JSONB DEFAULT '[]',
    categories JSONB DEFAULT '[]',
    skills JSONB DEFAULT '[]',
    
    -- Options
    barter_accepted BOOLEAN DEFAULT false,
    accept_remote_work BOOLEAN DEFAULT false,
    
    -- Stats
    profile_views INTEGER DEFAULT 0,
    rating DECIMAL(3,2) DEFAULT 0 CHECK (rating >= 0 AND rating <= 5),
    total_reviews INTEGER DEFAULT 0,
    
    -- Visibility
    is_public BOOLEAN DEFAULT true,
    
    CONSTRAINT fk_model_user FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE
);

CREATE TRIGGER set_timestamp_model_profiles
    BEFORE UPDATE ON model_profiles
    FOR EACH ROW
    EXECUTE PROCEDURE trigger_set_timestamp();

CREATE INDEX idx_model_profiles_user_id ON model_profiles(user_id);
CREATE INDEX idx_model_profiles_city ON model_profiles(city);
CREATE INDEX idx_model_profiles_rating ON model_profiles(rating DESC);
CREATE INDEX idx_model_profiles_is_public ON model_profiles(is_public) WHERE is_public = true;

-- Employer Profiles
CREATE TABLE IF NOT EXISTS employer_profiles (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    created_at TIMESTAMPTZ DEFAULT now(),
    updated_at TIMESTAMPTZ DEFAULT now(),
    
    user_id UUID NOT NULL UNIQUE,
    
    company_name VARCHAR(255) NOT NULL,
    company_type VARCHAR(100),
    
    description TEXT,
    website TEXT,
    contact_person VARCHAR(255),
    contact_phone VARCHAR(20),
    
    -- Location
    city VARCHAR(100),
    country VARCHAR(100),
    
    -- Stats
    rating DECIMAL(3,2) DEFAULT 0 CHECK (rating >= 0 AND rating <= 5),
    total_reviews INTEGER DEFAULT 0,
    castings_posted INTEGER DEFAULT 0,
    
    -- Verification
    is_verified BOOLEAN DEFAULT false,
    verified_at TIMESTAMPTZ,
    
    CONSTRAINT fk_employer_user FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE
);

CREATE TRIGGER set_timestamp_employer_profiles
    BEFORE UPDATE ON employer_profiles
    FOR EACH ROW
    EXECUTE PROCEDURE trigger_set_timestamp();

CREATE INDEX idx_employer_profiles_user_id ON employer_profiles(user_id);
CREATE INDEX idx_employer_profiles_city ON employer_profiles(city);
CREATE INDEX idx_employer_profiles_is_verified ON employer_profiles(is_verified);

-- ============================================================
-- 5. SUBSCRIPTION & PAYMENTS
-- ============================================================

CREATE TABLE IF NOT EXISTS subscription_plans (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    created_at TIMESTAMPTZ DEFAULT now(),
    updated_at TIMESTAMPTZ DEFAULT now(),
    
    name VARCHAR(100) NOT NULL UNIQUE,
    slug VARCHAR(100) NOT NULL UNIQUE,
    description TEXT,
    
    price DECIMAL(10,2) NOT NULL,
    currency VARCHAR(10) DEFAULT 'KZT',
    billing_period INTEGER NOT NULL, -- days
    
    features JSONB DEFAULT '{}',
    limits JSONB DEFAULT '{}',
    
    is_active BOOLEAN DEFAULT true,
    trial_days INTEGER DEFAULT 0
);

CREATE TRIGGER set_timestamp_subscription_plans
    BEFORE UPDATE ON subscription_plans
    FOR EACH ROW
    EXECUTE PROCEDURE trigger_set_timestamp();

CREATE INDEX idx_subscription_plans_is_active ON subscription_plans(is_active) WHERE is_active = true;

-- User Subscriptions
CREATE TABLE IF NOT EXISTS user_subscriptions (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    created_at TIMESTAMPTZ DEFAULT now(),
    updated_at TIMESTAMPTZ DEFAULT now(),
    
    user_id UUID NOT NULL,
    plan_id UUID NOT NULL,
    
    status subscription_status DEFAULT 'active',
    
    start_date TIMESTAMPTZ NOT NULL,
    end_date TIMESTAMPTZ NOT NULL,
    renewal_date TIMESTAMPTZ,
    
    auto_renew BOOLEAN DEFAULT true,
    cancelled_at TIMESTAMPTZ,
    
    current_usage JSONB DEFAULT '{}',
    
    invoice_id TEXT UNIQUE,
    
    CONSTRAINT fk_subscription_user FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE,
    CONSTRAINT fk_subscription_plan FOREIGN KEY (plan_id) REFERENCES subscription_plans(id) ON DELETE RESTRICT
);

CREATE TRIGGER set_timestamp_user_subscriptions
    BEFORE UPDATE ON user_subscriptions
    FOR EACH ROW
    EXECUTE PROCEDURE trigger_set_timestamp();

CREATE INDEX idx_user_subscriptions_user_id ON user_subscriptions(user_id);
CREATE INDEX idx_user_subscriptions_status ON user_subscriptions(status);
CREATE INDEX idx_user_subscriptions_end_date ON user_subscriptions(end_date) WHERE status = 'active';

-- Payment Transactions
CREATE TABLE IF NOT EXISTS payment_transactions (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    created_at TIMESTAMPTZ DEFAULT now(),
    updated_at TIMESTAMPTZ DEFAULT now(),
    
    user_id UUID NOT NULL,
    subscription_id UUID,
    
    amount DECIMAL(10,2) NOT NULL,
    currency VARCHAR(10) DEFAULT 'KZT',
    
    status payment_status DEFAULT 'pending',
    
    payment_method VARCHAR(50),
    transaction_id TEXT UNIQUE,
    invoice_id TEXT UNIQUE,
    
    paid_at TIMESTAMPTZ,
    
    description TEXT,
    metadata JSONB,
    
    CONSTRAINT fk_payment_user FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE,
    CONSTRAINT fk_payment_subscription FOREIGN KEY (subscription_id) REFERENCES user_subscriptions(id) ON DELETE SET NULL
);

CREATE TRIGGER set_timestamp_payment_transactions
    BEFORE UPDATE ON payment_transactions
    FOR EACH ROW
    EXECUTE PROCEDURE trigger_set_timestamp();

CREATE INDEX idx_payment_transactions_user_id ON payment_transactions(user_id);
CREATE INDEX idx_payment_transactions_status ON payment_transactions(status);

-- ============================================================
-- 6. FILE UPLOADS
-- ============================================================

CREATE TABLE IF NOT EXISTS uploads (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    created_at TIMESTAMPTZ DEFAULT now(),
    updated_at TIMESTAMPTZ DEFAULT now(),
    
    user_id UUID NOT NULL,
    
    entity_type VARCHAR(50),
    entity_id TEXT,
    
    file_type VARCHAR(50),
    usage VARCHAR(50), -- 'profile', 'portfolio', 'casting', 'message', etc.
    
    original_name TEXT,
    mime_type VARCHAR(100),
    
    path TEXT NOT NULL,
    url TEXT,
    
    size BIGINT,
    
    -- Image variants
    thumbnail_path TEXT,
    variants JSONB, -- {"small": "path", "medium": "path", "large": "path"}
    
    -- Metadata
    metadata JSONB, -- dimensions, duration, etc.
    
    storage_provider VARCHAR(50) DEFAULT 'local', -- 'local', 's3', 'cloudflare_r2'
    expires_at TIMESTAMPTZ,
    
    download_count INTEGER DEFAULT 0,
    last_accessed_at TIMESTAMPTZ,
    
    is_public BOOLEAN DEFAULT true,
    
    CONSTRAINT fk_upload_user FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE,
    CONSTRAINT chk_storage_provider CHECK (storage_provider IN ('local', 's3', 'cloudflare_r2'))
);

CREATE TRIGGER set_timestamp_uploads
    BEFORE UPDATE ON uploads
    FOR EACH ROW
    EXECUTE PROCEDURE trigger_set_timestamp();

CREATE INDEX idx_uploads_user_id ON uploads(user_id);
CREATE INDEX idx_uploads_entity ON uploads(entity_type, entity_id);
CREATE INDEX idx_uploads_usage ON uploads(usage);
CREATE INDEX idx_uploads_storage_provider ON uploads(storage_provider);
CREATE INDEX idx_uploads_expires_at ON uploads(expires_at) WHERE expires_at IS NOT NULL;
CREATE INDEX idx_uploads_created_at ON uploads(created_at DESC);

-- Portfolio Items
CREATE TABLE IF NOT EXISTS portfolio_items (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    created_at TIMESTAMPTZ DEFAULT now(),
    updated_at TIMESTAMPTZ DEFAULT now(),
    
    model_id UUID NOT NULL,
    upload_id UUID NOT NULL,
    
    title VARCHAR(255),
    description TEXT,
    order_index INTEGER DEFAULT 0,
    
    CONSTRAINT fk_portfolio_model FOREIGN KEY (model_id) REFERENCES model_profiles(id) ON DELETE CASCADE,
    CONSTRAINT fk_portfolio_upload FOREIGN KEY (upload_id) REFERENCES uploads(id) ON DELETE CASCADE
);

CREATE TRIGGER set_timestamp_portfolio_items
    BEFORE UPDATE ON portfolio_items
    FOR EACH ROW
    EXECUTE PROCEDURE trigger_set_timestamp();

CREATE INDEX idx_portfolio_items_model_id ON portfolio_items(model_id);
CREATE INDEX idx_portfolio_items_order ON portfolio_items(model_id, order_index);

-- ============================================================
-- 7. CASTINGS & RESPONSES
-- ============================================================

CREATE TABLE IF NOT EXISTS castings (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    created_at TIMESTAMPTZ DEFAULT now(),
    updated_at TIMESTAMPTZ DEFAULT now(),
    
    employer_id UUID NOT NULL,
    
    title VARCHAR(255) NOT NULL,
    description TEXT,
    
    -- Payment
    payment_min DECIMAL(10,2),
    payment_max DECIMAL(10,2),
    currency VARCHAR(10) DEFAULT 'KZT',
    
    -- Event details
    event_date TIMESTAMPTZ,
    event_time VARCHAR(50),
    
    -- Location
    address TEXT,
    city VARCHAR(100) NOT NULL,
    country VARCHAR(100),
    
    -- Filters
    categories JSONB DEFAULT '[]',
    gender VARCHAR(20),
    age_min INTEGER,
    age_max INTEGER,
    height_min DECIMAL(5,2),
    height_max DECIMAL(5,2),
    weight_min DECIMAL(5,2),
    weight_max DECIMAL(5,2),
    
    clothing_size VARCHAR(10),
    shoe_size VARCHAR(10),
    
    experience_level VARCHAR(50),
    languages JSONB DEFAULT '[]',
    
    -- Job type
    job_type VARCHAR(50), -- 'one-time', 'recurring', 'permanent'
    
    -- Status
    status casting_status DEFAULT 'draft',
    published_at TIMESTAMPTZ,
    closed_at TIMESTAMPTZ,
    
    -- Stats
    views INTEGER DEFAULT 0,
    response_count INTEGER DEFAULT 0,
    
    -- Budget
    max_responses INTEGER,
    
    CONSTRAINT fk_casting_employer FOREIGN KEY (employer_id) REFERENCES employer_profiles(id) ON DELETE CASCADE,
    CONSTRAINT chk_salary_range CHECK (payment_min IS NULL OR payment_max IS NULL OR payment_min <= payment_max),
    CONSTRAINT chk_age_range CHECK (age_min IS NULL OR age_max IS NULL OR age_min <= age_max),
    CONSTRAINT chk_height_range CHECK (height_min IS NULL OR height_max IS NULL OR height_min <= height_max),
    CONSTRAINT chk_weight_range CHECK (weight_min IS NULL OR weight_max IS NULL OR weight_min <= weight_max)
);

CREATE TRIGGER set_timestamp_castings
    BEFORE UPDATE ON castings
    FOR EACH ROW
    EXECUTE PROCEDURE trigger_set_timestamp();

CREATE INDEX idx_castings_employer_id ON castings(employer_id);
CREATE INDEX idx_castings_city ON castings(city);
CREATE INDEX idx_castings_status ON castings(status);
CREATE INDEX idx_castings_event_date ON castings(event_date) WHERE status = 'active';
CREATE INDEX idx_castings_published ON castings(published_at DESC) WHERE status = 'active';

-- Casting Responses
CREATE TABLE IF NOT EXISTS casting_responses (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    created_at TIMESTAMPTZ DEFAULT now(),
    updated_at TIMESTAMPTZ DEFAULT now(),
    
    casting_id UUID NOT NULL,
    model_id UUID NOT NULL,
    
    message TEXT,
    proposed_rate DECIMAL(10,2),
    
    status response_status DEFAULT 'pending',
    
    accepted_at TIMESTAMPTZ,
    rejected_at TIMESTAMPTZ,
    
    rating_given BOOLEAN DEFAULT false,
    
    CONSTRAINT fk_response_casting FOREIGN KEY (casting_id) REFERENCES castings(id) ON DELETE CASCADE,
    CONSTRAINT fk_response_model FOREIGN KEY (model_id) REFERENCES model_profiles(id) ON DELETE CASCADE,
    CONSTRAINT uq_casting_response UNIQUE(casting_id, model_id)
);

CREATE TRIGGER set_timestamp_casting_responses
    BEFORE UPDATE ON casting_responses
    FOR EACH ROW
    EXECUTE PROCEDURE trigger_set_timestamp();

CREATE INDEX idx_casting_responses_casting_id ON casting_responses(casting_id);
CREATE INDEX idx_casting_responses_model_id ON casting_responses(model_id);
CREATE INDEX idx_casting_responses_status ON casting_responses(status);

-- ============================================================
-- 8. REVIEWS & RATINGS
-- ============================================================

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
    
    approved_at TIMESTAMPTZ,
    rejected_reason TEXT,
    
    CONSTRAINT fk_review_model FOREIGN KEY (model_id) REFERENCES model_profiles(id) ON DELETE CASCADE,
    CONSTRAINT fk_review_employer FOREIGN KEY (employer_id) REFERENCES employer_profiles(id) ON DELETE CASCADE,
    CONSTRAINT fk_review_casting FOREIGN KEY (casting_id) REFERENCES castings(id) ON DELETE SET NULL,
    CONSTRAINT uq_review UNIQUE(model_id, employer_id, casting_id)
);

CREATE TRIGGER set_timestamp_reviews
    BEFORE UPDATE ON reviews
    FOR EACH ROW
    EXECUTE PROCEDURE trigger_set_timestamp();

CREATE INDEX idx_reviews_model_id ON reviews(model_id);
CREATE INDEX idx_reviews_employer_id ON reviews(employer_id);
CREATE INDEX idx_reviews_casting_id ON reviews(casting_id);
CREATE INDEX idx_reviews_status ON reviews(status);
CREATE INDEX idx_reviews_rating ON reviews(rating);

-- ============================================================
-- 9. NOTIFICATIONS
-- ============================================================

CREATE TABLE IF NOT EXISTS notifications (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    created_at TIMESTAMPTZ DEFAULT now(),
    updated_at TIMESTAMPTZ DEFAULT now(),
    
    user_id UUID NOT NULL,
    
    type VARCHAR(100) NOT NULL,
    title VARCHAR(255),
    message TEXT,
    
    data JSONB,
    
    is_read BOOLEAN DEFAULT false,
    read_at TIMESTAMPTZ,
    
    action_url TEXT,
    
    CONSTRAINT fk_notification_user FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE
);

CREATE TRIGGER set_timestamp_notifications
    BEFORE UPDATE ON notifications
    FOR EACH ROW
    EXECUTE PROCEDURE trigger_set_timestamp();

CREATE INDEX idx_notifications_user_id ON notifications(user_id);
CREATE INDEX idx_notifications_is_read ON notifications(is_read);
CREATE INDEX idx_notifications_created_at ON notifications(created_at DESC);

-- ============================================================
-- 10. USAGE TRACKING
-- ============================================================

CREATE TABLE IF NOT EXISTS usage_tracking (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    
    user_id UUID,
    
    event_type VARCHAR(100) NOT NULL,
    
    metadata JSONB,
    
    created_at TIMESTAMPTZ DEFAULT now(),
    
    CONSTRAINT fk_usage_tracking_user FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE SET NULL
);

CREATE INDEX idx_usage_tracking_user_id ON usage_tracking(user_id);
CREATE INDEX idx_usage_tracking_event_type ON usage_tracking(event_type);
CREATE INDEX idx_usage_tracking_created_at ON usage_tracking(created_at DESC);

-- ============================================================
-- 11. CHAT SYSTEM (in chat schema)
-- ============================================================

-- Dialogs (conversations)
CREATE TABLE IF NOT EXISTS chat.dialogs (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    created_at TIMESTAMPTZ DEFAULT now(),
    updated_at TIMESTAMPTZ DEFAULT now(),
    
    is_group BOOLEAN DEFAULT false,
    title VARCHAR(255),
    image_url TEXT,
    
    casting_id UUID,
    
    last_message_id UUID,
    
    CONSTRAINT fk_dialog_casting FOREIGN KEY (casting_id) REFERENCES public.castings(id) ON DELETE SET NULL
);

CREATE TRIGGER set_timestamp_chat_dialogs
    BEFORE UPDATE ON chat.dialogs
    FOR EACH ROW
    EXECUTE PROCEDURE trigger_set_timestamp();

CREATE INDEX idx_chat_dialogs_is_group ON chat.dialogs(is_group);
CREATE INDEX idx_chat_dialogs_casting_id ON chat.dialogs(casting_id);

-- Messages
CREATE TABLE IF NOT EXISTS chat.messages (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    created_at TIMESTAMPTZ DEFAULT now(),
    
    dialog_id UUID NOT NULL,
    sender_id UUID NOT NULL,
    
    type message_type DEFAULT 'text',
    content TEXT,
    
    attachment_url TEXT,
    attachment_name TEXT,
    attachment_size BIGINT,
    
    forward_from_id UUID,
    reply_to_id UUID,
    
    status message_status DEFAULT 'sent',
    
    edited_at TIMESTAMPTZ,
    deleted_at TIMESTAMPTZ,
    
    CONSTRAINT fk_message_dialog FOREIGN KEY (dialog_id) REFERENCES chat.dialogs(id) ON DELETE CASCADE,
    CONSTRAINT fk_message_sender FOREIGN KEY (sender_id) REFERENCES public.users(id) ON DELETE SET NULL,
    CONSTRAINT fk_message_forward FOREIGN KEY (forward_from_id) REFERENCES chat.messages(id) ON DELETE SET NULL,
    CONSTRAINT fk_message_reply FOREIGN KEY (reply_to_id) REFERENCES chat.messages(id) ON DELETE SET NULL
);

CREATE INDEX idx_chat_messages_dialog_id ON chat.messages(dialog_id);
CREATE INDEX idx_chat_messages_sender_id ON chat.messages(sender_id);
CREATE INDEX idx_chat_messages_created_at ON chat.messages(created_at DESC);

-- Add foreign key for last_message_id
DO $$
BEGIN
    IF NOT EXISTS (SELECT 1 FROM pg_constraint WHERE conname = 'fk_dialog_last_message') THEN
        ALTER TABLE chat.dialogs
        ADD CONSTRAINT fk_dialog_last_message
            FOREIGN KEY (last_message_id) REFERENCES chat.messages(id) ON DELETE SET NULL;
    END IF;
END;
$$;

-- Dialog Participants
CREATE TABLE IF NOT EXISTS chat.dialog_participants (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    created_at TIMESTAMPTZ DEFAULT now(),
    updated_at TIMESTAMPTZ DEFAULT now(),
    
    dialog_id UUID NOT NULL,
    user_id UUID NOT NULL,
    
    role VARCHAR(50) DEFAULT 'member', -- 'admin', 'member', 'moderator'
    
    joined_at TIMESTAMPTZ DEFAULT now(),
    last_seen_at TIMESTAMPTZ,
    
    is_muted BOOLEAN DEFAULT false,
    is_archived BOOLEAN DEFAULT false,
    
    typing_until TIMESTAMPTZ,
    left_at TIMESTAMPTZ,
    
    CONSTRAINT fk_participant_dialog FOREIGN KEY (dialog_id) REFERENCES chat.dialogs(id) ON DELETE CASCADE,
    CONSTRAINT fk_participant_user FOREIGN KEY (user_id) REFERENCES public.users(id) ON DELETE CASCADE,
    CONSTRAINT uq_dialog_participant UNIQUE(dialog_id, user_id)
);

CREATE TRIGGER set_timestamp_chat_dialog_participants
    BEFORE UPDATE ON chat.dialog_participants
    FOR EACH ROW
    EXECUTE PROCEDURE trigger_set_timestamp();

CREATE INDEX idx_chat_dialog_participants_dialog_id ON chat.dialog_participants(dialog_id);
CREATE INDEX idx_chat_dialog_participants_user_id ON chat.dialog_participants(user_id);

-- Message Attachments
CREATE TABLE IF NOT EXISTS chat.message_attachments (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    created_at TIMESTAMPTZ DEFAULT now(),
    
    message_id UUID NOT NULL,
    uploader_id UUID,
    
    file_type VARCHAR(50),
    mime_type VARCHAR(100),
    file_name TEXT,
    url TEXT,
    size BIGINT,
    
    CONSTRAINT fk_attachment_message FOREIGN KEY (message_id) REFERENCES chat.messages(id) ON DELETE CASCADE,
    CONSTRAINT fk_attachment_uploader FOREIGN KEY (uploader_id) REFERENCES public.users(id) ON DELETE SET NULL
);

CREATE INDEX idx_chat_message_attachments_message_id ON chat.message_attachments(message_id);

-- Message Reactions
CREATE TABLE IF NOT EXISTS chat.message_reactions (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    created_at TIMESTAMPTZ DEFAULT now(),
    
    message_id UUID NOT NULL,
    user_id UUID NOT NULL,
    
    emoji VARCHAR(20) NOT NULL,
    
    CONSTRAINT fk_reaction_message FOREIGN KEY (message_id) REFERENCES chat.messages(id) ON DELETE CASCADE,
    CONSTRAINT fk_reaction_user FOREIGN KEY (user_id) REFERENCES public.users(id) ON DELETE CASCADE,
    CONSTRAINT uq_message_reaction UNIQUE(message_id, user_id, emoji)
);

CREATE INDEX idx_chat_message_reactions_message_id ON chat.message_reactions(message_id);
CREATE INDEX idx_chat_message_reactions_user_id ON chat.message_reactions(user_id);

-- Message Read Receipts
CREATE TABLE IF NOT EXISTS chat.message_read_receipts (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    created_at TIMESTAMPTZ DEFAULT now(),
    
    message_id UUID NOT NULL,
    user_id UUID NOT NULL,
    
    read_at TIMESTAMPTZ DEFAULT now(),
    
    CONSTRAINT fk_read_receipt_message FOREIGN KEY (message_id) REFERENCES chat.messages(id) ON DELETE CASCADE,
    CONSTRAINT fk_read_receipt_user FOREIGN KEY (user_id) REFERENCES public.users(id) ON DELETE CASCADE,
    CONSTRAINT uq_read_receipt UNIQUE(message_id, user_id)
);

CREATE INDEX idx_chat_message_read_receipts_message_id ON chat.message_read_receipts(message_id);
CREATE INDEX idx_chat_message_read_receipts_user_id ON chat.message_read_receipts(user_id);

-- ============================================================
-- 12. REFRESH TOKENS
-- ============================================================

CREATE TABLE IF NOT EXISTS refresh_tokens (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    created_at TIMESTAMPTZ DEFAULT now(),
    updated_at TIMESTAMPTZ DEFAULT now(),
    
    user_id UUID NOT NULL,
    
    token TEXT NOT NULL UNIQUE,
    expires_at TIMESTAMPTZ NOT NULL,
    
    used_at TIMESTAMPTZ,
    revoked_at TIMESTAMPTZ,
    
    ip_address TEXT,
    user_agent TEXT,
    
    CONSTRAINT fk_refresh_token_user FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE
);

CREATE TRIGGER set_timestamp_refresh_tokens
    BEFORE UPDATE ON refresh_tokens
    FOR EACH ROW
    EXECUTE PROCEDURE trigger_set_timestamp();

CREATE INDEX idx_refresh_tokens_user_id ON refresh_tokens(user_id);
CREATE INDEX idx_refresh_tokens_token ON refresh_tokens(token);
CREATE INDEX idx_refresh_tokens_expires_at ON refresh_tokens(expires_at);

-- ============================================================
-- 13. SAMPLE DATA (Optional)
-- ============================================================

-- Insert subscription plans
INSERT INTO subscription_plans (name, slug, description, price, billing_period, features, limits, is_active)
VALUES
    ('Free', 'free', 'Get started with basic features', 0, 30, 
     '{"castings": 5, "responses": true, "reviews": true}'::jsonb,
     '{"max_castings": 5, "max_portfolio_items": 10}'::jsonb,
     true),
    ('MWork Start', 'mwork-start', 'Perfect for growing models and employers', 990, 30,
     '{"castings": 20, "responses": true, "reviews": true, "promotion": true, "analytics": true}'::jsonb,
     '{"max_castings": 20, "max_portfolio_items": 50}'::jsonb,
     true),
    ('MWork Pro', 'mwork-pro', 'Professional tier with unlimited access', 2990, 30,
     '{"castings": -1, "responses": true, "reviews": true, "promotion": true, "analytics": true, "support": "priority"}'::jsonb,
     '{"max_castings": -1, "max_portfolio_items": -1}'::jsonb,
     true)
ON CONFLICT (slug) DO NOTHING;

COMMIT;

-- ============================================================
-- SUCCESS
-- ============================================================
-- All tables created successfully with proper relationships, indexes, and constraints.
