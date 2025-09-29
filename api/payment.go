package api

import (
	"encoding/json"
	"io"
	"net/http"
	"strconv"
	"time"

	"github.com/danglnh07/ticket-system/service/payment"
	"github.com/gin-gonic/gin"
	"github.com/stripe/stripe-go/v82"
	"github.com/stripe/stripe-go/v82/webhook"
)

// Stripe config response struct
type StripeConfigResponse struct {
	PublishableKey string `json:"publishable_key"`
}

func (server *Server) StripeConfig(ctx *gin.Context) {
	ctx.JSON(http.StatusOK, StripeConfigResponse{server.config.StripePublishableKey})
}

type PaymentIntentResponse struct {
	SecretKey string `json:"secret_key"`
}

func (server *Server) CreatePaymentIntent(ctx *gin.Context) {
	// Get the amount from query string
	amount, err := strconv.ParseInt(ctx.Query("amount"), 10, 64)
	if err != nil {
		server.logger.Warn("POST /api/payment/intent: invalid amount query parameter", "error", err)
		ctx.JSON(http.StatusBadRequest, ErrorResponse{"invalid value for amount"})
		return
	}

	// Create payment intent
	clientSecret, err := payment.CreatePaymentIntent(amount)
	if err != nil {
		server.logger.Error("POST /api/payment/intent: failed to create payment intent", "error", err)
		ctx.JSON(http.StatusInternalServerError, ErrorResponse{"Internal server error"})
		return
	}

	// Return the client secret key back as an object the frontend expects
	ctx.JSON(http.StatusOK, PaymentIntentResponse{clientSecret})
}

type RefundRequest struct {
	PaymentIntentID string `json:"piID" binding:"required"`
	Reason          string `json:"reason" binding:"required"`
	Amount          int64  `json:"amount" binding:"min=1"`
}

type RefundResponse struct {
	ID        string    `json:"id"`
	Amount    int64     `json:"amount"`
	CreatedAt time.Time `json:"created_at"`
	Status    string    `json:"status"`
}

func (server *Server) Refund(ctx *gin.Context) {
	var req RefundRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		server.logger.Warn("POST /api/payment/intent: failed to get request body", "error", err)
		ctx.JSON(http.StatusBadRequest, ErrorResponse{"Invalid request body"})
		return
	}

	// Check if piID exists in database, amount is not exceed the total amount

	// Check if the reason match
	reason := payment.RefundReason(req.Reason)
	if reason != payment.Duplicate && reason != payment.Fraudulent && reason != payment.RequestedByCustomer {
		ctx.JSON(http.StatusBadRequest, ErrorResponse{"invalid reason"})
		return
	}

	// Refund
	refund, err := payment.CreateRefund(req.PaymentIntentID, reason, req.Amount)
	if err != nil {
		server.logger.Error("/api/payment/refund: failed to create a refund", "error", err)
		ctx.JSON(http.StatusInternalServerError, ErrorResponse{"Internal server error"})
		return
	}

	ctx.JSON(http.StatusOK, RefundResponse{
		ID:        refund.ID,
		Amount:    refund.Amount,
		CreatedAt: time.Unix(0, refund.Created),
		Status:    string(refund.Status),
	})
}

// Webhook handler for stripe
func (server *Server) PaymentWebhookHandler(ctx *gin.Context) {
	// Read the payload
	payload, err := io.ReadAll(ctx.Request.Body)
	if err != nil {
		server.logger.Error("/webhook: failed to read request body")
		return
	}
	defer ctx.Request.Body.Close()

	// Construct event
	event, err := webhook.ConstructEvent(
		payload, ctx.Request.Header.Get("Stripe-Signature"), server.config.StripeWebhookSecret)
	if err != nil {
		server.logger.Error("/webhook: failed to construct event", "error", err)
		return
	}

	// Act based on event type
	var pi stripe.PaymentIntent
	if err := json.Unmarshal(event.Data.Raw, &pi); err != nil {
		server.logger.Error("/webhook: failed to unmarshal the payment intent", "error", err)
		return
	}
	switch event.Type {
	case "payment_intent.succeeded":
		// --> IMPLEMENT LOGIC HERE
		server.logger.Info("/webhook: payment success", "id", pi.ID)
	case "payment_intent.payment_failed":
		// --> IMPLEMENT LOGIC HERE
		server.logger.Warn("/webhook: payment failed", "id", pi.ID)
	default:
		server.logger.Warn("/webhook: unsupported event", "type", event.Type)
	}
}
