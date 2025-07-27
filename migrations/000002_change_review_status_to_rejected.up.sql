-- Change review_status column to rejected boolean
ALTER TABLE events ADD COLUMN rejected BOOLEAN DEFAULT FALSE;

-- Migrate existing data: set rejected = true for events with review_status = 'rejected'
UPDATE events SET rejected = true WHERE review_status = 'rejected';

-- Drop the old review_status column
ALTER TABLE events DROP COLUMN review_status;

-- Drop the old index
DROP INDEX IF EXISTS idx_events_review_status;

-- Create new index for rejected status
CREATE INDEX IF NOT EXISTS idx_events_rejected ON events(rejected);