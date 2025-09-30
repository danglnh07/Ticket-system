package api

import (
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"os"
	"strings"

	"github.com/danglnh07/ticket-system/db"
	"github.com/danglnh07/ticket-system/service/security"
	"github.com/danglnh07/ticket-system/util"
	"github.com/gin-gonic/gin"
	"golang.org/x/oauth2"
	"golang.org/x/oauth2/google"
	"gorm.io/gorm"
)

type OAuthState struct {
	Role     db.Role          `json:"role"`
	Provider db.OauthProvider `json:"provider"`
}

func NewGoogleOAuth() *oauth2.Config {
	return &oauth2.Config{
		RedirectURL:  fmt.Sprintf("%s/oauth2/callback", os.Getenv(util.DOMAIN)),
		ClientID:     os.Getenv(util.GOOGLE_CLIENT_ID),
		ClientSecret: os.Getenv(util.GOOGLE_CLIENT_SECRET),
		Scopes: []string{
			"https://www.googleapis.com/auth/userinfo.email",
			"https://www.googleapis.com/auth/userinfo.profile",
			"https://www.googleapis.com/auth/calendar.events",
		},
		Endpoint: google.Endpoint,
	}

}

func (server *Server) HandleOAuth(ctx *gin.Context) {
	// Get role from request parameter
	role := db.Role(ctx.Query("role"))
	if role != db.User && role != db.Admin && role != db.SupportedStaff && role != db.Organiser {
		ctx.JSON(http.StatusBadRequest, ErrorResponse{"Invalid role"})
		return
	}

	// Get which provider user want to login
	provider := db.OauthProvider(ctx.Query("provider"))
	if provider != db.Google {
		ctx.JSON(http.StatusBadRequest, ErrorResponse{"Unsupported OAuth provider"})
		return
	}

	state := OAuthState{Role: role, Provider: provider}
	data, err := json.Marshal(state)
	if err != nil {
		server.logger.Error("GET /api/auth/oauth?role=ROLE: failed to marshal state", "error", err)
		ctx.JSON(http.StatusInternalServerError, ErrorResponse{"Internal server error"})
		return
	}

	url := server.oauthConfigs[provider].AuthCodeURL(string(data), oauth2.AccessTypeOffline, oauth2.ApprovalForce)
	ctx.Redirect(http.StatusTemporaryRedirect, url)
}

func (server *Server) HandleCallback(ctx *gin.Context) {
	// Get the state, validate and act based on each OAuth provider
	rawState := strings.TrimSpace(ctx.Query("state"))
	if rawState == "" {
		ctx.JSON(http.StatusBadRequest, ErrorResponse{"Invalide state"})
		return
	}

	var state OAuthState
	if err := json.Unmarshal([]byte(rawState), &state); err != nil {
		ctx.JSON(http.StatusBadRequest, ErrorResponse{"Invalid state"})
		return
	}

	// Get the code return by OAuth provider and exchange for token
	code := strings.TrimSpace(ctx.Query("code"))
	if code == "" {
		ctx.JSON(http.StatusBadRequest, ErrorResponse{"Code cannot be empty"})
		return
	}

	// Get user data from OAuth provider
	var account db.Account
	switch state.Provider {
	case db.Google:
		// Echange code for token
		token, err := server.oauthConfigs[state.Provider].Exchange(ctx, code)
		if err != nil {
			server.logger.Error("GET /oauth2/callback: failed to exchange code for token", "error", err)
			ctx.JSON(http.StatusInternalServerError, ErrorResponse{"Internal server error"})
			return
		}

		// Fetch user data
		client := server.oauthConfigs[state.Provider].Client(ctx, token)
		resp, err := client.Get("https://www.googleapis.com/oauth2/v2/userinfo")
		if err != nil {
			server.logger.Error("GET /oauth2/callback: failed to make request to googleapis/userinfo", "error", err)
			ctx.JSON(http.StatusInternalServerError, ErrorResponse{"Internal server error"})
			return
		}

		var data map[string]any
		if err = json.NewDecoder(resp.Body).Decode(&data); err != nil {
			server.logger.Error("GET /oauth2/callback: failed to marshal response body", "error", err)
			ctx.JSON(http.StatusInternalServerError, ErrorResponse{"Internal server error"})
			return
		}

		account.Username = data["name"].(string)
		account.Email = data["email"].(string)
		account.Avatar = data["picture"].(string)
		account.Role = state.Role
		account.Status = db.Active
		account.OauthProvider = sql.NullString{String: string(db.Google), Valid: true}
		account.OauthProviderID = sql.NullString{String: data["id"].(string), Valid: true}
		account.OauthAccessToken = sql.NullString{String: token.AccessToken, Valid: true}
		account.OauthRefreshToken = sql.NullString{String: token.RefreshToken, Valid: true}
	case db.Telegram:
		// Implement Telegram logic here
	default:
		ctx.JSON(http.StatusBadRequest, ErrorResponse{"Invalid state"})
		return
	}

	// Check if this account has been register
	result := server.queries.DB.
		Where("oauth_provider = ? AND oauth_provider_id = ?", account.OauthProvider, account.OauthProviderID).
		First(&account)
	if result.Error != nil {
		if errors.Is(result.Error, gorm.ErrRecordNotFound) {
			// If not found, then we create a new account
			result = server.queries.DB.Create(&account)
			if result.Error != nil {
				server.logger.Error("GET /oauth2/callback: failed to create new account", "error", result.Error)
				ctx.JSON(http.StatusInternalServerError, ErrorResponse{"Internal server error"})
				return
			}

			// NOT return here
		} else {
			// Other database error
			server.logger.Error("GET /oauth2/callback: failed to get account data", "error", result.Error)
			ctx.JSON(http.StatusInternalServerError, ErrorResponse{"Internal server error"})
			return
		}
	}

	// Create tokens and return data back to client
	// Generate tokens
	accessToken, err := server.jwtService.CreateToken(account.ID, account.Role, security.AccessToken, int(account.TokenVersion))
	if err != nil {
		server.logger.Error("GET /oauth2/callback: failed to create access token", "error", err)
		ctx.JSON(http.StatusInternalServerError, ErrorResponse{"Internal server error"})
		return
	}

	refreshToken, err := server.jwtService.CreateToken(account.ID, account.Role, security.RefreshToken, int(account.TokenVersion))
	if err != nil {
		server.logger.Error("GET /oauth2/callback: failed to create refresh token")
		ctx.JSON(http.StatusInternalServerError, ErrorResponse{"Internal server error"})
		return
	}

	// Return data back to client
	ctx.JSON(http.StatusOK, AuthResponse{
		ID:           account.ID,
		Username:     account.Username,
		Email:        account.Email,
		Avatar:       account.Avatar,
		Point:        account.Point,
		AccessToken:  accessToken,
		RefreshToken: refreshToken,
	})
}
