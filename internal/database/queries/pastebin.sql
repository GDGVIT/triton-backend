-- name: CreatePastebin :one
INSERT INTO pastebin (user_uuid, title, content, url_uuid, extension)
VALUES (@user_uuid, @title, @content, @url_uuid, @extension)
RETURNING *;

-- name: GetPastebin :one
SELECT p.user_uuid, p.title, p.content, p.extension
FROM pastebin p
INNER JOIN url u
ON p.url_uuid = u.url_uuid
WHERE u.url_name = @url::text;