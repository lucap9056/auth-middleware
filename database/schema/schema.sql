CREATE TABLE IF NOT EXISTS auth_user_devices (
    device_id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    device_name TEXT NOT NULL,
    user_email {{.UsersEmailType}} NOT NULL,
    secret TEXT NOT NULL,
    updated_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    CONSTRAINT fk_auth_user_devices FOREIGN KEY (user_email)
        REFERENCES {{.UsersTable}} ({{.UsersEmailColumn}})
        ON DELETE CASCADE ON UPDATE CASCADE
);

CREATE INDEX IF NOT EXISTS idx_auth_user_devices_user_email ON auth_user_devices(user_email);
CREATE INDEX IF NOT EXISTS idx_auth_user_devices_updated_at ON auth_user_devices(updated_at);
