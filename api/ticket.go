package api

import (
	"errors"
	"net/http"
	"time"

	"github.com/danglnh07/ticket-system/db"
	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

type IssueTicketRequest struct {
	EventID uint    `json:"event_id" binding:"required"`
	Rank    string  `json:"rank" binding:"required"`
	Total   uint    `json:"total" binding:"required;min=0"`
	Price   float64 `json:"price" binding:"required;min=0"`
	Status  string  `json:"status" binding:"required"`
}

type IssueTicketResponse struct {
	ID        uint    `json:"id"`
	EventID   uint    `json:"event_id"`
	Rank      string  `json:"rank"`
	Total     uint    `json:"total"`
	Available uint    `json:"available"`
	Price     float64 `json:"price"`
	Status    string  `json:"status"`
}

func (server *Server) IssueTicket(ctx *gin.Context) {
	// Get request body and validate
	var req IssueTicketRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		server.logger.Warn("POST /api/ticket: failed to bind request body", "error", err)
		ctx.JSON(http.StatusBadRequest, ErrorResponse{"Invalid request body"})
		return
	}

	// Check if status is correct
	status := db.EventStatus(req.Status)
	if status != db.Published && status != db.Draft {
		ctx.JSON(http.StatusBadRequest, ErrorResponse{"Invalid ticket status"})
		return
	}

	// Check if event is valid (status = published, start time not passed)
	var event db.Event
	result := server.queries.DB.Where("id = ?", req.EventID).First(&event)
	if result.Error != nil {
		// If event ID not match any record
		if errors.Is(result.Error, gorm.ErrRecordNotFound) {
			ctx.JSON(http.StatusBadRequest, ErrorResponse{"Event ID not found"})
			return
		}

		// Other database error
		server.logger.Error("POST /api/ticket: failed to get event data from database", "error", result.Error)
		ctx.JSON(http.StatusInternalServerError, ErrorResponse{"Internal server error"})
		return
	}

	if event.Status != db.Published {
		ctx.JSON(http.StatusBadRequest, ErrorResponse{"Event not published"})
		return
	}

	if time.Now().Add(time.Hour).After(event.StartTime) {
		ctx.JSON(http.StatusBadRequest, ErrorResponse{"Tickets must be issued at least 1 hour before event start"})
		return
	}

	// Insert into database
	var ticket = db.Ticket{
		EventID:   req.EventID,
		Rank:      req.Rank,
		Total:     req.Total,
		Available: req.Total,
		Price:     req.Price,
		Status:    status,
	}

	result = server.queries.DB.Create(&ticket)
	if result.Error != nil {
		server.logger.Error("POST /api/ticket: failed to issue ticket", "error", result.Error)
		ctx.JSON(http.StatusInternalServerError, ErrorResponse{"Internal server error"})
		return
	}

	// Return data back to client
	ctx.JSON(http.StatusCreated, IssueTicketResponse{
		ID:        ticket.ID,
		EventID:   ticket.EventID,
		Rank:      ticket.Rank,
		Total:     ticket.Total,
		Available: ticket.Available,
		Price:     ticket.Price,
		Status:    string(ticket.Status),
	})
}

func (server *Server) BookTicket(ctx *gin.Context) {

}
