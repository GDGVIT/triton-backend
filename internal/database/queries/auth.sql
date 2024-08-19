-- name: CreateUser :one
INSERT INTO users (auth_type, oauth_id)
VALUES (@auth_type, @oauth_id)
RETURNING uuid;

-- name: GetUserByOAuthID :one
SELECT uuid
FROM users
WHERE auth_type = @auth_type AND oauth_id = @oauth_id;

-- name: GetUserByToken :one
SELECT users.uuid, users.created_at, users.oauth_id
FROM users
INNER JOIN tokens
ON users.uuid = tokens.user_uuid
WHERE tokens.hash = $1
AND tokens.scope = $2
AND tokens.expiry > $3;