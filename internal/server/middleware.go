package server

import (
	"crypto/sha256"
	"errors"
	"strings"
	"time"
	"triton-backend/internal/controllers/v1/auth"
	"triton-backend/internal/database"
	"triton-backend/internal/merrors"

	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
)

func CORSMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Writer.Header().Set("Access-Control-Allow-Origin", "*")
		c.Writer.Header().Set("Access-Control-Allow-Credentials", "true")
		c.Writer.Header().Set("Access-Control-Allow-Headers", "Content-Type, Content-Length, Accept-Encoding, X-CSRF-Token, Authorization, accept, origin, Cache-Control, X-Requested-With")
		c.Writer.Header().Set("Access-Control-Allow-Methods", "POST, OPTIONS, GET, PUT")

		if c.Request.Method == "OPTIONS" {
			c.AbortWithStatus(204)
			return
		}

		c.Next()
	}
}

func (s *Server) authenticate() gin.HandlerFunc {
	return func(ctx *gin.Context) {
		ctx.Writer.Header().Add("Vary", "Authorization")

		authorizationHeader := ctx.Request.Header.Get("Authorization")

		if authorizationHeader == "" {
			ctx.Set("user", auth.AnonymousUser)
			ctx.Next()
			return
		}

		headerParts := strings.Split(authorizationHeader, " ")
		if len(headerParts) != 2 || headerParts[0] != "Bearer" {
			ctx.Writer.Header().Set("WWW-Authenticate", "Bearer")
			merrors.Unauthorized(ctx, "invalid or missing authentication token")
			return
		}

		token := headerParts[1]

		if v, err := auth.ValidateTokenPlaintext(token); !v {
			ctx.Writer.Header().Set("WWW-Authenticate", "Bearer")
			merrors.Unauthorized(ctx, err)
			return
		}
		hash := sha256.Sum256([]byte(token))

		q := database.New(s.db)

		user, err := q.GetUserByToken(ctx, database.GetUserByTokenParams{
			Hash:   hash[:],
			Scope:  "authentication",
			Expiry: pgtype.Timestamptz{Time: time.Now(), Valid: true},
		})
		if err != nil {
			switch {
			case errors.Is(err, pgx.ErrNoRows):
				ctx.Writer.Header().Set("WWW-Authenticate", "Bearer")
				merrors.Unauthorized(ctx, "invalid or missing authentication token")
			default:
				merrors.InternalServer(ctx, err.Error())
			}
			return
		}

		ctx.Set("user", &user)

		ctx.Next()
	}
}
