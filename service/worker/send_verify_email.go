package worker

import (
	"bytes"
	"embed"
	"encoding/json"
	"html/template"
)

type SendVerifyEmailPayload struct {
	Email    string `json:"email"`
	Username string `json:"username"`
	Link     string `json:"link"`
}

const SendVerifyEmail = "send-verify-email"

//go:embed verify_email.html
var fs embed.FS

func (processor *RedisTaskProcessor) SendVerifyEmail(pl []byte) error {
	// Marshal the payload
	var payload SendVerifyEmailPayload
	if err := json.Unmarshal(pl, &payload); err != nil {
		return err
	}

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
	processor.logger.Info("Task process successfully", "task name", SendVerifyEmail)
	return nil
}
