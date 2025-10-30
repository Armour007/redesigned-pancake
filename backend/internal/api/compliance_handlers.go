package api

import (
	"archive/zip"
	"bytes"
	"encoding/csv"
	"encoding/json"
	"net/http"
	"os"
	"strings"
	"time"

	database "github.com/Armour007/aura-backend/internal"
	"github.com/gin-gonic/gin"
)

// GET /organizations/:orgId/regulator/snapshot
func RegulatorSnapshot(c *gin.Context) {
	orgID := c.Param("orgId")
	if orgID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "missing org"})
		return
	}

	out := gin.H{"org_id": orgID}
	// Collect a few high-level counts; ignore individual errors to keep snapshot resilient.
	var nUsers, nAgents, nAPIKeys, nPolicies, nCerts int
	_ = database.DB.Get(&nUsers, "SELECT COUNT(1) FROM organization_members WHERE organization_id=$1", orgID)
	_ = database.DB.Get(&nAgents, "SELECT COUNT(1) FROM agents WHERE organization_id=$1", orgID)
	_ = database.DB.Get(&nAPIKeys, "SELECT COUNT(1) FROM api_keys WHERE organization_id=$1 AND revoked_at IS NULL", orgID)
	_ = database.DB.Get(&nPolicies, "SELECT COUNT(1) FROM policy_versions pv JOIN policies p ON p.id=pv.policy_id WHERE p.organization_id=$1", orgID)
	_ = database.DB.Get(&nCerts, "SELECT COUNT(1) FROM client_certs WHERE org_id=$1", orgID)
	out["counts"] = gin.H{"users": nUsers, "agents": nAgents, "api_keys": nAPIKeys, "policies": nPolicies, "client_certs": nCerts}

	var anchor gin.H = gin.H{}
	var date, hash, ext string
	_ = database.DB.QueryRowx("SELECT anchor_date, root_hash, COALESCE(external_ref,'') FROM audit_anchors WHERE org_id=$1 ORDER BY anchor_date DESC LIMIT 1", orgID).
		Scan(&date, &hash, &ext)
	if hash != "" {
		anchor = gin.H{"date": date, "root_hash": hash, "external_ref": ext}
	}
	out["latest_audit_anchor"] = anchor

	c.JSON(http.StatusOK, out)
}

// GET /organizations/:orgId/regulator/compliance-mapping
// Returns automated SOC2/GDPR mappings summary (counts by control/article) with references to evidence types.
func GetComplianceMapping(c *gin.Context) {
	orgID := c.Param("orgId")
	if orgID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "missing org"})
		return
	}

	// Minimal heuristic mapping: event name -> SOC2 controls and GDPR articles
	typeMap := map[string]struct {
		SOC2 []string
		GDPR []string
	}{
		"policy_version_approval":      {SOC2: []string{"CC5.2", "CC2.3"}},
		"policy_version_activated":     {SOC2: []string{"CC5.3"}},
		"federation_boundary_crossing": {SOC2: []string{"CC6.6"}, GDPR: []string{"Art.28"}},
		"trust_token_revoked":          {SOC2: []string{"CC6.1"}, GDPR: []string{"Art.32"}},
		"audit_anchor_set":             {SOC2: []string{"CC7.2"}},
		"webhook_delivery":             {SOC2: []string{"CC7.4"}},
		"smtp_send":                    {SOC2: []string{"CC5.2"}},
	}

	// Pull recent audit ledger and event logs
	type row struct {
		Event string `db:"event"`
	}
	rows := []row{}
	_ = database.DB.Select(&rows, `SELECT event FROM audit_ledger WHERE org_id=$1 AND created_at > NOW() - INTERVAL '90 days'`, orgID)
	r2 := []row{}
	_ = database.DB.Select(&r2, `SELECT event FROM event_logs WHERE organization_id=$1 AND created_at > NOW() - INTERVAL '90 days'`, orgID)
	rows = append(rows, r2...)

	soc2Counts := map[string]int{}
	gdprCounts := map[string]int{}
	for _, r := range rows {
		if m, ok := typeMap[r.Event]; ok {
			for _, cno := range m.SOC2 {
				soc2Counts[cno]++
			}
			for _, a := range m.GDPR {
				gdprCounts[a]++
			}
		}
	}
	c.JSON(http.StatusOK, gin.H{
		"org_id":        orgID,
		"soc2":          soc2Counts,
		"gdpr":          gdprCounts,
		"evidence":      []string{"audit_ledger", "event_logs", "decision_traces", "policy_versions", "client_certs", "trust_keys", "revocations"},
		"lookback_days": 90,
	})
}

// GET /organizations/:orgId/regulator/audit-bundle[?from=RFC3339&to=RFC3339&as=zip]
// Produces an exportable audit bundle of key evidence for regulators.
func ExportAuditBundle(c *gin.Context) {
	orgID := c.Param("orgId")
	if orgID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "missing org"})
		return
	}
	var from, to time.Time
	var err error
	if s := c.Query("from"); s != "" {
		from, err = time.Parse(time.RFC3339, s)
		if err != nil {
			c.JSON(400, gin.H{"error": "bad from"})
			return
		}
	}
	if s := c.Query("to"); s != "" {
		to, err = time.Parse(time.RFC3339, s)
		if err != nil {
			c.JSON(400, gin.H{"error": "bad to"})
			return
		}
	}
	if to.IsZero() {
		to = time.Now().UTC()
	}
	if from.IsZero() {
		from = to.AddDate(0, -1, 0)
	} // default 30 days

	type qrow struct{ JSON json.RawMessage }
	bundle := map[string]any{}

	// event_logs
	el := []map[string]any{}
	{
		rows, _ := database.DB.Queryx(`SELECT to_jsonb(e.*) AS json FROM event_logs e WHERE e.organization_id=$1 AND e.created_at BETWEEN $2 AND $3 ORDER BY e.created_at ASC`, orgID, from, to)
		defer func() { _ = rows.Close() }()
		for rows.Next() {
			var r qrow
			_ = rows.Scan(&r.JSON)
			var m map[string]any
			_ = json.Unmarshal(r.JSON, &m)
			el = append(el, m)
		}
		bundle["event_logs"] = el
	}
	// audit_ledger
	al := []map[string]any{}
	{
		rows, _ := database.DB.Queryx(`SELECT to_jsonb(a.*) AS json FROM audit_ledger a WHERE a.org_id=$1 AND a.created_at BETWEEN $2 AND $3 ORDER BY a.created_at ASC`, orgID, from, to)
		defer func() { _ = rows.Close() }()
		for rows.Next() {
			var r qrow
			_ = rows.Scan(&r.JSON)
			var m map[string]any
			_ = json.Unmarshal(r.JSON, &m)
			al = append(al, m)
		}
		bundle["audit_ledger"] = al
	}
	// decision_traces
	dt := []map[string]any{}
	{
		rows, _ := database.DB.Queryx(`SELECT to_jsonb(d.*) AS json FROM decision_traces d WHERE d.org_id=$1 AND d.created_at BETWEEN $2 AND $3 ORDER BY d.created_at ASC`, orgID, from, to)
		defer func() { _ = rows.Close() }()
		for rows.Next() {
			var r qrow
			_ = rows.Scan(&r.JSON)
			var m map[string]any
			_ = json.Unmarshal(r.JSON, &m)
			dt = append(dt, m)
		}
		bundle["decision_traces"] = dt
	}
	// policy_versions
	pv := []map[string]any{}
	{
		rows, _ := database.DB.Queryx(`SELECT to_jsonb(pv.*) AS json FROM policy_versions pv JOIN policies p ON p.id=pv.policy_id WHERE p.organization_id=$1 ORDER BY pv.created_at ASC`, orgID)
		defer func() { _ = rows.Close() }()
		for rows.Next() {
			var r qrow
			_ = rows.Scan(&r.JSON)
			var m map[string]any
			_ = json.Unmarshal(r.JSON, &m)
			pv = append(pv, m)
		}
		bundle["policy_versions"] = pv
	}
	// trust_keys
	tk := []map[string]any{}
	{
		rows, _ := database.DB.Queryx(`SELECT to_jsonb(t.*) AS json FROM trust_keys t WHERE t.org_id=$1 ORDER BY t.created_at ASC`, orgID)
		defer func() { _ = rows.Close() }()
		for rows.Next() {
			var r qrow
			_ = rows.Scan(&r.JSON)
			var m map[string]any
			_ = json.Unmarshal(r.JSON, &m)
			tk = append(tk, m)
		}
		bundle["trust_keys"] = tk
	}
	// revocations
	rv := []map[string]any{}
	{
		rows, _ := database.DB.Queryx(`SELECT to_jsonb(r.*) AS json FROM trust_token_revocations r WHERE r.org_id=$1 AND r.created_at BETWEEN $2 AND $3 ORDER BY r.created_at ASC`, orgID, from, to)
		defer func() { _ = rows.Close() }()
		for rows.Next() {
			var r qrow
			_ = rows.Scan(&r.JSON)
			var m map[string]any
			_ = json.Unmarshal(r.JSON, &m)
			rv = append(rv, m)
		}
		bundle["trust_token_revocations"] = rv
	}

	// respond as JSON or ZIP
	if c.Query("as") == "zip" {
		var buf bytes.Buffer
		zw := zip.NewWriter(&buf)
		addJSON := func(name string, v any) {
			b, _ := json.MarshalIndent(v, "", "  ")
			f, _ := zw.Create(name)
			_, _ = f.Write(b)
		}
		addJSON("bundle.json", bundle)
		_ = zw.Close()
		c.Data(http.StatusOK, "application/zip", buf.Bytes())
		return
	}
	c.JSON(http.StatusOK, gin.H{"from": from.Format(time.RFC3339), "to": to.Format(time.RFC3339), "bundle": bundle})
}

// GenerateAuditBundleZip creates a ZIP with bundle.json for reuse (e.g., scheduled exports)
func GenerateAuditBundleZip(orgID string, from, to time.Time) ([]byte, error) {
	// Reuse the same bundle-building logic as above
	type qrow struct{ JSON json.RawMessage }
	bundle := map[string]any{}
	// event_logs
	el := []map[string]any{}
	{
		rows, _ := database.DB.Queryx(`SELECT to_jsonb(e.*) AS json FROM event_logs e WHERE e.organization_id=$1 AND e.created_at BETWEEN $2 AND $3 ORDER BY e.created_at ASC`, orgID, from, to)
		defer func() { _ = rows.Close() }()
		for rows.Next() {
			var r qrow
			_ = rows.Scan(&r.JSON)
			var m map[string]any
			_ = json.Unmarshal(r.JSON, &m)
			el = append(el, m)
		}
		bundle["event_logs"] = el
	}
	// audit_ledger
	al := []map[string]any{}
	{
		rows, _ := database.DB.Queryx(`SELECT to_jsonb(a.*) AS json FROM audit_ledger a WHERE a.org_id=$1 AND a.created_at BETWEEN $2 AND $3 ORDER BY a.created_at ASC`, orgID, from, to)
		defer func() { _ = rows.Close() }()
		for rows.Next() {
			var r qrow
			_ = rows.Scan(&r.JSON)
			var m map[string]any
			_ = json.Unmarshal(r.JSON, &m)
			al = append(al, m)
		}
		bundle["audit_ledger"] = al
	}
	// decision_traces
	dt := []map[string]any{}
	{
		rows, _ := database.DB.Queryx(`SELECT to_jsonb(d.*) AS json FROM decision_traces d WHERE d.org_id=$1 AND d.created_at BETWEEN $2 AND $3 ORDER BY d.created_at ASC`, orgID, from, to)
		defer func() { _ = rows.Close() }()
		for rows.Next() {
			var r qrow
			_ = rows.Scan(&r.JSON)
			var m map[string]any
			_ = json.Unmarshal(r.JSON, &m)
			dt = append(dt, m)
		}
		bundle["decision_traces"] = dt
	}
	// policy_versions
	pv := []map[string]any{}
	{
		rows, _ := database.DB.Queryx(`SELECT to_jsonb(pv.*) AS json FROM policy_versions pv JOIN policies p ON p.id=pv.policy_id WHERE p.organization_id=$1 ORDER BY pv.created_at ASC`, orgID)
		defer func() { _ = rows.Close() }()
		for rows.Next() {
			var r qrow
			_ = rows.Scan(&r.JSON)
			var m map[string]any
			_ = json.Unmarshal(r.JSON, &m)
			pv = append(pv, m)
		}
		bundle["policy_versions"] = pv
	}
	// trust_keys
	tk := []map[string]any{}
	{
		rows, _ := database.DB.Queryx(`SELECT to_jsonb(t.*) AS json FROM trust_keys t WHERE t.org_id=$1 ORDER BY t.created_at ASC`, orgID)
		defer func() { _ = rows.Close() }()
		for rows.Next() {
			var r qrow
			_ = rows.Scan(&r.JSON)
			var m map[string]any
			_ = json.Unmarshal(r.JSON, &m)
			tk = append(tk, m)
		}
		bundle["trust_keys"] = tk
	}
	// revocations
	rv := []map[string]any{}
	{
		rows, _ := database.DB.Queryx(`SELECT to_jsonb(r.*) AS json FROM trust_token_revocations r WHERE r.org_id=$1 AND r.created_at BETWEEN $2 AND $3 ORDER BY r.created_at ASC`, orgID, from, to)
		defer func() { _ = rows.Close() }()
		for rows.Next() {
			var r qrow
			_ = rows.Scan(&r.JSON)
			var m map[string]any
			_ = json.Unmarshal(r.JSON, &m)
			rv = append(rv, m)
		}
		bundle["trust_token_revocations"] = rv
	}
	// zip
	var buf bytes.Buffer
	zw := zip.NewWriter(&buf)
	addJSON := func(name string, v any) {
		b, _ := json.MarshalIndent(v, "", "  ")
		f, _ := zw.Create(name)
		_, _ = f.Write(b)
	}
	addJSON("bundle.json", bundle)
	_ = zw.Close()
	return buf.Bytes(), nil
}

// GET /organizations/:orgId/regulator/audit-export?format=csv|json&as=zip
// Streams CSV files or JSON bundle, optionally zipped.
func ExportAuditData(c *gin.Context) {
	orgID := c.Param("orgId")
	if orgID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "missing org"})
		return
	}
	format := strings.ToLower(strings.TrimSpace(c.Query("format")))
	if format == "" {
		format = "json"
	}
	var from, to time.Time
	var err error
	if s := c.Query("from"); s != "" {
		if from, err = time.Parse(time.RFC3339, s); err != nil {
			c.JSON(400, gin.H{"error": "bad from"})
			return
		}
	}
	if s := c.Query("to"); s != "" {
		if to, err = time.Parse(time.RFC3339, s); err != nil {
			c.JSON(400, gin.H{"error": "bad to"})
			return
		}
	}
	if to.IsZero() {
		to = time.Now().UTC()
	}
	if from.IsZero() {
		from = to.AddDate(0, -1, 0)
	}

	if format == "json" {
		// Reuse bundle
		zipb, _ := GenerateAuditBundleZip(orgID, from, to)
		if c.Query("as") == "zip" {
			c.Data(http.StatusOK, "application/zip", zipb)
			return
		}
		// Unzip not implemented here; return JSON by rebuilding like ExportAuditBundle
		ExportAuditBundle(c)
		return
	}
	if format != "csv" {
		c.JSON(400, gin.H{"error": "unsupported format"})
		return
	}
	// Build CSV zip with multiple files
	var buf bytes.Buffer
	zw := zip.NewWriter(&buf)
	writeCSV := func(name string, headers []string, rows [][]string) {
		f, _ := zw.Create(name)
		w := csv.NewWriter(f)
		_ = w.Write(headers)
		for _, r := range rows {
			_ = w.Write(r)
		}
		w.Flush()
	}
	// event_logs (subset of fields)
	{
		rows, _ := database.DB.Queryx(`SELECT id, organization_id, agent_id, event_type, timestamp FROM event_logs WHERE organization_id=$1 AND timestamp BETWEEN $2 AND $3 ORDER BY timestamp ASC`, orgID, from, to)
		defer func() { _ = rows.Close() }()
		out := [][]string{}
		for rows.Next() {
			var id, org, evt string
			var agent *string
			var ts time.Time
			_ = rows.Scan(&id, &org, &agent, &evt, &ts)
			a := ""
			if agent != nil {
				a = *agent
			}
			out = append(out, []string{id, org, a, evt, ts.Format(time.RFC3339)})
		}
		writeCSV("event_logs.csv", []string{"id", "org_id", "agent_id", "event", "timestamp"}, out)
	}
	// audit_ledger
	{
		rows, _ := database.DB.Queryx(`SELECT id, org_id, event, created_at FROM audit_ledger WHERE org_id=$1 AND created_at BETWEEN $2 AND $3 ORDER BY created_at ASC`, orgID, from, to)
		defer func() { _ = rows.Close() }()
		out := [][]string{}
		for rows.Next() {
			var id, org, evt string
			var ts time.Time
			_ = rows.Scan(&id, &org, &evt, &ts)
			out = append(out, []string{id, org, evt, ts.Format(time.RFC3339)})
		}
		writeCSV("audit_ledger.csv", []string{"id", "org_id", "event", "created_at"}, out)
	}
	_ = zw.Close()
	c.Data(http.StatusOK, "application/zip", buf.Bytes())
}

// GenerateAuditCSVZip builds a CSV ZIP similar to ExportAuditData(format=csv)
func GenerateAuditCSVZip(orgID string, from, to time.Time) ([]byte, error) {
	var buf bytes.Buffer
	zw := zip.NewWriter(&buf)
	writeCSV := func(name string, headers []string, rows [][]string) {
		f, _ := zw.Create(name)
		w := csv.NewWriter(f)
		_ = w.Write(headers)
		for _, r := range rows {
			_ = w.Write(r)
		}
		w.Flush()
	}
	// event_logs (subset of fields)
	{
		rows, _ := database.DB.Queryx(`SELECT id, organization_id, agent_id, event_type, timestamp FROM event_logs WHERE organization_id=$1 AND timestamp BETWEEN $2 AND $3 ORDER BY timestamp ASC`, orgID, from, to)
		defer func() { _ = rows.Close() }()
		out := [][]string{}
		for rows.Next() {
			var id, org, evt string
			var agent *string
			var ts time.Time
			_ = rows.Scan(&id, &org, &agent, &evt, &ts)
			a := ""
			if agent != nil {
				a = *agent
			}
			out = append(out, []string{id, org, a, evt, ts.Format(time.RFC3339)})
		}
		writeCSV("event_logs.csv", []string{"id", "org_id", "agent_id", "event", "timestamp"}, out)
	}
	// audit_ledger
	{
		rows, _ := database.DB.Queryx(`SELECT id, org_id, event, created_at FROM audit_ledger WHERE org_id=$1 AND created_at BETWEEN $2 AND $3 ORDER BY created_at ASC`, orgID, from, to)
		defer func() { _ = rows.Close() }()
		out := [][]string{}
		for rows.Next() {
			var id, org, evt string
			var ts time.Time
			_ = rows.Scan(&id, &org, &evt, &ts)
			out = append(out, []string{id, org, evt, ts.Format(time.RFC3339)})
		}
		writeCSV("audit_ledger.csv", []string{"id", "org_id", "event", "created_at"}, out)
	}
	_ = zw.Close()
	return buf.Bytes(), nil
}

// POST /organizations/:orgId/regulator/audit-export/schedule
// Body: { cron: "0 2 * * *", dest_type: "webhook"|"file", dest: "https://..."|"/tmp/exports", format: "json"|"csv", lookback: "720h" }
// Ephemeral (in-memory) until persisted in DB.
func SetAuditExportSchedule(c *gin.Context) {
	orgID := c.Param("orgId")
	if orgID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "missing org"})
		return
	}
	var body struct {
		Cron     string `json:"cron"`
		DestType string `json:"dest_type"`
		Dest     string `json:"dest"`
		Format   string `json:"format"`
		Lookback string `json:"lookback"`
	}
	if err := c.ShouldBindJSON(&body); err != nil {
		c.JSON(400, gin.H{"error": "bad json"})
		return
	}
	if strings.TrimSpace(body.Cron) == "" {
		// allow a simple fallback via env for demo
		body.Cron = os.Getenv("AURA_AUDIT_EXPORT_CRON")
	}
	if strings.TrimSpace(body.Cron) == "" || strings.TrimSpace(body.DestType) == "" || strings.TrimSpace(body.Dest) == "" {
		c.JSON(400, gin.H{"error": "cron, dest_type and dest required"})
		return
	}
	if body.Format == "" {
		body.Format = "json"
	}
	if body.Lookback == "" {
		body.Lookback = "720h"
	}
	cfg := AuditExportCfg{Cron: body.Cron, DestType: body.DestType, Dest: body.Dest, Format: body.Format, Lookback: body.Lookback}
	if err := setOrgAuditSchedule(orgID, cfg); err != nil {
		c.JSON(400, gin.H{"error": err.Error()})
		return
	}
	c.JSON(200, gin.H{"ok": true})
}

// GET /organizations/:orgId/regulator/audit-export/schedule
func GetAuditExportSchedule(c *gin.Context) {
	orgID := c.Param("orgId")
	schedMu.Lock()
	defer schedMu.Unlock()
	if cfg, ok := orgCfg[orgID]; ok {
		c.JSON(200, gin.H{"org_id": orgID, "schedule": cfg})
		return
	}
	c.JSON(200, gin.H{"org_id": orgID, "schedule": nil})
}
