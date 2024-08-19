-- CREATE TABLE tokens (
--     id SERIAL PRIMARY KEY,
--     token TEXT NOT NULL UNIQUE,
--     user_uuid UUID REFERENCES users(uuid) ON DELETE CASCADE,
--     refresh_token TEXT NOT NULL,
--     new_access_token TEXT NOT NULL,
--     is_valid BOOLEAN DEFAULT true,
--     created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
-- );

CREATE TABLE IF NOT EXISTS tokens (
    id SERIAL PRIMARY KEY,
    hash bytea NOT NULL,
    user_uuid uuid REFERENCES users(uuid) ON DELETE CASCADE,
    expiry timestamp(0) with time zone NOT NULL,
    scope text NOT NULL,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);