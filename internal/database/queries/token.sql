-- name: DeleteTokenByToken :exec
DELETE FROM tokens
WHERE hash = @hash;

-- -- name: GetNewAccessTokenByRefreshToken :one
-- SELECT new_access_token
-- FROM tokens
-- WHERE refresh_token = @refresh_token AND is_valid = true;

-- name: CreateNewToken :exec
INSERT INTO tokens (hash,user_uuid,expiry,scope)
VALUES (@hash, @user_uuid, @expiry, @scope);

-- name: DeleteTokenForUser :exec
DELETE FROM tokens
WHERE user_uuid = @user_uuid;