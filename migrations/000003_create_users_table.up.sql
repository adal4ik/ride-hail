CREATE TABLE IF NOT EXISTS users (
    user_id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    created_at TIMESTAMPTZ DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMPTZ DEFAULT CURRENT_TIMESTAMP,
    username TEXT NOT NULL,
    email TEXT UNIQUE NOT NULL,
    password_hash VARCHAR(255) NOT NULL,
    role roles DEFAULT 'PASSENGER',
    status user_status DEFAULT 'ACTIVE',
    attrs JSONB DEFAULT '{}'::JSONB
);
