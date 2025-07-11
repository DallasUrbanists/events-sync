-- Drop trigger first
DROP TRIGGER IF EXISTS update_events_updated_at ON events;

-- Drop function
DROP FUNCTION IF EXISTS update_updated_at_column();

-- Drop indexes
DROP INDEX IF EXISTS idx_events_start_time;
DROP INDEX IF EXISTS idx_events_review_status;
DROP INDEX IF EXISTS idx_events_organization;
DROP INDEX IF EXISTS idx_events_uid;

-- Drop table
DROP TABLE IF EXISTS events;