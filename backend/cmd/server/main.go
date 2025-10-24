package main

import (
	"log"

	"github.com/gin-gonic/gin"

	// Import your database and api packages - Adjust paths
	database "github.com/Armour007/aura-backend/internal"
	"github.com/Armour007/aura-backend/internal/api" // Assuming Armour007 is correct
)

func main() {
	database.Connect()
	log.Println("Starting AURA backend server on :8080...")
	router := gin.Default()

	// --- Public Routes (No Auth Needed) ---
	authRoutes := router.Group("/auth")
	{
		authRoutes.POST("/register", api.RegisterUser)
		authRoutes.POST("/login", api.LoginUser)
	}
	coreRoutes := router.Group("/v1")
	{
		// NOTE: /v1/verify uses API Key Auth, NOT User JWT Auth.
		// We'll add separate middleware for API Keys later.
		coreRoutes.POST("/verify", api.HandleVerifyRequest) // Keep public for now
	}

	// --- Protected Routes (Require User JWT Auth) ---
	// Create a new group for routes that need authentication
	protectedRoutes := router.Group("/")
	protectedRoutes.Use(api.AuthMiddleware()) // Apply the middleware HERE
	{
		// All routes defined within this group will now require a valid JWT
		orgRoutes := protectedRoutes.Group("/organizations/:orgId")
		{
			agentRoutes := orgRoutes.Group("/agents")
			{
				agentRoutes.POST("", api.CreateAgent)
				agentRoutes.GET("", api.GetAgents)
				agentRoutes.GET("/:agentId", api.GetAgents)
				agentRoutes.PUT("/:agentId", api.UpdateAgent)
				agentRoutes.DELETE("/:agentId", api.DeleteAgent)

				permRoutes := agentRoutes.Group("/:agentId/permissions")
				{
					permRoutes.POST("", api.AddPermissionRule)
					permRoutes.GET("", api.GetPermissionRules)
					permRoutes.DELETE("/:ruleId", api.DeletePermissionRule)
				}
			}

			apiKeyRoutes := orgRoutes.Group("/apikeys")
			{
				apiKeyRoutes.POST("", api.CreateAPIKey)
				apiKeyRoutes.GET("", api.GetAPIKeys)
				apiKeyRoutes.DELETE("/:keyId", api.DeleteAPIKey)
			}

			logRoutes := orgRoutes.Group("/logs")
			{
				logRoutes.GET("", api.GetEventLogs)
			}
		}
		// Add other protected routes here if needed (e.g., /user/profile)
	}

	err := router.Run(":8080")
	if err != nil {
		log.Fatal("Failed to start server:", err)
	}
}
