package notify

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"time"
)

type GoogleCalendar struct {
	// Google client ID and client secret
	ClientID     string
	ClientSecret string

	// The number of minutes that Google calendar will send a notification before an event start
	EmailNoti int
	PopupNoti int
}

func NewGoogleCalendar(clientID, clientSecret string, emailNoti, popupNoti int) *GoogleCalendar {
	return &GoogleCalendar{
		ClientID:     clientID,
		ClientSecret: clientSecret,
		EmailNoti:    emailNoti,
		PopupNoti:    popupNoti,
	}
}

// Method to request a new access token for user who use Google login. expireIn count in seconds
func (calendar *GoogleCalendar) RefreshToken(refreshToken string) (accessToken string, expiredIn int, err error) {
	// Refresh token
	payload := url.Values{}
	payload.Set("client_id", calendar.ClientID)
	payload.Set("client_secret", calendar.ClientSecret)
	payload.Set("refresh_token", refreshToken)
	payload.Set("grant_type", "refresh_token")
	payload.Set("scope", "openid email profile https://www.googleapis.com/auth/calendar.events")

	resp, err := http.PostForm("https://oauth2.googleapis.com/token", payload)
	if err != nil {
		return
	}
	defer resp.Body.Close()

	if 200 > resp.StatusCode || resp.StatusCode >= 300 {
		data, _ := io.ReadAll(resp.Body)
		fmt.Println("Error message", string(data))
		return "", 0, fmt.Errorf("request new access token failed: %s", resp.Status)
	}

	var newToken struct {
		AccessToken string `json:"access_token"`
		ExpiresIn   int    `json:"expires_in"` // Seconds
	}
	if err = json.NewDecoder(resp.Body).Decode(&newToken); err != nil {
		return
	}

	accessToken = newToken.AccessToken
	expiredIn = newToken.ExpiresIn
	return
}

type CalendarPayload struct {
	Title       string    `json:"title"`
	Location    string    `json:"location"`
	Description string    `json:"description"`
	Start       time.Time `json:"start"`
	End         time.Time `json:"end"`
}

// Helper method: create the data for sending to Google Calendar
func (calendar *GoogleCalendar) CreateEventData(payload CalendarPayload) map[string]any {
	return map[string]any{
		"summary":     payload.Title,
		"location":    payload.Location,
		"description": payload.Description,
		"start": map[string]any{
			"dateTime": payload.Start,
			"timeZone": "UTC",
		},
		"end": map[string]any{
			"dateTime": payload.End,
			"timeZone": "UTC",
		},
		"reminders": map[string]any{
			"useDefault": false,
			"overrides": []map[string]any{
				{"method": "email", "minutes": calendar.EmailNoti}, // Auto sent email x minutes before the event start
				{"method": "popup", "minutes": calendar.PopupNoti}, // UI popup (mobile) x minutes before the event start
			},
		},
	}
}

// Create event in Google Calendar in the primary calendar.
// The string return is the event ID, which is needed to update/delete it
func (calendar *GoogleCalendar) CreateEvent(accessToken string, payload CalendarPayload) (string, error) {
	// Create event data
	data := calendar.CreateEventData(payload)

	// Make request
	body, err := json.Marshal(data)
	if err != nil {
		return "", err
	}
	url := "https://www.googleapis.com/calendar/v3/calendars/primary/events"
	req, err := http.NewRequest("POST", url, bytes.NewBuffer(body))
	if err != nil {
		return "", err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+accessToken)
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	// Check response status
	if 200 > resp.StatusCode || resp.StatusCode >= 300 {
		message, _ := io.ReadAll(resp.Body)
		return "", fmt.Errorf("request failed: %s\nMessage: %s", resp.Status, message)
	}

	// Read response body and return the event ID (needed for update/delete event)
	respData := map[string]any{}
	if err = json.NewDecoder(resp.Body).Decode(&respData); err != nil {
		return "", err
	}
	fmt.Println(resp.Status)

	return respData["id"].(string), nil
}

// Update event in Google Calendar
func (calendar *GoogleCalendar) UpdateEvent(accessToken, eventID string, payload CalendarPayload) error {
	// Create event data
	data := calendar.CreateEventData(payload)

	// Make request
	body, err := json.Marshal(data)
	if err != nil {
		return err
	}
	url := fmt.Sprintf("https://www.googleapis.com/calendar/v3/calendars/primary/events/%s", eventID)
	req, err := http.NewRequest("PUT", url, bytes.NewBuffer(body))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+accessToken)
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	// Check response status
	if 200 > resp.StatusCode || resp.StatusCode >= 300 {
		return fmt.Errorf("request failed: %s", resp.Status)
	}
	fmt.Println(resp.Status)

	return nil
}

// Delete event in Google Calendar
func (calendar *GoogleCalendar) DeleteEvent(accessToken, eventID string) error {
	// Make request
	url := fmt.Sprintf("https://www.googleapis.com/calendar/v3/calendars/primary/events/%s", eventID)
	req, err := http.NewRequest("DELETE", url, nil)
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+accessToken)
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	// Check response status
	if 200 > resp.StatusCode || resp.StatusCode >= 300 {
		return fmt.Errorf("request failed: %s", resp.Status)
	}
	fmt.Println(resp.Status)

	return nil
}
