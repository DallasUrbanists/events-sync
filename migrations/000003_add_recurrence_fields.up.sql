-- Add recurrence-related fields to events table
ALTER TABLE events
ADD COLUMN sequence INTEGER DEFAULT 0,
ADD COLUMN recurrence_id VARCHAR(255),
ADD COLUMN rrule TEXT,
ADD COLUMN rdate TEXT,
ADD COLUMN exdate TEXT;

-- Create composite index for UID and recurrence_id for efficient lookups
CREATE INDEX IF NOT EXISTS idx_events_uid_recurrence_id ON events(uid, recurrence_id);

-- Create index for sequence-based queries
CREATE INDEX IF NOT EXISTS idx_events_sequence ON events(sequence);