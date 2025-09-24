package util

import (
	"os"
	"strconv"
	"time"

	"github.com/joho/godotenv"
)

type Config struct {
	// Database config
	DBConn string

	// Redis config
	RedisAddr string

	// Email config
	SMTPHost    string
	SMTPPort    string
	Email       string
	AppPassword string

	// Security config.
	SecretKey              []byte
	TokenExpiration        time.Duration
	RefreshTokenExpiration time.Duration

	// OAuth2 config
	GoogleClientID     string
	GoogleClientSecret string

	// Rate limiting config
	MaxRequest int
	RefillRate time.Duration

	// Stripe API key
	StripePublishableKey string
	StripeSecretKey      string
	StripeWebhookSecret  string
}

func LoadConfig(path string) *Config {
	err := godotenv.Load(path)
	if err != nil {
		return &Config{
			DBConn:                 os.Getenv("DB_CONN"),
			RedisAddr:              os.Getenv("REDIS_ADDRESS"),
			SMTPHost:               "smtp.gmail.com",
			SMTPPort:               "587",
			Email:                  os.Getenv("EMAIL"),
			AppPassword:            os.Getenv("APP_PASSWORD"),
			SecretKey:              []byte(os.Getenv("SECRET_KEY")),
			TokenExpiration:        time.Hour,
			RefreshTokenExpiration: time.Hour * 24,
			GoogleClientID:         os.Getenv("GOOGLE_CLIENT_ID"),
			GoogleClientSecret:     os.Getenv("GOOGLE_CLIENT_SECRET"),
			MaxRequest:             100,
			RefillRate:             time.Second * 10,
			StripePublishableKey:   os.Getenv("STRIPE_PUBLISHABLE_KEY"),
			StripeSecretKey:        os.Getenv("STRIPE_SECRET_KEY"),
			StripeWebhookSecret:    os.Getenv("STRIPE_WEBHOOK_SECRET"),
		}
	}

	// Try get and parse data
	tokenExpiration, err := strconv.Atoi(os.Getenv("TOKEN_EXPIRATION"))
	if err != nil {
		// Fallback to default value (60 minutes)
		tokenExpiration = 60
	}

	refreshTokenExpiration, err := strconv.Atoi(os.Getenv("REFRESH_TOKEN_EXPIRATION"))
	if err != nil {
		// Fallback to default value (1440 minutes = 24 hours)
		refreshTokenExpiration = 1440
	}

	maxRequest, err := strconv.Atoi(os.Getenv("MAX_REQUEST"))
	if err != nil {
		maxRequest = 100
	}

	refillRate, err := strconv.Atoi(os.Getenv("REFILL_RATE"))
	if err != nil {
		// Fallback to default value (10 seconds)
		refillRate = 10
	}

	return &Config{
		DBConn:                 os.Getenv("DB_CONN"),
		RedisAddr:              os.Getenv("REDIS_ADDRESS"),
		SMTPHost:               os.Getenv("SMTP_HOST"),
		SMTPPort:               os.Getenv("SMTP_PORT"),
		Email:                  os.Getenv("EMAIL"),
		AppPassword:            os.Getenv("APP_PASSWORD"),
		SecretKey:              []byte(os.Getenv("SECRET_KEY")),
		TokenExpiration:        time.Minute * time.Duration(tokenExpiration),
		RefreshTokenExpiration: time.Minute * time.Duration(refreshTokenExpiration),
		GoogleClientID:         os.Getenv("GOOGLE_CLIENT_ID"),
		GoogleClientSecret:     os.Getenv("GOOGLE_CLIENT_SECRET"),
		MaxRequest:             maxRequest,
		RefillRate:             time.Second * time.Duration(refillRate),
		StripePublishableKey:   os.Getenv("STRIPE_PUBLISHABLE_KEY"),
		StripeSecretKey:        os.Getenv("STRIPE_SECRET_KEY"),
		StripeWebhookSecret:    os.Getenv("STRIPE_WEBHOOK_SECRET"),
	}
}
