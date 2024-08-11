-- name: CreateURL :one
INSERT INTO url (url_name)
VALUES (@url_name)
RETURNING url_uuid;