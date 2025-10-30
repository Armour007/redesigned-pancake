package api

import (
	"bytes"
	"context"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	database "github.com/Armour007/aura-backend/internal"
	"github.com/robfig/cron/v3"
)

// In-memory per-org audit export schedule (ephemeral)
// For persistence, migrate to a DB table and load on startup.

type AuditExportCfg struct {
	Cron     string // cron spec
	DestType string // webhook|file
	Dest     string // URL or file directory
	Format   string // json|csv (json bundled)
	Lookback string // Go duration (e.g., 720h for 30d)
}

var (
	cronOnce sync.Once
	sched    *cron.Cron
	schedMu  sync.Mutex
	orgJobs  = map[string]cron.EntryID{}
	orgCfg   = map[string]AuditExportCfg{}
)

func StartAuditScheduler() {
	cronOnce.Do(func() {
		sched = cron.New(cron.WithParser(cron.NewParser(cron.Minute | cron.Hour | cron.Dom | cron.Month | cron.Dow)))
		sched.Start()
		// Load persisted schedules from DB
		go LoadAuditSchedulesFromDB()
	})
}

// Set or update schedule for an org (ephemeral)
func setOrgAuditSchedule(orgID string, cfg AuditExportCfg) error {
	schedMu.Lock()
	defer schedMu.Unlock()
	if sched == nil {
		StartAuditScheduler()
	}
	// remove existing job if any
	if id, ok := orgJobs[orgID]; ok {
		sched.Remove(id)
		delete(orgJobs, orgID)
	}
	orgCfg[orgID] = cfg
	// persist to DB (upsert)
	_, err := database.DB.Exec(`INSERT INTO audit_export_schedules(org_id,cron,dest_type,dest,format,lookback,updated_at)
		VALUES($1,$2,$3,$4,$5,$6,NOW())
		ON CONFLICT (org_id) DO UPDATE SET cron=EXCLUDED.cron,dest_type=EXCLUDED.dest_type,dest=EXCLUDED.dest,format=EXCLUDED.format,lookback=EXCLUDED.lookback,updated_at=NOW()`,
		orgID, cfg.Cron, cfg.DestType, cfg.Dest, cfg.Format, cfg.Lookback)
	if err != nil {
		return err
	}
	// add new job
	id, err2 := sched.AddFunc(cfg.Cron, func() {
		// compute time range
		lb := 30 * 24 * time.Hour
		if d, err := time.ParseDuration(cfg.Lookback); err == nil && d > 0 {
			lb = d
		}
		to := time.Now().UTC()
		from := to.Add(-lb)
		// generate payload
		switch strings.ToLower(cfg.Format) {
		case "json":
			if zipBytes, err := GenerateAuditBundleZip(orgID, from, to); err == nil {
				deliver(orgID, cfg, zipBytes, "application/zip")
			} else {
				log.Printf("audit schedule: json bundle error for org=%s: %v", orgID, err)
			}
		case "csv":
			if zipBytes, err := GenerateAuditCSVZip(orgID, from, to); err == nil {
				deliver(orgID, cfg, zipBytes, "application/zip")
			} else {
				log.Printf("audit schedule: csv bundle error for org=%s: %v", orgID, err)
			}
		}
	})
	if err2 == nil {
		orgJobs[orgID] = id
	}
	return err2
}

func deliver(orgID string, cfg AuditExportCfg, body []byte, ctype string) {
	switch strings.ToLower(cfg.DestType) {
	case "webhook":
		client := &http.Client{Timeout: 10 * time.Second}
		req, _ := http.NewRequestWithContext(context.Background(), http.MethodPost, cfg.Dest, bytes.NewReader(body))
		req.Header.Set("Content-Type", ctype)
		_, _ = client.Do(req)
	case "file":
		_ = os.MkdirAll(cfg.Dest, 0o755)
		fn := time.Now().UTC().Format("20060102-150405") + "-" + orgID + ".zip"
		_ = os.WriteFile(filepath.Join(cfg.Dest, fn), body, 0o644)
	}
}

// LoadAuditSchedulesFromDB reads all schedules and registers cron jobs
func LoadAuditSchedulesFromDB() {
	type row struct {
		OrgID    string `db:"org_id"`
		Cron     string `db:"cron"`
		DestType string `db:"dest_type"`
		Dest     string `db:"dest"`
		Format   string `db:"format"`
		Lookback string `db:"lookback"`
	}
	rows := []row{}
	if err := database.DB.Select(&rows, `SELECT org_id::text, cron, dest_type, dest, format, lookback FROM audit_export_schedules`); err != nil {
		log.Printf("audit schedule: load failed: %v", err)
		return
	}
	for _, r := range rows {
		_ = setOrgAuditSchedule(r.OrgID, AuditExportCfg{Cron: r.Cron, DestType: r.DestType, Dest: r.Dest, Format: r.Format, Lookback: r.Lookback})
	}
}
