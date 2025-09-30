package notify

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"

	"github.com/danglnh07/ticket-system/util"
)

/*
 * Telegram bot API docs: https://core.telegram.org/bots/api
 */

type Chatbot struct {
	server  string
	webhook string
}

func NewChatbot(token string) (*Chatbot, error) {
	bot := &Chatbot{
		server:  "https://api.telegram.org/bot" + token,
		webhook: fmt.Sprintf("%s/api/bot/webhook", os.Getenv(util.DOMAIN)),
	}

	/*
	 * Since Telegram only accept one webhook for each chatbot register, and webhook can
	 * only be assign if none exists:
	 * 1. We get the webhook using getWebhookInfo
	 * 2.1. If the webhook match the current DOMAIN, do nothing
	 * 2.2. If the DOMAIN provide is different, we delete the webhook, and set the new one
	 */

	// Get webhook info
	webhook, err := bot.GetWebhook()
	if err != nil {
		return nil, err
	}

	if webhook != bot.webhook {
		// Delete webhoook first
		if err := bot.DeleteWebhook(); err != nil {
			return nil, err
		}

		// Assign new webhoook with provided value
		if err := bot.SetWebhook(bot.webhook); err != nil {
			return nil, err
		}
	}

	return bot, nil
}

func (bot *Chatbot) GetInfo() (string, error) {
	resp, err := http.Get(fmt.Sprintf("%s/getMe", bot.server))
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if 200 > resp.StatusCode || resp.StatusCode >= 300 {
		return "", fmt.Errorf("failed to get request: %s", resp.Status)
	}

	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}

	return string(data), nil
}

func (bot *Chatbot) SetWebhook(url string) error {
	data, _ := json.Marshal(map[string]string{"url": url})

	resp, err := http.Post(
		fmt.Sprintf("%s/setWebhook", bot.server), "application/json", bytes.NewBuffer(data))

	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if 200 > resp.StatusCode || resp.StatusCode >= 300 {
		message, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("failed to set webhook: %s - Message: %s", resp.Status, message)
	}

	return nil
}

func (bot *Chatbot) DeleteWebhook() error {
	resp, err := http.Post(
		fmt.Sprintf("%s/deleteWebhook", bot.server), "application/json", nil)

	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if 200 > resp.StatusCode || resp.StatusCode >= 300 {
		message, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("failed to delete webhook: %s - Message: %s", resp.Status, message)
	}

	return nil
}

func (bot *Chatbot) GetWebhook() (string, error) {
	resp, err := http.Get(fmt.Sprintf("%s/getWebhookInfo", bot.server))
	if err != nil {
		return "", err
	}

	data := map[string]any{}
	if err := json.NewDecoder(resp.Body).Decode(&data); err != nil {
		return "", err
	}

	if data["ok"].(bool) {
		result := data["result"].(map[string]any)
		return result["url"].(string), nil
	}

	return "", fmt.Errorf("failed to get webhook info: %d %s",
		data["error_code"], data["description"])
}

type MessagePayload struct {
	ChatID int    `json:"chat_id"`
	Text   string `json:"text"`
}

func (bot *Chatbot) SendMessage(payload MessagePayload) error {
	data, _ := json.Marshal(payload)
	resp, err := http.Post(
		fmt.Sprintf("%s/sendMessage", bot.server), "application/json", bytes.NewBuffer(data))
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if 200 > resp.StatusCode || resp.StatusCode >= 300 {
		message, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("failed to send message to client: %s - Message: %s", resp.Status, message)
	}

	return nil
}
