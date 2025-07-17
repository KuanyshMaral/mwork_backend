-- UUID генерация (если требуется)
CREATE EXTENSION IF NOT EXISTS "pgcrypto";

-- Пользователи
CREATE TABLE users (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    email TEXT NOT NULL UNIQUE,
    password_hash TEXT NOT NULL,
    role TEXT NOT NULL,
    subscription TEXT DEFAULT 'free',
    status TEXT NOT NULL DEFAULT 'pending',
    is_verified BOOLEAN DEFAULT FALSE,
    verification_token TEXT,
    reset_token TEXT,
    reset_token_exp TIMESTAMP,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- Профили моделей
CREATE TABLE model_profiles (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    name TEXT NOT NULL,
    age INT NOT NULL,
    height FLOAT NOT NULL,
    weight FLOAT NOT NULL,
    gender TEXT NOT NULL,
    experience INT DEFAULT 0,
    hourly_rate NUMERIC(10,2) DEFAULT 0.0,
    description TEXT,
    clothing_size TEXT,
    shoe_size TEXT,
    city TEXT NOT NULL,
    languages TEXT[] DEFAULT '{}',
    categories TEXT[] DEFAULT '{}',
    barter_accepted BOOLEAN DEFAULT FALSE,
    profile_views INT DEFAULT 0,
    rating NUMERIC(3,2) DEFAULT 0.0,
    is_public BOOLEAN DEFAULT TRUE,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- Работодатели (employer_profiles)
CREATE TABLE employer_profiles (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    company_name TEXT NOT NULL,
    contact_person TEXT,
    phone TEXT,
    website TEXT,
    city TEXT NOT NULL,
    company_type TEXT,
    description TEXT,
    is_verified BOOLEAN DEFAULT FALSE,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- Кастинги
CREATE TABLE castings (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    employer_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    title TEXT NOT NULL,
    description TEXT,
    categories TEXT[] DEFAULT '{}',
    gender TEXT,
    age_min INT,
    age_max INT,
    height_min FLOAT,
    height_max FLOAT,
    weight_min FLOAT,
    weight_max FLOAT,
    clothing_size TEXT,
    shoe_size TEXT,
    experience_level TEXT,
    languages TEXT[] DEFAULT '{}',
    city TEXT NOT NULL,
    job_type TEXT DEFAULT 'one-time', -- or 'permanent'
    casting_date TIMESTAMP,
    casting_time TEXT,
    address TEXT,
    payment_min NUMERIC(10,2),
    payment_max NUMERIC(10,2),
    views INT DEFAULT 0,
    status TEXT DEFAULT 'draft',
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- Отклики моделей
CREATE TABLE responses (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    casting_id UUID REFERENCES castings(id) ON DELETE CASCADE,
    model_id UUID REFERENCES model_profiles(id) ON DELETE CASCADE,
    message TEXT,
    status TEXT DEFAULT 'pending',
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- Сообщения между пользователями
CREATE TABLE messages (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    sender_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    receiver_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    casting_id UUID,
    content TEXT NOT NULL,
    is_read BOOLEAN DEFAULT FALSE,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- Подписки
CREATE TABLE subscription_plans (
    id TEXT PRIMARY KEY,
    name TEXT NOT NULL,
    price NUMERIC NOT NULL,
    currency TEXT NOT NULL,
    duration TEXT NOT NULL,
    features TEXT[] DEFAULT '{}',
    limits JSONB,
    created_at TIMESTAMP DEFAULT NOW(),
    updated_at TIMESTAMP DEFAULT NOW()
);

CREATE TABLE user_subscriptions (
    id TEXT PRIMARY KEY,
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    plan_id TEXT NOT NULL REFERENCES subscription_plans(id),
    status TEXT NOT NULL,
    start_date TIMESTAMP,
    end_date TIMESTAMP,
    auto_renew BOOLEAN DEFAULT TRUE,
    usage JSONB,
    created_at TIMESTAMP DEFAULT NOW(),
    updated_at TIMESTAMP DEFAULT NOW()
);

-- Файлы
CREATE TABLE uploads (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    entity_type TEXT NOT NULL,
    entity_id TEXT NOT NULL,
    file_type TEXT NOT NULL,
    usage TEXT NOT NULL,
    path TEXT NOT NULL,
    mime_type TEXT NOT NULL,
    size BIGINT NOT NULL,
    is_public BOOLEAN NOT NULL DEFAULT TRUE,
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
);

-- Оценки моделей
CREATE TABLE ratings (
    id SERIAL PRIMARY KEY,
    model_id UUID NOT NULL REFERENCES model_profiles(id) ON DELETE CASCADE,
    score INT CHECK (score BETWEEN 1 AND 5),
    comment TEXT,
    author_id UUID NOT NULL REFERENCES users(id) ON DELETE SET NULL,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE refresh_tokens (
    id UUID PRIMARY KEY,
    user_id UUID REFERENCES users(id),
    token TEXT UNIQUE NOT NULL,
    created_at TIMESTAMP NOT NULL DEFAULT NOW(),
    expires_at TIMESTAMP NOT NULL
);

-- Индексы
CREATE INDEX idx_model_profile_city ON model_profiles(city);
CREATE INDEX idx_casting_city ON castings(city);
CREATE INDEX idx_response_casting ON responses(casting_id);
