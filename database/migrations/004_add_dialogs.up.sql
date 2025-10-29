-- Create dialogs table for chat conversations
CREATE TABLE IF NOT EXISTS dialogs (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    participant1_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    participant2_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    casting_id UUID REFERENCES castings(id) ON DELETE SET NULL,
    created_at TIMESTAMP DEFAULT NOW(),
    UNIQUE(participant1_id, participant2_id)
);

-- Add indexes
CREATE INDEX idx_dialogs_participant1 ON dialogs(participant1_id);
CREATE INDEX idx_dialogs_participant2 ON dialogs(participant2_id);
