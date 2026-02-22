-- Users table
CREATE TABLE IF NOT EXISTS users
(
    id            SERIAL PRIMARY KEY,
    username      VARCHAR(50) UNIQUE  NOT NULL,
    email         VARCHAR(255) UNIQUE NOT NULL,
    password_hash VARCHAR(255)        NOT NULL,
    display_name  VARCHAR(100),
    avatar_url    VARCHAR(500),
    created_at    TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at    TIMESTAMP DEFAULT CURRENT_TIMESTAMP
    );

-- Messages table
CREATE TABLE IF NOT EXISTS messages
(
    id           SERIAL PRIMARY KEY,
    sender_id    INTEGER NOT NULL REFERENCES users (id) ON DELETE CASCADE,
    recipient_id INTEGER NOT NULL REFERENCES users (id) ON DELETE CASCADE,
    content      TEXT    NOT NULL,
    is_read      BOOLEAN   DEFAULT FALSE,
    created_at   TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- Create indexes for messages table
CREATE INDEX IF NOT EXISTS idx_sender ON messages(sender_id);
CREATE INDEX IF NOT EXISTS idx_recipient ON messages(recipient_id);
CREATE INDEX IF NOT EXISTS idx_created_at ON messages(created_at);

-- Create updated_at trigger function
CREATE OR REPLACE FUNCTION update_updated_at_column()
    RETURNS TRIGGER AS
$$
BEGIN
    NEW.updated_at = CURRENT_TIMESTAMP;
RETURN NEW;
END;
$$ language 'plpgsql';

-- Apply trigger to users table
CREATE TRIGGER update_users_updated_at
    BEFORE UPDATE
    ON users
    FOR EACH ROW
    EXECUTE FUNCTION update_updated_at_column();