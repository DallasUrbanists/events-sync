-- Update primary key to be combination of uid and recurrence_id
-- First, drop the existing unique constraint on uid
ALTER TABLE events DROP CONSTRAINT IF EXISTS events_uid_key;

-- Drop the existing primary key constraint
ALTER TABLE events DROP CONSTRAINT IF EXISTS events_pkey;

-- Create a unique constraint on (uid, recurrence_id) to handle NULL values properly
-- For events without recurrence_id, we'll use a special value to ensure uniqueness
UPDATE events SET recurrence_id = '' WHERE recurrence_id IS NULL;

-- Add the new composite primary key
ALTER TABLE events ADD PRIMARY KEY (uid, recurrence_id);

-- Drop the old index since it's no longer needed (primary key creates an index)
DROP INDEX IF EXISTS idx_events_uid;

-- Update the composite index to be more efficient
DROP INDEX IF EXISTS idx_events_uid_recurrence_id;

-- Create a new index for uid-only lookups (for events without recurrence_id)
CREATE INDEX IF NOT EXISTS idx_events_uid_null_recurrence ON events(uid) WHERE recurrence_id = '';

-- Create a new index for uid + recurrence_id lookups
CREATE INDEX IF NOT EXISTS idx_events_uid_recurrence_id ON events(uid, recurrence_id);

-- Keep the existing indexes for other fields
-- (organization, review_status, start_time, sequence indexes remain unchanged)