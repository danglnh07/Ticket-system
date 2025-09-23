package worker

import (
	"log/slog"
	"os"
	"testing"

	"github.com/danglnh07/ticket-system/ticket-system/db"
	"github.com/danglnh07/ticket-system/ticket-system/service/mail"
	"github.com/hibiken/asynq"
)

var (
	queries     *db.Queries
	mailService mail.MailService
	logger      = slog.New(slog.NewTextHandler(os.Stdout, nil))
	distributor TaskDistributor
	processor   TaskProcessor
)

type MockMailService struct {
	logger *slog.Logger
}

func (mock *MockMailService) SendEmail(to, subject, body string) error {
	logger.Info("Email sent successfully", "to", to, "subject", subject, "body", body)
	return nil
}

func TestMain(m *testing.M) {
	// Create mock mail service
	mailService = &MockMailService{logger}

	// Create, connect and run database migration
	conn := os.Getenv("DB_CONN")
	queries = db.NewQueries()
	if err := queries.ConnectDB(conn); err != nil {
		logger.Error("test worker: failed to connect to database", "error", err)
		os.Exit(1)
	}
	if err := queries.AutoMigration(); err != nil {
		logger.Error("test worker: failed to run auto migration", "error", err)
		os.Exit(1)
	}

	// Connect to Redis
	redisOpt := asynq.RedisClientOpt{
		Addr: os.Getenv("REDIS_ADDRESS"),
	}
	distributor = NewRedisTaskDistributor(redisOpt, logger)

	// Run the task processor in goroutine (since the asynq.Start will block the main thread)
	go StartBackgroundProcessor(redisOpt, queries, mailService, logger)

	os.Exit(m.Run())
}

func StartBackgroundProcessor(
	redisOpts asynq.RedisClientOpt,
	queries *db.Queries,
	mailService mail.MailService,
	logger *slog.Logger,
) error {
	processor = NewRedisTaskProcessor(redisOpts, queries, mailService, logger)
	return processor.Start()
}
