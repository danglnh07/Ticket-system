package api

import (
	"database/sql"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/danglnh07/ticket-system/db"
	"github.com/danglnh07/ticket-system/service/security"
	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

type CreateEventRequest struct {
	Name         string    `json:"name" binding:"required"`
	Desription   string    `json:"description" binding:"required"`
	Location     string    `json:"location" binding:"required"`
	StartTime    time.Time `json:"start_time" binding:"required"`
	EndTime      time.Time `json:"end_time" binding:"required"`
	PreviewImage string    `json:"preview_image"`
	Status       string    `json:"status"`
}

type EventResponse struct {
	ID           uint      `json:"id"`
	Name         string    `json:"name"`
	Description  string    `json:"description"`
	Location     string    `json:"location"`
	StartTime    time.Time `json:"start_time"`
	EndTime      time.Time `json:"end_time"`
	PreviewImage string    `json:"preview_image"`
	Status       string    `json:"status"`
}

func (server *Server) validateEventTime(start, end time.Time) error {
	// Check if the start time is before end time at least 1 hour
	if start.Add(time.Hour).After(end) {
		return fmt.Errorf("Event duration must be at least 1 hour")
	}

	// Check if start time is at least 1 week in the future
	if time.Now().Add(time.Hour * 24 * 7).After(start) {
		return fmt.Errorf("Event's start time must be at least 1 week in the future")
	}

	return nil
}

func (server *Server) CreateEvent(ctx *gin.Context) {
	// Get request body and validate
	var req CreateEventRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		server.logger.Warn("POST /api/event: failed to bind JSON request body", "error", err)
		ctx.JSON(http.StatusBadRequest, ErrorResponse{"Invalid request body"})
		return
	}

	// Time validation
	if err := server.validateEventTime(req.StartTime, req.EndTime); err != nil {
		ctx.JSON(http.StatusBadRequest, ErrorResponse{err.Error()})
		return
	}

	// Get account ID from claims
	claims, _ := ctx.Get(claimsKey)
	accountID := claims.(*security.CustomClaims).ID

	// Check status
	status := db.EventStatus(req.Status)
	if status != db.Published && status != db.Draft {
		ctx.JSON(http.StatusBadRequest, ErrorResponse{"Invalid value for status"})
		return
	}

	// Check if client provide preview_image
	var image sql.NullString
	if req.PreviewImage == "" {
		image = sql.NullString{String: "", Valid: false}
	} else {
		image = sql.NullString{String: req.PreviewImage, Valid: true}
	}

	// Create event in database
	var event = db.Event{
		HostID:       accountID,
		Name:         req.Name,
		Description:  req.Desription,
		Location:     req.Location,
		StartTime:    req.StartTime,
		EndTime:      req.EndTime,
		PreviewImage: image,
		Status:       status,
	}
	result := server.queries.DB.Create(&event)
	if result.Error != nil {
		server.logger.Error("POST /api/event: failed to create event", "error", result.Error)
		ctx.JSON(http.StatusInternalServerError, ErrorResponse{"Internal server error"})
		return
	}

	// Return data back to client
	ctx.JSON(http.StatusCreated, EventResponse{
		ID:           event.ID,
		Name:         event.Name,
		Description:  event.Description,
		Location:     event.Location,
		StartTime:    event.StartTime,
		EndTime:      event.EndTime,
		PreviewImage: event.PreviewImage.String,
		Status:       string(event.Status),
	})
}

type UpdateEventRequest struct {
	Name         string    `json:"name"`
	Description  string    `json:"description"`
	Location     string    `json:"location"`
	StartTime    time.Time `json:"start_time"`
	EndTime      time.Time `json:"end_time"`
	PreviewImage string    `json:"preview_image"`
	Status       string    `json:"status"`
}

func (server *Server) UpdateEvent(ctx *gin.Context) {
	// Get event ID from request parameter
	eventID := ctx.Param("id")

	// Get event from database
	var event db.Event
	result := server.queries.DB.Where("id = ?", eventID).First(&event)
	if result.Error != nil {
		// If ID not found
		if errors.Is(result.Error, gorm.ErrRecordNotFound) {
			ctx.JSON(http.StatusBadRequest, ErrorResponse{"Event ID not found"})
			return
		}

		// Other database error
		server.logger.Error("PUT /api/event: failed to get event data from database", "error", result.Error)
		ctx.JSON(http.StatusInternalServerError, ErrorResponse{"Internal server error"})
		return
	}

	// Check if the requester is the event's host
	claims, _ := ctx.Get(claimsKey)
	requesterID := claims.(*security.CustomClaims).ID

	if requesterID != event.HostID {
		ctx.JSON(http.StatusUnauthorized, ErrorResponse{"You are not authorized to perform this action"})
		return
	}

	// Get and validate request body
	var req UpdateEventRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		server.logger.Warn("PUT /api/event: failed to bind request body", "error", err)
		ctx.JSON(http.StatusBadRequest, ErrorResponse{"Invalid request body"})
		return
	}

	if err := server.validateEventTime(req.StartTime, req.EndTime); err != nil {
		ctx.JSON(http.StatusBadRequest, ErrorResponse{err.Error()})
		return
	}

	status := db.EventStatus(req.Status)
	if status != db.Published && status != db.Draft && status != db.Canceled {
		ctx.JSON(http.StatusBadRequest, ErrorResponse{"Invalid value for event status"})
		return
	}

	if req.Name = strings.TrimSpace(req.Name); req.Name != "" {
		event.Name = req.Name
	}

	if req.Description = strings.TrimSpace(req.Description); req.Description != "" {
		event.Description = req.Description
	}

	if req.Location = strings.TrimSpace(req.Location); req.Location != "" {
		event.Location = req.Location
	}

	if req.PreviewImage = strings.TrimSpace(req.PreviewImage); req.PreviewImage != "" {
		event.PreviewImage = sql.NullString{String: req.PreviewImage, Valid: true}
	}

	if !req.StartTime.IsZero() {
		event.StartTime = req.StartTime
	}

	if !req.EndTime.IsZero() {
		event.EndTime = req.EndTime
	}

	// Save new event
	result = server.queries.DB.Save(&event)
	if result.Error != nil {
		server.logger.Error("PUT /api/event/:id: failed to update event into database", "error", result.Error)
		ctx.JSON(http.StatusInternalServerError, ErrorResponse{"Internal server error"})
		return
	}

	// Return new event data back to client
	ctx.JSON(http.StatusOK, EventResponse{
		ID:           event.ID,
		Name:         event.Name,
		Description:  event.Description,
		Location:     event.Location,
		StartTime:    event.StartTime,
		EndTime:      event.EndTime,
		PreviewImage: event.PreviewImage.String,
		Status:       string(event.Status),
	})
}

func (server *Server) GetEvent(ctx *gin.Context) {
	// Get event ID
	eventID := ctx.Param("id")

	// Get event from database
	var event db.Event
	result := server.queries.DB.Where("id = ?", eventID).First(&event)
	if result.Error != nil {
		// If ID not match any record
		if errors.Is(result.Error, gorm.ErrRecordNotFound) {
			ctx.JSON(http.StatusNotFound, ErrorResponse{"Event ID not found"})
			return
		}

		// Other database error
		server.logger.Error("GET /api/event/:id: failed to get event data", "error", result.Error)
		ctx.JSON(http.StatusBadRequest, ErrorResponse{"Internal server error"})
		return
	}

	// Check if the current requester has the right to see this event based on status
	// If owner or admin -> can see regardless of status
	// Other organisers, users -> can only see if published
	claims, _ := ctx.Get(claimsKey)
	requesterID := claims.(*security.CustomClaims).ID
	role := claims.(*security.CustomClaims).Role
	if requesterID != event.HostID && role != db.Admin && event.Status != db.Published {
		ctx.JSON(http.StatusUnauthorized, ErrorResponse{"You haven't been authorized to perform this action"})
		return
	}

	// Return result back to client
	ctx.JSON(http.StatusOK, EventResponse{
		ID:           event.ID,
		Name:         event.Name,
		Description:  event.Description,
		Location:     event.Location,
		StartTime:    event.StartTime,
		EndTime:      event.EndTime,
		PreviewImage: event.PreviewImage.String,
		Status:       string(event.Status),
	})
}
