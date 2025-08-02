-- Remove recurrence-related fields from events table
DROP INDEX IF EXISTS idx_events_sequence;
DROP INDEX IF EXISTS idx_events_uid_recurrence_id;

ALTER TABLE events
DROP COLUMN IF EXISTS sequence,
DROP COLUMN IF EXISTS recurrence_id,
DROP COLUMN IF EXISTS rrule,
DROP COLUMN IF EXISTS rdate,
DROP COLUMN IF EXISTS exdate;