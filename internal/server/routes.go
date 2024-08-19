package server

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
)

func (s *Server) RegisterRoutes() http.Handler {
	r := gin.Default()
	r.Use(CORSMiddleware())

	r.GET("/", s.HelloWorldHandler)
	r.GET("/health", s.healthHandler)

	v1 := r.Group("/v1")
	v1.Use(s.authenticate())
	v1.POST("/pastebin/create", s.PastebinHandler.CreatePastebin)
	v1.GET("/pastebin/:url", s.PastebinHandler.GetPastebin)

	authGroup := v1.Group("/auth")
	{
		// OAuth Login
		authGroup.GET("/login/oauth", s.AuthHandler.GoogleLoginHandler)

		// OAuth Callback
		authGroup.GET("/callback/oauth", s.AuthHandler.GoogleCallbackHandler)

		// Register User via OAuth
		authGroup.POST("/register/oauth", s.AuthHandler.RegisterOAuthUser)

		// Register User Anonymously
		authGroup.POST("/register/anonymous", s.AuthHandler.RegisterAnonymousUser)

		// Get User by OAuth ID
		authGroup.POST("/get/oauth", s.AuthHandler.GetUserByOAuthID)

		// Get User by Anonymous ID
		authGroup.POST("/get/anonymous", s.AuthHandler.GetUserByAnonymousID)

		// Logout
		authGroup.POST("/logout", s.AuthHandler.LogoutHandler) // Needs implementation

		// Refresh OAuth Token
		// authGroup.POST("/token/refresh", s.AuthHandler.RefreshTokenHandler) // Needs implementation
	}

	return r
}

func (s *Server) HelloWorldHandler(c *gin.Context) {
	resp := make(map[string]string)
	resp["message"] = "Hello World"

	c.JSON(http.StatusOK, resp)
}

func (s *Server) healthHandler(c *gin.Context) {
	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
	defer cancel()

	stats := make(map[string]string)

	err := s.db.Ping(ctx)
	if err != nil {
		stats["status"] = "down"
		stats["error"] = fmt.Sprintf("db down: %v", err)
		log.Fatalf(fmt.Sprintf("db down: %v", err)) // Log the error and terminate the program
		c.JSON(http.StatusInternalServerError, stats)
		return
	}

	// Database is up, add more statistics
	stats["status"] = "up"
	stats["message"] = "It's healthy"
	c.JSON(http.StatusOK, stats)
}
