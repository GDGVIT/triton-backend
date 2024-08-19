package auth

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/base32"
	"time"
	"triton-backend/internal/database"

	"github.com/google/uuid"
)

type Token struct {
	Plaintext string    `json:"token"`
	Hash      []byte    `json:"-"`
	UserID    uuid.UUID `json:"-"`
	Expiry    time.Time `json:"expiry"`
	Scope     string    `json:"-"`
}

var AnonymousUser = &database.GetUserByTokenRow{}

func ValidateTokenPlaintext(tokenPlaintext string) (bool, string) {
	if tokenPlaintext == "" {
		return false, "token must be provided"
	}
	if len(tokenPlaintext) != 26 {
		return false, "token must be 26 bytes long"
	}
	return true, ""
}

func generateToken(userID uuid.UUID, ttl time.Duration, scope string) (*Token, error) {
	token := &Token{
		UserID: userID,
		Expiry: time.Now().Add(ttl),
		Scope:  scope,
	}

	randomBytes := make([]byte, 16)

	_, err := rand.Read(randomBytes)
	if err != nil {
		return nil, err
	}

	token.Plaintext = base32.StdEncoding.WithPadding(base32.NoPadding).EncodeToString(randomBytes)

	hash := sha256.Sum256([]byte(token.Plaintext))
	token.Hash = hash[:]
	return token, nil
}
