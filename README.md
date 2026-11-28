# Chirpy

Chirpy is a small social microblogging API written in Go. It supports user accounts, JWT and refresh-token authentication, short posts (chirps), profanity filtering, post lookup and sorting, admin metrics, and a premium Chirpy Red webhook.

## Features

- Create, read, and delete chirps
- Filter chirp content for selected profane words
- Create and update user accounts
- Password hashing with Argon2
- JWT access tokens and refresh tokens
- PostgreSQL storage with SQLC-generated database code
- Static app serving and admin metrics
- Chirpy Red premium upgrade webhook

## Tech Stack

- Go
- PostgreSQL
- [SQLC](https://sqlc.dev/) for type-safe database queries
- [Argon2](https://github.com/alexedwards/argon2id) for password hashing
- [JWT](https://github.com/golang-jwt/jwt) for access tokens
- `net/http` for the HTTP server
- HTML static assets

## Project Structure

```text
chirpy/
├── main.go                 # Server setup and API handlers
├── index.html              # Simple static page
├── assets/                 # Static assets
├── internal/
│   ├── auth/               # Password, JWT, API key, and refresh-token helpers
│   ├── database/           # SQLC-generated models and query methods
│   ├── handlers/           # Shared HTTP handlers
│   └── helpers/            # JSON and error response helpers
├── sql/
│   ├── schema/             # Goose-style PostgreSQL migrations
│   └── queries/            # SQL queries generated with SQLC
└── sqlc.yaml               # SQLC configuration
```

## Prerequisites

- Go 1.27 or newer
- PostgreSQL
- [SQLC](https://sqlc.dev/docs/overview/install) (only needed when changing SQL queries)
- [Goose](https://github.com/pressly/goose) (optional, for running migrations)

## Environment Variables

Create a `.env` file in the project root:

```dotenv
DB_URL=postgres://username:password@localhost:5432/chirpy?sslmode=disable
PLATFORM=dev
SECRET=a-long-random-jwt-secret
POLKA_KEY=a-long-random-webhook-api-key
```

| Variable | Purpose |
| --- | --- |
| `DB_URL` | PostgreSQL connection string |
| `PLATFORM` | Set to `dev` to enable the local database reset endpoint |
| `SECRET` | Secret key used to sign and verify JWT access tokens |
| `POLKA_KEY` | API key used to authenticate the Chirpy Red webhook |

## Database Setup

Create a PostgreSQL database, then apply the migrations in `sql/schema` in numerical order with Goose:

```bash
createdb chirpy

cd sql/schema
goose postgres "$DB_URL" up
```

To regenerate the SQLC Go package after editing a query or migration:

```bash
sqlc generate
```

## Run the Server

```bash
go mod download
go run .
```

The server listens on `http://localhost:8080`.

Open `http://localhost:8080/app/` for the static app page.

## API Overview

| Method | Endpoint | Description |
| --- | --- | --- |
| `GET` | `/api/healthz` | Health check |
| `POST` | `/api/users` | Create a user |
| `PUT` | `/api/users` | Update email or password |
| `POST` | `/api/login` | Login and receive access and refresh tokens |
| `POST` | `/api/refresh` | Exchange a refresh token for a new access token |
| `POST` | `/api/revoke` | Revoke a refresh token |
| `POST` | `/api/chirps` | Create a chirp |
| `GET` | `/api/chirps` | List chirps; supports `author_id` and `sort=desc` |
| `GET` | `/api/chirps/{chirpID}` | Get one chirp |
| `DELETE` | `/api/chirps/{chirpID}` | Delete your own chirp |
| `GET` | `/admin/metrics` | View static file request count |
| `POST` | `/admin/reset` | Reset metrics and users in development |
| `POST` | `/api/polka/webhooks` | Upgrade a user to Chirpy Red |

Protected endpoints expect a bearer token:

```text
Authorization: Bearer <token>
```

## Example Requests

Create a user:

```bash
curl -X POST http://localhost:8080/api/users \
  -H 'Content-Type: application/json' \
  -d '{"email":"user@example.com","password":"secure-password"}'
```

Create a chirp:

```bash
curl -X POST http://localhost:8080/api/chirps \
  -H 'Content-Type: application/json' \
  -H 'Authorization: Bearer <access-token>' \
  -d '{"body":"Hello from Chirpy!"}'
```

List chirps in descending order:

```bash
curl 'http://localhost:8080/api/chirps?sort=desc'
```

## Tests

Run the Go test suite from the project root:

```bash
go test ./...
```

## License

No license has been specified for this project.
