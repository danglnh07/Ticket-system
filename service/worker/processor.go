package worker

import (
	"context"
	"log/slog"

	"github.com/danglnh07/ticket-system/ticket-system/db"
	"github.com/danglnh07/ticket-system/ticket-system/service/mail"
	"github.com/hibiken/asynq"
)

// Task processor interface
type TaskProcessor interface {
	Start() error
	ProcessTaskSendVerifyEmail(ctx context.Context, task *asynq.Task) (err error)
}

// Redis task processor
type RedisTaskProcessor struct {
	// Asynq server
	server *asynq.Server

	// Dependencies
	queries     *db.Queries
	mailService mail.MailService

	// Logger for debugging
	logger *slog.Logger
}

// Constructor method for Redis task processor
func NewRedisTaskProcessor(
	redisOpts asynq.RedisClientOpt,
	queries *db.Queries,
	mailService mail.MailService,
	logger *slog.Logger,
) TaskProcessor {
	return &RedisTaskProcessor{
		server:      asynq.NewServer(redisOpts, asynq.Config{}),
		queries:     queries,
		mailService: mailService,
		logger:      logger,
	}
}

// Method to start the worker server
func (processor *RedisTaskProcessor) Start() error {
	mux := asynq.NewServeMux()

	mux.HandleFunc(SendVerifyEmail, processor.ProcessTaskSendVerifyEmail)

	return processor.server.Start(mux)
}
