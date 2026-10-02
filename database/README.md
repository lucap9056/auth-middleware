# Database Module

This module stores device sessions for the `auth-middleware` project in PostgreSQL. User accounts are owned by your application; this module only references them by email.

## Requirements

- **PostgreSQL 13+**
- **An existing users table** whose email column is a `PRIMARY KEY` or has a `UNIQUE` constraint, so a foreign key can reference it

## Installation

```bash
go get github.com/lucap9056/auth-middleware/database/v2
```

## Usage

```go
import "github.com/lucap9056/auth-middleware/database/v2"

userRef, err := database.WithUserEmailReference("users(email)")
if err != nil {
	return err
}

db, err := database.NewDatabase(dsn,
	database.WithAutoCreateSchema(true),
	userRef,
)
if err != nil {
	return err
}
defer db.Close()
```

## Options

| Option | Default | Description |
|---|---|---|
| `WithMaxOpenConns(n)` | `20` | Maximum number of open connections |
| `WithMaxIdleConns(n)` | `15` | Maximum number of idle connections |
| `WithConnMaxLifetime(d)` | `5m` | Maximum lifetime of a connection |
| `WithConnMaxIdleTime(d)` | `2m` | Maximum idle time of a connection |
| `WithCleanupInterval(d)` | `24h` | Interval for deleting devices not updated in the last 7 days; `0` disables cleanup |
| `WithAutoCreateSchema(bool)` | `false` | Create the `auth_user_devices` table and its indexes on startup |
| `WithUserEmailReference(ref)` | `users(email)` | Users table and email column referenced by the foreign key |

### User Email Reference

Format: `[<schema>.]<table>(<column>)[:<type>]`

| Example | Meaning |
|---|---|
| `users(email)` | `users.email`, type `text` |
| `users(email):citext` | `users.email`, type `citext` |
| `auth.members(mail):varchar(255)` | `auth.members.mail`, type `varchar(255)` |

The reference is case-insensitive and normalized to lowercase. `<type>` must match the type of the referenced column.

### Auto Create Schema

When enabled, `NewDatabase` creates the schema, which requires DDL privileges. When disabled, create the schema yourself using `schema.Generate`, or with the SQL below.

## Schema

```sql
CREATE TABLE IF NOT EXISTS auth_user_devices (
    device_id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    device_name TEXT NOT NULL,
    user_email TEXT NOT NULL,
    secret TEXT NOT NULL,
    generation INT NOT NULL DEFAULT 1,
    updated_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    CONSTRAINT fk_auth_user_devices FOREIGN KEY (user_email)
        REFERENCES users (email)
        ON DELETE CASCADE ON UPDATE CASCADE
);

CREATE INDEX IF NOT EXISTS idx_auth_user_devices_user_email ON auth_user_devices(user_email);
CREATE INDEX IF NOT EXISTS idx_auth_user_devices_updated_at ON auth_user_devices(updated_at);
```

## API

| Method | Description |
|---|---|
| `SaveDeviceSecret(userEmail, deviceName, secret)` | Creates a device and returns its `device_id`. Returns `ErrUserNotFound` if the email does not exist in the users table |
| `GetDeviceSecret(deviceID)` | Returns the device's `secret` and current `generation` |
| `UpdateDeviceSecret(deviceID)` | Increments the device's `generation`, refreshes `updated_at`, and returns the new `generation` |
| `DeleteDevice(userEmail, deviceID)` | Deletes one device belonging to the user |
| `DeleteAllDevices(userEmail)` | Deletes all devices of the user |
| `DeleteAllDevicesReturningIDs(userEmail)` | Deletes all devices of the user and returns their IDs |
| `Close()` | Stops the cleanup worker and closes the connection pool |
