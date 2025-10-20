CREATE TABLE IF NOT EXISTS  jwt_tokens (
    jwt_token_id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    created_at TIMESTAMPTZ DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMPTZ DEFAULT CURRENT_TIMESTAMP,
    user_id UUID REFERENCES users(user_id) ON DELETE CASCADE,
    refresh_token TEXT NOT NULL
);