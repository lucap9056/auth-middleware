# Auth Middleware

A modular Go-based authentication and authorization middleware system designed for flexibility and security.

## Architecture

This project is structured as a set of independent, Go-module-based components:

- **[`/database`](./database/README.md)**: Data persistence layer using PostgreSQL. Provides schema initialization and a standard interface for device secret management.
- **[`/jwt`](./jwt/README.md)**: Core JWT management module. Handles token generation, validation, and utilizes the `Database` interface for secure, device-bound token lifecycles.
- **[`/oauth2`](./oauth2/README.md)**: An authentication server module that implements OAuth2 flows, integrating `database` and `jwt` modules for robust session management.

## Getting Started

1.  **Database Setup**: Initialize your PostgreSQL database using `database/init.sql`.
2.  **Configuration**: Configure each module according to its respective `README.md`.
3.  **Deployment**: Deploy the `oauth2` module as your authentication service, or integrate `jwt` and `database` modules directly into your existing Go application.
