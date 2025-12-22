-- Create users table
CREATE TABLE IF NOT EXISTS users (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    email VARCHAR(255) UNIQUE,
    password_hash VARCHAR(255),
    expo_device_id VARCHAR(255) UNIQUE,
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    last_login_at TIMESTAMP WITH TIME ZONE,

    -- Constraints to ensure valid account types
    CONSTRAINT check_account_type CHECK (
        -- Must have either email/password OR expo_device_id OR both
        (email IS NOT NULL AND password_hash IS NOT NULL) OR
        (expo_device_id IS NOT NULL)
    ),
    CONSTRAINT check_email_with_password CHECK (
        -- If email is present, password must be present
        (email IS NULL AND password_hash IS NULL) OR
        (email IS NOT NULL AND password_hash IS NOT NULL)
    )
);

-- Create indexes for users table
CREATE INDEX IF NOT EXISTS idx_users_email ON users(email) WHERE email IS NOT NULL;
CREATE INDEX IF NOT EXISTS idx_users_expo_device_id ON users(expo_device_id) WHERE expo_device_id IS NOT NULL;
CREATE INDEX IF NOT EXISTS idx_users_created_at ON users(created_at);

-- Create function to update updated_at timestamp
CREATE OR REPLACE FUNCTION update_updated_at_column()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = NOW();
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

-- Create trigger to automatically update updated_at
CREATE TRIGGER update_users_updated_at
    BEFORE UPDATE ON users
    FOR EACH ROW
    EXECUTE FUNCTION update_updated_at_column();

-- Add comments for documentation
COMMENT ON TABLE users IS 'Stores user account information for email/password and mobile device authentication';
COMMENT ON COLUMN users.email IS 'User email address - NULL for mobile-only accounts';
COMMENT ON COLUMN users.password_hash IS 'Bcrypt hash of user password - NULL for mobile-only accounts';
COMMENT ON COLUMN users.expo_device_id IS 'Expo Application Installation ID - NULL for email-only accounts';
COMMENT ON COLUMN users.last_login_at IS 'Timestamp of last successful login';
