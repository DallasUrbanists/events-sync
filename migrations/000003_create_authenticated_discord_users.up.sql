-- Create authenticated_discord_users table
CREATE TABLE IF NOT EXISTS authenticated_discord_users (
    id SERIAL PRIMARY KEY,
    discord_id VARCHAR(255) UNIQUE NOT NULL,
    username VARCHAR(255) NOT NULL,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

-- Index for efficient Discord ID lookups
CREATE INDEX IF NOT EXISTS idx_authenticated_discord_users_discord_id ON authenticated_discord_users(discord_id);

-- Trigger to automatically update updated_at
CREATE TRIGGER update_authenticated_discord_users_updated_at
    BEFORE UPDATE ON authenticated_discord_users
    FOR EACH ROW
    EXECUTE FUNCTION update_updated_at_column();