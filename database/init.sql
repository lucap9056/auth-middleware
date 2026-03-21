CREATE EXTENSION IF NOT EXISTS "uuid-ossp";

CREATE TABLE IF NOT EXISTS users (
    user_id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    username TEXT NOT NULL,
    email TEXT NOT NULL UNIQUE
);

CREATE TABLE IF NOT EXISTS user_devices (
    device_id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    device_name TEXT NOT NULL,
    user_id UUID NOT NULL,
    secret TEXT NOT NULL,
    updated_at TIMESTAMPTZ DEFAULT CURRENT_TIMESTAMP,
    CONSTRAINT fk_user
        FOREIGN KEY(user_id) 
        REFERENCES users(user_id) 
        ON DELETE CASCADE
);

CREATE INDEX IF NOT EXISTS idx_user_devices_updated_at 
ON user_devices (updated_at);