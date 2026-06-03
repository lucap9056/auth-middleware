# Database Module

This module handles data persistence for the `auth-middleware` project. It uses PostgreSQL as the underlying database.

## Prerequisites

- **PostgreSQL**: A running PostgreSQL instance.
- **UUID Extension**: The database must support the `uuid-ossp` extension.

## Initialization

You must execute the `init.sql` script to create the necessary tables and extensions:

```bash
# Example using psql
psql -h <host> -U <username> -d <database_name> -f init.sql
```

## Schema Details

The module relies on two tables:

1.  **`users`**: Stores user information.
2.  **`user_devices`**: Stores device information, including the `secret` used for JWT signing. 

> **Important Note**: Ensure your database user has sufficient privileges to execute `CREATE EXTENSION IF NOT EXISTS "uuid-ossp";` and to perform CRUD operations on these tables.

## Implementation Requirements

This module is designed to be injected into other modules (like `jwt`). Ensure your implementation of the data access layer properly handles:

- **Connection Pooling**: Use a library like `pgx` with a connection pool for better performance.
- **Error Handling**: Properly map database errors to application-level errors, especially for missing records (e.g., when `GetDeviceSecret` fails).
