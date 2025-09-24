package api

import (
	"encoding/json"
	"io"
	"net/http"
	"strconv"

	"github.com/danglnh07/ticket-system/ticket-system/service/payment"
	"github.com/gin-gonic/gin"
	"github.com/stripe/stripe-go/v82"
	"github.com/stripe/stripe-go/v82/webhook"
)

// Return Stripe publishable key
func (server *Server) StripeConfig(ctx *gin.Context) {
	// Return the stripe publishable key as an object the frontend expects
	ctx.JSON(http.StatusOK, gin.H{"publishableKey": server.config.StripePublishableKey})
}

// Create payment intent
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
	ctx.JSON(http.StatusOK, gin.H{"clientSecret": clientSecret})
}

// Webhook handler for stripe
func (server *Server) WebhookHandler(ctx *gin.Context) {
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
