package main

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"encoding/json"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/redis/go-redis/v9"
	"go.opentelemetry.io/contrib/instrumentation/github.com/gin-gonic/gin/otelgin"

	database "github.com/Armour007/aura-backend/internal"
	"github.com/Armour007/aura-backend/internal/api"
	"github.com/Armour007/aura-backend/internal/audit"
	"github.com/Armour007/aura-backend/internal/mesh"
	"github.com/Armour007/aura-backend/internal/policy"
	"github.com/Armour007/aura-backend/internal/rel"
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
	// Start background codegen worker if queue enabled; manage cancellable context
	if os.Getenv("AURA_QUEUE_ENABLE") != "" {
		wctx, cancel := context.WithCancel(context.Background())
		go api.StartCodegenWorker(wctx)
		// Trap SIGINT/SIGTERM to cancel worker gracefully
		go func() {
			sigc := make(chan os.Signal, 1)
			signal.Notify(sigc, syscall.SIGINT, syscall.SIGTERM)
			<-sigc
			log.Println("signal received, cancelling worker...")
			cancel()
		}()
	}
	// Security headers
	router.Use(api.CSPMiddleware())
	router.Use(api.HSTSMiddlewareFromEnv())
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
	strict := os.Getenv("AURA_CORS_STRICT")
	config := cors.Config{
		AllowAllOrigins:  strict == "", // default permissive in dev; set AURA_CORS_STRICT=1 for strict mode
		AllowMethods:     []string{"GET", "POST", "PUT", "PATCH", "DELETE", "OPTIONS"},
		AllowHeaders:     []string{"Origin", "Content-Length", "Content-Type", "Authorization", "X-API-Key", "X-Request-ID", "Idempotency-Key", "AURA-Version", "X-Aura-Signature", "X-Aura-Timestamp", "X-Aura-Nonce"},
		ExposeHeaders:    []string{"Content-Length", "X-Request-ID"},
		AllowCredentials: false,
		MaxAge:           12 * time.Hour,
	}
	// Strict origins via env (comma-separated) or AURA_FRONTEND_BASE_URL
	if origins := os.Getenv("AURA_CORS_ORIGINS"); origins != "" || os.Getenv("AURA_FRONTEND_BASE_URL") != "" {
		config.AllowAllOrigins = false
		// Build allow list
		parts := []string{}
		if o := os.Getenv("AURA_CORS_ORIGINS"); o != "" {
			parts = append(parts, strings.Split(o, ",")...)
		}
		if f := os.Getenv("AURA_FRONTEND_BASE_URL"); f != "" {
			parts = append(parts, f)
		}
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

	// Background job: deactivate trust keys past their grace deadline
	go func() {
		ticker := time.NewTicker(30 * time.Second)
		defer ticker.Stop()
		for range ticker.C {
			if database.DB != nil {
				_, _ = database.DB.Exec(`UPDATE trust_keys SET active=false, deactivate_after=NULL WHERE active=true AND deactivate_after IS NOT NULL AND deactivate_after <= NOW()`)
			}
		}
	}()

	// Background job: periodically anchor federation gossip head per topic
	go func() {
		anchorOrg := strings.TrimSpace(os.Getenv("AURA_FEDERATION_ANCHOR_ORG_ID"))
		if anchorOrg == "" {
			return
		}
		ticker := time.NewTicker(5 * time.Minute)
		defer ticker.Stop()
		for range ticker.C {
			if database.DB == nil {
				continue
			}
			for _, topic := range []string{"revocation", "key_rotation", "org_registry"} {
				var head string
				_ = database.DB.Get(&head, `SELECT hash FROM federation_gossip WHERE topic=$1 ORDER BY ts DESC, created_at DESC LIMIT 1`, topic)
				if head == "" {
					continue
				}
				date := time.Now().UTC().Format("2006-01-02")
				ext := "federation:" + topic
				// Upsert anchor similar to SetAuditAnchor
				_, err := database.DB.Exec(`INSERT INTO audit_anchors(org_id, anchor_date, root_hash, external_ref) VALUES ($1,$2,$3,$4)
					ON CONFLICT (org_id, anchor_date) DO UPDATE SET root_hash=EXCLUDED.root_hash, external_ref=EXCLUDED.external_ref`, anchorOrg, date, head, ext)
				if err == nil {
					_ = audit.Append(context.Background(), uuid.MustParse(anchorOrg), "audit_anchor_set", gin.H{"date": date, "root_hash": head, "external_ref": ext}, nil, nil)
				}
			}
		}
	}()

	// Optional: background publish to peers
	go func() {
		if os.Getenv("AURA_GOSSIP_PUBLISH_ENABLE") != "1" {
			return
		}
		client := &http.Client{Timeout: 5 * time.Second}
		ticker := time.NewTicker(1 * time.Minute)
		defer ticker.Stop()
		for range ticker.C {
			if database.DB == nil {
				continue
			}
			// Load peers
			type peer struct {
				URL string `db:"url"`
			}
			peers := []peer{}
			_ = database.DB.Select(&peers, `SELECT url FROM federation_peers`)
			if len(peers) == 0 {
				continue
			}
			// Gather recent messages (last minute) in batches
			var rows []struct {
				OrgID   string    `db:"org_id"`
				Topic   string    `db:"topic"`
				Ts      time.Time `db:"ts"`
				Nonce   string    `db:"nonce"`
				Payload []byte    `db:"payload"`
				JWS     string    `db:"jws"`
			}
			_ = database.DB.Select(&rows, `SELECT org_id::text, topic, ts, nonce, payload, jws FROM federation_gossip WHERE ts > NOW() - INTERVAL '1 minute' ORDER BY ts ASC LIMIT 500`)
			if len(rows) == 0 {
				continue
			}
			// Prepare payload
			type msg struct {
				OrgID   string          `json:"org_id"`
				Topic   string          `json:"topic"`
				Ts      time.Time       `json:"ts"`
				Nonce   string          `json:"nonce"`
				Payload json.RawMessage `json:"payload"`
				JWS     string          `json:"jws"`
			}
			batch := make([]msg, 0, len(rows))
			for _, r := range rows {
				batch = append(batch, msg{OrgID: r.OrgID, Topic: r.Topic, Ts: r.Ts, Nonce: r.Nonce, Payload: json.RawMessage(r.Payload), JWS: r.JWS})
			}
			body, _ := json.Marshal(map[string]any{"messages": batch})
			for _, p := range peers {
				req, _ := http.NewRequest(http.MethodPost, strings.TrimRight(p.URL, "/")+"/v2/federation/gossip", strings.NewReader(string(body)))
				req.Header.Set("Content-Type", "application/json")
				// best-effort send; ignore errors
				_, _ = client.Do(req)
			}

			// Attempt to auto-merge when multiple heads exist for a topic to aid convergence
			anchorOrg := strings.TrimSpace(os.Getenv("AURA_FEDERATION_ANCHOR_ORG_ID"))
			if anchorOrg != "" {
				for _, topic := range []string{"revocation", "key_rotation", "org_registry"} {
					var heads []string
					_ = database.DB.Select(&heads, `
						SELECT g1.hash FROM federation_gossip g1
						WHERE g1.topic=$1 AND NOT EXISTS (
							SELECT 1 FROM federation_gossip g2
							WHERE g2.topic=$1 AND (g2.prev_hash=g1.hash OR (g2.parents IS NOT NULL AND g1.hash = ANY(g2.parents)))
						) ORDER BY g1.ts DESC LIMIT 5`, topic)
					if len(heads) >= 2 {
						payload := json.RawMessage(`{"merge":true}`)
						_, _, _, _, _ = api.CreateMergeGossip(context.Background(), anchorOrg, topic, heads, payload, "")
					}
				}
			}
		}
	}()

	// Initialize Trust Graph client (SpiceDB via env, else local) with TTL cache
	{
		posTTL := 2 * time.Second
		negTTL := 500 * time.Millisecond
		if v := os.Getenv("AURA_REL_CACHE_TTL_MS"); v != "" {
			if d, err := time.ParseDuration(v + "ms"); err == nil {
				posTTL = d
			}
		}
		if v := os.Getenv("AURA_REL_NEG_CACHE_TTL_MS"); v != "" {
			if d, err := time.ParseDuration(v + "ms"); err == nil {
				negTTL = d
			}
		}
		backend := os.Getenv("AURA_REL_BACKEND")
		var inner rel.GraphClient
		if backend == "spicedb" {
			ep := os.Getenv("AURA_SPICEDB_ENDPOINT")
			tok := os.Getenv("AURA_SPICEDB_TOKEN")
			if ep != "" && tok != "" {
				if gc, err := rel.NewSpiceDBFromEnv(ep, tok); err == nil {
					inner = gc
					log.Println("Trust Graph: using SpiceDB backend")
				} else {
					log.Printf("Trust Graph: SpiceDB init failed (%v), falling back to local", err)
				}
			}
		}
		if inner == nil {
			inner = rel.NewLocalGraph()
			log.Println("Trust Graph: using local SQL backend")
		}
		api.SetGraphClient(api.NewCachedGraph(inner, posTTL, negTTL))
	}

	// Initialize Mesh Bus (NATS via env when built with nats tag; else LocalBus)
	{
		var b mesh.Bus
		if url := os.Getenv("AURA_NATS_URL"); url != "" {
			if nb, err := mesh.NewNatsBus(url); err == nil {
				b = nb
				log.Println("Mesh Bus: using NATS backend")
			} else {
				log.Printf("Mesh Bus: NATS init failed (%v), falling back to local", err)
			}
		}
		if b == nil {
			b = mesh.NewLocalBus()
			log.Println("Mesh Bus: using local backend")
		}
		api.SetBus(b)
		// Subscriptions: graph.invalidate => ClearGraphCache; policy.invalidate => DeleteCompiled
		_, _ = b.Subscribe(mesh.TopicGraphInvalidate, func(ctx context.Context, e mesh.Event) {
			api.ClearGraphCache()
		})
		_, _ = b.Subscribe(mesh.TopicPolicyInvalidate, func(ctx context.Context, e mesh.Event) {
			var pl struct {
				PolicyID string `json:"policy_id"`
			}
			_ = json.Unmarshal(e.Payload, &pl)
			if pl.PolicyID != "" {
				if id, err := uuid.Parse(pl.PolicyID); err == nil {
					policy.DeleteCompiled(id, 0)
				}
			}
		})
	}
	// --- Public Routes (No Auth Needed) ---
	authRoutes := router.Group("/auth")
	{
		authRoutes.POST("/register", api.RegisterUser)
		authRoutes.POST("/login", api.LoginUser)
		// SPIFFE-based attestation token minting (guarded via env)
		authRoutes.POST("/attest", api.Attest)
	}
	coreRoutes := router.Group("/v1")
	coreRoutes.Use(api.AttestOrAPIKeyAuthMiddleware())
	// Optional HMAC request signing (no-op if AURA_REQUEST_HMAC_SECRET unset)
	coreRoutes.Use(api.RequestSigningMiddleware())
	// Apply rate limiting to core verification endpoints (env-configurable, optional Redis)
	coreRoutes.Use(api.RateLimitMiddlewareFromEnv())
	// Optional mTLS enforcement for v1 core routes
	if os.Getenv("AURA_MTLS_REQUIRE_V1") == "1" {
		coreRoutes.Use(api.AgentCertBindingMiddleware())
	}
	{
		// /v1/verify uses API Key Auth via middleware above
		coreRoutes.POST("/verify", api.HandleVerifyRequest)
		// Agent CSR issue (MVP): protected by attestation or API key; returns client cert
		coreRoutes.POST("/agents/:agentId/csr", api.AcceptAgentCSR)
		// Trust Graph v1 endpoints (batch tuples, relation check, expand)
		coreRoutes.POST("/tuples", api.UpsertTuplesV1)
		coreRoutes.POST("/check", api.CheckRelationV1)
		coreRoutes.GET("/trust/graph/expand", api.ExpandTrustGraphV1)
	}

	// Experimental v2 verification with policy/relationship prototype
	v2 := router.Group("/v2")
	v2.Use(api.AttestOrAPIKeyAuthMiddleware())
	v2.Use(api.RequestSigningMiddleware())
	v2.Use(api.RateLimitMiddlewareFromEnv())
	// Optional mTLS enforcement for v2 endpoints
	if os.Getenv("AURA_MTLS_REQUIRE_V2") == "1" {
		v2.Use(api.AgentCertBindingMiddleware())
	}
	{
		v2.POST("/verify", api.HandleVerifyV2)
		v2.GET("/decisions/search", api.GetRecentDecisionTraces)
		v2.GET("/decisions/:traceId", api.GetDecisionTrace)
		// Cognitive Firewall endpoints
		v2.POST("/guard", api.InlineGuard)
		v2.POST("/policy/author/nl-compile", api.CompilePolicyFromNL)
		v2.POST("/policy/tests/run", api.RunPolicyTests)
		v2.POST("/policy/preview", api.PreviewPolicyAgainstTraces)
		// Attestation and certs
		v2.POST("/attest", api.HandleAttest)
		v2.POST("/certs/issue", api.IssueClientCert)
		v2.GET("/certs", api.ListClientCerts)
		v2.POST("/certs/:serial/revoke", api.RevokeClientCert)
		v2.GET("/certs/crl.pem", api.GetCRL)
		// Runtime approvals polling
		v2.GET("/approvals/:traceId", api.GetApprovalStatus)
		v2.GET("/signals/risk", api.GetRiskSignals)
		v2.POST("/signals/risk/alerts", api.RaiseRiskAlert)
		v2.DELETE("/signals/risk/alerts", api.ClearRiskAlert)
		v2.GET("/reputation", api.GetReputation)
		v2.GET("/policy/recommendations", api.GetPolicyRecommendations)
		v2.GET("/policy/packs", api.ListPolicyPacks)
		v2.GET("/policy/packs/:packId", api.GetPolicyPack)
		v2.GET("/policies/:policyId/versions", api.ListPolicyVersionsV2)
		v2.GET("/audit/ledger", api.GetAuditLedger)
		v2.GET("/audit/verify", api.VerifyAuditChain)
		v2.POST("/audit/anchor", api.SetAuditAnchor)
		v2.GET("/audit/anchor", api.GetAuditAnchor)
		v2.GET("/network/info", api.GetNetworkInfo)
		v2.POST("/tokens/introspect", api.IntrospectTrustToken)
		v2.POST("/federation/contracts", api.CreateFederationContract)
		v2.GET("/federation/contracts", api.ListFederationContracts)
		v2.POST("/federation/events", api.RecordFederationBoundaryEvent)
		v2.POST("/federation/delegations", api.CreateFederationDelegation)

		// Federation gossip protocol (minimal proofs + hash-chaining)
		v2.POST("/federation/gossip", api.IngestGossip)
		v2.GET("/federation/gossip", api.PullGossip)
		v2.POST("/federation/gossip/publish", api.PublishGossip)
		v2.POST("/federation/gossip/merge", api.MergeGossip)

		// Federation peers management
		v2.GET("/federation/peers", api.ListFederationPeers)
		v2.POST("/federation/peers", api.AddFederationPeer)
		v2.DELETE("/federation/peers/:peerId", api.DeleteFederationPeer)

		// Trust Registry
		v2.POST("/registry/orgs", api.CreateRegistryOrg)
		v2.GET("/registry/orgs", api.ListRegistryOrgs)
		v2.POST("/registry/agents", api.CreateRegistryAgent)
		v2.GET("/registry/agents", api.ListRegistryAgents)
	}

	// Public endpoints with rate limiting: JWKS and public registry
	public := router.Group("")
	public.Use(api.RateLimitPublicMiddlewareFromEnv())
	{
		// Well-known JWKS for trust token verification (public)
		public.GET("/.well-known/aura-jwks.json", api.JWKS)
		// Org-scoped JWKS
		public.GET("/.well-known/aura/:orgId/jwks.json", api.OrgJWKS)
		// DID resolver for did:aura:org:<orgId>
		public.GET("/resolve", api.ResolveDID)
		public.GET("/resolve/:did", api.ResolveDID)
		// Public Trust Registry
		public.GET("/registry/public/orgs", api.PublicListRegistryOrgs)
		public.GET("/registry/public/agents", api.PublicListRegistryAgents)
		// Public AURA-ID registry endpoint (rate-limited)
		public.GET("/aura-id/:id", api.GetAuraID)
		// Slack interactive webhook for runtime approvals (no auth; restrict via Slack app config)
		public.POST("/v2/approvals/slack", api.SlackApprovalWebhook)
		// SSO entry points (stubbed unless AURA_SSO_ENABLE=1)
		public.GET("/sso/:provider/login", api.SSOLogin)
		public.GET("/sso/:provider/callback", api.SSOCallback)
	}

	// --- Protected Routes (Require User JWT Auth) ---
	// Create a new group for routes that need authentication
	// Health and readiness
	// API Docs
	router.GET("/metrics", func(c *gin.Context) { promhttp.Handler().ServeHTTP(c.Writer, c.Request) })
	router.GET("/openapi.json", api.OpenAPIJSON)
	router.GET("/docs", api.SwaggerUI)
	router.GET("/healthz", func(c *gin.Context) { c.Status(200) })
	router.GET("/readyz", func(c *gin.Context) {
		ctx, cancel := context.WithTimeout(c.Request.Context(), 300*time.Millisecond)
		defer cancel()
		if err := database.DB.DB.PingContext(ctx); err != nil {
			c.JSON(503, gin.H{"status": "not ready", "error": err.Error()})
			return
		}
		// If queue is enabled, require Redis to be reachable
		if os.Getenv("AURA_QUEUE_ENABLE") != "" {
			addr := os.Getenv("AURA_REDIS_ADDR")
			if addr == "" {
				addr = os.Getenv("REDIS_ADDR")
			}
			if addr == "" {
				c.JSON(503, gin.H{"status": "not ready", "error": "redis addr not configured"})
				return
			}
			rdb := redis.NewClient(&redis.Options{Addr: addr, Password: os.Getenv("AURA_REDIS_PASSWORD")})
			rctx, rcancel := context.WithTimeout(c.Request.Context(), 300*time.Millisecond)
			defer rcancel()
			if err := rdb.Ping(rctx).Err(); err != nil {
				c.JSON(503, gin.H{"status": "not ready", "error": "redis ping failed"})
				_ = rdb.Close()
				return
			}
			_ = rdb.Close()
		}
		c.JSON(200, gin.H{"status": "ready"})
	})

	// Public signed download for generated SDKs (no auth) when signed URL is present
	router.GET("/sdk/public/download-generated/:jobId", api.DownloadGeneratedSDKPublic)

	// Identities (AURA-ID) - v1 convenience aliases
	coreRoutes.POST("/identities", api.CreateAuraID)
	coreRoutes.GET("/identities/:id", api.GetAuraID)
	// Trust Tokens (issue/verify/revoke)
	coreRoutes.POST("/token/issue", api.IssueTrustTokenV1)
	coreRoutes.POST("/token/verify", api.VerifyTrustTokenV1)
	coreRoutes.POST("/token/revoke", api.RevokeTrustTokenV1)
	// SDK utilities (protected): download curated SDK bundles
	// These endpoints require user authentication
	// Example: GET /sdk/download?lang=node|python|go
	// Response: application/zip attachment with the selected SDK bundle
	protectedSDK := router.Group("/sdk")
	protectedSDK.Use(api.AuthMiddleware())
	{
		protectedSDK.GET("/download", api.DownloadSDK)
		protectedSDK.GET("/supported-langs", api.GetSupportedLangs)
		protectedSDK.POST("/generate", api.GenerateSDK)
		protectedSDK.GET("/generate/:jobId", api.GetSDKJob)
		protectedSDK.GET("/download-generated/:jobId", api.DownloadGeneratedSDK)
	}

	// Admin utilities
	admin := router.Group("/admin")
	admin.Use(api.AuthMiddleware())
	{
		admin.POST("/test-smtp", api.TestSMTP)
		admin.GET("/health", api.AdminHealth)
		admin.POST("/queue/drain", api.QueueDrain)
		admin.GET("/queue/drain/status", api.QueueDrainStatus)
		admin.GET("/queue/drain/complete", api.QueueDrainComplete)
		admin.GET("/queue/dlq", api.ListDLQ)
		admin.POST("/queue/dlq/requeue", api.RequeueDLQ)
		admin.POST("/queue/dlq/delete", api.DeleteDLQ)
		admin.GET("/webhooks/dlq", api.ListWebhookDLQ)
		admin.POST("/webhooks/dlq/requeue", api.RequeueWebhookDLQ)
		admin.POST("/webhooks/dlq/delete", api.DeleteWebhookDLQ)
		// Devices (read-only list)
		admin.GET("/devices", api.AdminListDevices)
		admin.GET("/devices/detail", api.AdminListDevicesDetail)
		// Trust tuple admin utilities
		admin.GET("/rel/tuples", api.AdminListTuples)
		admin.DELETE("/rel/tuples", api.AdminDeleteTuples)
	}

	protectedRoutes := router.Group("/")
	protectedRoutes.Use(api.AuthMiddleware()) // Apply the middleware HERE
	// Apply CSRF for state-changing dashboard calls if enabled
	protectedRoutes.Use(api.CSRFMiddlewareFromEnv())
	{
		// User profile endpoints
		protectedRoutes.GET("/me", api.GetMe)
		protectedRoutes.PUT("/me", api.UpdateMe)
		protectedRoutes.PUT("/me/password", api.UpdatePassword)

		protectedRoutes.GET("/organizations/mine", api.GetMyOrganizations)
		// All routes defined within this group will now require a valid JWT
		orgRoutes := protectedRoutes.Group("/organizations/:orgId")
		orgRoutes.Use(api.OrgMemberMiddleware())
		// Enforce least-privilege for non-admin roles on state-changing actions
		orgRoutes.Use(api.ReadOnlyGuardMiddleware())
		// Apply idempotency for POST requests with Idempotency-Key header
		orgRoutes.Use(api.IdempotencyMiddlewareFromEnv())
		{
			// Organization settings (admin only)
			orgRoutes.GET("", api.GetOrganizationByID)
			orgRoutes.PUT("", api.RequireOrgAdmin(), api.UpdateOrganization)
			orgRoutes.PUT("/settings", api.RequireOrgAdmin(), api.UpdateOrganizationSettings)

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

			// Policy prototype endpoints under org
			polRoutes := orgRoutes.Group("/policies")
			{
				polRoutes.POST("", api.RequireOrgAdmin(), api.CreatePolicy)
				polRoutes.POST(":policyId/versions", api.RequireOrgAdmin(), api.AddPolicyVersion)
				polRoutes.POST(":policyId/versions/:version/approve", api.RequireOrgAdmin(), api.ApprovePolicyVersion)
				polRoutes.POST(":policyId/assignments", api.RequireOrgAdmin(), api.AssignPolicy)
				polRoutes.POST(":policyId/versions/:version/simulate", api.RequireOrgAdmin(), api.SimulatePolicyVersion)
				polRoutes.POST(":policyId/versions/:version/activate", api.RequireOrgAdmin(), api.ActivatePolicyVersion)
				polRoutes.GET(":policyId/versions", api.RequireOrgAdmin(), api.ListPolicyVersions)
			}

			// Relationship prototype endpoints
			relRoutes := orgRoutes.Group("/rel")
			{
				relRoutes.POST("/tuples", api.RequireOrgAdmin(), api.UpsertTuples)
				relRoutes.POST("/check", api.CheckRelation)
			}

			apiKeyRoutes := orgRoutes.Group("/apikeys")
			{
				apiKeyRoutes.POST("", api.RequireOrgAdmin(), api.CreateAPIKey)
				apiKeyRoutes.GET("", api.RequireOrgAdmin(), api.GetAPIKeys)
				apiKeyRoutes.DELETE("/:keyId", api.RequireOrgAdmin(), api.DeleteAPIKey)
				apiKeyRoutes.POST("/:keyId/rotate", api.RequireOrgAdmin(), api.RotateAPIKey)
			}

			// Trust keys management (admin)
			tk := orgRoutes.Group("/trust-keys")
			{
				tk.GET("", api.RequireOrgAdmin(), api.ListTrustKeys)
				tk.POST("", api.RequireOrgAdmin(), api.CreateTrustKey)
				tk.POST("/rotate", api.RequireOrgAdmin(), api.RotateTrustKey)
				tk.POST("/:keyId/activate", api.RequireOrgAdmin(), api.ActivateTrustKey)
				tk.POST("/:keyId/deactivate", api.RequireOrgAdmin(), api.DeactivateTrustKey)
			}

			// Trust token revocations
			revRoutes := orgRoutes.Group("/trust-tokens")
			{
				revRoutes.GET("/revocations", api.RequireOrgAdmin(), api.GetRevocations)
				revRoutes.POST("/revocations", api.RequireOrgAdmin(), api.RevokeTrustToken)
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

			// AURA-ID issuance and listing (admin)
			orgRoutes.POST("/aura-identities", api.RequireOrgAdmin(), api.CreateAuraID)
			orgRoutes.GET("/aura-identities", api.RequireOrgAdmin(), api.ListAuraIDs)

			// Trust DNA & reputation graph (admin)
			orgRoutes.POST("/trust-dna", api.RequireOrgAdmin(), api.SubmitTrustDNA)
			// Regulator view endpoints (admin)
			orgRoutes.GET("/regulator/snapshot", api.RequireOrgAdmin(), api.RegulatorSnapshot)
			orgRoutes.GET("/regulator/audit-bundle", api.RequireOrgAdmin(), api.ExportAuditBundle)
			orgRoutes.GET("/regulator/compliance-mapping", api.RequireOrgAdmin(), api.GetComplianceMapping)
			orgRoutes.GET("/regulator/audit-export", api.RequireOrgAdmin(), api.ExportAuditData)
			orgRoutes.POST("/regulator/audit-export/schedule", api.RequireOrgAdmin(), api.SetAuditExportSchedule)
			orgRoutes.GET("/regulator/audit-export/schedule", api.RequireOrgAdmin(), api.GetAuditExportSchedule)
			orgRoutes.POST("/trust-dna/near", api.RequireOrgAdmin(), api.GetTrustDNANear)
			orgRoutes.GET("/trust-dna/aggregate", api.RequireOrgAdmin(), api.GetTrustDNAAggregate)
		}
		// Add other protected routes here if needed (e.g., /user/profile)
	}

	// SCIM API (token-protected)
	scim := router.Group("/scim/v2")
	{
		scim.GET("/Users", api.SCIMListUsers)
		scim.POST("/Users", api.SCIMCreateUser)
		scim.PATCH("/Users/:id", api.SCIMPatchUser)
		scim.GET("/Groups", api.SCIMListGroups)
	}

	// Optional TLS with client auth
	certFile := os.Getenv("AURA_TLS_CERT_FILE")
	keyFile := os.Getenv("AURA_TLS_KEY_FILE")
	clientCAFile := os.Getenv("AURA_CLIENT_CA_FILE")
	clientAuthMode := os.Getenv("AURA_TLS_CLIENT_AUTH") // require|verify|off
	if certFile != "" && keyFile != "" {
		srv := &http.Server{Addr: ":" + port, Handler: router}
		// TLS config
		cfg := &tls.Config{}
		if clientCAFile != "" {
			caBytes, err := ioutil.ReadFile(clientCAFile)
			if err == nil {
				pool := x509.NewCertPool()
				if pool.AppendCertsFromPEM(caBytes) {
					cfg.ClientCAs = pool
				}
			}
		}
		switch strings.ToLower(clientAuthMode) {
		case "require":
			cfg.ClientAuth = tls.RequireAndVerifyClientCert
		case "verify":
			cfg.ClientAuth = tls.VerifyClientCertIfGiven
		default:
			cfg.ClientAuth = tls.NoClientCert
		}
		srv.TLSConfig = cfg
		log.Println("HTTPS server with TLS enabled on :" + port)
		err := srv.ListenAndServeTLS(certFile, keyFile)
		if err != nil {
			log.Fatal("Failed to start TLS server:", err)
		}
		return
	}
	// Start background audit scheduler if configured
	api.StartAuditScheduler()

	err := router.Run(":" + port)
	if err != nil {
		log.Fatal("Failed to start server:", err)
	}
}
