-- Remove type field from events table
-- Drop the index first
DROP INDEX IF EXISTS idx_events_type;

-- Drop the check constraint
ALTER TABLE events DROP CONSTRAINT IF EXISTS check_type;

-- Remove the column
ALTER TABLE events DROP COLUMN IF EXISTS type;
