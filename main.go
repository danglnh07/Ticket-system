// @title Ticket and event management API
// @version 1.0
// @description This is the API for ticket and event management system
// @host localhost:8080
// @BasePath /
// @securityDefinitions.apikey BearerAuth
// @in header
// @name Authorization
package main

import (
	"log/slog"
	"os"
	"strconv"
	"time"

	"github.com/danglnh07/ticket-system/api"
	"github.com/danglnh07/ticket-system/db"
	"github.com/danglnh07/ticket-system/service/mail"
	"github.com/danglnh07/ticket-system/service/notify"
	"github.com/danglnh07/ticket-system/service/payment"
	"github.com/danglnh07/ticket-system/service/scheduler"
	"github.com/danglnh07/ticket-system/service/security"
	"github.com/danglnh07/ticket-system/service/worker"
	"github.com/danglnh07/ticket-system/util"
	"github.com/hibiken/asynq"
	"github.com/redis/go-redis/v9"
)

func main() {
	// Initialize logger
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))

	// Load config
	if err := util.LoadConfig(".env.dev"); err != nil {
		logger.Error("Error loading configuration", "error", err)
	}

	// Connect to database, run database migration and seed initial data
	queries := db.NewQueries()
	if err := queries.ConnectDB(os.Getenv(util.DB_CONN)); err != nil {
		logger.Error("Error connecting to database", "error", err, "db_conn", os.Getenv(util.DB_CONN))
		os.Exit(1)
	}
	if err := queries.AutoMigration(); err != nil {
		logger.Error("Error running auto migration", "error", err)
		os.Exit(1)
	}

	// Connect to Redis Cache
	cacheOpts := redis.Options{
		Addr:     os.Getenv(util.REDIS_ADDRESS),
		Password: "", // No password Setup
		DB:       0,  // Default DB
	}
	queries.ConnectRedis(&cacheOpts)

	// Setup stripe secret key globally
	payment.InitStripe()

	// Run the cron
	s := scheduler.NewScheduler()

	// Add job
	// s.AddJob("@every 15m", func() { /*Add your job to be run here*/ })

	// Run the cron job in separate goroutine
	s.RunCronJobs()

	/* Create dependencies for server */

	// Create mail service
	mailService := mail.NewEmailService(os.Getenv(util.SYSTEM_EMAIL), os.Getenv(util.EMAIL_APP_PASSWORD))

	// Create JWT service
	tokenExpiration, err := strconv.Atoi(os.Getenv(util.TOKEN_EXPIRATION))
	if err != nil {
		logger.Error("Invalid value for token expiration. Fall to default value")
		tokenExpiration = 60
	}

	refreshTokenExpiration, err := strconv.Atoi(os.Getenv(util.REFRESH_TOKEN_EXPIRATION))
	if err != nil {
		logger.Error("Invalid value for refresh token expiration. Fall to default value")
		refreshTokenExpiration = 1440
	}

	jwtService := security.NewJWTService(
		[]byte(os.Getenv(util.SECRET_KEY)),
		time.Duration(tokenExpiration)*time.Minute,
		time.Duration(refreshTokenExpiration)*time.Minute)

	// Create the hub
	hub := notify.NewHub(logger)

	// Create distributor and start processor in the background
	opts := asynq.RedisClientOpt{Addr: os.Getenv(util.REDIS_ADDRESS)}
	distributor := worker.NewRedisTaskDistributor(opts, logger)
	for range 2 {
		go func() {
			err := StartBackgroundProcessor(opts, queries, mailService, hub, logger)
			if err != nil {
				logger.Error("Task failed", "error", err)
			}
		}()
	}

	// Create Google calendar
	calendar := notify.NewGoogleCalendar(
		os.Getenv(util.GOOGLE_CLIENT_ID),
		os.Getenv(util.GOOGLE_CLIENT_SECRET),
		45, 60,
	)

	// Create chatbot
	bot, err := notify.NewChatbot(os.Getenv(util.TELEGRAM_TOKEN))
	if err != nil {
		logger.Error("Failed to create Telegram bot", "error", err)
		os.Exit(1)
	}

	// Start server
	server := api.NewServer(queries, mailService, jwtService, distributor, calendar, hub, bot, logger)
	if err := server.Start(); err != nil {
		logger.Error("Failed to start server", "error", err)
		os.Exit(1)
	}
}

func StartBackgroundProcessor(
	redisOpts asynq.RedisClientOpt,
	queries *db.Queries,
	mailService mail.MailService,
	hub *notify.Hub,
	logger *slog.Logger,
) error {
	// Create the processor
	processor := worker.NewRedisTaskProcessor(redisOpts, queries, mailService, hub, logger)

	// Start process tasks
	return processor.Start()
}
