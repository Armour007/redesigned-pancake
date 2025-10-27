package api

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

// OpenAPIJSON serves an OpenAPI v3 document describing the AURA API.
func OpenAPIJSON(c *gin.Context) {
	spec := map[string]any{
		"openapi": "3.0.3",
		"info": map[string]any{
			"title":       "AURA API",
			"version":     "1.0.0",
			"description": "API-first trust and authorization for autonomous systems.",
		},
		"components": map[string]any{
			"securitySchemes": map[string]any{
				"bearerAuth": map[string]any{"type": "http", "scheme": "bearer", "bearerFormat": "JWT"},
				"apiKeyAuth": map[string]any{"type": "apiKey", "in": "header", "name": "X-API-Key"},
			},
			"parameters": map[string]any{
				"AURAVersion": map[string]any{
					"name": "AURA-Version", "in": "header", "required": false,
					"description": "Optional API version header. Defaults to 2025-10-01.",
					"schema":      map[string]any{"type": "string", "example": "2025-10-01"},
				},
				"IdempotencyKey": map[string]any{
					"name": "Idempotency-Key", "in": "header", "required": false,
					"description": "Provide for POST mutations to safely retry. First 2xx response is cached for 24h.",
					"schema":      map[string]any{"type": "string", "example": "req_6a84c5e9e2a14d0a"},
				},
			},
			"schemas": map[string]any{
				"RegisterRequest": map[string]any{"type": "object", "required": []string{"full_name", "email", "password"}, "properties": map[string]any{
					"full_name": map[string]any{"type": "string"},
					"email":     map[string]any{"type": "string", "format": "email"},
					"password":  map[string]any{"type": "string", "format": "password"},
				}},
				"LoginRequest": map[string]any{"type": "object", "required": []string{"email", "password"}, "properties": map[string]any{
					"email":    map[string]any{"type": "string", "format": "email"},
					"password": map[string]any{"type": "string", "format": "password"},
				}},
				"VerifyRequest": map[string]any{"type": "object", "required": []string{"agent_id", "request_context"}, "properties": map[string]any{
					"agent_id":        map[string]any{"type": "string", "format": "uuid"},
					"request_context": map[string]any{"type": "object", "additionalProperties": true},
				}},
				"VerifyResponse": map[string]any{"type": "object", "properties": map[string]any{
					"decision": map[string]any{"type": "string", "enum": []any{"ALLOWED", "DENIED"}},
					"reason":   map[string]any{"type": "string"},
				}},
				"Agent": map[string]any{"type": "object", "properties": map[string]any{
					"id":              map[string]any{"type": "string", "format": "uuid"},
					"organization_id": map[string]any{"type": "string", "format": "uuid"},
					"name":            map[string]any{"type": "string"},
					"description":     map[string]any{"type": "string", "nullable": true},
					"created_at":      map[string]any{"type": "string", "format": "date-time"},
				}},
				"Permission": map[string]any{"type": "object", "properties": map[string]any{
					"id":         map[string]any{"type": "string", "format": "uuid"},
					"agent_id":   map[string]any{"type": "string", "format": "uuid"},
					"rule":       map[string]any{"type": "object", "additionalProperties": true},
					"is_active":  map[string]any{"type": "boolean"},
					"created_at": map[string]any{"type": "string", "format": "date-time"},
				}},
				"APIKeyInfo": map[string]any{"type": "object", "properties": map[string]any{
					"id":           map[string]any{"type": "string", "format": "uuid"},
					"name":         map[string]any{"type": "string"},
					"key_prefix":   map[string]any{"type": "string"},
					"created_at":   map[string]any{"type": "string", "format": "date-time"},
					"last_used_at": map[string]any{"type": "string", "format": "date-time", "nullable": true},
					"expires_at":   map[string]any{"type": "string", "format": "date-time", "nullable": true},
				}},
				"EventLog": map[string]any{"type": "object", "properties": map[string]any{
					"id":                  map[string]any{"type": "integer", "format": "int64"},
					"organization_id":     map[string]any{"type": "string", "format": "uuid"},
					"agent_id":            map[string]any{"type": "string", "format": "uuid", "nullable": true},
					"timestamp":           map[string]any{"type": "string", "format": "date-time"},
					"event_type":          map[string]any{"type": "string"},
					"api_key_prefix_used": map[string]any{"type": "string", "nullable": true},
					"decision":            map[string]any{"type": "string"},
					"request_details":     map[string]any{"type": "object", "additionalProperties": true},
					"decision_reason":     map[string]any{"type": "string", "nullable": true},
					"client_ip_address":   map[string]any{"type": "string"},
					"request_id":          map[string]any{"type": "string", "nullable": true},
					"user_agent":          map[string]any{"type": "string", "nullable": true},
					"path":                map[string]any{"type": "string", "nullable": true},
					"status_code":         map[string]any{"type": "integer", "nullable": true},
				}},
				"WebhookEndpoint": map[string]any{"type": "object", "properties": map[string]any{
					"id":         map[string]any{"type": "string", "format": "uuid"},
					"url":        map[string]any{"type": "string", "format": "uri"},
					"is_active":  map[string]any{"type": "boolean"},
					"created_at": map[string]any{"type": "string", "format": "date-time"},
				}},
				"User": map[string]any{"type": "object", "properties": map[string]any{
					"id":         map[string]any{"type": "string", "format": "uuid"},
					"full_name":  map[string]any{"type": "string"},
					"email":      map[string]any{"type": "string", "format": "email"},
					"created_at": map[string]any{"type": "string", "format": "date-time"},
					"updated_at": map[string]any{"type": "string", "format": "date-time"},
				}},
			},
		},
		"paths": map[string]any{
			"/auth/register": map[string]any{
				"post": map[string]any{
					"summary":     "Register user",
					"requestBody": map[string]any{"required": true, "content": map[string]any{"application/json": map[string]any{"schema": map[string]any{"$ref": "#/components/schemas/RegisterRequest"}}}},
					"responses":   map[string]any{"201": map[string]any{"description": "Created"}},
				},
			},
			"/auth/login": map[string]any{
				"post": map[string]any{
					"summary":     "Login user",
					"requestBody": map[string]any{"required": true, "content": map[string]any{"application/json": map[string]any{"schema": map[string]any{"$ref": "#/components/schemas/LoginRequest"}}}},
					"responses":   map[string]any{"200": map[string]any{"description": "OK"}},
				},
			},
			"/v1/verify": map[string]any{
				"post": map[string]any{
					"summary":     "Verify authorization (API key auth)",
					"security":    []any{map[string]any{"apiKeyAuth": []any{}}},
					"parameters":  []any{map[string]any{"$ref": "#/components/parameters/AURAVersion"}},
					"requestBody": map[string]any{"required": true, "content": map[string]any{"application/json": map[string]any{"schema": map[string]any{"$ref": "#/components/schemas/VerifyRequest"}}}},
					"responses":   map[string]any{"200": map[string]any{"description": "OK", "content": map[string]any{"application/json": map[string]any{"schema": map[string]any{"$ref": "#/components/schemas/VerifyResponse"}}}}},
				},
			},
			"/organizations/mine": map[string]any{
				"get": map[string]any{"summary": "List my organizations (JWT)", "security": []any{map[string]any{"bearerAuth": []any{}}}},
			},
			"/organizations/{orgId}": map[string]any{
				"parameters": []any{map[string]any{"name": "orgId", "in": "path", "required": true, "schema": map[string]any{"type": "string", "format": "uuid"}}},
				"get":        map[string]any{"summary": "Get org", "security": []any{map[string]any{"bearerAuth": []any{}}}},
				"put":        map[string]any{"summary": "Update org", "security": []any{map[string]any{"bearerAuth": []any{}}}},
			},
			"/organizations/{orgId}/agents": map[string]any{
				"parameters": []any{map[string]any{"name": "orgId", "in": "path", "required": true, "schema": map[string]any{"type": "string", "format": "uuid"}}},
				"get":        map[string]any{"summary": "List agents", "security": []any{map[string]any{"bearerAuth": []any{}}}},
				"post":       map[string]any{"summary": "Create agent", "security": []any{map[string]any{"bearerAuth": []any{}}}, "parameters": []any{map[string]any{"$ref": "#/components/parameters/IdempotencyKey"}, map[string]any{"$ref": "#/components/parameters/AURAVersion"}}},
			},
			"/organizations/{orgId}/agents/{agentId}": map[string]any{
				"parameters": []any{
					map[string]any{"name": "orgId", "in": "path", "required": true, "schema": map[string]any{"type": "string", "format": "uuid"}},
					map[string]any{"name": "agentId", "in": "path", "required": true, "schema": map[string]any{"type": "string", "format": "uuid"}},
				},
				"get":    map[string]any{"summary": "Get agent", "security": []any{map[string]any{"bearerAuth": []any{}}}},
				"put":    map[string]any{"summary": "Update agent", "security": []any{map[string]any{"bearerAuth": []any{}}}},
				"delete": map[string]any{"summary": "Delete agent", "security": []any{map[string]any{"bearerAuth": []any{}}}},
			},
			"/organizations/{orgId}/agents/{agentId}/permissions": map[string]any{
				"parameters": []any{
					map[string]any{"name": "orgId", "in": "path", "required": true, "schema": map[string]any{"type": "string", "format": "uuid"}},
					map[string]any{"name": "agentId", "in": "path", "required": true, "schema": map[string]any{"type": "string", "format": "uuid"}},
				},
				"get":  map[string]any{"summary": "List rules", "security": []any{map[string]any{"bearerAuth": []any{}}}},
				"post": map[string]any{"summary": "Add rule", "security": []any{map[string]any{"bearerAuth": []any{}}}, "parameters": []any{map[string]any{"$ref": "#/components/parameters/IdempotencyKey"}, map[string]any{"$ref": "#/components/parameters/AURAVersion"}}},
			},
			"/organizations/{orgId}/agents/{agentId}/permissions/{ruleId}": map[string]any{
				"parameters": []any{
					map[string]any{"name": "orgId", "in": "path", "required": true, "schema": map[string]any{"type": "string", "format": "uuid"}},
					map[string]any{"name": "agentId", "in": "path", "required": true, "schema": map[string]any{"type": "string", "format": "uuid"}},
					map[string]any{"name": "ruleId", "in": "path", "required": true, "schema": map[string]any{"type": "string", "format": "uuid"}},
				},
				"delete": map[string]any{"summary": "Delete rule", "security": []any{map[string]any{"bearerAuth": []any{}}}},
			},
			"/organizations/{orgId}/apikeys": map[string]any{
				"parameters": []any{map[string]any{"name": "orgId", "in": "path", "required": true, "schema": map[string]any{"type": "string", "format": "uuid"}}},
				"get":        map[string]any{"summary": "List API keys", "security": []any{map[string]any{"bearerAuth": []any{}}}},
				"post":       map[string]any{"summary": "Create API key", "security": []any{map[string]any{"bearerAuth": []any{}}}, "parameters": []any{map[string]any{"$ref": "#/components/parameters/IdempotencyKey"}, map[string]any{"$ref": "#/components/parameters/AURAVersion"}}},
			},
			"/organizations/{orgId}/apikeys/{keyId}": map[string]any{
				"parameters": []any{
					map[string]any{"name": "orgId", "in": "path", "required": true, "schema": map[string]any{"type": "string", "format": "uuid"}},
					map[string]any{"name": "keyId", "in": "path", "required": true, "schema": map[string]any{"type": "string", "format": "uuid"}},
				},
				"delete": map[string]any{"summary": "Revoke API key", "security": []any{map[string]any{"bearerAuth": []any{}}}},
			},
			"/organizations/{orgId}/logs": map[string]any{
				"parameters": []any{map[string]any{"name": "orgId", "in": "path", "required": true, "schema": map[string]any{"type": "string", "format": "uuid"}}},
				"get":        map[string]any{"summary": "List event logs", "security": []any{map[string]any{"bearerAuth": []any{}}}},
			},
			"/organizations/{orgId}/webhooks": map[string]any{
				"parameters": []any{map[string]any{"name": "orgId", "in": "path", "required": true, "schema": map[string]any{"type": "string", "format": "uuid"}}},
				"get":        map[string]any{"summary": "List webhook endpoints", "security": []any{map[string]any{"bearerAuth": []any{}}}},
				"post":       map[string]any{"summary": "Create webhook endpoint", "security": []any{map[string]any{"bearerAuth": []any{}}}, "parameters": []any{map[string]any{"$ref": "#/components/parameters/IdempotencyKey"}, map[string]any{"$ref": "#/components/parameters/AURAVersion"}}},
			},
			"/organizations/{orgId}/webhooks/{webhookId}": map[string]any{
				"parameters": []any{
					map[string]any{"name": "orgId", "in": "path", "required": true, "schema": map[string]any{"type": "string", "format": "uuid"}},
					map[string]any{"name": "webhookId", "in": "path", "required": true, "schema": map[string]any{"type": "string", "format": "uuid"}},
				},
				"delete": map[string]any{"summary": "Delete webhook endpoint", "security": []any{map[string]any{"bearerAuth": []any{}}}},
			},
			"/me": map[string]any{
				"get": map[string]any{"summary": "Get current user", "security": []any{map[string]any{"bearerAuth": []any{}}}},
				"put": map[string]any{"summary": "Update current user", "security": []any{map[string]any{"bearerAuth": []any{}}}},
			},
			"/me/password": map[string]any{
				"put": map[string]any{"summary": "Change password", "security": []any{map[string]any{"bearerAuth": []any{}}}},
			},
			"/healthz": map[string]any{"get": map[string]any{"summary": "Liveness"}},
			"/readyz":  map[string]any{"get": map[string]any{"summary": "Readiness"}},
		},
	}
	c.JSON(http.StatusOK, spec)
}

// SwaggerUI serves a lightweight Swagger UI page referencing /openapi.json.
func SwaggerUI(c *gin.Context) {
	html := `<!DOCTYPE html>
<html>
<head>
  <meta charset="utf-8" />
  <meta name="viewport" content="width=device-width, initial-scale=1" />
  <title>AURA API Docs</title>
  <link rel="stylesheet" href="https://unpkg.com/swagger-ui-dist@5/swagger-ui.css" />
  <style>body { margin:0; background:#0b0b0b } .swagger-ui .topbar { display:none }</style>
  </head>
<body>
  <div id="swagger-ui"></div>
  <script src="https://unpkg.com/swagger-ui-dist@5/swagger-ui-bundle.js"></script>
  <script>
    window.ui = SwaggerUIBundle({
      url: '/openapi.json',
      dom_id: '#swagger-ui',
      presets: [SwaggerUIBundle.presets.apis],
      layout: 'BaseLayout'
    });
  </script>
</body>
</html>`
	c.Data(http.StatusOK, "text/html; charset=utf-8", []byte(html))
}
