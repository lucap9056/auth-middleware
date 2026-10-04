CREATE TABLE IF NOT EXISTS {{.UsersTable}} (
    user_id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    username TEXT NOT NULL,
    {{.UsersEmailColumn}} {{.UsersEmailType}} NOT NULL UNIQUE
);