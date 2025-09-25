package api

import (
	"encoding/json"
	"io"
	"net/http"
	"strconv"

	"github.com/danglnh07/ticket-system/service/payment"
	"github.com/gin-gonic/gin"
	"github.com/stripe/stripe-go/v82"
	"github.com/stripe/stripe-go/v82/webhook"
)

// Stripe config response struct
type StripeConfigResponse struct {
	PublishableKey string `json:"publishable_key"`
}

// StripeConfig godoc
// @Summary      Get stripe publishable key
// @Description  Get stripe publishable key
// @Tags         payment
// @Produce      json
// @Success      200      {object}  StripeConfigResponse
// @Security     BearerAuth
// @Router       /api/payment/config [get]
func (server *Server) StripeConfig(ctx *gin.Context) {
	ctx.JSON(http.StatusOK, StripeConfigResponse{server.config.StripePublishableKey})
}

type PaymentIntentResponse struct {
	SecretKey string `json:"secret_key"`
}

// CreatePaymentIntent godoc
// @Summary      Create a new payment intent
// @Description  Creates a new payment intent with an amount. Here, we expect amount to be in cent
// @Tags         payment
// @Produce      json
// @Param        amount  query      int  true  "The amount for payment (cents)"
// @Success      200      {object}  PaymentIntentResponse
// @Failure      400      {object}  ErrorResponse      "Invalid request body or invalid deadline"
// @Failure      500      {object}  ErrorResponse      "Internal server error"
// @Security     BearerAuth
// @Router       /api/payment/intent [post]
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
