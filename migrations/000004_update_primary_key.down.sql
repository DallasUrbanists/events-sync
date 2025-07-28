-- Revert primary key changes
-- Drop the new composite primary key
ALTER TABLE events DROP CONSTRAINT IF EXISTS events_pkey;

-- Restore the original primary key on id
ALTER TABLE events ADD PRIMARY KEY (id);

-- Restore the unique constraint on uid
ALTER TABLE events ADD CONSTRAINT events_uid_key UNIQUE (uid);

-- Restore NULL values for empty recurrence_id
UPDATE events SET recurrence_id = NULL WHERE recurrence_id = '';

-- Drop the new indexes
DROP INDEX IF EXISTS idx_events_uid_null_recurrence;
DROP INDEX IF EXISTS idx_events_uid_recurrence_id;

-- Restore the original uid index
CREATE INDEX IF NOT EXISTS idx_events_uid ON events(uid);

-- Restore the original composite index
CREATE INDEX IF NOT EXISTS idx_events_uid_recurrence_id ON events(uid, recurrence_id);