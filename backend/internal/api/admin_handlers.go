package api

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/smtp"
	"os"
	"strconv"
	"strings"
	"time"

	database "github.com/Armour007/aura-backend/internal"
	"github.com/Armour007/aura-backend/internal/utils"
	"github.com/gin-gonic/gin"
	"github.com/redis/go-redis/v9"
)

// TestSMTP sends a simple test email using current SMTP_* environment variables.
// POST /admin/test-smtp { "to": "you@example.com" }
func TestSMTP(c *gin.Context) {
	var req struct {
		To string `json:"to"`
	}
	if err := c.BindJSON(&req); err != nil || req.To == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid json or missing 'to'"})
		return
	}
	// Read SMTP config
	host := os.Getenv("SMTP_HOST")
	port := os.Getenv("SMTP_PORT")
	user := os.Getenv("SMTP_USER")
	pass := os.Getenv("SMTP_PASS")
	from := os.Getenv("SMTP_FROM")
	if host == "" || port == "" || from == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "SMTP not configured: set SMTP_HOST, SMTP_PORT, SMTP_FROM (and optionally SMTP_USER, SMTP_PASS)"})
		return
	}
	addr := host + ":" + port
	msg := []byte("To: " + req.To + "\r\n" +
		"Subject: Aura SMTP test\r\n" +
		"MIME-version: 1.0\r\n" +
		"Content-Type: text/plain; charset=UTF-8\r\n\r\n" +
		"This is a test email from Aura. If you received this, your SMTP settings are working.\r\n")
	var auth smtp.Auth
	if user != "" {
		auth = smtp.PlainAuth("", user, pass, host)
	}
	tout := 5 * time.Second
	if v := os.Getenv("AURA_SMTP_TIMEOUT_MS"); v != "" {
		if ms, err := strconv.Atoi(v); err == nil && ms > 0 {
			tout = time.Duration(ms) * time.Millisecond
		}
	}
	smtpCB := GetBreaker("smtp_send")
	if !smtpCB.Allow() {
		c.JSON(http.StatusBadRequest, gin.H{"ok": false, "error": "smtp circuit open"})
		return
	}
	start := time.Now()
	err := sendMailWithTimeout(tout, func() error { return smtp.SendMail(addr, auth, from, []string{req.To}, msg) })
	success := err == nil
	RecordExternalOp("smtp_send", time.Since(start), success)
	if success {
		smtpCB.ReportSuccess()
	} else {
		smtpCB.ReportFailure()
	}
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"ok": false, "error": fmt.Sprintf("send failed: %v", err)})
		return
	}
	c.JSON(http.StatusOK, gin.H{"ok": true})
}

// ListDLQ returns recent messages from the codegen DLQ stream.
// GET /admin/queue/dlq?count=50
func ListDLQ(c *gin.Context) {
	if redisClient == nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "queue disabled"})
		return
	}
	count := 50
	if v := c.Query("count"); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n > 0 {
			if n > 500 {
				n = 500 // cap to avoid huge responses
			}
			count = n
		}
	}
	// Optional pagination: before_id or older_than (unix seconds)
	max := "+"
	if bid := c.Query("before_id"); bid != "" {
		// Exclusive upper bound to avoid duplicating last seen id
		max = "(" + bid
	} else if ot := c.Query("older_than"); ot != "" {
		if sec, err := strconv.ParseInt(ot, 10, 64); err == nil && sec > 0 {
			// Use stream ID time portion for bound: (<ms-0
			ms := sec * 1000
			max = fmt.Sprintf("(%d-0", ms)
		}
	}
	// newest first
	msgs, err := redisClient.XRevRangeN(c, codegenDLQStream, max, "-", int64(count)).Result()
	if err != nil && err != redis.Nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	// Update DLQ depth gauge and include total size
	total, _ := redisClient.XLen(c, codegenDLQStream).Result()
	SetDLQDepth("codegen", total)
	type item struct {
		ID         string      `json:"id"`
		Payload    interface{} `json:"payload"`
		Reason     string      `json:"reason"`
		Deliveries int64       `json:"deliveries"`
		At         int64       `json:"at"`
	}
	out := make([]item, 0, len(msgs))
	for _, m := range msgs {
		it := item{ID: m.ID}
		if v, ok := m.Values["payload"]; ok {
			it.Payload = v
		}
		if v, ok := m.Values["reason"].(string); ok {
			it.Reason = v
		}
		switch d := m.Values["deliveries"].(type) {
		case int64:
			it.Deliveries = d
		case int:
			it.Deliveries = int64(d)
		case string:
			if n, err := strconv.ParseInt(d, 10, 64); err == nil {
				it.Deliveries = n
			}
		}
		switch at := m.Values["at"].(type) {
		case int64:
			it.At = at
		case int:
			it.At = int64(at)
		case string:
			if n, err := strconv.ParseInt(at, 10, 64); err == nil {
				it.At = n
			}
		}
		out = append(out, it)
	}
	nextBefore := ""
	if len(out) > 0 {
		// XREVRANGE returns newest->oldest; last element is the next page cursor (older)
		nextBefore = out[len(out)-1].ID
	}
	c.JSON(http.StatusOK, gin.H{"items": out, "count": len(out), "total": total, "next_before_id": nextBefore})
}

// ListWebhookDLQ returns recent webhook DLQ entries.
// GET /admin/webhooks/dlq?count=50
func ListWebhookDLQ(c *gin.Context) {
	if redisClient == nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "queue disabled"})
		return
	}
	count := 50
	if v := c.Query("count"); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n > 0 {
			if n > 500 {
				n = 500
			}
			count = n
		}
	}
	// Optional pagination: before_id or older_than (unix seconds)
	max := "+"
	if bid := c.Query("before_id"); bid != "" {
		// Exclusive upper bound to avoid duplicating last seen id
		max = "(" + bid
	} else if ot := c.Query("older_than"); ot != "" {
		if sec, err := strconv.ParseInt(ot, 10, 64); err == nil && sec > 0 {
			// Use stream ID time portion for bound: (<ms-0
			ms := sec * 1000
			max = fmt.Sprintf("(%d-0", ms)
		}
	}
	msgs, err := redisClient.XRevRangeN(c, "aura:webhooks:dlq", max, "-", int64(count)).Result()
	if err != nil && err != redis.Nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	total, _ := redisClient.XLen(c, "aura:webhooks:dlq").Result()
	SetDLQDepth("webhooks", total)
	out := make([]gin.H, 0, len(msgs))
	for _, m := range msgs {
		out = append(out, gin.H{
			"id":        m.ID,
			"org_id":    m.Values["org_id"],
			"endpoint":  m.Values["endpoint"],
			"url":       m.Values["url"],
			"event":     m.Values["event"],
			"attempts":  m.Values["attempts"],
			"last_code": m.Values["last_code"],
			"at":        m.Values["at"],
		})
	}
	nextBefore := ""
	if len(out) > 0 {
		nextBefore = out[len(out)-1]["id"].(string)
	}
	c.JSON(http.StatusOK, gin.H{"items": out, "count": len(out), "total": total, "next_before_id": nextBefore})
}

// RequeueDLQ re-enqueues DLQ messages back to the main codegen stream and removes them from DLQ.
// POST /admin/queue/dlq/requeue { ids?: [], all?: bool, count?: number }
func RequeueDLQ(c *gin.Context) {
	if redisClient == nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "queue disabled"})
		return
	}
	var req struct {
		IDs   []string `json:"ids"`
		All   bool     `json:"all"`
		Count int      `json:"count"`
	}
	if err := c.BindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid json"})
		return
	}
	if req.Count <= 0 {
		req.Count = 200
	}
	if req.Count > 1000 {
		req.Count = 1000
	}
	ids := req.IDs
	// If requeuing all, fetch up to Count oldest first
	if req.All {
		msgs, err := redisClient.XRangeN(c, codegenDLQStream, "-", "+", int64(req.Count)).Result()
		if err != nil && err != redis.Nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		ids = make([]string, 0, len(msgs))
		for _, m := range msgs {
			ids = append(ids, m.ID)
		}
	}
	if len(ids) == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "no ids provided and all=false"})
		return
	}
	// Requeue
	requeued := 0
	failed := 0
	for _, id := range ids {
		// Read message to get payload
		xs, err := redisClient.XRange(c, codegenDLQStream, id, id).Result()
		if err != nil || len(xs) == 0 {
			failed++
			continue
		}
		m := xs[0]
		var payloadStr string
		switch v := m.Values["payload"].(type) {
		case string:
			payloadStr = v
		case []byte:
			payloadStr = string(v)
		default:
			// try json marshal
			if b, err := json.Marshal(v); err == nil {
				payloadStr = string(b)
			}
		}
		if payloadStr == "" {
			failed++
			continue
		}
		// Add back to main stream
		if err := redisClient.XAdd(c, &redis.XAddArgs{Stream: codegenStream, Values: map[string]any{"payload": payloadStr}}).Err(); err != nil {
			failed++
			continue
		}
		// Delete from DLQ
		_ = redisClient.XDel(c, codegenDLQStream, id).Err()
		requeued++
	}
	// Update DLQ depth gauge
	if total, err := redisClient.XLen(c, codegenDLQStream).Result(); err == nil {
		SetDLQDepth("codegen", total)
	}
	c.JSON(http.StatusOK, gin.H{"requeued": requeued, "failed": failed})
}

// DeleteDLQ removes DLQ messages without requeue.
// POST /admin/queue/dlq/delete { ids?: [], all?: bool, count?: number }
func DeleteDLQ(c *gin.Context) {
	if redisClient == nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "queue disabled"})
		return
	}
	var req struct {
		IDs   []string `json:"ids"`
		All   bool     `json:"all"`
		Count int      `json:"count"`
	}
	if err := c.BindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid json"})
		return
	}
	if req.Count <= 0 {
		req.Count = 200
	}
	if req.Count > 1000 {
		req.Count = 1000
	}

	ids := req.IDs
	if req.All {
		msgs, err := redisClient.XRangeN(c, codegenDLQStream, "-", "+", int64(req.Count)).Result()
		if err != nil && err != redis.Nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		ids = make([]string, 0, len(msgs))
		for _, m := range msgs {
			ids = append(ids, m.ID)
		}
	}
	if len(ids) == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "no ids provided and all=false"})
		return
	}
	deleted := 0
	for _, id := range ids {
		if _, err := redisClient.XDel(c, codegenDLQStream, id).Result(); err == nil {
			deleted++
		}
	}
	if total, err := redisClient.XLen(c, codegenDLQStream).Result(); err == nil {
		SetDLQDepth("codegen", total)
	}
	c.JSON(http.StatusOK, gin.H{"deleted": deleted})
}

// AdminHealth returns a full-state health snapshot for dashboards and readiness gates.
// GET /admin/health
// Includes: DB ping, Redis ping, queue stream lengths, group pending, and DLQ size.
func AdminHealth(c *gin.Context) {
	// DB ping with timeout
	dbOK := false
	dbMs := int64(0)
	{
		start := time.Now()
		ctx, cancel := context.WithTimeout(c.Request.Context(), 300*time.Millisecond)
		defer cancel()
		if err := database.DB.DB.PingContext(ctx); err == nil {
			dbOK = true
		}
		dbMs = time.Since(start).Milliseconds()
	}

	// Redis ping (prefer existing queue redisClient; else try env-configured client briefly)
	redisOK := false
	redisMs := int64(0)
	var rdb *redis.Client
	if redisClient != nil {
		rdb = redisClient
	} else {
		addr := os.Getenv("AURA_REDIS_ADDR")
		if addr == "" {
			addr = os.Getenv("REDIS_ADDR")
		}
		if addr != "" {
			rdb = redis.NewClient(&redis.Options{Addr: addr, Password: os.Getenv("AURA_REDIS_PASSWORD")})
			defer func() { _ = rdb.Close() }()
		}
	}
	if rdb != nil {
		start := time.Now()
		ctx, cancel := context.WithTimeout(c.Request.Context(), 300*time.Millisecond)
		defer cancel()
		if err := rdb.Ping(ctx).Err(); err == nil {
			redisOK = true
		}
		redisMs = time.Since(start).Milliseconds()
	}

	// Queue snapshot
	queue := gin.H{"enabled": redisClient != nil}
	if rdb != nil {
		ctx, cancel := context.WithTimeout(c.Request.Context(), 400*time.Millisecond)
		defer cancel()
		// Stream lengths
		codegenLen, _ := rdb.XLen(ctx, codegenStream).Result()
		dlqLen, _ := rdb.XLen(ctx, codegenDLQStream).Result()
		// Group pending summary
		pendingTotal := int64(0)
		if p, err := rdb.XPending(ctx, codegenStream, codegenGroup).Result(); err == nil && p != nil {
			pendingTotal = p.Count
		}
		queue = gin.H{
			"enabled":               true,
			"codegen_stream_len":    codegenLen,
			"codegen_group_pending": pendingTotal,
			"dlq_len":               dlqLen,
		}
		// Keep DLQ depth gauge fresh
		SetDLQDepth("codegen", dlqLen)
	}

	status := http.StatusOK
	overall := "ok"
	if !dbOK || (rdb != nil && !redisOK) {
		status = http.StatusServiceUnavailable
		overall = "degraded"
	}
	c.JSON(status, gin.H{
		"status": overall,
		"db":     gin.H{"ok": dbOK, "ping_ms": dbMs},
		"redis":  gin.H{"ok": redisOK, "ping_ms": redisMs},
		"queue":  queue,
		"ts":     time.Now().UTC().Format(time.RFC3339),
	})
}

// QueueDrain toggles drain mode to stop reading new messages while allowing pending to finish.
// POST /admin/queue/drain { "enable": true|false }
func QueueDrain(c *gin.Context) {
	var req struct {
		Enable bool `json:"enable"`
	}
	if err := c.BindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid json"})
		return
	}
	drainMode = req.Enable
	// Return status
	status := gin.H{"drain": drainMode}
	if redisClient != nil {
		if p, err := redisClient.XPending(c, codegenStream, codegenGroup).Result(); err == nil && p != nil {
			status["codegen_pending"] = p.Count
		}
		if x, err := redisClient.XLen(c, codegenDLQStream).Result(); err == nil {
			status["dlq_len"] = x
		}
	}
	c.JSON(http.StatusOK, status)
}

// QueueDrainStatus returns the current drain status and queue snapshot.
// GET /admin/queue/drain/status
func QueueDrainStatus(c *gin.Context) {
	status := gin.H{"drain": drainMode}
	if redisClient != nil {
		if p, err := redisClient.XPending(c, codegenStream, codegenGroup).Result(); err == nil && p != nil {
			status["codegen_pending"] = p.Count
		}
		if x, err := redisClient.XLen(c, codegenDLQStream).Result(); err == nil {
			status["dlq_len"] = x
		}
	}
	c.JSON(http.StatusOK, status)
}

// QueueDrainComplete returns whether the worker has drained (no pending for threshold ticks)
// GET /admin/queue/drain/complete
func QueueDrainComplete(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{"drained": drainedComplete})
}

// RequeueWebhookDLQ retries once by re-dispatching DLQ entries and deletes on success
// POST /admin/webhooks/dlq/requeue { ids?: [], all?: bool, count?: number }
func RequeueWebhookDLQ(c *gin.Context) {
	if redisClient == nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "queue disabled"})
		return
	}
	var req struct {
		IDs   []string `json:"ids"`
		All   bool     `json:"all"`
		Count int      `json:"count"`
	}
	if err := c.BindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid json"})
		return
	}
	if req.Count <= 0 {
		req.Count = 100
	}
	if req.Count > 1000 {
		req.Count = 1000
	}

	ids := req.IDs
	if req.All {
		msgs, err := redisClient.XRangeN(c, "aura:webhooks:dlq", "-", "+", int64(req.Count)).Result()
		if err != nil && err != redis.Nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		ids = make([]string, 0, len(msgs))
		for _, m := range msgs {
			ids = append(ids, m.ID)
		}
	}
	if len(ids) == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "no ids provided and all=false"})
		return
	}
	requeued := 0
	failed := 0
	for _, id := range ids {
		xs, err := redisClient.XRange(c, "aura:webhooks:dlq", id, id).Result()
		if err != nil || len(xs) == 0 {
			failed++
			continue
		}
		m := xs[0]
		endpointID, _ := m.Values["endpoint"].(string)
		url, _ := m.Values["url"].(string)
		event, _ := m.Values["event"].(string)
		payloadStr, _ := m.Values["payload"].(string)
		// Fetch endpoint secret to sign again (if still present)
		var ep struct {
			ID     string `db:"id"`
			Secret string `db:"secret"`
		}
		_ = database.DB.Get(&ep, `SELECT id, secret FROM webhook_endpoints WHERE id=$1`, endpointID)
		// Retry once
		success := false
		if payloadStr != "" && url != "" && ep.Secret != "" {
			ts := time.Now().Unix()
			sig := utils.ComputeWebhookSignature(ep.Secret, ts, []byte(payloadStr))
			req2, err2 := http.NewRequest("POST", url, strings.NewReader(payloadStr))
			if err2 == nil {
				req2.Header.Set("Content-Type", "application/json")
				req2.Header.Set("AURA-Event", event)
				req2.Header.Set("AURA-Webhook-ID", endpointID)
				req2.Header.Set("AURA-Event-ID", id)
				req2.Header.Set("Idempotency-Key", id)
				req2.Header.Set("AURA-Signature", fmt.Sprintf("t=%d,v1=%s", ts, sig))
				start := time.Now()
				resp, err3 := (&http.Client{Timeout: 3 * time.Second}).Do(req2)
				dur := time.Since(start)
				RecordExternalOp("webhook_send", dur, err3 == nil && resp != nil && resp.StatusCode/100 == 2)
				if err3 == nil {
					if resp.StatusCode/100 == 2 {
						success = true
					}
					_ = resp.Body.Close()
				}
			}
		}
		if success {
			_, _ = redisClient.XDel(c, "aura:webhooks:dlq", id).Result()
			requeued++
		} else {
			failed++
		}
	}
	if total, err := redisClient.XLen(c, "aura:webhooks:dlq").Result(); err == nil {
		SetDLQDepth("webhooks", total)
	}
	c.JSON(http.StatusOK, gin.H{"requeued": requeued, "failed": failed})
}

// DeleteWebhookDLQ removes entries from the webhook DLQ
// POST /admin/webhooks/dlq/delete { ids?: [], all?: bool, count?: number }
func DeleteWebhookDLQ(c *gin.Context) {
	if redisClient == nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "queue disabled"})
		return
	}
	var req struct {
		IDs   []string `json:"ids"`
		All   bool     `json:"all"`
		Count int      `json:"count"`
	}
	if err := c.BindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid json"})
		return
	}
	if req.Count <= 0 {
		req.Count = 100
	}
	if req.Count > 1000 {
		req.Count = 1000
	}
	ids := req.IDs
	if req.All {
		msgs, err := redisClient.XRangeN(c, "aura:webhooks:dlq", "-", "+", int64(req.Count)).Result()
		if err != nil && err != redis.Nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		ids = make([]string, 0, len(msgs))
		for _, m := range msgs {
			ids = append(ids, m.ID)
		}
	}
	if len(ids) == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "no ids provided and all=false"})
		return
	}
	deleted := 0
	for _, id := range ids {
		if _, err := redisClient.XDel(c, "aura:webhooks:dlq", id).Result(); err == nil {
			deleted++
		}
	}
	if total, err := redisClient.XLen(c, "aura:webhooks:dlq").Result(); err == nil {
		SetDLQDepth("webhooks", total)
	}
	c.JSON(http.StatusOK, gin.H{"deleted": deleted})
}
