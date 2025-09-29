package api

import (
	"log/slog"

	"github.com/danglnh07/ticket-system/db"
	_ "github.com/danglnh07/ticket-system/docs"
	"github.com/danglnh07/ticket-system/service/mail"
	"github.com/danglnh07/ticket-system/service/notify"
	"github.com/danglnh07/ticket-system/service/security"
	"github.com/danglnh07/ticket-system/service/worker"
	"github.com/danglnh07/ticket-system/util"
	"github.com/gin-gonic/gin"
	swaggerFiles "github.com/swaggo/files"
	ginSwagger "github.com/swaggo/gin-swagger"
)

// Server struct, holds the router, dependencies, system config and logger
type Server struct {
	// API router
	router *gin.Engine

	// Queries
	queries *db.Queries

	// Dependencies
	mailService mail.MailService
	jwtService  *security.JWTService
	distributor worker.TaskDistributor
	hub         *notify.Hub
	bot         *notify.Chatbot

	// Server's config and logger
	config *util.Config
	logger *slog.Logger
}

// Constructor method for server struct
func NewServer(
	queries *db.Queries,
	mailService mail.MailService,
	jwtService *security.JWTService,
	distributor worker.TaskDistributor,
	hub *notify.Hub,
	bot *notify.Chatbot,
	config *util.Config,
	logger *slog.Logger,
) *Server {
	return &Server{
		router:      gin.Default(),
		queries:     queries,
		mailService: mailService,
		jwtService:  jwtService,
		distributor: distributor,
		hub:         hub,
		bot:         bot,
		config:      config,
		logger:      logger,
	}
}

// Helper method to register handler for API
func (server *Server) RegisterHandler() {
	server.router.Use(server.CORSMiddleware())

	// API routes
	api := server.router.Group("/api")
	{
		payment := api.Group("/payment")
		{
			payment.GET("/config", server.StripeConfig)
			payment.POST("/intent", server.CreatePaymentIntent)
			payment.POST("/refund", server.Refund)
			payment.POST("/webhook", server.PaymentWebhookHandler)
		}

		bot := api.Group("/bot")
		{
			bot.POST("/webhook", server.BotWebhook)
		}
	}

	// Swagger docs
	server.router.GET("/swagger/*any", ginSwagger.WrapHandler(swaggerFiles.Handler))
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
