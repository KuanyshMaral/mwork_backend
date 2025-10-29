-- Add usage tracking table for free tier users
CREATE TABLE IF NOT EXISTS usage (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    publications_used INT DEFAULT 0,
    responses_used INT DEFAULT 0,
    period_start TIMESTAMP NOT NULL DEFAULT NOW(),
    created_at TIMESTAMP DEFAULT NOW(),
    updated_at TIMESTAMP DEFAULT NOW(),
    UNIQUE(user_id)
);

-- Add index for faster lookups
CREATE INDEX idx_usage_user_id ON usage(user_id);

-- Add plan limits structure (example plans)
INSERT INTO subscription_plans (id, name, price, duration_days, limits, features, created_at, updated_at)
VALUES 
    ('free-tier', 'Free', 0, 30, 
     '{"max_publications": 5, "max_responses": 0, "can_promote": false, "analytics": false}'::jsonb,
     '["5 publications per month", "Unlimited responses", "Basic profile"]'::jsonb,
     NOW(), NOW()),
    ('mwork-start', 'MWork Start', 990, 30,
     '{"max_publications": 20, "max_responses": 0, "can_promote": true, "analytics": true}'::jsonb,
     '["20 publications per month", "Unlimited responses", "Promotion", "Analytics"]'::jsonb,
     NOW(), NOW()),
    ('mwork-pro', 'MWork Pro', 2990, 30,
     '{"max_publications": 0, "max_responses": 0, "can_promote": true, "analytics": true, "priority_support": true}'::jsonb,
     '["Unlimited publications", "Unlimited responses", "Priority promotion", "Advanced analytics", "Priority support"]'::jsonb,
     NOW(), NOW())
ON CONFLICT (id) DO NOTHING;
