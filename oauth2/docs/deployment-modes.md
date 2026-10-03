# Deployment Modes

The mode is selected by `DATABASE_URL` and `DB_USER_EMAIL_REFERENCE`. See the [comparison table](../README.md#deployment-modes).

## Stateless proxy

A lightweight OAuth2 client that keeps the client secret, `state`, and PKCE verifier on the server and hands the provider's tokens to the client. No database is needed; `DB_*`, `JWT_*`, `ALLOW_REGISTRATION`, and `ALLOW_UNVERIFIED_EMAIL` have no effect. Downstream services must validate the provider's tokens themselves.

## Managed users

This server owns a `users` table:

```sql
CREATE TABLE IF NOT EXISTS users (
    user_id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    username TEXT NOT NULL,
    email TEXT NOT NULL UNIQUE
);
```

With `ALLOW_REGISTRATION=true`, a user who signs in for the first time is inserted with the provider's display name as `username`. Without it, only emails already in `users` can sign in. When `DB_AUTO_CREATE_SCHEMA` is not `true`, the table must already exist; this is checked on startup.

## External users

Your application owns the users table, and `DB_USER_EMAIL_REFERENCE` tells this server where it is, e.g. `public.accounts(email):citext`. The email column must be a `PRIMARY KEY` or `UNIQUE`. This server never writes to that table: only emails that already exist can sign in. Deleting a user in your application removes their sessions through the foreign key.

To put a display name in the `username` claim, set `DB_USER_USERNAME_COLUMN` to a column of the same table. This requires `SELECT` privilege on it, is checked on startup, and `NULL` becomes an empty string.
