# Pantry Scrolls API

This repository contains the backend API for Pantry Scrolls.

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
