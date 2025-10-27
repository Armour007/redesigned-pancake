package main

import (
	"context"
	"log"
	"os"
	"strings"
	"time"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"go.opentelemetry.io/contrib/instrumentation/github.com/gin-gonic/gin/otelgin"

	database "github.com/Armour007/aura-backend/internal"
	"github.com/Armour007/aura-backend/internal/api"
)

func main() {
	database.Connect()

	// Determine listen port from environment (PORT or AURA_PORT), default to 8080
	port := os.Getenv("PORT")
	if port == "" {
		port = os.Getenv("AURA_PORT")
	}
	if port == "" {
		// Use 8081 as the local default to avoid common collisions with other dev services
		port = "8081"
	}
	log.Println("Starting AURA backend server on :" + port + "...")
	router := gin.Default()
	// OpenTelemetry tracing (optional)
	if shutdown, ok := api.SetupOTelFromEnv(); ok {
		defer shutdown(context.Background())
		router.Use(otelgin.Middleware("aura-backend"))
	}
	// Metrics
	router.Use(api.MetricsMiddleware())
	// Assign a Request ID to every request for tracing
	router.Use(api.RequestIDMiddleware())
	// API versioning header middleware
	router.Use(api.VersionMiddleware("2025-10-01"))
	// --- START CORS MIDDLEWARE ---
	// Apply CORS middleware. Default() allows all origins for development.
	// For production, configure specific origins: cors.New(cors.Config{...})
	// Replace the simple cors.Default() with this configuration:
	config := cors.Config{
		AllowAllOrigins:  true, // Allow requests from any origin (good for development)
		AllowMethods:     []string{"GET", "POST", "PUT", "PATCH", "DELETE", "OPTIONS"},
		AllowHeaders:     []string{"Origin", "Content-Length", "Content-Type", "Authorization", "X-API-Key", "X-Request-ID", "Idempotency-Key", "AURA-Version"}, // Allow API key, request ID, idempotency and version headers
		ExposeHeaders:    []string{"Content-Length", "X-Request-ID"},
		AllowCredentials: true,
		MaxAge:           12 * time.Hour,
	}
	// Override allowed origins via env (comma-separated)
	if origins := os.Getenv("AURA_CORS_ORIGINS"); origins != "" {
		config.AllowAllOrigins = false
		// Trim spaces around each origin
		parts := strings.Split(origins, ",")
		allow := make([]string, 0, len(parts))
		for _, p := range parts {
			if s := strings.TrimSpace(p); s != "" {
				allow = append(allow, s)
			}
		}
		if len(allow) > 0 {
			config.AllowOrigins = allow
		}
	}
	router.Use(cors.New(config))
	// Optionally configure trusted proxies (comma-separated CIDRs or IPs)
	if tp := os.Getenv("AURA_TRUSTED_PROXIES"); tp != "" {
		parts := strings.Split(tp, ",")
		for i := range parts {
			parts[i] = strings.TrimSpace(parts[i])
		}
		if err := router.SetTrustedProxies(parts); err != nil {
			log.Printf("warning: failed to set trusted proxies: %v", err)
		}
	}
	// --- END CORS MIDDLEWARE ---
	// --- Public Routes (No Auth Needed) ---
	authRoutes := router.Group("/auth")
	{
		authRoutes.POST("/register", api.RegisterUser)
		authRoutes.POST("/login", api.LoginUser)
	}
	coreRoutes := router.Group("/v1")
	coreRoutes.Use(api.ApiKeyAuthMiddleware())
	// Apply rate limiting to core verification endpoints (env-configurable, optional Redis)
	coreRoutes.Use(api.RateLimitMiddlewareFromEnv())
	{
		// /v1/verify uses API Key Auth via middleware above
		coreRoutes.POST("/verify", api.HandleVerifyRequest)
	}

	// --- Protected Routes (Require User JWT Auth) ---
	// Create a new group for routes that need authentication
	// Health and readiness
	router.GET("/healthz", func(c *gin.Context) { c.Status(200) })
	router.GET("/readyz", func(c *gin.Context) {
		ctx, cancel := context.WithTimeout(c.Request.Context(), 300*time.Millisecond)
		defer cancel()
		if err := database.DB.DB.PingContext(ctx); err != nil {
			c.JSON(503, gin.H{"status": "not ready", "error": err.Error()})
			return
		}
		c.JSON(200, gin.H{"status": "ready"})
	})

	// OpenAPI JSON, Swagger UI, and Prometheus metrics
	router.GET("/openapi.json", api.OpenAPIJSON)
	router.GET("/docs", api.SwaggerUI)
	router.GET("/metrics", gin.WrapH(promhttp.Handler()))
	// Tiny redirect to frontend Quick Start for first-time onboarding
	router.GET("/quickstart", api.QuickstartRedirect)

	protectedRoutes := router.Group("/")
	protectedRoutes.Use(api.AuthMiddleware()) // Apply the middleware HERE
	{
		// User profile endpoints
		protectedRoutes.GET("/me", api.GetMe)
		protectedRoutes.PUT("/me", api.UpdateMe)
		protectedRoutes.PUT("/me/password", api.UpdatePassword)

		protectedRoutes.GET("/organizations/mine", api.GetMyOrganizations)
		// All routes defined within this group will now require a valid JWT
		orgRoutes := protectedRoutes.Group("/organizations/:orgId")
		orgRoutes.Use(api.OrgMemberMiddleware())
		// Apply idempotency for POST requests with Idempotency-Key header
		orgRoutes.Use(api.IdempotencyMiddlewareFromEnv())
		{
			// Organization settings (admin only)
			orgRoutes.GET("", api.GetOrganizationByID)
			orgRoutes.PUT("", api.RequireOrgAdmin(), api.UpdateOrganization)

			agentRoutes := orgRoutes.Group("/agents")
			{
				// Admin-only mutations
				agentRoutes.POST("", api.RequireOrgAdmin(), api.CreateAgent)
				agentRoutes.GET("", api.GetAgents)
				agentRoutes.GET("/:agentId", api.GetAgentByID)
				agentRoutes.PUT("/:agentId", api.RequireOrgAdmin(), api.UpdateAgent)
				agentRoutes.DELETE("/:agentId", api.RequireOrgAdmin(), api.DeleteAgent)

				permRoutes := agentRoutes.Group("/:agentId/permissions")
				{
					permRoutes.POST("", api.RequireOrgAdmin(), api.AddPermissionRule)
					permRoutes.GET("", api.GetPermissionRules)
					permRoutes.DELETE("/:ruleId", api.RequireOrgAdmin(), api.DeletePermissionRule)
				}
			}

			apiKeyRoutes := orgRoutes.Group("/apikeys")
			{
				apiKeyRoutes.POST("", api.RequireOrgAdmin(), api.CreateAPIKey)
				apiKeyRoutes.GET("", api.RequireOrgAdmin(), api.GetAPIKeys)
				apiKeyRoutes.DELETE("/:keyId", api.RequireOrgAdmin(), api.DeleteAPIKey)
			}

			// Webhook endpoints management
			webhookRoutes := orgRoutes.Group("/webhooks")
			{
				webhookRoutes.POST("", api.RequireOrgAdmin(), api.CreateWebhookEndpoint)
				webhookRoutes.GET("", api.RequireOrgAdmin(), api.ListWebhookEndpoints)
				webhookRoutes.DELETE("/:webhookId", api.RequireOrgAdmin(), api.DeleteWebhookEndpoint)
			}

			logRoutes := orgRoutes.Group("/logs")
			{
				logRoutes.GET("", api.GetEventLogs)
			}
		}
		// Add other protected routes here if needed (e.g., /user/profile)
	}

	err := router.Run(":" + port)
	if err != nil {
		log.Fatal("Failed to start server:", err)
	}
}
