package api

import (
	"context"
	"crypto/ed25519"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"regexp"
	"strconv"
	"strings"
	"time"

	database "github.com/Armour007/aura-backend/internal"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"github.com/Armour007/aura-backend/internal/attest"
	"github.com/Armour007/aura-backend/internal/audit"
	kms "github.com/Armour007/aura-backend/internal/crypto"
	"github.com/Armour007/aura-backend/internal/policy"
	polrepo "github.com/Armour007/aura-backend/internal/policy"
	"github.com/Armour007/aura-backend/internal/rel"
	"github.com/Armour007/aura-backend/internal/risk"
	"github.com/Armour007/aura-backend/internal/utils"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
)

// VerifyV2Request allows richer input but stays optional for prototype
type VerifyV2Request struct {
	AgentID           uuid.UUID       `json:"agent_id"`
	Action            string          `json:"action,omitempty"`
	Resource          string          `json:"resource,omitempty"`
	RequestContext    json.RawMessage `json:"request_context"`
	TargetOrgID       string          `json:"target_org_id,omitempty"`
	IncludeTrustToken bool            `json:"include_trust_token,omitempty"`
}

type VerifyV2Response struct {
	Allow   bool   `json:"allow"`
	Reason  string `json:"reason,omitempty"`
	TraceID string `json:"trace_id,omitempty"`
	Token   string `json:"token,omitempty"`
}

// Risk tracker singleton for prototype
var rt = risk.NewTracker(30*time.Second, parseRiskThreshold())

func getRiskTracker() *risk.Tracker { return rt }

func parseRiskThreshold() int {
	if v := os.Getenv("AURA_RISK_SPIKE_THRESHOLD"); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n > 0 {
			return n
		}
	}
	return 50
}

// mergeSignals injects risk signals into the request context JSON
func mergeSignals(input json.RawMessage, sig risk.Signals) json.RawMessage {
	if len(input) == 0 {
		m := map[string]any{"risk": map[string]any{"score": sig.Score, "flags": sig.Flags}}
		b, _ := json.Marshal(m)
		return b
	}
	var m map[string]any
	if err := json.Unmarshal(input, &m); err != nil {
		// if not an object, wrap into one
		m = map[string]any{"_": input}
	}
	m["risk"] = map[string]any{"score": sig.Score, "flags": sig.Flags}
	b, _ := json.Marshal(m)
	return b
}

func HandleVerifyV2(c *gin.Context) {
	// Backpressure: simple inflight limiter
	if !acquireVerifySlot() {
		IncVerifyQuickReject()
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "verify overloaded"})
		return
	}
	defer releaseVerifySlot()

	ctx, span := otel.Tracer("aura-backend").Start(c.Request.Context(), "verify")
	defer span.End()

	orgID := c.GetString("orgID")
	var req VerifyV2Request
	if err := c.ShouldBindJSON(&req); err != nil {
		span.SetStatus(codes.Error, "bad_request")
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Principal (prototype): from headers or fallback to provided agent
	pr := attest.FromRequest(c.Request, orgID, req.AgentID.String())

	// Policy selection: support multiple assignments with deterministic bucketing
	assignCtx, assignSpan := otel.Tracer("aura-backend").Start(ctx, "db.get_active_assignments")
	assignments, err := polrepo.GetActiveAssignmentsForOrg(assignCtx, uuid.MustParse(orgID))
	assignSpan.End()
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, "db_error_assignments")
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	if len(assignments) == 0 {
		c.JSON(http.StatusOK, VerifyV2Response{Allow: false, Reason: "No active policy assignment"})
		return
	}
	// Choose which policy to evaluate using a stable bucket on (org, agent, action, resource)
	agentStrForSel := req.AgentID.String()
	if agentStrForSel == uuid.Nil.String() {
		agentStrForSel = c.GetString("agentID")
	}
	selKey := orgID + "|" + agentStrForSel + "|" + req.Action + "|" + req.Resource
	idx := bucket(selKey, "") % len(assignments)
	p := &assignments[idx].Policy
	v := &assignments[idx].Version

	// Federation boundary, contract scope enforcement, and relationship checks
	// If TargetOrgID is set and differs from caller org, require an active federation contract and enforce scope
	var fedScope struct {
		AllowedActions   []string `json:"allowed_actions"`
		AllowedResources []string `json:"allowed_resources"`
	}
	if req.TargetOrgID != "" && req.TargetOrgID != orgID {
		var scope json.RawMessage
		if err := database.DB.Get(&scope, `SELECT scope FROM federation_contracts WHERE org_id=$1 AND counterparty_org_id=$2 AND active=true ORDER BY created_at DESC LIMIT 1`, orgID, req.TargetOrgID); err != nil || len(scope) == 0 {
			_ = audit.Append(c.Request.Context(), uuid.MustParse(orgID), "federation_boundary_crossing", gin.H{"to_org_id": req.TargetOrgID, "allowed": false, "reason": "no_contract"}, nil, nil)
			c.JSON(http.StatusOK, VerifyV2Response{Allow: false, Reason: "No federation contract"})
			return
		}
		_ = json.Unmarshal(scope, &fedScope)
		// Enforce action scope if provided
		if req.Action != "" && len(fedScope.AllowedActions) > 0 && !matchesAllowed(req.Action, fedScope.AllowedActions) {
			_ = audit.Append(c.Request.Context(), uuid.MustParse(orgID), "federation_scope_denied", gin.H{"to_org_id": req.TargetOrgID, "reason": "action_not_allowed", "action": req.Action}, nil, nil)
			c.JSON(http.StatusOK, VerifyV2Response{Allow: false, Reason: "Federation: action not allowed by contract"})
			return
		}
		// Enforce resource scope if provided
		if req.Resource != "" && len(fedScope.AllowedResources) > 0 && !matchesAllowed(req.Resource, fedScope.AllowedResources) {
			_ = audit.Append(c.Request.Context(), uuid.MustParse(orgID), "federation_scope_denied", gin.H{"to_org_id": req.TargetOrgID, "reason": "resource_not_allowed", "resource": req.Resource}, nil, nil)
			c.JSON(http.StatusOK, VerifyV2Response{Allow: false, Reason: "Federation: resource not allowed by contract"})
			return
		}
		_ = audit.Append(c.Request.Context(), uuid.MustParse(orgID), "federation_boundary_crossing", gin.H{"to_org_id": req.TargetOrgID, "allowed": true}, nil, nil)
		// Zero trust cross-org: require attestation auth if enabled
		if os.Getenv("AURA_ZERO_TRUST_CROSS_ORG") == "1" {
			if kind := c.GetString("authKind"); kind != "attest" {
				c.JSON(http.StatusOK, VerifyV2Response{Allow: false, Reason: "Cross-org requires attestation auth"})
				return
			}
		}
	}

	// Relationship check via Graph: if provided resource, example gate: agent can_act_for org (or target org when set)
	if req.Resource != "" {
		relOrg := orgID
		if req.TargetOrgID != "" {
			relOrg = req.TargetOrgID
		}
		gctx, gspan := otel.Tracer("aura-backend").Start(ctx, "graph.check")
		allowed, _, err := getGraph().Check(gctx,
			rel.RelationRef{Namespace: "agent", ObjectID: pr.AgentID},
			"can_act_for",
			rel.RelationRef{Namespace: "org", ObjectID: relOrg},
		)
		if err != nil || !allowed {
			if err != nil {
				gspan.RecordError(err)
			}
			gspan.End()
			c.JSON(http.StatusOK, VerifyV2Response{Allow: false, Reason: "No delegation to act for org"})
			return
		}
		gspan.End()
	}

	// Canary rollout selection: if a rollout is active for this policy, bucket the agent deterministically
	var rollout struct {
		Version int `db:"version"`
		Percent int `db:"percent"`
	}
	_, rollSpan := otel.Tracer("aura-backend").Start(ctx, "db.get_policy_rollout")
	if err := database.DB.Get(&rollout, `SELECT version, percent FROM policy_rollouts WHERE org_id=$1 AND policy_id=$2 AND active=true ORDER BY created_at DESC LIMIT 1`, orgID, p.ID); err == nil && rollout.Percent > 0 {
		// Determine agent string for bucketing
		agentStr := req.AgentID.String()
		if agentStr == uuid.Nil.String() {
			agentStr = c.GetString("agentID")
		}
		if bucket(orgID, agentStr, p.ID.String()) < rollout.Percent {
			// switch to rollout version
			if rv, err := polrepo.GetVersion(c.Request.Context(), p.ID, rollout.Version); err == nil {
				v = &rv
			}
		}
	}
	rollSpan.End()

	// Record risk hit and compute runtime signals
	agentStr := req.AgentID.String()
	if agentStr == uuid.Nil.String() {
		agentStr = c.GetString("agentID")
	}
	signals := getRiskTracker().Get(orgID, agentStr, time.Now())
	mergedCtx := mergeSignals(req.RequestContext, signals)
	// Inject federation scope and counterparty for policy evaluation when applicable
	if req.TargetOrgID != "" && req.TargetOrgID != orgID {
		var m map[string]any
		if len(mergedCtx) == 0 {
			m = map[string]any{}
		} else if err := json.Unmarshal(mergedCtx, &m); err != nil {
			m = map[string]any{"_": string(mergedCtx)}
		}
		m["federation"] = map[string]any{
			"counterparty_org_id": req.TargetOrgID,
			"scope": map[string]any{
				"allowed_actions":   fedScope.AllowedActions,
				"allowed_resources": fedScope.AllowedResources,
			},
		}
		if b, err := json.Marshal(m); err == nil {
			mergedCtx = b
		}
	}

	// compile policy version body and evaluate
	e := evalRegistry[p.EngineType]
	if e == nil {
		c.JSON(http.StatusOK, VerifyV2Response{Allow: false, Reason: "Unsupported engine"})
		return
	}
	var comp policy.CompiledPolicy
	if cpc, ok := policy.GetCompiled(v.PolicyID, v.Version); ok {
		comp = cpc
	} else {
		_, compSpan := otel.Tracer("aura-backend").Start(ctx, "policy.compile")
		cp, err := e.Compile(v.Body)
		compSpan.End()
		if err != nil {
			span.RecordError(err)
			span.SetStatus(codes.Error, "compile_error")
			c.JSON(http.StatusOK, VerifyV2Response{Allow: false, Reason: err.Error()})
			return
		}
		policy.PutCompiled(v.PolicyID, v.Version, cp)
		comp = cp
	}
	// Canonicalize context for stable token hashing; use canonicalized for eval too to keep parity
	canonCtx := utils.CanonicalizeJSON(mergedCtx)
	_, evalSpan := otel.Tracer("aura-backend").Start(ctx, "policy.evaluate")
	dec, err := e.Evaluate(comp, canonCtx)
	evalSpan.End()
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, "eval_error")
		c.JSON(http.StatusOK, VerifyV2Response{Allow: false, Reason: err.Error()})
		return
	}

	// Enrich trace with policy and principal context
	if dec.Trace != nil {
		dec.Trace.PolicyID = v.PolicyID
		dec.Trace.PolicyVersion = v.Version
		dec.Trace.Principal = &policy.PrincipalTrace{
			OrgID:           pr.OrgID,
			AgentID:         pr.AgentID,
			SPIFFEID:        pr.SPIFFEID,
			AuthnKind:       pr.AuthnKind,
			CertFingerprint: pr.CertFingerprint,
		}
	}

	RecordDecision(map[bool]string{true: "ALLOWED", false: "DENIED"}[dec.Allow], orgID)
	RecordDecisionReason(dec.Reason, map[bool]string{true: "ALLOWED", false: "DENIED"}[dec.Allow], orgID)

	// Update simple reputation model: if denied with rate_spike flag, increment violations and adjust reliability
	go func() {
		if !dec.Allow && dec.Trace != nil && dec.Trace.Principal != nil {
			flags := signals.Flags
			seenSpike := false
			for _, f := range flags {
				if f == "rate_spike" {
					seenSpike = true
					break
				}
			}
			if seenSpike {
				// upsert agent_reputation
				_, _ = database.DB.Exec(`INSERT INTO agent_reputation(org_id, agent_id, violations, last_violation_at, reliability_score)
					VALUES ($1,$2,1,NOW(),95)
					ON CONFLICT (org_id, agent_id)
					DO UPDATE SET violations=agent_reputation.violations+1, last_violation_at=NOW(), reliability_score=GREATEST(agent_reputation.reliability_score-5,0)`,
					uuid.MustParse(orgID), uuid.MustParse(dec.Trace.Principal.AgentID))
			}
		}
	}()
	// persist decision trace for retrieval
	go func() {
		var traceJSON json.RawMessage
		if dec.Trace != nil {
			if b, err := json.Marshal(dec.Trace); err == nil {
				traceJSON = b
			}
		}
		var agentPtr *uuid.UUID
		if req.AgentID != uuid.Nil {
			agentPtr = &req.AgentID
		}
		_, _ = database.DB.Exec(`INSERT INTO decision_traces (org_id, trace_id, policy_id, policy_version, agent_id, allow, reason, trace) VALUES ($1,$2,$3,$4,$5,$6,$7,$8)`,
			orgID, dec.TraceID, v.PolicyID, v.Version, agentPtr, dec.Allow, dec.Reason, traceJSON)
		// Append audit event referencing this trace for compliance replay
		_ = audit.Append(c.Request.Context(), uuid.MustParse(orgID), "decision_trace_recorded", map[string]any{"trace_id": dec.TraceID, "policy_id": v.PolicyID, "version": v.Version, "allow": dec.Allow, "reason": dec.Reason}, nil, agentPtr)
	}()

	// minimal event side-effect placeholder
	_ = time.Now()

	// Optional trust token
	resp := VerifyV2Response{Allow: dec.Allow, Reason: dec.Reason, TraceID: dec.TraceID}
	if req.IncludeTrustToken {
		token := buildTrustToken(ctx, orgID, pr.AgentID, v.PolicyID.String(), v.Version, dec.Allow, dec.Reason, canonCtx, dec.TraceID)
		if token != "" {
			resp.Token = token
		}
	}
	c.JSON(http.StatusOK, resp)
}

// bucket returns a deterministic 0-99 value for canary/selection
func bucket(a, b string, rest ...string) int {
	input := a
	if b != "" {
		input += ":" + b
	}
	for _, r := range rest {
		input += ":" + r
	}
	h := sha256.Sum256([]byte(input))
	// use first two bytes to create 0-65535, then mod 100
	v := int(h[0])<<8 | int(h[1])
	return v % 100
}

// buildTrustToken signs a short-lived JWT with decision details and a hash of the evaluated context
func buildTrustToken(ctx context.Context, orgID, agentID, policyID string, version int, allow bool, reason string, reqCtx json.RawMessage, traceID string) string {
	// Observability
	tr := otel.Tracer("aura")
	ctx, span := tr.Start(ctx, "trust_token.build")
	defer span.End()
	// hash of evaluation context
	sum := sha256.Sum256(reqCtx)
	ctxHash := base64.RawURLEncoding.EncodeToString(sum[:])
	ttl := 120
	if v := os.Getenv("AURA_TRUST_TOKEN_TTL_SECONDS"); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n > 0 {
			ttl = n
		}
	}
	exp := time.Now().Add(time.Duration(ttl) * time.Second).Unix()
	// Prefer org-scoped KMS/local key if configured, else env Ed25519, else fallback to HS256. Include JTI for replay prevention.
	// Attempt to load active trust key for org
	var tk struct {
		Alg  string          `db:"alg"`
		Kid  string          `db:"kid"`
		Prov *string         `db:"provider"`
		Ref  *string         `db:"key_ref"`
		Ver  *string         `db:"key_version"`
		Cfg  json.RawMessage `db:"provider_config"`
		Enc  *string         `db:"ed25519_private_key_base64"`
		JWK  json.RawMessage `db:"jwk_pub"`
	}
	if err := database.DB.Get(&tk, `SELECT alg, COALESCE(kid,''), provider, key_ref, key_version, COALESCE(provider_config,'{}'::jsonb), ed25519_private_key_base64, COALESCE(jwk_pub,'{}'::jsonb) FROM trust_keys WHERE org_id=$1 AND active=true ORDER BY created_at DESC LIMIT 1`, orgID); err == nil && tk.Alg != "" {
		jti := uuid.New().String()
		header := fmt.Sprintf(`{"alg":"%s","typ":"JWT","kid":"%s"}`, tk.Alg, tk.Kid)
		payload := fmt.Sprintf(`{"org_id":"%s","agent_id":"%s","policy_id":"%s","policy_version":%d,"allow":%t,"reason":%q,"context_hash":"%s","trace_id":"%s","exp":%d,"jti":"%s"}`, orgID, agentID, policyID, version, allow, reason, ctxHash, traceID, exp, jti)
		hb := base64.RawURLEncoding.EncodeToString([]byte(header))
		pb := base64.RawURLEncoding.EncodeToString([]byte(payload))
		unsigned := hb + "." + pb
		// Local Ed25519 path
		if (tk.Prov == nil || *tk.Prov == "" || strings.ToLower(*tk.Prov) == "local") && tk.Alg == "EdDSA" && tk.Enc != nil && *tk.Enc != "" {
			if priv, pub, _ := loadOrgEd25519Key(orgID); priv != nil && pub != nil {
				span.SetAttributes(attribute.String("source", "org_local"), attribute.String("alg", "EdDSA"), attribute.String("kid", tk.Kid))
				sig := ed25519.Sign(priv, []byte(unsigned))
				sb := base64.RawURLEncoding.EncodeToString(sig)
				tok := unsigned + "." + sb
				if os.Getenv("AURA_TRUST_JTI_WRITE") == "1" {
					_, _ = database.DB.Exec(`INSERT INTO trust_token_jti(org_id, jti, exp_at) VALUES ($1,$2,to_timestamp($3)) ON CONFLICT DO NOTHING`, orgID, jti, exp)
				}
				RecordTrustToken("org_local", "EdDSA", true, orgID)
				span.SetAttributes(attribute.Bool("success", true))
				return tok
			}
		}
		// KMS-backed signing via signer interface
		rec := kms.TrustKeyRecord{Provider: strPtrVal(tk.Prov), KeyRef: strPtrVal(tk.Ref), KeyVersion: strPtrVal(tk.Ver), Alg: tk.Alg, Kid: tk.Kid, EncPriv: strPtrVal(tk.Enc), ProviderConfig: tk.Cfg, JWKPub: tk.JWK}
		if signer, err := kms.NewSignerFromRecord(rec); err == nil {
			span.SetAttributes(attribute.String("source", "org_kms"), attribute.String("alg", tk.Alg), attribute.String("kid", tk.Kid))
			if sig, err := signer.Sign(ctx, []byte(unsigned)); err == nil {
				sb := base64.RawURLEncoding.EncodeToString(sig)
				tok := unsigned + "." + sb
				if os.Getenv("AURA_TRUST_JTI_WRITE") == "1" {
					_, _ = database.DB.Exec(`INSERT INTO trust_token_jti(org_id, jti, exp_at) VALUES ($1,$2,to_timestamp($3)) ON CONFLICT DO NOTHING`, orgID, jti, exp)
				}
				RecordTrustToken("org_kms", tk.Alg, true, orgID)
				span.SetAttributes(attribute.Bool("success", true))
				return tok
			}
			// signer.Sign failed
			RecordTrustToken("org_kms", tk.Alg, false, orgID)
		}
	}
	// Fallbacks remain the same
	if priv, pub, kid := loadEd25519KeyFromEnv(); priv != nil && pub != nil {
		jti := uuid.New().String()
		header := fmt.Sprintf(`{"alg":"EdDSA","typ":"JWT","kid":"%s"}`, kid)
		payload := fmt.Sprintf(`{"org_id":"%s","agent_id":"%s","policy_id":"%s","policy_version":%d,"allow":%t,"reason":%q,"context_hash":"%s","trace_id":"%s","exp":%d,"jti":"%s"}`, orgID, agentID, policyID, version, allow, reason, ctxHash, traceID, exp, jti)
		hb := base64.RawURLEncoding.EncodeToString([]byte(header))
		pb := base64.RawURLEncoding.EncodeToString([]byte(payload))
		unsigned := hb + "." + pb
		span.SetAttributes(attribute.String("source", "env_local"), attribute.String("alg", "EdDSA"), attribute.String("kid", kid))
		sig := ed25519.Sign(priv, []byte(unsigned))
		sb := base64.RawURLEncoding.EncodeToString(sig)
		tok := unsigned + "." + sb
		if os.Getenv("AURA_TRUST_JTI_WRITE") == "1" {
			_, _ = database.DB.Exec(`INSERT INTO trust_token_jti(org_id, jti, exp_at) VALUES ($1,$2,to_timestamp($3)) ON CONFLICT DO NOTHING`, orgID, jti, exp)
		}
		RecordTrustToken("env_local", "EdDSA", true, orgID)
		span.SetAttributes(attribute.Bool("success", true))
		return tok
	}
	// Fallback: HS256
	secret := os.Getenv("AURA_TRUST_TOKEN_SIGNING_KEY")
	if secret == "" {
		secret = os.Getenv("JWT_SECRET")
	}
	if secret == "" {
		RecordTrustToken("env_hs256", "HS256", false, orgID)
		span.SetAttributes(attribute.String("source", "env_hs256"), attribute.String("alg", "HS256"), attribute.Bool("success", false))
		return ""
	}
	jti := uuid.New().String()
	header := `{"alg":"HS256","typ":"JWT"}`
	payload := fmt.Sprintf(`{"org_id":"%s","agent_id":"%s","policy_id":"%s","policy_version":%d,"allow":%t,"reason":%q,"context_hash":"%s","trace_id":"%s","exp":%d,"jti":"%s"}`, orgID, agentID, policyID, version, allow, reason, ctxHash, traceID, exp, jti)
	hb := base64.RawURLEncoding.EncodeToString([]byte(header))
	pb := base64.RawURLEncoding.EncodeToString([]byte(payload))
	unsigned := hb + "." + pb
	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write([]byte(unsigned))
	sig := mac.Sum(nil)
	sb := base64.RawURLEncoding.EncodeToString(sig)
	tok := unsigned + "." + sb
	if os.Getenv("AURA_TRUST_JTI_WRITE") == "1" {
		_, _ = database.DB.Exec(`INSERT INTO trust_token_jti(org_id, jti, exp_at) VALUES ($1,$2,to_timestamp($3)) ON CONFLICT DO NOTHING`, orgID, jti, exp)
	}
	RecordTrustToken("env_hs256", "HS256", true, orgID)
	span.SetAttributes(attribute.String("source", "env_hs256"), attribute.String("alg", "HS256"), attribute.Bool("success", true))
	return tok
}

// strPtrVal safely dereferences a *string, returning empty string when nil
func strPtrVal(p *string) string {
	if p == nil {
		return ""
	}
	return *p
}

// matchesAllowed checks if s matches any of the allowed entries using simple globbing:
// '*' allows all, a pattern ending with '*' is treated as prefix match, otherwise exact match.
func matchesAllowed(s string, allowed []string) bool {
	for _, a := range allowed {
		if a == "*" {
			return true
		}
		if strings.HasSuffix(a, "*") {
			prefix := strings.TrimSuffix(a, "*")
			if strings.HasPrefix(s, prefix) {
				return true
			}
		} else if strings.HasPrefix(a, "re:") {
			pat := strings.TrimPrefix(a, "re:")
			if r, err := regexp.Compile(pat); err == nil {
				if r.MatchString(s) {
					return true
				}
			}
		} else if s == a {
			return true
		}
	}
	return false
}

// --- Backpressure control for verify ---
var (
	verifySem     chan struct{}
	verifySemOnce bool
)

func initVerifyLimiter() {
	if verifySemOnce {
		return
	}
	max := 0
	if v := os.Getenv("AURA_VERIFY_MAX_INFLIGHT"); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n > 0 {
			max = n
		}
	}
	if max > 0 {
		verifySem = make(chan struct{}, max)
	} else {
		verifySem = nil
	}
	verifySemOnce = true
}

func acquireVerifySlot() bool {
	if !verifySemOnce {
		initVerifyLimiter()
	}
	if verifySem == nil {
		return true
	}
	select {
	case verifySem <- struct{}{}:
		SetVerifyInflight(len(verifySem))
		return true
	default:
		return false
	}
}

func releaseVerifySlot() {
	if verifySem == nil {
		return
	}
	select {
	case <-verifySem:
		SetVerifyInflight(len(verifySem))
	default:
	}
}
