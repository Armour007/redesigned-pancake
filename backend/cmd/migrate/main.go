package main

import (
	"fmt"
	"io/fs"
	"log"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	database "github.com/Armour007/aura-backend/internal"
)

func main() {
	// Connect to DB using existing env-based configuration
	database.Connect()

	// Ensure schema_migrations table exists
	ensureMigrationsTable()

	// Find and sort migration files
	migDir := filepath.Join("db", "migrations")
	files := collectSQLFiles(migDir)
	if len(files) == 0 {
		log.Println("No migration files found, skipping.")
		return
	}

	applied := getAppliedMigrations()

	for _, f := range files {
		name := filepath.Base(f)
		if applied[name] {
			continue // already applied
		}
		upSQL, err := extractGooseUp(f)
		if err != nil {
			log.Fatalf("Failed extracting Up section from %s: %v", name, err)
		}
		if strings.TrimSpace(upSQL) == "" {
			log.Printf("Skipping empty Up migration: %s", name)
			markApplied(name)
			continue
		}
		log.Printf("Applying migration: %s", name)
		if err := execStatements(upSQL); err != nil {
			log.Fatalf("Migration %s failed: %v", name, err)
		}
		markApplied(name)
	}
	log.Println("Migrations applied successfully.")
}

func ensureMigrationsTable() {
	_, err := database.DB.Exec(`
        CREATE TABLE IF NOT EXISTS schema_migrations (
            version TEXT PRIMARY KEY,
            applied_at timestamptz NOT NULL DEFAULT now()
        )
    `)
	if err != nil {
		log.Fatalf("Unable to ensure schema_migrations table: %v", err)
	}
}

func getAppliedMigrations() map[string]bool {
	rows, err := database.DB.Queryx("SELECT version FROM schema_migrations")
	if err != nil {
		log.Fatalf("Unable to query schema_migrations: %v", err)
	}
	defer rows.Close()
	applied := make(map[string]bool)
	for rows.Next() {
		var v string
		if err := rows.Scan(&v); err != nil {
			log.Fatalf("Scan error: %v", err)
		}
		applied[v] = true
	}
	return applied
}

func markApplied(version string) {
	_, err := database.DB.Exec("INSERT INTO schema_migrations(version, applied_at) VALUES ($1, $2) ON CONFLICT (version) DO NOTHING", version, time.Now())
	if err != nil {
		log.Fatalf("Failed marking migration applied %s: %v", version, err)
	}
}

func collectSQLFiles(dir string) []string {
	var files []string
	_ = filepath.WalkDir(dir, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			return nil
		}
		if strings.HasSuffix(strings.ToLower(d.Name()), ".sql") {
			files = append(files, path)
		}
		return nil
	})
	sort.Strings(files)
	return files
}

func extractGooseUp(path string) (string, error) {
	b, err := os.ReadFile(path)
	if err != nil {
		return "", err
	}
	content := string(b)
	// Find -- +goose Up ... until -- +goose Down or end
	lower := strings.ToLower(content)
	upIdx := strings.Index(lower, "-- +goose up")
	if upIdx == -1 {
		// If no markers, assume whole file is up
		return content, nil
	}
	// slice from end of line after the marker
	rest := content[upIdx:]
	// find next line break after marker
	nl := strings.Index(rest, "\n")
	if nl != -1 {
		rest = rest[nl+1:]
	} else {
		rest = ""
	}
	downMarker := strings.Index(strings.ToLower(rest), "-- +goose down")
	if downMarker != -1 {
		rest = rest[:downMarker]
	}
	return rest, nil
}

// execStatements splits SQL by ';' and executes sequentially, ignoring benign "already exists" errors.
func execStatements(sql string) error {
	// naive split: good enough for simple CREATE/DROP statements here
	stmts := strings.Split(sql, ";")
	for _, raw := range stmts {
		stmt := strings.TrimSpace(raw)
		if stmt == "" {
			continue
		}
		if _, err := database.DB.Exec(stmt); err != nil {
			// ignore common idempotent errors
			msg := strings.ToLower(err.Error())
			if strings.Contains(msg, "already exists") || strings.Contains(msg, "duplicate") {
				log.Printf("Ignoring idempotent error for statement: %s -> %v", short(stmt), err)
				continue
			}
			return fmt.Errorf("statement failed: %v", err)
		}
	}
	return nil
}

func short(s string) string {
	if len(s) > 120 {
		return s[:120] + "..."
	}
	return s
}
