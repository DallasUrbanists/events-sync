-- Create events table with review status and change tracking
CREATE TABLE IF NOT EXISTS events (
    id SERIAL PRIMARY KEY,
    uid VARCHAR(255) UNIQUE NOT NULL,
    organization VARCHAR(255) NOT NULL,
    summary TEXT NOT NULL,
    description TEXT,
    location TEXT,
    start_time TIMESTAMP WITH TIME ZONE NOT NULL,
    end_time TIMESTAMP WITH TIME ZONE NOT NULL,
    created_time TIMESTAMP WITH TIME ZONE,
    modified_time TIMESTAMP WITH TIME ZONE,
    status VARCHAR(50),
    transparency VARCHAR(50),
    review_status VARCHAR(20) DEFAULT 'pending' CHECK (review_status IN ('reviewed', 'rejected', 'pending')),
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

-- Index for efficient UID lookups
CREATE INDEX IF NOT EXISTS idx_events_uid ON events(uid);

-- Index for organization queries
CREATE INDEX IF NOT EXISTS idx_events_organization ON events(organization);

-- Index for review status queries
CREATE INDEX IF NOT EXISTS idx_events_review_status ON events(review_status);

-- Index for time-based queries
CREATE INDEX IF NOT EXISTS idx_events_start_time ON events(start_time);

-- Function to update the updated_at timestamp
CREATE OR REPLACE FUNCTION update_updated_at_column()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = NOW();
    RETURN NEW;
END;
$$ language 'plpgsql';

-- Trigger to automatically update updated_at
CREATE TRIGGER update_events_updated_at
    BEFORE UPDATE ON events
    FOR EACH ROW
    EXECUTE FUNCTION update_updated_at_column();