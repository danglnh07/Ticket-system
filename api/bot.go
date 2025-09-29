package api

import (
	"fmt"

	"github.com/danglnh07/ticket-system/service/notify"
	"github.com/gin-gonic/gin"
)

type Message struct {
	ID   int `json:"message_id"`
	From struct {
		ID           int    `json:"id"`
		IsBot        bool   `json:"is_bot"`
		FirstName    string `json:"first_name"`
		LanguageCode string `json:"language_code"`
	} `json:"from"`
	Chat struct {
		ID        int    `json:"id"`
		FirstName string `json:"first_name"`
		Type      string `json:"type"`
	} `json:"chat"`
	Date uint64 `json:"date"`
	Text string `json:"text"`
}

type Update struct {
	ID      int     `json:"update_id"`
	Message Message `json:"message"`
}

func (server *Server) BotWebhook(ctx *gin.Context) {
	server.logger.Info("Webhook called")

	var update Update
	if err := ctx.ShouldBindJSON(&update); err != nil {
		server.logger.Warn("/api/bot/webhook: failed to parse update", "error", err)
		return
	}

	server.logger.Info("Receive update", "chat-id", update.Message.Chat.ID, "message", update.Message.Text)

	// Simple echo logic
	err := server.bot.SendMessage(notify.MessagePayload{
		ChatID: update.Message.Chat.ID,
		Text:   fmt.Sprintf("You have sent a text: **%s**", update.Message.Text),
	})
	if err != nil {
		server.logger.Error("/api/bot/webhook: failed to echo message back", "error", err)
		return
	}

	server.logger.Info("Echo message successfully")
}
