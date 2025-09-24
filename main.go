package main

import (
	"log/slog"
	"os"

	"github.com/danglnh07/ticket-system/ticket-system/api"
	"github.com/danglnh07/ticket-system/ticket-system/db"
	"github.com/danglnh07/ticket-system/ticket-system/service/mail"
	"github.com/danglnh07/ticket-system/ticket-system/service/notify"
	"github.com/danglnh07/ticket-system/ticket-system/service/payment"
	"github.com/danglnh07/ticket-system/ticket-system/service/scheduler"
	"github.com/danglnh07/ticket-system/ticket-system/service/security"
	"github.com/danglnh07/ticket-system/ticket-system/service/worker"
	"github.com/danglnh07/ticket-system/ticket-system/util"
	"github.com/hibiken/asynq"
)

func main() {
	// Initialize logger
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))

	// Load config
	config := util.LoadConfig(".env")

	// Connect to database and run database migration
	queries := db.NewQueries()
	if err := queries.ConnectDB(config.DBConn); err != nil {
		logger.Error("Error connecting to database", "error", err)
		os.Exit(1)
	}
	if err := queries.AutoMigration(); err != nil {
		logger.Error("Error running auto migration", "error", err)
		os.Exit(1)
	}

	// Setup stripe secret key globally
	payment.InitStripe(config.StripeSecretKey)

	// Run the cron
	s := scheduler.NewScheduler()

	// Add job
	// s.AddJob("@every 15m", func() { /*Add your job to be run here*/ })

	// Run the cron job in separate goroutine
	s.RunCronJobs()

	// Create dependencies for server
	mailService := mail.NewEmailService(config)
	jwtService := security.NewJWTService(config)
	distributor := worker.NewRedisTaskDistributor(asynq.RedisClientOpt{
		Addr: config.RedisAddr,
	}, logger)
	hub := notify.NewHub(logger)

	// Start the background server in separate goroutine (since it's will block the main thread)
	go StartBackgroundProcessor(asynq.RedisClientOpt{Addr: config.RedisAddr}, queries, mailService, hub, logger)

	// Start server
	server := api.NewServer(mailService, jwtService, distributor, hub, config, logger)
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
