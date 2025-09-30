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

/*
 * Google Calendar API docs: https://developers.google.com/workspace/calendar/api/v3/reference/events
 */

// Google Calendar struct
type GoogleCalendar struct {
	// Google client ID and client secret
	ClientID     string
	ClientSecret string

	// The number of minutes that Google calendar will send a notification before an event start
	EmailNoti int
	PopupNoti int
}

// Google Calendar constructor
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
	// Set query parameters
	payload := url.Values{}
	payload.Set("client_id", calendar.ClientID)
	payload.Set("client_secret", calendar.ClientSecret)
	payload.Set("refresh_token", refreshToken)
	payload.Set("grant_type", "refresh_token")
	payload.Set("scope", "openid email profile https://www.googleapis.com/auth/calendar.events")

	// Make a POST request to refresh token endpoint
	resp, err := http.PostForm("https://oauth2.googleapis.com/token", payload)
	if err != nil {
		return
	}
	defer resp.Body.Close()

	// Check response status code
	if 200 > resp.StatusCode || resp.StatusCode >= 300 {
		data, _ := io.ReadAll(resp.Body)
		fmt.Println("Error message", string(data))
		return "", 0, fmt.Errorf("request new access token failed: %s", resp.Status)
	}

	// Marshal response to get token and its expiration
	var data map[string]any
	if err = json.NewDecoder(resp.Body).Decode(&data); err != nil {
		return
	}

	accessToken = data["access_token"].(string)
	expiredIn = data["expires_in"].(int)
	return
}

// Payload for creating event in Google Calendar
type CalendarPayload struct {
	Title       string    `json:"title"`
	Location    string    `json:"location"`
	Description string    `json:"description"`
	Start       time.Time `json:"start"`
	End         time.Time `json:"end"`
}

// Helper method: create the data for sending to Google Calendar
func (calendar *GoogleCalendar) createEventData(payload CalendarPayload) map[string]any {
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

// Helper method for making request to Google Calendar API.
// If this is POST request (create event), the string return is the event ID, else it would be an empty string
func (calendar *GoogleCalendar) makeRequest(method, url, accessToken string, body io.Reader) (string, error) {
	// Make request to Google calendar API
	req, err := http.NewRequest(method, url, body)
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
		// If response status code is 403 (access token expired)
		if resp.StatusCode == 403 {
			return "", fmt.Errorf("Access token expired")
		}

		// Other error
		message, _ := io.ReadAll(resp.Body)
		return "", fmt.Errorf("request failed: %s\nMessage: %s", resp.Status, message)
	}

	// Parse response body. Only POST that we care about the response
	if method == "POST" {
		respData := map[string]any{}
		if err = json.NewDecoder(resp.Body).Decode(&respData); err != nil {
			return "", err
		}
		return respData["id"].(string), nil
	}

	return "", nil
}

// Create event in Google Calendar in the primary calendar.
// The string return is the event ID, which is needed to update/delete it
func (calendar *GoogleCalendar) CreateEvent(accessToken string, payload CalendarPayload) (string, error) {
	// Create event data
	data := calendar.createEventData(payload)

	// Make POST request to calendar endpoint
	body, err := json.Marshal(data)
	if err != nil {
		return "", err
	}
	url := "https://www.googleapis.com/calendar/v3/calendars/primary/events"

	return calendar.makeRequest("POST", url, accessToken, bytes.NewBuffer(body))
}

// Update event in Google Calendar
func (calendar *GoogleCalendar) UpdateEvent(accessToken, eventID string, payload CalendarPayload) error {
	// Create event data
	data := calendar.createEventData(payload)

	// Make PUT request to calendar endpoint
	body, err := json.Marshal(data)
	if err != nil {
		return err
	}
	url := fmt.Sprintf("https://www.googleapis.com/calendar/v3/calendars/primary/events/%s", eventID)

	_, err = calendar.makeRequest("PUT", url, accessToken, bytes.NewBuffer(body))
	return err
}

// Delete event in Google Calendar
func (calendar *GoogleCalendar) DeleteEvent(accessToken, eventID string) error {
	// Make request
	url := fmt.Sprintf("https://www.googleapis.com/calendar/v3/calendars/primary/events/%s", eventID)
	_, err := calendar.makeRequest("DELETE", url, accessToken, nil)
	return err
}
