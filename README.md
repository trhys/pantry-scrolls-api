# Pantry Scrolls API

This repository contains the backend API for Pantry Scrolls.

## Quick start

### Prerequisites

- [Go 1.25+](https://go.dev/dl/)
- [Docker and Docker Compose](https://docs.docker.com/get-docker/) (for running the full stack)
- [goose](https://github.com/pressly/goose) (for database migrations)
- AWS credentials (S3 bucket and SES for image storage and email)
- A PostgreSQL instance (handled by Docker Compose)

### Environment variables

Copy the example below into a `.env` file at the project root and fill in the values.

```env
DB=******localhost:5432/recipe_repo?sslmode=disable
SECRET=<jwt-signing-secret>
JWT_DUR=3600
S3_BUCKET=<bucket-name>
S3_REGION=<aws-region>
S3_CDN=<cdn-base-url>
IMAGE_PLACEHOLDER=<s3-key-for-placeholder-image>
USERPW=<hashed-seed-user-password>
REACTURL=<frontend-origin-url>
AWS_ACCESS_KEY_ID=<aws-access-key>
AWS_SECRET_ACCESS_KEY=<aws-secret-key>
```

### Run with Docker Compose

The compose file starts the API server, a PostgreSQL database with schema migrations and seed data, a Caddy reverse proxy, and optionally Prometheus and Grafana.

```sh
docker compose up --build
```

The API is then available at `https://localhost/api/`.

### Build and run locally

Install Go dependencies:

```sh
go mod download
```

Build the binary:

```sh
go build -o reciperepo .
```

Run the server (requires the environment variables above to be set):

```sh
./reciperepo
```

The server listens on `0.0.0.0:8080` by default.

### Lint and test

```sh
go vet ./...
go test ./...
```

Integration tests in `internal/server/tests` require a running database and a populated `.env` file.

## Production

### Image publishing

The API image is built and published to GHCR automatically on every push to `main` via `.github/workflows/publish.yml`.

Images are tagged with an immutable commit SHA (`sha-<short-sha>`) and `latest` (for `main` builds only).

You can also trigger a build manually from the **Actions** tab using the `workflow_dispatch` event.

Pull the image:

```sh
docker pull ghcr.io/trhys/pantry-scrolls-api:sha-<commit-sha>
```

### Required runtime environment variables

All of the following must be set in the production environment. The app will exit with an error if any are missing.

| Variable | Description |
|---|---|
| `DB` | PostgreSQL connection string (`******host:5432/db?sslmode=...`) |
| `SECRET` | JWT signing secret |
| `JWT_DUR` | JWT lifetime in seconds (e.g. `3600`) |
| `S3_BUCKET` | S3 bucket name for image storage |
| `S3_REGION` | AWS region for S3 and SES |
| `S3_CDN` | CDN base URL for serving images |
| `IMAGE_PLACEHOLDER` | S3 key used for the default placeholder image |
| `REACTURL` | Frontend origin URL for CORS (e.g. `https://app.example.com`) |
| `AWS_ACCESS_KEY_ID` | AWS access key |
| `AWS_SECRET_ACCESS_KEY` | AWS secret key |
| `USERPW` | Seed user password (used by `dbinit` for local seeding; still required at API startup) |

### Health endpoint

The API exposes an unauthenticated health endpoint at `GET /healthz`.

```sh
curl http://localhost:8080/healthz
# {"status":"ok"}
```

The Docker image includes a `HEALTHCHECK` that polls this endpoint every 30 seconds.

### Database migrations

Migrations are managed with [goose](https://github.com/pressly/goose) and live in `sql/schema/`.

For production, run migrations as an explicit step before deploying a new image:

```sh
goose -dir sql/schema postgres "$DB" up
```

Or use the migration binary that is built into the `Dockerfile.db` image:

```sh
# From a container with access to the database:
./goose -dir sql/schema postgres "$DB_URL" up
```

**Do not rely on automatic migration at container startup for production deployments.**

### Seed data (local/dev only)

The `cmd/dbinit` binary seeds ingredients and a root recipe user. It is **not for production**.

When running the local dev stack via Docker Compose, seeding is opt-in. Set `SEED=true` on the `db` service to enable it:

```sh
SEED=true docker compose up --build
```

Without `SEED=true` the DB container will run migrations only and skip seeding.

## API overview

- Base path: `/api`
- Content type: `application/json` unless noted.
- Auth:
  - Protected endpoints use JWT auth middleware.
  - JWT can be sent as `jwt` cookie or an `Authorization` bearer token.
  - Refresh token can be sent as `refresh_token` cookie or bearer token on refresh/revoke routes.

## Response shape notes

- Success responses are JSON.
- `204` responses send no body.
- Error responses return plain text messages (not JSON objects).

## Data shapes

### User
```json
{
  "id": "uuid",
  "name": "string",
  "image_url": "string"
}
```

### Session
```json
{
  "id": "uuid",
  "name": "string",
  "image_url": "string",
  "email": "string",
  "token": "jwt",
  "refresh_token": "token"
}
```

### Refresh session
```json
{
  "id": "uuid",
  "name": "string",
  "image_url": "string",
  "email": "string"
}
```

### Recipe card list
```json
{
  "recipes": [
    {
      "id": "uuid",
      "title": "string",
      "created_at": "timestamp",
      "updated_at": "timestamp",
      "image_url": "string",
      "user_id": "uuid",
      "author": "string"
    }
  ]
}
```

### Recipe full
```json
{
  "id": "uuid",
  "title": "string",
  "created_at": "timestamp",
  "updated_at": "timestamp",
  "user_id": "uuid",
  "author": "string",
  "description": "string",
  "image_url": "string",
  "ingredients": [
    {
      "id": "uuid",
      "name": "string",
      "quantity": 1.5,
      "unit": "string"
    }
  ],
  "instructions": "string"
}
```

### Shopping list
```json
{
  "id": "uuid",
  "name": "string",
  "created_at": "timestamp",
  "updated_at": "timestamp",
  "recipes": [
    {
      "id": "uuid",
      "title": "string",
      "created_at": "timestamp",
      "updated_at": "timestamp",
      "user_id": "uuid",
      "author": "string",
      "quantity": 1
    }
  ]
}
```

### User shopping list collection
```json
{
  "shopping_lists": [
    {
      "id": "uuid",
      "name": "string",
      "created_at": "timestamp",
      "updated_at": "timestamp"
    }
  ]
}
```

### Ingredient base
```json
{
  "ingredients": [
    {
      "id": "uuid",
      "name": "string"
    }
  ]
}
```

### Ingredient units
```json
{
  "units": [
    { "name": "string" }
  ]
}
```

### Printable shopping list
```json
{
  "name": "string",
  "items": [
    {
      "id": "uuid",
      "name": "string",
      "quantity": 1,
      "unit": "string"
    }
  ]
}
```

### Message
```json
{
  "id": "uuid",
  "user_email": "string",
  "message": "string",
  "resolved": false,
  "status": "unread|read|archived"
}
```

## Endpoints

## Metrics

### `GET /api/metrics`
- Auth: none
- Function: Prometheus metrics endpoint.
- Response: Prometheus text exposition format.

## Users and auth

### `POST /api/users`
- Auth: none
- Function: create a user and trigger verification email.
- Request:
```json
{
  "email": "user@example.com",
  "password": "string (min 5 chars)",
  "name": "string (max 30 chars)"
}
```
- Response: `201` (empty JSON body)

### `POST /api/sessions`
- Auth: none
- Function: login and issue JWT + refresh token.
- Request:
```json
{
  "email": "user@example.com",
  "password": "string"
}
```
- Response: `200` with Session shape; also sets `jwt` and `refresh_token` cookies.

### `GET /api/sessions`
- Auth: required
- Function: return session user details from JWT subject.
- Response: `200` with Refresh session shape.

### `GET /api/users/{user_id}`
- Auth: optional JWT (public response without owner auth)
- Function: fetch user profile and recipes. Returns private view when requester is same user.
- Response:
  - Public:
  ```json
  {
    "id": "uuid",
    "name": "string",
    "image_url": "string",
    "recipes": ["Recipe card objects"]
  }
  ```
  - Private (owner):
  ```json
  {
    "id": "uuid",
    "name": "string",
    "image_url": "string",
    "email": "string",
    "created_at": "timestamp",
    "recipes": ["Recipe card objects"],
    "shopping_lists": ["Shopping list objects"]
  }
  ```

### `PUT /api/users`
- Auth: required
- Function: upload/update current user profile image.
- Request: `multipart/form-data` with `image` file (`image/jpeg` or `image/png`).
- Response: `204`

### `PUT /api/users/{user_id}`
- Auth: required (must match `{user_id}`)
- Function: update username.
- Request:
```json
{ "name": "string (max 30 chars)" }
```
- Response: `204`

### `PUT /api/users/{user_id}/deactivate`
- Auth: required (must match `{user_id}`)
- Function: mark account for deactivation and send cancellation email.
- Request: none
- Response: `204`

### `PUT /api/deactivation/cancel`
- Auth: none
- Function: cancel pending account deactivation.
- Request:
```json
{ "token": "string" }
```
- Response: `204`

### `GET /api/verify/{token}`
- Auth: none
- Function: verify user email with token.
- Request: none
- Response: `200`

### `POST /api/resetpassword`
- Auth: none
- Function: send reset password email and token.
- Request:
```json
{ "email": "user@example.com" }
```
- Response: `204`

### `PUT /api/resetpassword`
- Auth: none
- Function: update password using token.
- Request:
```json
{
  "token": "string",
  "password": "string (min 5 chars)"
}
```
- Response: `204`

### `GET /api/users`
- Auth: none
- Function: get total user count.
- Response:
```json
{ "total": 123 }
```

## Recipes

### `GET /api/recipes`
- Auth: none
- Function: list recipe cards.
- Query params:
  - `total=true` returns only total count.
- Response:
  - default: Recipe card list shape
  - with `total=true`:
  ```json
  { "total": 123 }
  ```

### `GET /api/recipes/{recipe_id}`
- Auth: none
- Function: fetch full recipe with ingredients.
- Response: Recipe full shape.

### `POST /api/recipes`
- Auth: required
- Function: create recipe (optionally with image upload).
- Request: `multipart/form-data`
  - `payload` (JSON string):
  ```json
  {
    "title": "string",
    "description": "string",
    "ingredients": [
      {
        "id": "ingredient uuid",
        "quantity": 1.5,
        "unit": "string"
      }
    ],
    "instructions": "string"
  }
  ```
  - `image` file (optional, jpeg/png)
- Response: `200` with Recipe full shape.

### `PUT /api/recipes/{recipe_id}`
- Auth: required (must own recipe)
- Function: update recipe and replace ingredient links.
- Request: `multipart/form-data` using same `payload` + optional `image` as create recipe.
- Response: `204`

### `DELETE /api/recipes/{recipe_id}`
- Auth: required (must own recipe)
- Function: delete recipe.
- Request: none
- Response: `204`

### `GET /api/recipes/explore`
- Auth: none
- Function: search recipe cards.
- Query params:
  - `search` (optional, sanitized, max length 100)
- Response: Recipe card list shape.

## Ingredients

### `GET /api/ingredients`
- Auth: none
- Function: list base ingredients.
- Response: Ingredient base shape.

### `GET /api/ingredients/{ingredient_id}/units`
- Auth: none
- Function: list available units for a specific ingredient.
- Response: Ingredient units shape.

## Shopping lists

### `GET /api/shoppinglists`
- Auth: required
- Function: list shopping lists for authenticated user.
- Response: User shopping list collection shape.

### `POST /api/shoppinglists`
- Auth: required
- Function: create shopping list.
- Request:
```json
{ "name": "string" }
```
- Response:
```json
{
  "id": "uuid",
  "name": "string",
  "created_at": "timestamp",
  "updated_at": "timestamp"
}
```

### `GET /api/shoppinglists/{shopping_list_id}`
- Auth: required (must own list)
- Function: fetch list with recipe entries.
- Response: Shopping list shape.

### `POST /api/shoppinglists/{shopping_list_id}`
- Auth: required
- Function: add/update recipe quantity on shopping list.
- Request:
```json
{
  "recipe_id": "uuid",
  "quantity": 1
}
```
- Response: `204`

### `GET /api/shoppinglists/{shopping_list_id}/print`
- Auth: required (must own list)
- Function: get ingredient-level printable list with retail unit conversion.
- Response: Printable shopping list shape.

### `DELETE /api/shoppinglists/{shopping_list_id}`
- Auth: required (must own list)
- Function: delete shopping list.
- Response: `204`

## Tokens

### `GET /api/tokens/refresh`
- Auth: refresh token required (cookie or bearer)
- Function: issue new JWT.
- Response:
```json
{ "token": "jwt" }
```
- Also sets `jwt` cookie.

### `GET /api/tokens/revoke`
- Auth: refresh token required (cookie or bearer)
- Function: revoke refresh token and clear auth cookies.
- Response: `204`

## Messages

### `POST /api/messages`
- Auth: none
- Function: submit contact/feedback message.
- Request:
```json
{
  "email": "user@example.com",
  "message": "string (1-1000 chars)"
}
```
- Response: `204`

### `GET /api/messages`
- Auth: none
- Function: list messages.
- Query params:
  - `tag` optional (`unread`, `read`, `archived`, or omitted/`none` for all).
- Response:
```json
[
  {
    "id": "uuid",
    "user_email": "string",
    "message": "string",
    "resolved": false,
    "status": "unread"
  }
]
```

### `POST /api/messages/{message_id}`
- Auth: none
- Function: update message status.
- Request:
```json
{ "status": "unread|read|archived" }
```
- Response: `204`
