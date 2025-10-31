package api

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"os"
	"strings"
	"time"

	database "github.com/Armour007/aura-backend/internal"
	"github.com/Armour007/aura-backend/internal/utils"
	"github.com/coreos/go-oidc/v3/oidc"
	"github.com/gin-gonic/gin"
	jwtlib "github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"golang.org/x/oauth2"
)

type oidcProviderConfig struct {
	Issuer       string
	ClientID     string
	ClientSecret string
	RedirectURL  string
	Scopes       []string
}

func getEnvAny(keys ...string) string {
	for _, k := range keys {
		if v := strings.TrimSpace(os.Getenv(k)); v != "" {
			return v
		}
	}
	return ""
}

func loadOIDCConfig(provider string) (*oidcProviderConfig, error) {
	p := strings.ToLower(provider)
	upper := strings.ToUpper(p)
	issuer := getEnvAny("OIDC_ISSUER_"+upper, "AURA_OIDC_ISSUER_"+upper, "OIDC_ISSUER")
	clientID := getEnvAny("OIDC_CLIENT_ID_"+upper, "AURA_OIDC_CLIENT_ID_"+upper, "OIDC_CLIENT_ID")
	clientSecret := getEnvAny("OIDC_CLIENT_SECRET_"+upper, "AURA_OIDC_CLIENT_SECRET_"+upper, "OIDC_CLIENT_SECRET")
	redirectURL := getEnvAny("OIDC_REDIRECT_URL_"+upper, "AURA_OIDC_REDIRECT_URL_"+upper, "OIDC_REDIRECT_URL")
	// Provider presets (issuer) if not provided
	if issuer == "" {
		switch p {
		case "google":
			issuer = "https://accounts.google.com"
		case "azure":
			if tid := strings.TrimSpace(os.Getenv("AURA_AZURE_TENANT_ID")); tid != "" {
				issuer = fmt.Sprintf("https://login.microsoftonline.com/%s/v2.0", tid)
			}
		case "okta":
			if dom := strings.TrimSpace(os.Getenv("AURA_OKTA_DOMAIN")); dom != "" {
				issuer = fmt.Sprintf("https://%s/oauth2/default", dom)
			}
		}
	}
	// Default redirect: API base + /sso/:provider/callback
	if redirectURL == "" {
		if api := strings.TrimRight(getEnvAny("AURA_API_BASE_URL", "PUBLIC_API_BASE", "API_BASE"), "/"); api != "" {
			redirectURL = api + "/sso/" + p + "/callback"
		}
	}
	if issuer == "" || clientID == "" || clientSecret == "" || redirectURL == "" {
		return nil, fmt.Errorf("missing OIDC configuration for provider %s", p)
	}
	scopes := []string{"openid", "email", "profile"}
	if extra := strings.TrimSpace(os.Getenv("AURA_OIDC_EXTRA_SCOPES")); extra != "" {
		scopes = append(scopes, strings.Split(extra, ",")...)
	}
	return &oidcProviderConfig{Issuer: issuer, ClientID: clientID, ClientSecret: clientSecret, RedirectURL: redirectURL, Scopes: scopes}, nil
}

func signState(claims jwtlib.MapClaims) (string, error) {
	sec, err := utils.GetJwtSecretString()
	if err != nil || strings.TrimSpace(sec) == "" {
		return "", errors.New("JWT_SECRET not set")
	}
	token := jwtlib.NewWithClaims(jwtlib.SigningMethodHS256, claims)
	return token.SignedString([]byte(sec))
}

func verifyState(state string) (jwtlib.MapClaims, error) {
	sec, err := utils.GetJwtSecretString()
	if err != nil || strings.TrimSpace(sec) == "" {
		return nil, errors.New("JWT_SECRET not set")
	}
	tok, err := jwtlib.Parse(state, func(token *jwtlib.Token) (interface{}, error) {
		if token.Method.Alg() != jwtlib.SigningMethodHS256.Alg() {
			return nil, fmt.Errorf("unexpected signing method: %s", token.Header["alg"])
		}
		return []byte(sec), nil
	})
	if err != nil || !tok.Valid {
		if err == nil {
			err = errors.New("invalid state token")
		}
		return nil, err
	}
	if cl, ok := tok.Claims.(jwtlib.MapClaims); ok {
		// standard exp check is handled by Parse if using RegisteredClaims; for MapClaims, check manually
		if expv, ok := cl["exp"].(float64); ok {
			if time.Unix(int64(expv), 0).Before(time.Now().Add(-5 * time.Second)) {
				return nil, errors.New("state expired")
			}
		}
		return cl, nil
	}
	return nil, errors.New("invalid claims type")
}

// GET /sso/:provider/login
func SSOLogin(c *gin.Context) {
	prov := strings.ToLower(c.Param("provider"))
	if os.Getenv("AURA_SSO_ENABLE") != "1" {
		c.JSON(http.StatusNotImplemented, gin.H{"error": "SSO not enabled"})
		return
	}
	cfg, err := loadOIDCConfig(prov)
	if err != nil {
		c.JSON(500, gin.H{"error": err.Error()})
		return
	}
	// Discover provider
	ctx := context.Background()
	provider, err := oidc.NewProvider(ctx, cfg.Issuer)
	if err != nil {
		c.JSON(500, gin.H{"error": "failed to discover OIDC provider"})
		return
	}
	// OAuth2 config
	oauthCfg := &oauth2.Config{
		ClientID:     cfg.ClientID,
		ClientSecret: cfg.ClientSecret,
		Endpoint:     provider.Endpoint(),
		RedirectURL:  cfg.RedirectURL,
		Scopes:       cfg.Scopes,
	}
	// Build state
	nonce := fmt.Sprintf("%d", time.Now().UnixNano())
	st, err := signState(jwtlib.MapClaims{
		"typ":   "sso_state",
		"prov":  prov,
		"nonce": nonce,
		"exp":   time.Now().Add(5 * time.Minute).Unix(),
	})
	if err != nil {
		c.JSON(500, gin.H{"error": "failed to sign state"})
		return
	}
	authURL := oauthCfg.AuthCodeURL(st, oauth2.SetAuthURLParam("nonce", nonce))
	c.Redirect(http.StatusFound, authURL)
}

// GET /sso/:provider/callback
func SSOCallback(c *gin.Context) {
	if os.Getenv("AURA_SSO_ENABLE") != "1" {
		c.JSON(http.StatusNotImplemented, gin.H{"error": "SSO not enabled"})
		return
	}
	prov := strings.ToLower(c.Param("provider"))
	cfg, err := loadOIDCConfig(prov)
	if err != nil {
		c.JSON(500, gin.H{"error": err.Error()})
		return
	}
	// Verify state
	state := c.Query("state")
	if state == "" {
		c.JSON(400, gin.H{"error": "missing state"})
		return
	}
	_, err = verifyState(state)
	if err != nil {
		c.JSON(400, gin.H{"error": "invalid state"})
		return
	}
	code := c.Query("code")
	if code == "" {
		c.JSON(400, gin.H{"error": "missing code"})
		return
	}
	// Discover & exchange code
	ctx := context.Background()
	provider, err := oidc.NewProvider(ctx, cfg.Issuer)
	if err != nil {
		c.JSON(500, gin.H{"error": "provider discovery failed"})
		return
	}
	oauthCfg := &oauth2.Config{
		ClientID:     cfg.ClientID,
		ClientSecret: cfg.ClientSecret,
		Endpoint:     provider.Endpoint(),
		RedirectURL:  cfg.RedirectURL,
		Scopes:       cfg.Scopes,
	}
	tok, err := oauthCfg.Exchange(ctx, code)
	if err != nil {
		c.JSON(400, gin.H{"error": "token exchange failed"})
		return
	}
	// Verify ID Token
	rawID, ok := tok.Extra("id_token").(string)
	if !ok || rawID == "" {
		c.JSON(400, gin.H{"error": "missing id_token"})
		return
	}
	verifier := provider.Verifier(&oidc.Config{ClientID: cfg.ClientID})
	idt, err := verifier.Verify(ctx, rawID)
	if err != nil {
		c.JSON(400, gin.H{"error": "invalid id_token"})
		return
	}
	var claims struct {
		Email         string   `json:"email"`
		EmailVerified bool     `json:"email_verified"`
		Name          string   `json:"name"`
		GivenName     string   `json:"given_name"`
		FamilyName    string   `json:"family_name"`
		Groups        []string `json:"groups"`
	}
	if err := idt.Claims(&claims); err != nil {
		c.JSON(400, gin.H{"error": "cannot parse claims"})
		return
	}
	email := strings.ToLower(strings.TrimSpace(claims.Email))
	if email == "" {
		c.JSON(400, gin.H{"error": "email not provided by IdP"})
		return
	}
	fullName := strings.TrimSpace(func() string {
		if claims.Name != "" {
			return claims.Name
		}
		return strings.TrimSpace(claims.GivenName + " " + claims.FamilyName)
	}())
	// Upsert user
	var uidStr string
	_ = database.DB.Get(&uidStr, `SELECT id::text FROM users WHERE email=$1`, email)
	if uidStr == "" {
		// create user with empty password (generate UUID in app to avoid DB extension dependency)
		newUID := uuid.New().String()
		_, _ = database.DB.Exec(`INSERT INTO users(id, full_name, email, hashed_password, created_at, updated_at)
			VALUES($1,$2,$3,'',NOW(),NOW())`, newUID, fullName, email)
		uidStr = newUID
	} else if fullName != "" {
		_, _ = database.DB.Exec(`UPDATE users SET full_name=$1, updated_at=NOW() WHERE id=$2`, fullName, uidStr)
	}
	// Ensure membership: if user has no orgs, try domain->org mapping first else create personal org
	var orgID string
	_ = database.DB.Get(&orgID, `SELECT organization_id::text FROM organization_members WHERE user_id=$1 ORDER BY joined_at LIMIT 1`, uidStr)
	if orgID == "" {
		// domain->org mapping via env, e.g., AURA_SSO_DOMAIN_ORG_MAP="example.com=<orgId>;acme.com=<orgId>"
		if at := strings.LastIndex(email, "@"); at > 0 {
			dom := strings.ToLower(email[at+1:])
			if m := strings.TrimSpace(os.Getenv("AURA_SSO_DOMAIN_ORG_MAP")); m != "" {
				pairs := strings.Split(m, ";")
				for _, p := range pairs {
					kv := strings.SplitN(strings.TrimSpace(p), "=", 2)
					if len(kv) == 2 && strings.EqualFold(strings.TrimSpace(kv[0]), dom) {
						mapped := strings.TrimSpace(kv[1])
						// ensure org exists
						var exists int
						_ = database.DB.Get(&exists, `SELECT 1 FROM organizations WHERE id=$1`, mapped)
						if exists == 1 {
							_, _ = database.DB.Exec(`INSERT INTO organization_members(organization_id,user_id,role,joined_at) VALUES ($1,$2,'member',NOW()) ON CONFLICT DO NOTHING`, mapped, uidStr)
							orgID = mapped
							break
						}
					}
				}
			}
		}
		if orgID == "" {
			// fallback: create personal org and owner membership
			newOrg := uuid.New().String()
			_, _ = database.DB.Exec(`INSERT INTO organizations(id,name,owner_id,created_at,updated_at) VALUES ($1,$2,$3,NOW(),NOW())`, newOrg, fullName+"'s Organization", uidStr)
			_, _ = database.DB.Exec(`INSERT INTO organization_members(organization_id,user_id,role,joined_at) VALUES ($1,$2,'owner',NOW())`, newOrg, uidStr)
			orgID = newOrg
		}
	}
	// Optional group-to-role mapping via env: AURA_SSO_GROUP_ROLE_MAP="okta-admin=admin;auditors=auditor"
	if m := strings.TrimSpace(os.Getenv("AURA_SSO_GROUP_ROLE_MAP")); m != "" && len(claims.Groups) > 0 {
		// choose first matching mapping and upsert role
		pairs := strings.Split(m, ";")
		for _, p := range pairs {
			kv := strings.SplitN(strings.TrimSpace(p), "=", 2)
			if len(kv) != 2 {
				continue
			}
			g, role := strings.TrimSpace(kv[0]), strings.TrimSpace(kv[1])
			for _, ugrp := range claims.Groups {
				if strings.EqualFold(ugrp, g) {
					_, _ = database.DB.Exec(`UPDATE organization_members SET role=$1 WHERE organization_id=$2 AND user_id=$3`, role, orgID, uidStr)
					goto mapped
				}
			}
		}
	}
mapped:
	// Mint app JWT and redirect to frontend
	// Mint app JWT using user UUID
	uid, _ := uuid.Parse(uidStr)
	jwtStr, err := utils.GenerateJWT(uid)
	if err != nil {
		c.JSON(500, gin.H{"error": "failed to mint app token"})
		return
	}
	fe := strings.TrimRight(getEnvAny("AURA_FRONTEND_BASE_URL", "FRONTEND_BASE_URL"), "/")
	path := os.Getenv("AURA_SSO_REDIRECT_PATH")
	if path == "" {
		path = "/login/sso-callback"
	}
	// Append token as query param
	u, _ := url.Parse(fe + path)
	q := u.Query()
	q.Set("token", jwtStr)
	q.Set("orgId", orgID)
	u.RawQuery = q.Encode()
	c.Redirect(http.StatusFound, u.String())
}
