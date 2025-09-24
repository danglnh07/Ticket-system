package worker

import (
	"context"
	"encoding/json"
	"log/slog"

	"github.com/danglnh07/ticket-system/db"
	"github.com/danglnh07/ticket-system/service/mail"
	"github.com/danglnh07/ticket-system/service/notify"
	"github.com/hibiken/asynq"
)

// Task processor interface
type TaskProcessor interface {
	Start() error
	ProcessTask(ctx context.Context, task *asynq.Task, handle func(payload any) error) error
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
		return processor.ProcessTask(ctx, t, processor.SendVerifyEmail)
	})
	mux.HandleFunc(SendNotification, func(ctx context.Context, t *asynq.Task) error {
		return processor.ProcessTask(ctx, t, processor.SendNotification)
	})

	return processor.server.Start(mux)
}

func (processor *RedisTaskProcessor) ProcessTask(
	ctx context.Context,
	task *asynq.Task,
	handle func(payload any) error,
) error {
	// Unmarshal the payload
	var payload SendVerifyEmailPayload
	if err := json.Unmarshal(task.Payload(), &payload); err != nil {
		return err
	}

	return handle(payload)
}
