package worker

import (
	"bytes"
	"context"
	"embed"
	"encoding/json"
	"html/template"

	"github.com/hibiken/asynq"
)

type SendVerifyEmailPayload struct {
	Email    string `json:"email"`
	Username string `json:"username"`
	Link     string `json:"link"`
}

const SendVerifyEmail = "send-welcome-email"

//go:embed verify_email.html
var fs embed.FS

func (distributor *RedisTaskDistributor) DistributeTaskSendVerifyEmail(
	ctx context.Context,
	payload SendVerifyEmailPayload,
	opts ...asynq.Option,
) (err error) {
	// Marshal payload
	data, err := json.Marshal(payload)
	if err != nil {
		return err
	}

	// Create new task
	task := asynq.NewTask(SendVerifyEmail, data, opts...)

	// Send task to Redis queue
	info, err := distributor.client.EnqueueContext(ctx, task)
	if err != nil {
		return err
	}

	// Log task info
	distributor.logger.Info("Task info", "task_name", SendVerifyEmail, "queue", info.Queue, "max_retry", info.MaxRetry)

	return nil
}

func (processor *RedisTaskProcessor) ProcessTaskSendVerifyEmail(ctx context.Context, task *asynq.Task) (err error) {
	// Unmarshal the payload
	var payload SendVerifyEmailPayload
	err = json.Unmarshal(task.Payload(), &payload)
	if err != nil {
		return err
	}

	// Process task -> decoupling method for unit test
	processor.SendVerifyEmail(payload)

	return nil
}

func (processor *RedisTaskProcessor) SendVerifyEmail(payload SendVerifyEmailPayload) error {
	// Prepare the HTML email body
	tmpl, err := template.ParseFS(fs, "verify_email.html")
	if err != nil {
		return err
	}
	var buffer bytes.Buffer
	if err = tmpl.Execute(&buffer, payload); err != nil {
		return err
	}

	// Send email
	err = processor.mailService.SendEmail(payload.Email, "Welcome to Ticket - Verify your account", buffer.String())
	if err != nil {
		return err
	}
	return nil
}
