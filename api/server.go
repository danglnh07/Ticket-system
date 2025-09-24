package api

import (
	"log/slog"

	"github.com/danglnh07/ticket-system/ticket-system/service/mail"
	"github.com/danglnh07/ticket-system/ticket-system/service/notify"
	"github.com/danglnh07/ticket-system/ticket-system/service/security"
	"github.com/danglnh07/ticket-system/ticket-system/service/worker"
	"github.com/danglnh07/ticket-system/ticket-system/util"
	"github.com/gin-gonic/gin"
)

// Server struct, holds the router, dependencies, system config and logger
type Server struct {
	// API router
	router *gin.Engine

	// Dependencies
	mailService mail.MailService
	jwtService  *security.JWTService
	distributor worker.TaskDistributor
	hub         *notify.Hub

	// Server's config and logger
	config *util.Config
	logger *slog.Logger
}

// Constructor method for server struct
func NewServer(
	mailService mail.MailService,
	jwtService *security.JWTService,
	distributor worker.TaskDistributor,
	hub *notify.Hub,
	config *util.Config,
	logger *slog.Logger,
) *Server {
	return &Server{
		router:      gin.Default(),
		mailService: mailService,
		jwtService:  jwtService,
		distributor: distributor,
		hub:         hub,
		config:      config,
		logger:      logger,
	}
}

// Helper method to register handler for API
func (server *Server) RegisterHandler() {
	api := server.router.Group("/api")
	{
		payment := api.Group("/payment")
		{
			payment.GET("/config", server.StripeConfig)
			payment.POST("/intent", server.CreatePaymentIntent)
		}
	}

	server.router.POST("/webhook", server.WebhookHandler)
}

// Start server
func (server *Server) Start() error {
	server.RegisterHandler()
	return server.router.Run(":8080")
}

// Error response struct
type ErrorResponse struct {
	Message string `json:"error"`
}
