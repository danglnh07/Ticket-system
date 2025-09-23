package worker

import (
	"fmt"
)

type SendNotificationPayload struct {
	ReceiverID uint   `json:"receiver_id"`
	Title      string `json:"title"`
	Content    string `json:"content"`
}

const SendNotification = "send-notification"

func (processor *RedisTaskProcessor) SendNotification(pl any) error {
	// Check if the payload type is correct
	payload, ok := pl.(SendNotificationPayload)
	if !ok {
		return fmt.Errorf("invalid payload type for this task")
	}

	// Check if user is online
	if processor.hub.IsUserOnline(payload.ReceiverID) {
		// Send in app notification
		err := processor.hub.Publish(payload.ReceiverID, map[string]string{
			"title":   payload.Title,
			"content": payload.Content,
		})

		if err != nil {
			return err
		}

		return nil
	}

	// If user is not online, we'll choose another notification method (just print the message for now)
	processor.logger.Info("Send notification to offline user",
		"message", fmt.Sprintf("Title: %s - Content: %s", payload.Title, payload.Content))

	return nil
}
