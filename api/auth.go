package api

import (
	"database/sql"
	"errors"
	"fmt"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/danglnh07/ticket-system/db"
	"github.com/danglnh07/ticket-system/service/security"
	"github.com/danglnh07/ticket-system/service/worker"
	"github.com/danglnh07/ticket-system/util"
	"github.com/gin-gonic/gin"
	"github.com/hibiken/asynq"
	"gorm.io/gorm"
)

type RegisterRequest struct {
	Username string `json:"username" binding:"required"`
	Email    string `json:"email" binding:"required,email"`
	Password string `json:"password" binding:"required"`
	Role     string `json:"role" binding:"required"`
}

type AuthResponse struct {
	ID           uint   `json:"id"`
	Username     string `json:"username"`
	Email        string `json:"email"`
	Avatar       string `json:"avatar"`
	Rank         string `json:"rank"`
	Point        uint   `json:"point"`
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
}

func (server *Server) Register(ctx *gin.Context) {
	// Get request body and validate
	var req RegisterRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		server.logger.Warn("POST /api/auth/register: failed to parse request body", "error", err)
		ctx.JSON(http.StatusBadRequest, ErrorResponse{"Invalid request body"})
		return
	}

	// Check if role is valid
	role := db.Role(req.Role)
	if role != db.Admin && role != db.User && role != db.Organiser && role != db.SupportedStaff {
		ctx.JSON(http.StatusBadRequest, ErrorResponse{"Invalid value for role!"})
		return
	}

	// Check if this username, email exists
	var account db.Account
	result := server.queries.DB.
		Where("username = ? OR email = ?", req.Username, req.Email).
		First(&account)
	if result.Error == nil {
		// If username or email exists
		resp := ErrorResponse{}
		if req.Email == account.Email {
			resp.Message = "This email has been registered. Please login instead"
		} else if req.Username == account.Username {
			resp.Message = "This username has been taken"
		}

		ctx.JSON(http.StatusBadRequest, resp)
		return
	} else if result.Error != nil && !errors.Is(result.Error, gorm.ErrRecordNotFound) {
		server.logger.Error("POST /api/auth/register: failed to get user data", "error", result.Error)
		ctx.JSON(http.StatusInternalServerError, ErrorResponse{"Internal server error"})
		return
	}

	// Hash password
	hashed, err := security.BcryptHash(req.Password)
	if err != nil {
		server.logger.Error("POST /api/auth/register: failed to user password", "error", err)
		ctx.JSON(http.StatusInternalServerError, ErrorResponse{"Internal server error"})
		return
	}

	// Insert into database
	account.Username = req.Username
	account.Email = req.Email
	account.Password = sql.NullString{String: hashed, Valid: true}
	account.Avatar = "default_avatar.png"
	account.Role = role
	account.Status = db.Inactive
	result = server.queries.DB.Create(&account)
	if result.Error != nil {
		server.logger.Error("POST /api/auth/register: failed to create new account", "error", result.Error)
		ctx.JSON(http.StatusInternalServerError, ErrorResponse{"Internal server error"})
		return
	}

	// Create background task: send verification email to client's email
	token := security.Encode(fmt.Sprintf("%d.%d.%s", account.ID, time.Now().UnixNano(), security.Hash(os.Getenv(util.SECRET_KEY))))
	link := fmt.Sprintf("%s/api/auth/verify?token=%s", os.Getenv(os.Getenv(util.DOMAIN)), token)
	err = server.distributor.DistributeTask(ctx, worker.SendVerifyEmail, worker.SendVerifyEmailPayload{
		Email:    account.Email,
		Username: account.Username,
		Link:     link,
	}, asynq.MaxRetry(1))
	if err != nil {
		server.logger.Error("POST /api/auth/register: failed to distribute send verification mail task", "error", err)
		ctx.JSON(http.StatusInternalServerError, ErrorResponse{"Account created successfully, but failed to send verification email"})
		return
	}

	// Return result back to client
	ctx.JSON(http.StatusCreated, gin.H{
		"message": "Account created successfully, please activate your account through the email we have sent",
	})
}

func (server *Server) VerifyAccount(ctx *gin.Context) {
	// Get the token
	token := strings.TrimSpace(ctx.Query("token"))
	if token == "" {
		server.logger.Warn("GET /api/auth/verify: empty token")
		ctx.JSON(http.StatusBadRequest, ErrorResponse{"Invalid link"})
		return
	}

	// Decode token
	decodeData, err := security.Decode(token)
	if err != nil {
		server.logger.Warn("GET /api/auth/verify: failed to decode token", "error", err)
		ctx.JSON(http.StatusBadRequest, ErrorResponse{"Invalid link"})
		return
	}
	data := strings.Split(decodeData, ".")
	if len(data) != 3 {
		server.logger.Warn("GET /api/auth/verify: token portion is not 3", "len", len(data))
		ctx.JSON(http.StatusBadRequest, ErrorResponse{"Invalid link"})
		return
	}

	// Check if token is valid by checking the hash
	if security.Hash(os.Getenv(util.SECRET_KEY)) != data[2] {
		server.logger.Warn("GET /api/auth/verify: signature not match")
		ctx.JSON(http.StatusBadRequest, ErrorResponse{"Invalid link"})
		return
	}

	// Check if the token has expired
	issueAt, err := strconv.ParseInt(data[1], 10, 64)
	if err != nil {
		server.logger.Warn("GET /api/auth/verify: failed to parse timestampt", "error", err)
		ctx.JSON(http.StatusBadRequest, ErrorResponse{"Invalid link"})
		return
	}

	if time.Unix(0, issueAt).Add(24 * time.Hour).Before(time.Now()) {
		server.logger.Warn("GET /api/auth/verify: token expired")
		ctx.JSON(http.StatusBadRequest, ErrorResponse{"Verification link expired"})
		return
	}

	// Update account status
	result := server.queries.DB.Table("accounts").Where("id = ?", data[0]).Update("status", db.Active)
	if result.Error != nil {
		// If ID not match any record
		if errors.Is(result.Error, gorm.ErrRecordNotFound) {
			ctx.JSON(http.StatusBadRequest, ErrorResponse{"Invalid link"})
			return
		}

		// Other database error
		server.logger.Error("GET /api/auth/verify: failed to update account status", "error", err)
		ctx.JSON(http.StatusInternalServerError, ErrorResponse{"Internal server error"})
		return
	}

	// Return result to client
	ctx.JSON(http.StatusOK, gin.H{"message": "Account activate successfully! You can login to start our service now"})
}

type LoginRequest struct {
	Username string `json:"username"`
	Email    string `json:"email"`
	Password string `json:"password" binding:"required"`
}

func (server *Server) Login(ctx *gin.Context) {
	// Get request body and validate
	var req LoginRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		server.logger.Warn("POST /api/auth/login: failed to parse request body", "error", err)
		ctx.JSON(http.StatusBadRequest, ErrorResponse{"Invalid request body"})
		return
	}

	// Check if at least username or email exists
	if req.Username == "" && req.Email == "" {
		ctx.JSON(http.StatusBadRequest, ErrorResponse{"Invalid request body! Provide at least username or email"})
		return
	}

	// Get account by either username or email
	var account db.Account
	result := server.queries.DB.Where("username = ? OR email = ?", req.Username, req.Email).First(&account)
	if result.Error != nil {
		// If not match
		if errors.Is(result.Error, gorm.ErrRecordNotFound) {
			ctx.JSON(http.StatusBadRequest, ErrorResponse{"Incorrect login credential"})
			return
		}

		// Other database error
		server.logger.Error("POST /api/auth/login: failed to get account", "error", result.Error)
		ctx.JSON(http.StatusInternalServerError, ErrorResponse{"Internal server error"})
		return
	}

	// Check if account is activated
	if account.Status != db.Active {
		ctx.JSON(http.StatusBadRequest, ErrorResponse{"Account not active"})
		return
	}

	// Compare password
	if !security.BcryptCompare(account.Password.String, req.Password) {
		ctx.JSON(http.StatusBadRequest, ErrorResponse{"Incorrect login credential"})
		return
	}

	// Generate tokens
	accessToken, err := server.jwtService.CreateToken(account.ID, account.Role, security.AccessToken, int(account.TokenVersion))
	if err != nil {
		server.logger.Error("POST /api/auth/login: failed to create access token", "error", err)
		ctx.JSON(http.StatusInternalServerError, ErrorResponse{"Internal server error"})
		return
	}

	refreshToken, err := server.jwtService.CreateToken(account.ID, account.Role, security.RefreshToken, int(account.TokenVersion))
	if err != nil {
		server.logger.Error("POST /api/auth/login: failed to create refresh token")
		ctx.JSON(http.StatusInternalServerError, ErrorResponse{"Internal server error"})
		return
	}

	// Add access token into cache
	server.queries.SetCache(ctx, fmt.Sprintf("%d", account.ID), accessToken, time.Hour)

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
