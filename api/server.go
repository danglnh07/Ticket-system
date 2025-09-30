package api

import (
	"log/slog"
	"net/http"

	"github.com/danglnh07/ticket-system/db"
	_ "github.com/danglnh07/ticket-system/docs"
	"github.com/danglnh07/ticket-system/service/mail"
	"github.com/danglnh07/ticket-system/service/notify"
	"github.com/danglnh07/ticket-system/service/security"
	"github.com/danglnh07/ticket-system/service/worker"
	"github.com/gin-gonic/gin"
	swaggerFiles "github.com/swaggo/files"
	ginSwagger "github.com/swaggo/gin-swagger"
	"golang.org/x/oauth2"
)

// Server struct, holds the router, dependencies, system config and logger
type Server struct {
	// API router
	router *gin.Engine

	// Queries
	queries *db.Queries

	// Dependencies
	oauthConfigs map[db.OauthProvider]*oauth2.Config
	mailService  mail.MailService
	jwtService   *security.JWTService
	distributor  worker.TaskDistributor
	calendar     *notify.GoogleCalendar
	hub          *notify.Hub
	bot          *notify.Chatbot

	// Server's config and logger
	logger *slog.Logger
}

// Constructor method for server struct
func NewServer(
	queries *db.Queries,
	mailService mail.MailService,
	jwtService *security.JWTService,
	distributor worker.TaskDistributor,
	calendar *notify.GoogleCalendar,
	hub *notify.Hub,
	bot *notify.Chatbot,
	logger *slog.Logger,
) *Server {
	// Create the list of oauth configs
	configs := map[db.OauthProvider]*oauth2.Config{
		db.Google: NewGoogleOAuth(),
	}

	return &Server{
		router:       gin.Default(),
		queries:      queries,
		oauthConfigs: configs,
		mailService:  mailService,
		jwtService:   jwtService,
		distributor:  distributor,
		calendar:     calendar,
		hub:          hub,
		bot:          bot,
		logger:       logger,
	}
}

// Helper method to register handler for API
func (server *Server) RegisterHandler() {
	server.router.Use(server.CORSMiddleware())

	// API routes
	api := server.router.Group("/api")
	{
		api.GET("", func(ctx *gin.Context) {
			ctx.JSON(http.StatusOK, gin.H{"message": "hello world"})
		})
		// Auth endpoints
		auth := api.Group("/auth")
		{
			auth.POST("/register", server.Register)
			auth.GET("/verify", server.VerifyAccount)
			auth.POST("/login", server.Login)
			auth.GET("/oauth", server.HandleOAuth)
		}

		// Event endpoints
		event := api.Group("/event", server.AuthMiddleware())
		{
			event.POST("", server.CreateEvent, server.AuthorizeMiddleware(db.Organiser))
			event.PUT("/:id", server.UpdateEvent, server.AuthorizeMiddleware(db.Organiser))
			event.GET("/:id", server.GetEvent)
		}

		// Ticket endpoints
		ticket := api.Group("/ticket", server.AuthMiddleware())
		{
			ticket.POST("", server.AuthorizeMiddleware(db.Organiser), server.IssueTicket)
			ticket.POST("/book/:id", server.AuthorizeMiddleware(db.User), server.BookTicket)

		}

		// Payment endpoints
		payment := api.Group("/payment")
		{
			payment.GET("/config", server.StripeConfig)
			payment.POST("/intent", server.CreatePaymentIntent)
			payment.POST("/refund", server.Refund)
			payment.POST("/webhook", server.PaymentWebhookHandler)
		}

		// Chatbot endpoints
		bot := api.Group("/bot")
		{
			bot.POST("/webhook", server.BotWebhook)
		}
	}

	// OAuth2 callback
	server.router.GET("/oauth2/callback", server.HandleCallback)

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
