-- Create portfolio_items table
CREATE TABLE IF NOT EXISTS portfolio_items (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    model_id UUID NOT NULL REFERENCES model_profiles(id) ON DELETE CASCADE,
    upload_id UUID NOT NULL REFERENCES uploads(id) ON DELETE CASCADE,
    title VARCHAR(255),
    description TEXT,
    order_index INT DEFAULT 0,
    created_at TIMESTAMP DEFAULT NOW()
);

-- Create reviews table
CREATE TABLE IF NOT EXISTS reviews (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    model_id UUID NOT NULL REFERENCES model_profiles(id) ON DELETE CASCADE,
    employer_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    casting_id UUID REFERENCES castings(id) ON DELETE SET NULL,
    rating INT NOT NULL CHECK (rating >= 1 AND rating <= 5),
    review_text TEXT,
    created_at TIMESTAMP DEFAULT NOW(),
    updated_at TIMESTAMP DEFAULT NOW(),
    UNIQUE(employer_id, model_id, casting_id)
);

-- Add indexes
CREATE INDEX idx_portfolio_model_id ON portfolio_items(model_id);
CREATE INDEX idx_portfolio_order ON portfolio_items(model_id, order_index);
CREATE INDEX idx_reviews_model_id ON reviews(model_id);
CREATE INDEX idx_reviews_employer_id ON reviews(employer_id);
CREATE INDEX idx_reviews_rating ON reviews(rating);
