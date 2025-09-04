-- Add type field to events table
-- Default value is 'social_gathering' for all existing events
ALTER TABLE events ADD COLUMN type VARCHAR(50) NOT NULL DEFAULT 'social_gathering';

-- Add check constraint to ensure valid event types
ALTER TABLE events ADD CONSTRAINT check_type CHECK (type IN ('civic_meeting', 'social_gathering', 'volunteer_action'));

-- Create index for efficient event type queries
CREATE INDEX IF NOT EXISTS idx_events_type ON events(type);

-- Update all existing events to have the default type
UPDATE events SET type = 'social_gathering' WHERE type IS NULL;
