package worker

import (
	"context"
	"log/slog"

	"github.com/danglnh07/ticket-system/db"
	"github.com/danglnh07/ticket-system/service/mail"
	"github.com/danglnh07/ticket-system/service/notify"
	"github.com/hibiken/asynq"
)

// Task processor interface
type TaskProcessor interface {
	Start() error
}

// Redis task processor
type RedisTaskProcessor struct {
	// Asynq server
	server *asynq.Server

	// Dependencies
	queries     *db.Queries
	mailService mail.MailService
	hub         *notify.Hub

	// Logger for debugging
	logger *slog.Logger
}

// Constructor method for Redis task processor
func NewRedisTaskProcessor(
	redisOpts asynq.RedisClientOpt,
	queries *db.Queries,
	mailService mail.MailService,
	hub *notify.Hub,
	logger *slog.Logger,
) TaskProcessor {
	return &RedisTaskProcessor{
		server:      asynq.NewServer(redisOpts, asynq.Config{}),
		queries:     queries,
		mailService: mailService,
		hub:         hub,
		logger:      logger,
	}
}

// Method to start the worker server
func (processor *RedisTaskProcessor) Start() error {
	mux := asynq.NewServeMux()

	mux.HandleFunc(SendVerifyEmail, func(ctx context.Context, t *asynq.Task) error {
		return processor.SendVerifyEmail(t.Payload())
	})
	mux.HandleFunc(SendNotification, func(ctx context.Context, t *asynq.Task) error {
		return processor.SendNotification(t.Payload())
	})

	return processor.server.Start(mux)
}
