package server

import (
	"fmt"
	"net/http"
	"os"
	"strconv"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	_ "github.com/joho/godotenv/autoload"

	"triton-backend/internal/controllers/v1/auth"
	"triton-backend/internal/controllers/v1/pastebin"
	"triton-backend/internal/database"
)

type Server struct {
	port            int
	db              *pgxpool.Pool
	PastebinHandler *pastebin.PastebinHandler
	AuthHandler     *auth.AuthHandler
}

func NewServer() *http.Server {
	port, _ := strconv.Atoi(os.Getenv("PORT"))
	db := database.NewService()
	server := &Server{
		port:            port,
		db:              db,
		PastebinHandler: pastebin.Handler(db),
		AuthHandler:     auth.Handler(db),
	}

	// Create a new HTTP server
	return &http.Server{
		Addr:         fmt.Sprintf(":%d", server.port),
		Handler:      server.RegisterRoutes(),
		IdleTimeout:  time.Minute,
		ReadTimeout:  10 * time.Second,
		WriteTimeout: 30 * time.Second,
	}
}
