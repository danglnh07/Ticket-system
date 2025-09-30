package security

import (
	"fmt"
	"time"

	"github.com/danglnh07/ticket-system/db"
	"github.com/golang-jwt/jwt/v5"
)

type JWTService struct {
	secretKey              []byte
	TokenExpiration        time.Duration
	RefreshTokenExpiration time.Duration
}

type TokenType string

const (
	Issuer = "ticket-system"

	AccessToken  TokenType = "access-token"
	RefreshToken TokenType = "refresh-token"
)

type CustomClaims struct {
	ID                   uint      `json:"id"`
	Role                 db.Role   `json:"role"`
	TokenType            TokenType `json:"token_type"`
	Version              int       `json:"version"`
	jwt.RegisteredClaims           // Embed the JWT Registered claims
}

func NewJWTService(secretKey []byte, tokenExpiration, refreshTokenExpiration time.Duration) *JWTService {
	return &JWTService{
		secretKey:              secretKey,
		TokenExpiration:        tokenExpiration,
		RefreshTokenExpiration: refreshTokenExpiration,
	}
}

func (service *JWTService) CreateToken(id uint, role db.Role, tokenType TokenType, version int) (string, error) {
	// Check token type and decide expiration time based on type
	var expiration time.Duration
	switch tokenType {
	case AccessToken:
		expiration = service.TokenExpiration
	case RefreshToken:
		expiration = service.RefreshTokenExpiration
	default:
		return "", fmt.Errorf("invalid token type")
	}

	// Create custom JWT claim
	claims := CustomClaims{
		ID:        id,
		Role:      role,
		TokenType: tokenType,
		Version:   version,
		RegisteredClaims: jwt.RegisteredClaims{
			Issuer:    Issuer,                                         // Who issue this token
			Subject:   fmt.Sprintf("%d", id),                          // Whom the token is about
			IssuedAt:  jwt.NewNumericDate(time.Now()),                 // When the token is created
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(expiration)), // When the token is expired
		},
	}

	// Generate token
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)

	// Sign token
	tokenStr, err := token.SignedString(service.secretKey)
	if err != nil {
		return "", err
	}

	return tokenStr, nil
}

func (service *JWTService) VerifyToken(signedToken string) (*CustomClaims, error) {
	// Use custom parser with deley to 30 secs
	parser := jwt.NewParser(jwt.WithLeeway(30 * time.Second))

	// Parse token
	parsedToken, err := parser.ParseWithClaims(signedToken, &CustomClaims{}, func(token *jwt.Token) (any, error) {
		// Check for signing method to avoid [alg: none] trick
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
		}
		return service.secretKey, nil
	})

	// Check if token parsing success
	if err != nil {
		return nil, err
	}

	// Extract claims from token
	claims, ok := parsedToken.Claims.(*CustomClaims)
	if !(ok && parsedToken.Valid) {
		return nil, jwt.ErrTokenInvalidClaims
	}

	// Check if this is the correct issuer
	if claims.Issuer != Issuer {
		return nil, fmt.Errorf("invalid issuer: %s", claims.Issuer)
	}

	// Check if the token type is correct
	if claims.TokenType != AccessToken && claims.TokenType != RefreshToken {
		return nil, fmt.Errorf("invalid token type: %s", claims.TokenType)
	}

	// Check if the role is role we define in database
	role := db.Role(claims.Role)
	if role != db.Admin && role != db.Organiser && role != db.SupportedStaff && role != db.User {
		return nil, fmt.Errorf("invalid role: %s", role)
	}

	return claims, nil
}
