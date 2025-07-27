-- Revert: Add back review_status column
ALTER TABLE events ADD COLUMN review_status VARCHAR(20) DEFAULT 'pending' CHECK (review_status IN ('reviewed', 'rejected', 'pending'));

-- Migrate data back: set review_status based on rejected value
UPDATE events SET review_status = 'rejected' WHERE rejected = true;
UPDATE events SET review_status = 'reviewed' WHERE rejected = false;

-- Drop the rejected column
ALTER TABLE events DROP COLUMN rejected;

-- Drop the new index
DROP INDEX IF EXISTS idx_events_rejected;

-- Recreate the old index
CREATE INDEX IF NOT EXISTS idx_events_review_status ON events(review_status);