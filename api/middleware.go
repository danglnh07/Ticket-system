package api

import (
	"fmt"
	"net/http"
	"strconv"
	"strings"

	"github.com/danglnh07/ticket-system/db"
	"github.com/danglnh07/ticket-system/service/security"
	"github.com/gin-gonic/gin"
)

const claimsKey = "claims-key"

func (server *Server) AuthMiddleware() gin.HandlerFunc {
	return func(ctx *gin.Context) {
		// Get token from request header
		token := strings.TrimSpace(strings.TrimPrefix(ctx.Request.Header.Get("Authorization"), "Bearer"))

		// Verify token
		claims, err := server.jwtService.VerifyToken(token)
		if err != nil {
			ctx.AbortWithStatusJSON(http.StatusUnauthorized, ErrorResponse{err.Error()})
			return
		}

		// Check if token version valid
		var tokenVersion int
		rawToken, err := server.queries.GetCache(ctx, fmt.Sprintf("%d", claims.ID))

		if err == nil {
			tokenVersion, _ = strconv.Atoi(rawToken)
		} else {
			server.logger.Warn("AuthMiddleware", "error", err)
			server.logger.Info("AuthMiddleware: fallback to database")

			// Call to database
			result := server.queries.DB.
				Table("accounts").
				Where("id = ?", claims.ID).
				Select("token_version").
				Scan(&tokenVersion)
			if result.Error != nil {
				server.logger.Error("AuthMiddleware: failed to get token version", "error", err)
				ctx.AbortWithStatusJSON(http.StatusInternalServerError, ErrorResponse{"Internal server error"})
				return
			}
		}

		if claims.Version != tokenVersion {
			ctx.AbortWithStatusJSON(http.StatusUnauthorized, ErrorResponse{"token version not match"})
			return
		}

		// If match, put the claims to context before forward to the next handler
		ctx.Set(claimsKey, claims)
		ctx.Next()
	}
}

func (server *Server) CORSMiddleware() gin.HandlerFunc {
	return func(ctx *gin.Context) {
		ctx.Header("Access-Control-Allow-Origin", "*")
		ctx.Header("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
		ctx.Header("Access-Control-Allow-Headers", "Content-Type, Authorization, X-Requested-With")

		// Handle preflight and return immediately so Gin doesn't respond 404 for OPTIONS
		if ctx.Request.Method == http.MethodOptions {
			ctx.AbortWithStatus(http.StatusOK)
			return
		}

		ctx.Next()
	}
}

func (server *Server) AuthorizeMiddleware(role db.Role) gin.HandlerFunc {
	return func(ctx *gin.Context) {
		// Get role from claims
		claims, _ := ctx.Get(claimsKey)
		rl := claims.(*security.CustomClaims).Role
		if rl != role {
			ctx.AbortWithStatusJSON(http.StatusUnauthorized, ErrorResponse{"You have no authorization to perform this action"})
			return
		}
		ctx.Next()
	}
}

func (server *Server) RateLimitMiddleware() gin.HandlerFunc {
	return func(ctx *gin.Context) {}
}
