package auth

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/url"
	"os"
	"time"
	"triton-backend/internal/database"
	"triton-backend/internal/merrors"
	"triton-backend/internal/utils"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/jackc/pgerrcode"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"
	"golang.org/x/oauth2"
	"golang.org/x/oauth2/google"
)

type AuthHandler struct {
	db                *pgxpool.Pool
	googleOauthConfig *oauth2.Config
	oauthStateString  string
	frontendCallback  string
}

func Handler(db *pgxpool.Pool) *AuthHandler {
	return &AuthHandler{
		db: db,
		googleOauthConfig: &oauth2.Config{
			ClientID:     os.Getenv("GOOGLE_CLIENT_ID"),
			ClientSecret: os.Getenv("GOOGLE_CLIENT_SECRET"),
			RedirectURL:  os.Getenv("GOOGLE_REDIRECT_URI"),
			Scopes:       []string{"https://www.googleapis.com/auth/userinfo.email", "https://www.googleapis.com/auth/userinfo.profile"},
			Endpoint:     google.Endpoint,
		},
		oauthStateString: "random_state_string", // Ideally, generate dynamically
		frontendCallback: os.Getenv("FRONTEND_CALLBACK_URL"),
	}
}

// GoogleLoginHandler initiates the OAuth2 login process
func (a *AuthHandler) GoogleLoginHandler(c *gin.Context) {
	url := a.googleOauthConfig.AuthCodeURL(a.oauthStateString, oauth2.AccessTypeOffline)
	c.Redirect(http.StatusTemporaryRedirect, url)
}

// GoogleCallbackHandler handles the callback from Google
func (a *AuthHandler) GoogleCallbackHandler(c *gin.Context) {
	state := c.Request.FormValue("state")
	if state != a.oauthStateString {
		merrors.Validation(c, "Invalid OAuth state")
		return
	}

	code := c.Request.FormValue("code")
	if code == "" {
		merrors.Validation(c, "Code not found")
		return
	}

	token, err := a.googleOauthConfig.Exchange(c, code)
	if err != nil {
		merrors.InternalServer(c, err.Error())
		return
	}

	client := a.googleOauthConfig.Client(c, token)
	response, err := client.Get("https://www.googleapis.com/oauth2/v2/userinfo")
	if err != nil {
		merrors.InternalServer(c, err.Error())
		return
	}
	defer response.Body.Close()

	userInfo, err := io.ReadAll(response.Body)
	if err != nil {
		merrors.InternalServer(c, err.Error())
		return
	}

	var googleUser map[string]interface{}
	err = json.Unmarshal(userInfo, &googleUser)
	if err != nil {
		merrors.InternalServer(c, err.Error())
		return
	}

	oauthID := googleUser["id"].(string)

	var userUUID uuid.UUID

	tx, err := a.db.Begin(c)
	if err != nil {
		merrors.InternalServer(c, err.Error())
		return
	}
	defer tx.Rollback(c)
	qtx := database.New(a.db).WithTx(tx)

	// Check if the user exists
	userUUID, err = qtx.GetUserByOAuthID(c, database.GetUserByOAuthIDParams{
		AuthType: "oauth",
		OauthID:  oauthID,
	})

	if errors.Is(err, pgx.ErrNoRows) {
		// If not, register a new user
		userUUID, err = qtx.CreateUser(c, database.CreateUserParams{
			AuthType: "oauth",
			OauthID:  oauthID,
		})
		var e *pgconn.PgError
		if errors.As(err, &e) && e.Code == pgerrcode.UniqueViolation {
			merrors.Validation(c, "User already exists with this OAuth ID!")
			return
		} else if err != nil {
			merrors.InternalServer(c, err.Error())
			return
		}
	}

	tok, err := generateToken(userUUID, 24*time.Hour, "authentication")
	if err != nil {
		merrors.InternalServer(c, "error in token generation")
		return
	}

	err = qtx.CreateNewToken(c, database.CreateNewTokenParams{
		Hash:     tok.Hash,
		UserUuid: tok.UserID,
		Expiry:   pgtype.Timestamptz{Time: tok.Expiry, Valid: true},
		Scope:    tok.Scope,
	})
	if err != nil {
		merrors.InternalServer(c, err.Error())
		return
	}

	err = tx.Commit(c)
	if err != nil {
		merrors.InternalServer(c, err.Error())
		return
	}

	queryVal := url.Values{
		"token": {tok.Plaintext},
	}
	reqUrl := a.frontendCallback + "?" + queryVal.Encode()
	c.Redirect(http.StatusTemporaryRedirect, reqUrl)

	// c.JSON(http.StatusOK, utils.BaseResponse{
	// 	Success:    true,
	// 	Message:    "OAuth user successfully authenticated",
	// 	Data:       tok,
	// 	StatusCode: http.StatusOK,
	// })
}

func (a *AuthHandler) RegisterOAuthUser(c *gin.Context) {
	var input struct {
		OAuthID string `json:"oauth_id" binding:"required"`
	}
	err := c.ShouldBindJSON(&input)
	if err != nil {
		merrors.Validation(c, err.Error())
		return
	}

	tx, err := a.db.Begin(c)
	if err != nil {
		merrors.InternalServer(c, err.Error())
		return
	}
	defer tx.Rollback(c)

	qtx := database.New(a.db).WithTx(tx)

	// Try to create a new user in the database
	_, err = qtx.CreateUser(c, database.CreateUserParams{
		AuthType: "oauth",
		OauthID:  input.OAuthID,
	})
	var e *pgconn.PgError
	if errors.As(err, &e) && e.Code == pgerrcode.UniqueViolation {
		merrors.Validation(c, "User already exists with this OAuth ID!")
		return
	} else if err != nil {
		merrors.InternalServer(c, err.Error())
		return
	}

	err = tx.Commit(c)
	if err != nil {
		merrors.InternalServer(c, err.Error())
		return
	}

	c.JSON(http.StatusOK, utils.BaseResponse{
		Success:    true,
		Message:    "OAuth user successfully registered",
		StatusCode: http.StatusOK,
	})
}

func (a *AuthHandler) GetUserByOAuthID(c *gin.Context) {
	var input struct {
		OAuthID string `json:"oauth_id" binding:"required"`
	}
	err := c.ShouldBindJSON(&input)
	if err != nil {
		merrors.Validation(c, err.Error())
		return
	}

	q := database.New(a.db)
	userUUID, err := q.GetUserByOAuthID(c, database.GetUserByOAuthIDParams{
		AuthType: "oauth",
		OauthID:  input.OAuthID,
	})
	if errors.Is(err, pgx.ErrNoRows) {
		merrors.NotFound(c, "User not found!")
		return
	} else if err != nil {
		merrors.InternalServer(c, err.Error())
		return
	}

	c.JSON(http.StatusOK, utils.BaseResponse{
		Success:    true,
		Message:    "OAuth user successfully retrieved",
		Data:       userUUID,
		StatusCode: http.StatusOK,
	})
}

func (a *AuthHandler) RegisterAnonymousUser(c *gin.Context) {
	// For anonymous auth, we generate a UUID and register it as a user.
	userUUID := uuid.New()

	tx, err := a.db.Begin(c)
	if err != nil {
		merrors.InternalServer(c, err.Error())
		return
	}
	defer tx.Rollback(c)

	qtx := database.New(a.db).WithTx(tx)

	// Try to create a new user in the database
	_, err = qtx.CreateUser(c, database.CreateUserParams{
		AuthType: "anonymous",
	})
	var e *pgconn.PgError
	if errors.As(err, &e) && e.Code == pgerrcode.UniqueViolation {
		merrors.Validation(c, "user already exists with this ID!")
		return
	} else if err != nil {
		merrors.InternalServer(c, err.Error())
		return
	}

	err = tx.Commit(c)
	if err != nil {
		merrors.InternalServer(c, err.Error())
		return
	}

	c.JSON(http.StatusOK, utils.BaseResponse{
		Success:    true,
		Message:    "Anonymous user successfully registered",
		Data:       userUUID,
		StatusCode: http.StatusOK,
	})
}

func (a *AuthHandler) GetUserByAnonymousID(c *gin.Context) {
	var input struct {
		UserID uuid.UUID `json:"user_id" binding:"required"`
	}
	err := c.ShouldBindJSON(&input)
	if err != nil {
		merrors.Validation(c, err.Error())
		return
	}

	q := database.New(a.db)
	userUUID, err := q.GetUserByOAuthID(c, database.GetUserByOAuthIDParams{
		AuthType: "anonymous",
		OauthID:  input.UserID.String(), // You might need to adjust this part based on your actual query setup.
	})
	if errors.Is(err, pgx.ErrNoRows) {
		merrors.NotFound(c, "user not found!")
		return
	} else if err != nil {
		merrors.InternalServer(c, err.Error())
		return
	}

	c.JSON(http.StatusOK, utils.BaseResponse{
		Success:    true,
		Message:    "Anonymous user successfully retrieved",
		Data:       userUUID,
		StatusCode: http.StatusOK,
	})
}

func (a *AuthHandler) LogoutHandler(c *gin.Context) {
	// Assuming you use a token-based authentication mechanism like JWT

	// Invalidate the token (You might need to remove the token from a store or mark it as invalid in the DB)
	u, _ := c.Get("user")
	user, _ := u.(*database.GetUserByTokenRow)
	if user == AnonymousUser {
		merrors.Unauthorized(c, "You are not logged in")
		return
	}

	// Example: remove token from the database
	err := a.invalidateToken(c, user.Uuid)
	if err != nil {
		merrors.InternalServer(c, "Error invalidating token")
		return
	}

	c.JSON(http.StatusOK, utils.BaseResponse{
		Success:    true,
		Message:    "Successfully logged out",
		StatusCode: http.StatusOK,
	})
}

func (a *AuthHandler) invalidateToken(ctx context.Context, userUUID uuid.UUID) error {
	// Create a new instance of database.Queries
	q := database.New(a.db)

	// Invalidate the token (e.g., delete it from the database)
	err := q.DeleteTokenForUser(ctx, userUUID)
	if err != nil {
		return err
	}

	return nil
}

/*

func (a *AuthHandler) RefreshTokenHandler(c *gin.Context) {
	var input struct {
		RefreshToken string `json:"refresh_token" binding:"required"`
	}
	err := c.ShouldBindJSON(&input)
	if err != nil {
		merrors.Validation(c, err.Error())
		return
	}

	// Validate and refresh the token
	newToken, err := a.refreshAccessToken(c, input.RefreshToken)
	if err != nil {
		merrors.Unauthorized(c, "Invalid or expired refresh token")
		return
	}

	c.JSON(http.StatusOK, utils.BaseResponse{
		Success:    true,
		Message:    "Token refreshed successfully",
		Data:       newToken,
		StatusCode: http.StatusOK,
	})
}

func (a *AuthHandler) refreshAccessToken(ctx context.Context, refreshToken string) (string, error) {
	// Create a new instance of database.Queries
	q := database.New(a.db)

	// Get the new access token using the refresh token
	newToken, err := q.GetNewAccessTokenByRefreshToken(ctx, refreshToken)
	if err != nil {
		return "", err
	}

	return newToken, nil
}

*/
