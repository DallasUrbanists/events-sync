-- Add overlay column to events table for manual property overrides
ALTER TABLE events ADD COLUMN overlay JSON;
