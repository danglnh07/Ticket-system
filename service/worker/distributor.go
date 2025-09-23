package worker

import (
	"context"
	"encoding/json"
	"log/slog"

	"github.com/hibiken/asynq"
)

// Task distributor interface
type TaskDistributor interface {
	DistributeTask(ctx context.Context, taskName string, payload any, opts ...asynq.Option) error
}

// Redis task distributor
type RedisTaskDistributor struct {
	client *asynq.Client
	logger *slog.Logger
}

// Constructor method for Redis task distributor
func NewRedisTaskDistributor(redisOpt asynq.RedisClientOpt, logger *slog.Logger) TaskDistributor {
	client := asynq.NewClient(redisOpt)
	return &RedisTaskDistributor{
		client: client,
		logger: logger,
	}
}

func (distributor *RedisTaskDistributor) DistributeTask(
	ctx context.Context,
	taskName string,
	payload any,
	opts ...asynq.Option,
) error {
	// Marshal payload
	data, err := json.Marshal(payload)
	if err != nil {
		return err
	}

	// Create new task
	task := asynq.NewTask(taskName, data, opts...)

	// Send task to Redis queue
	info, err := distributor.client.EnqueueContext(ctx, task)
	if err != nil {
		return err
	}

	// Log task info
	distributor.logger.Info("Task info", "task_name", taskName, "queue", info.Queue, "max_retry", info.MaxRetry)

	return nil
}
