package api

import (
	"archive/zip"
	"bytes"
	"context"
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/smtp"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
)

// DownloadSDK streams a zip of a curated SDK folder (node|python|go) to the user.
// Example: GET /sdk/download?lang=node
// Requires user authentication (handled by route group).
func DownloadSDK(c *gin.Context) {
	lang := strings.ToLower(strings.TrimSpace(c.Query("lang")))
	if lang == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "missing lang; use one of: node, python, go"})
		return
	}

	var subdir string
	switch lang {
	case "node", "javascript", "ts", "typescript":
		subdir = "node"
	case "python", "py":
		subdir = "python"
	case "go", "golang":
		subdir = filepath.Join("go", "aura")
	default:
		c.JSON(http.StatusBadRequest, gin.H{"error": "unsupported lang; use one of: node, python, go"})
		return
	}

	// Resolve SDK dir relative to backend working dir: ../sdks/<subdir>
	base := filepath.Clean(filepath.Join("..", "sdks", subdir))
	info, err := os.Stat(base)
	if err != nil || !info.IsDir() {
		c.JSON(http.StatusNotFound, gin.H{"error": "sdk not found on server"})
		return
	}

	// Zip the directory into memory (sufficient for small SDKs)
	var buf bytes.Buffer
	zw := zip.NewWriter(&buf)
	// Inject a tailored README and manifest if provided
	agentID := strings.TrimSpace(c.Query("agent_id"))
	action := strings.TrimSpace(c.Query("action"))
	baseURL := strings.TrimRight(os.Getenv("AURA_API_BASE_URL"), "/")
	if baseURL == "" {
		// Fallback to local default backend when not set
		baseURL = "http://localhost:8081"
	}
	// README with plug-in snippet per language
	readme := curatedReadmeForLang(lang, baseURL, agentID, action)
	if f, err := zw.Create("README.md"); err == nil {
		_, _ = f.Write([]byte(readme))
	}
	// manifest
	manifest := map[string]string{"lang": lang, "agent_id": agentID, "action": action, "base_url": baseURL}
	if mf, err := zw.Create("manifest.json"); err == nil {
		b, _ := json.MarshalIndent(manifest, "", "  ")
		_, _ = mf.Write(b)
	}
	// Walk directory and add files
	err = filepath.Walk(base, func(path string, info os.FileInfo, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if info.IsDir() {
			return nil
		}
		// Derive a relative path inside the zip (without leading ../sdks/)
		rel, err := filepath.Rel(filepath.Dir(base), path)
		if err != nil {
			return err
		}
		// Normalize to forward slashes in zip entries
		zipPath := filepath.ToSlash(rel)
		f, err := zw.Create(zipPath)
		if err != nil {
			return err
		}
		src, err := os.Open(path)
		if err != nil {
			return err
		}
		defer src.Close()
		_, err = io.Copy(f, src)
		return err
	})
	if err != nil {
		_ = zw.Close()
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to package sdk"})
		return
	}
	_ = zw.Close()

	filename := "aura-sdk-" + strings.ReplaceAll(lang, " ", "_") + ".zip"
	c.Header("Content-Type", "application/zip")
	c.Header("Content-Disposition", "attachment; filename=\""+filename+"\"")
	c.Header("Content-Length", strconv.Itoa(buf.Len()))
	c.Data(http.StatusOK, "application/zip", buf.Bytes())
}

// --- Helpers and async codegen simulation ---

// orDash returns v or "—" if empty
func orDash(v string) string {
	if v == "" {
		return "—"
	}
	return v
}

type sdkJob struct {
	ID             string `json:"id"`
	Lang           string `json:"lang"`
	AgentID        string `json:"agent_id,omitempty"`
	OrganizationID string `json:"organization_id,omitempty"`
	Email          string `json:"email,omitempty"`
	Status         string `json:"status"` // queued|running|ready|error
	Error          string `json:"error,omitempty"`
}

var sdkJobs = struct {
	mu   sync.Mutex
	jobs map[string]*sdkJob
	zips map[string][]byte
	// files persists generated artifacts to disk; value is absolute file path
	files map[string]string
}{jobs: map[string]*sdkJob{}, zips: map[string][]byte{}, files: map[string]string{}}

func newJobID() string {
	var b [12]byte
	_, _ = rand.Read(b[:])
	return hex.EncodeToString(b[:])
}

// GenerateSDK starts an async codegen job for other languages and returns a job id.
// POST /sdk/generate { lang, agent_id, organization_id, email }
func GenerateSDK(c *gin.Context) {
	var req struct {
		Lang           string `json:"lang"`
		AgentID        string `json:"agent_id"`
		OrganizationID string `json:"organization_id"`
		Email          string `json:"email"`
	}
	if err := c.BindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid json"})
		return
	}
	lang := strings.ToLower(strings.TrimSpace(req.Lang))
	if lang == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "missing lang"})
		return
	}
	// Accept a wider set for codegen, optionally restricted via env AURA_SDK_CODEGEN_LANGS (comma-separated)
	if !allowedCodegenLang(lang) {
		c.JSON(http.StatusBadRequest, gin.H{"error": "unsupported lang for async codegen"})
		return
	}
	job := &sdkJob{ID: newJobID(), Lang: lang, AgentID: strings.TrimSpace(req.AgentID), OrganizationID: strings.TrimSpace(req.OrganizationID), Email: strings.TrimSpace(req.Email), Status: "queued"}
	sdkJobs.mu.Lock()
	sdkJobs.jobs[job.ID] = job
	sdkJobs.mu.Unlock()

	// If queue is enabled, publish job; else run inline goroutine
	if os.Getenv("AURA_QUEUE_ENABLE") != "" {
		if err := EnqueueCodegen(job); err != nil {
			// fallback to local goroutine
			go runCodegen(job)
		}
	} else {
		go runCodegen(job)
	}

	c.JSON(http.StatusAccepted, gin.H{"job_id": job.ID, "status": job.Status})
}

func runCodegen(job *sdkJob) {
	setJobStatus(job.ID, "running", "")

	// Determine repo root (parent of backend dir)
	cwd, _ := os.Getwd() // typically <repo>/backend
	repoRoot := filepath.Clean(filepath.Join(cwd, ".."))
	codegenDir := filepath.Join(repoRoot, "sdks", "codegen")

	// Ensure OpenAPI spec file at backend/static/openapi.json
	if err := fetchOpenAPISpec(filepath.Join(cwd, "static", "openapi.json")); err != nil {
		setJobStatus(job.ID, "error", "failed to fetch openapi: "+err.Error())
		return
	}

	// Language mapping for openapi-generator
	genMap := map[string]struct{ generator, outRel string }{
		"java":   {"java", "../java"},
		"csharp": {"csharp", "../csharp"},
		"ruby":   {"ruby", "../ruby"},
		"php":    {"php", "../php"},
		"rust":   {"rust", "../rust"},
		"swift":  {"swift5", "../swift"},
		"kotlin": {"kotlin", "../kotlin"},
		"dart":   {"dart-dio", "../dart"},
		"cpp":    {"cpp-httplib", "../cpp"},
	}
	cfg, ok := genMap[job.Lang]
	if !ok {
		setJobStatus(job.ID, "error", "unsupported language for codegen")
		return
	}

	// Circuit breaker for docker codegen
	dockerCB := GetBreaker("docker_codegen")
	if !dockerCB.Allow() {
		// Short-circuit with placeholder
		placeholder := buildPlaceholderZip(job)
		path, perr := persistJobZip(job.ID, placeholder)
		sdkJobs.mu.Lock()
		if perr == nil {
			sdkJobs.files[job.ID] = path
		} else {
			sdkJobs.zips[job.ID] = placeholder
		}
		setJobStatusLocked(job.ID, "ready", "")
		sdkJobs.mu.Unlock()
		RecordExternalOp("docker_codegen", 0, false)
		return
	}

	// Run docker openapi-generator with timeout and a single retry
	var out []byte
	var err error
	start := time.Now()
	for attempt := 0; attempt < 2; attempt++ {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
		cmd := exec.CommandContext(ctx, "docker", "run", "--rm",
			"-v", fmt.Sprintf("%s:/work", repoRoot),
			"-w", "/work/sdks/codegen",
			"openapitools/openapi-generator-cli:latest",
			"generate",
			"-c", "openapi-generator.yaml",
			"-g", cfg.generator,
			"-o", cfg.outRel,
		)
		cmd.Dir = codegenDir
		out, err = cmd.CombinedOutput()
		cancel()
		if err == nil {
			break
		}
		// Backoff before retrying
		time.Sleep(2 * time.Second)
	}
	success := err == nil
	RecordExternalOp("docker_codegen", time.Since(start), success)
	if success {
		dockerCB.ReportSuccess()
	} else {
		dockerCB.ReportFailure()
	}
	if err != nil {
		// Fallback: placeholder zip and persist
		placeholder := buildPlaceholderZip(job)
		path, perr := persistJobZip(job.ID, placeholder)
		sdkJobs.mu.Lock()
		if perr == nil {
			sdkJobs.files[job.ID] = path
		} else {
			sdkJobs.zips[job.ID] = placeholder
		}
		setJobStatusLocked(job.ID, "ready", "")
		sdkJobs.mu.Unlock()
		fmt.Printf("codegen failed: %v\n%s\n", err, string(out))
		_ = sendEmailIfConfigured(job, len(placeholder))
		return
	}

	// Zip generated directory with extras
	outAbs := filepath.Clean(filepath.Join(codegenDir, cfg.outRel))
	data, err := zipDirWithExtras(outAbs, job)
	if err != nil {
		setJobStatus(job.ID, "error", "failed to package sdk: "+err.Error())
		return
	}

	// Persist to disk for signed URL downloads; fall back to memory if disk write fails
	path, perr := persistJobZip(job.ID, data)
	sdkJobs.mu.Lock()
	if perr == nil {
		sdkJobs.files[job.ID] = path
	} else {
		sdkJobs.zips[job.ID] = data
	}
	setJobStatusLocked(job.ID, "ready", "")
	sdkJobs.mu.Unlock()

	_ = sendEmailIfConfigured(job, len(data))
}

func fetchOpenAPISpec(targetPath string) error {
	base := strings.TrimRight(os.Getenv("AURA_API_BASE_URL"), "/")
	if base == "" {
		base = "http://localhost:8081"
	}
	url := base + "/openapi.json"
	resp, err := http.Get(url)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		return fmt.Errorf("unexpected status %d", resp.StatusCode)
	}
	if err := os.MkdirAll(filepath.Dir(targetPath), 0o755); err != nil {
		return err
	}
	f, err := os.Create(targetPath)
	if err != nil {
		return err
	}
	defer f.Close()
	_, err = io.Copy(f, resp.Body)
	return err
}

func zipDirWithExtras(dir string, job *sdkJob) ([]byte, error) {
	var buf bytes.Buffer
	zw := zip.NewWriter(&buf)
	err := filepath.Walk(dir, func(path string, info os.FileInfo, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if info.IsDir() {
			return nil
		}
		rel, err := filepath.Rel(filepath.Dir(dir), path)
		if err != nil {
			return err
		}
		f, err := zw.Create(filepath.ToSlash(rel))
		if err != nil {
			return err
		}
		src, err := os.Open(path)
		if err != nil {
			return err
		}
		defer src.Close()
		_, err = io.Copy(f, src)
		return err
	})
	if err != nil {
		_ = zw.Close()
		return nil, err
	}
	baseURL := strings.TrimRight(os.Getenv("AURA_API_BASE_URL"), "/")
	if baseURL == "" {
		baseURL = "http://localhost:8081"
	}
	readme := fmt.Sprintf("# Aura %s SDK (generated)\n\n- Agent ID: %s\n- Organization ID: %s\n- Backend: %s\n\nNext steps:\n1. Install per-language dependencies.\n2. Export AURA_API_KEY with a valid key.\n3. Call the verify endpoint using this client.\n", strings.Title(job.Lang), orDash(job.AgentID), orDash(job.OrganizationID), baseURL)
	if f, err := zw.Create("README.md"); err == nil {
		_, _ = f.Write([]byte(readme))
	}
	manifest := map[string]string{"lang": job.Lang, "agent_id": job.AgentID, "organization_id": job.OrganizationID, "base_url": baseURL}
	if mf, err := zw.Create("manifest.json"); err == nil {
		b, _ := json.MarshalIndent(manifest, "", "  ")
		_, _ = mf.Write(b)
	}
	_ = zw.Close()
	return buf.Bytes(), nil
}

func buildPlaceholderZip(job *sdkJob) []byte {
	var buf bytes.Buffer
	zw := zip.NewWriter(&buf)
	baseURL := strings.TrimRight(os.Getenv("AURA_API_BASE_URL"), "/")
	if baseURL == "" {
		baseURL = "http://localhost:8081"
	}
	readme := fmt.Sprintf("# Aura %s SDK (generated)\n\n- Agent ID: %s\n- Organization ID: %s\n- Backend: %s\n\nThis is a placeholder generated SDK bundle. Replace with real codegen output in production.\n", strings.Title(job.Lang), orDash(job.AgentID), orDash(job.OrganizationID), baseURL)
	if f, err := zw.Create("README.md"); err == nil {
		_, _ = f.Write([]byte(readme))
	}
	manifest := map[string]string{"lang": job.Lang, "agent_id": job.AgentID, "organization_id": job.OrganizationID, "base_url": baseURL}
	if mf, err := zw.Create("manifest.json"); err == nil {
		b, _ := json.MarshalIndent(manifest, "", "  ")
		_, _ = mf.Write(b)
	}
	_ = zw.Close()
	return buf.Bytes()
}

func sendEmailIfConfigured(job *sdkJob, size int) error {
	if job.Email == "" {
		return nil
	}
	host := os.Getenv("SMTP_HOST")
	port := os.Getenv("SMTP_PORT")
	user := os.Getenv("SMTP_USER")
	pass := os.Getenv("SMTP_PASS")
	from := os.Getenv("SMTP_FROM")
	if host == "" || port == "" || from == "" {
		return fmt.Errorf("smtp not configured")
	}
	apiBase := strings.TrimRight(os.Getenv("AURA_API_BASE_URL"), "/")
	if apiBase == "" {
		apiBase = "http://localhost:8081"
	}
	// Prefer a signed public URL if signing key is configured
	link := fmt.Sprintf("%s/sdk/download-generated/%s", apiBase, job.ID)
	if key := os.Getenv("AURA_DOWNLOAD_SIGNING_KEY"); key != "" {
		// 24h expiry
		exp := time.Now().Add(24 * time.Hour).Unix()
		sig := signDownload(job.ID, exp, key)
		link = fmt.Sprintf("%s/sdk/public/download-generated/%s?exp=%d&sig=%s", apiBase, job.ID, exp, sig)
	}
	subject := fmt.Sprintf("Your Aura %s SDK is ready", strings.ToUpper(job.Lang))
	body := fmt.Sprintf("Hello,\n\nYour SDK for %s is ready. Download (%d bytes):\n%s\n\n— Aura", job.Lang, size, link)
	msg := []byte("To: " + job.Email + "\r\n" +
		"Subject: " + subject + "\r\n" +
		"MIME-version: 1.0\r\n" +
		"Content-Type: text/plain; charset=UTF-8\r\n\r\n" +
		body)
	addr := host + ":" + port
	var auth smtp.Auth
	if user != "" {
		auth = smtp.PlainAuth("", user, pass, host)
	}
	// Send with a soft timeout and metrics
	tout := 5 * time.Second
	if v := os.Getenv("AURA_SMTP_TIMEOUT_MS"); v != "" {
		if ms, err := strconv.Atoi(v); err == nil && ms > 0 {
			tout = time.Duration(ms) * time.Millisecond
		}
	}
	smtpCB := GetBreaker("smtp_send")
	if !smtpCB.Allow() {
		RecordExternalOp("smtp_send", 0, false)
		return fmt.Errorf("smtp circuit open")
	}
	start := time.Now()
	err := sendMailWithTimeout(tout, func() error { return smtp.SendMail(addr, auth, from, []string{job.Email}, msg) })
	success := err == nil
	RecordExternalOp("smtp_send", time.Since(start), success)
	if success {
		smtpCB.ReportSuccess()
	} else {
		smtpCB.ReportFailure()
	}
	return err
}

func setJobStatus(id, status, errMsg string) {
	sdkJobs.mu.Lock()
	defer sdkJobs.mu.Unlock()
	setJobStatusLocked(id, status, errMsg)
}
func setJobStatusLocked(id, status, errMsg string) {
	if j, ok := sdkJobs.jobs[id]; ok {
		j.Status = status
		j.Error = errMsg
	}
}

// GetSDKJob returns status for a job id
// GET /sdk/generate/:jobId
func GetSDKJob(c *gin.Context) {
	id := c.Param("jobId")
	sdkJobs.mu.Lock()
	defer sdkJobs.mu.Unlock()
	if j, ok := sdkJobs.jobs[id]; ok {
		c.JSON(http.StatusOK, j)
		return
	}
	c.JSON(http.StatusNotFound, gin.H{"error": "job not found"})
}

// DownloadGeneratedSDK returns the generated zip when ready
// GET /sdk/download-generated/:jobId
func DownloadGeneratedSDK(c *gin.Context) {
	id := c.Param("jobId")
	sdkJobs.mu.Lock()
	j, jok := sdkJobs.jobs[id]
	path := sdkJobs.files[id]
	data, ok := sdkJobs.zips[id]
	sdkJobs.mu.Unlock()
	if !jok {
		c.JSON(http.StatusNotFound, gin.H{"error": "job not found"})
		return
	}
	filename := "aura-sdk-" + j.Lang + "-generated.zip"
	if path != "" {
		// Serve from disk
		http.ServeFile(c.Writer, c.Request, path)
		return
	}
	if ok {
		c.Header("Content-Type", "application/zip")
		c.Header("Content-Disposition", "attachment; filename=\""+filename+"\"")
		c.Header("Content-Length", strconv.Itoa(len(data)))
		c.Data(http.StatusOK, "application/zip", data)
		return
	}
	c.JSON(http.StatusNotFound, gin.H{"error": "not ready"})
}

// DownloadGeneratedSDKPublic allows unauthenticated, signed downloads via query params exp and sig
// GET /sdk/public/download-generated/:jobId?exp=<unix>&sig=<hmac>
func DownloadGeneratedSDKPublic(c *gin.Context) {
	id := c.Param("jobId")
	expStr := c.Query("exp")
	sig := c.Query("sig")
	key := os.Getenv("AURA_DOWNLOAD_SIGNING_KEY")
	if key == "" || expStr == "" || sig == "" {
		c.JSON(http.StatusForbidden, gin.H{"error": "invalid or missing signature"})
		return
	}
	exp, err := strconv.ParseInt(expStr, 10, 64)
	if err != nil || time.Now().Unix() > exp {
		c.JSON(http.StatusForbidden, gin.H{"error": "link expired"})
		return
	}
	want := signDownload(id, exp, key)
	if !hmac.Equal([]byte(sig), []byte(want)) {
		c.JSON(http.StatusForbidden, gin.H{"error": "bad signature"})
		return
	}
	// Delegate to the same underlying storage
	sdkJobs.mu.Lock()
	j, jok := sdkJobs.jobs[id]
	path := sdkJobs.files[id]
	data, ok := sdkJobs.zips[id]
	sdkJobs.mu.Unlock()
	if !jok {
		c.JSON(http.StatusNotFound, gin.H{"error": "job not found"})
		return
	}
	filename := "aura-sdk-" + j.Lang + "-generated.zip"
	if path != "" {
		http.ServeFile(c.Writer, c.Request, path)
		return
	}
	if ok {
		c.Header("Content-Type", "application/zip")
		c.Header("Content-Disposition", "attachment; filename=\""+filename+"\"")
		c.Header("Content-Length", strconv.Itoa(len(data)))
		c.Data(http.StatusOK, "application/zip", data)
		return
	}
	c.JSON(http.StatusNotFound, gin.H{"error": "not ready"})
}

// GetSupportedLangs returns curated SDK langs and env-filtered codegen langs
// GET /sdk/supported-langs
func GetSupportedLangs(c *gin.Context) {
	curated := []string{"node", "python", "go"}
	allCodegen := []string{"java", "csharp", "ruby", "php", "rust", "swift", "kotlin", "dart", "cpp"}
	// Filter according to allowedCodegenLang
	codegen := make([]string, 0, len(allCodegen))
	for _, l := range allCodegen {
		if allowedCodegenLang(l) {
			codegen = append(codegen, l)
		}
	}
	c.JSON(http.StatusOK, gin.H{
		"curated": curated,
		"codegen": codegen,
	})
}

// persistJobZip writes job artifact to disk and returns absolute path
func persistJobZip(jobID string, data []byte) (string, error) {
	cwd, _ := os.Getwd()
	repoRoot := filepath.Clean(filepath.Join(cwd, ".."))
	outDir := filepath.Join(repoRoot, "sdks", "generated", "jobs")
	if err := os.MkdirAll(outDir, 0o755); err != nil {
		return "", err
	}
	out := filepath.Join(outDir, jobID+".zip")
	if err := os.WriteFile(out, data, 0o644); err != nil {
		return "", err
	}
	return out, nil
}

// allowedCodegenLang checks against env-configured subset; defaults to full supported set
func allowedCodegenLang(lang string) bool {
	supported := map[string]bool{"java": true, "csharp": true, "ruby": true, "php": true, "rust": true, "swift": true, "kotlin": true, "dart": true, "cpp": true}
	if !supported[lang] {
		return false
	}
	csv := strings.TrimSpace(os.Getenv("AURA_SDK_CODEGEN_LANGS"))
	if csv == "" {
		return true
	}
	// Build a set from csv
	allowed := map[string]bool{}
	for _, p := range strings.Split(csv, ",") {
		if s := strings.ToLower(strings.TrimSpace(p)); s != "" {
			allowed[s] = true
		}
	}
	return allowed[lang]
}

// signDownload creates a URL-safe signature for jobID and expiry
func signDownload(jobID string, exp int64, key string) string {
	mac := hmac.New(sha256.New, []byte(key))
	mac.Write([]byte(jobID))
	mac.Write([]byte{"."[0]})
	mac.Write([]byte(strconv.FormatInt(exp, 10)))
	sum := mac.Sum(nil)
	return base64.RawURLEncoding.EncodeToString(sum)
}

// curatedReadmeForLang returns a language-specific README content showcasing the plug-in adapter pattern
func curatedReadmeForLang(lang, baseURL, agentID, action string) string {
	title := strings.Title(lang)
	switch lang {
	case "python", "py":
		return fmt.Sprintf(`# Aura Python SDK (curated)

Welcome! This bundle includes a minimal SDK and a 1‑minute plug‑in decorator.

Backend: %s
Agent ID: %s
Example action: %s

Quick start
1) Install locally after extracting this zip:

	pip install -e ./python

2) Set env and decorate your code:

	export AURA_API_KEY=your_aura_sk_...
	export AURA_AGENT_ID=%s
	export AURA_API_BASE_URL=%s

	from aura_sdk import protect

	@protect()
	def dangerous_function(user_id: str):
		# ... sensitive work ...
		return f"deleted {user_id}"

	print(dangerous_function('123'))

Notes
- @protect() reads AURA_API_KEY/AURA_AGENT_ID/AURA_API_BASE_URL
- Customize with on_deny or context_builder if needed
`, baseURL, orDash(agentID), orDash(action), agentID, baseURL)
	case "node", "javascript", "ts", "typescript":
		return fmt.Sprintf(`# Aura Node SDK (curated)

This bundle provides a client and tiny adapters for functions and Express.

Backend: %s
Agent ID: %s
Example action: %s

Quick start
1) Use directly from this bundle (no publish required):

	// ESM
	import { AuraClient, protect } from './node/src/index.js'
	const client = new AuraClient({ apiKey: process.env.AURA_API_KEY, baseURL: '%s' })
	const secured = protect({ client, agentId: '%s' })(async function dangerous(userId){ return 'deleted ' + userId })
	await secured('123')

Or, if published: import { AuraClient, protect } from '@auraai/sdk-node'

Express route

	import express from 'express'
	import { AuraClient, protectExpress } from './node/src/index.js'
	const app = express(); app.use(express.json())
	const client = new AuraClient({ apiKey: process.env.AURA_API_KEY, baseURL: '%s' })
	app.post('/danger', protectExpress({ client, agentId: '%s' }), (req, res) => res.json({ ok: true }))

Env
- AURA_API_KEY=aura_sk_...
- AURA_API_BASE_URL=%s
`, baseURL, orDash(agentID), orDash(action), baseURL, agentID, baseURL, agentID, baseURL)
	case "go", "golang":
		return fmt.Sprintf(`# Aura Go SDK (curated)

Use a single middleware to guard handlers.

Backend: %s
Agent ID: %s
Example action: %s

Quick start

	package main

	import (
		"net/http"
		"os"
		aura "github.com/Armour007/aura/sdks/go/aura"
	)

	func main(){
		os.Setenv("AURA_API_BASE_URL", "%s")
		c := aura.NewClient(os.Getenv("AURA_API_KEY"), os.Getenv("AURA_API_BASE_URL"), "")
		agent := "%s"
		mux := http.NewServeMux()
		mux.Handle("/danger", aura.ProtectHTTP(agent, c, func(r *http.Request) any { return map[string]any{"path": r.URL.Path} }, nil)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request){ w.Write([]byte("ok")) })))
		http.ListenAndServe(":3000", mux)
	}

Notes
- Import path can be adjusted via go.work or replace if using the local bundle
`, baseURL, orDash(agentID), orDash(action), baseURL, agentID)
	default:
		return fmt.Sprintf(`# Aura %s SDK (curated)

This archive contains a curated %s SDK starter.

- Backend Base URL: %s
- Agent ID: %s
- Example Action: %s

Quick start
1) Install dependencies inside this folder.
2) Set AURA_API_KEY and call /v1/verify with your agent.
`, title, title, baseURL, orDash(agentID), orDash(action))
	}
}
