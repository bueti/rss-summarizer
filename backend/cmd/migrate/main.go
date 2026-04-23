package main

import (
	"database/sql"
	"errors"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/joho/godotenv"
	_ "github.com/lib/pq"
)

func main() {
	_ = godotenv.Load()

	dbURL := os.Getenv("DATABASE_URL")
	if dbURL == "" {
		log.Fatal("DATABASE_URL not set")
	}

	db, err := sql.Open("postgres", dbURL)
	if err != nil {
		log.Fatalf("Failed to connect to database: %v", err)
	}
	defer db.Close()

	// Create migrations table
	if err := createMigrationsTable(db); err != nil {
		log.Fatalf("Failed to create migrations table: %v", err)
	}

	if len(os.Args) < 2 {
		log.Fatal("Usage: migrate [up|down|create]")
	}

	command := os.Args[1]

	switch command {
	case "up":
		if err := migrateUp(db); err != nil {
			log.Fatalf("Migration up failed: %v", err)
		}
		log.Println("Migration up completed successfully")
	case "down":
		if err := migrateDown(db); err != nil {
			log.Fatalf("Migration down failed: %v", err)
		}
		log.Println("Migration down completed successfully")
	case "create":
		if len(os.Args) < 3 {
			log.Fatal("Usage: migrate create <name>")
		}
		if err := createMigration(os.Args[2]); err != nil {
			log.Fatalf("Failed to create migration: %v", err)
		}
	default:
		log.Fatalf("Unknown command: %s", command)
	}
}

func createMigrationsTable(db *sql.DB) error {
	query := `
		CREATE TABLE IF NOT EXISTS schema_migrations (
			version VARCHAR(255) PRIMARY KEY,
			applied_at TIMESTAMP NOT NULL DEFAULT NOW()
		)
	`
	_, err := db.Exec(query)
	return err
}

func migrateUp(db *sql.DB) error {
	migrations, err := getMigrationFiles()
	if err != nil {
		return err
	}

	for _, migration := range migrations {
		applied, err := isMigrationApplied(db, migration.version)
		if err != nil {
			return err
		}
		if applied {
			continue
		}

		content, err := os.ReadFile(migration.upPath)
		if err != nil {
			return fmt.Errorf("failed to read migration %s: %w", migration.version, err)
		}

		if _, err := db.Exec(string(content)); err != nil {
			return fmt.Errorf("failed to apply migration %s: %w", migration.version, err)
		}

		if err := markMigrationApplied(db, migration.version); err != nil {
			return err
		}

		log.Printf("Applied migration: %s", migration.version)
	}

	return nil
}

func migrateDown(db *sql.DB) error {
	lastVersion, err := getLastAppliedMigration(db)
	if err != nil {
		return err
	}
	if lastVersion == "" {
		log.Println("No migrations to rollback")
		return nil
	}

	migrations, err := getMigrationFiles()
	if err != nil {
		return err
	}

	for _, migration := range migrations {
		if migration.version == lastVersion {
			content, err := os.ReadFile(migration.downPath)
			if err != nil {
				return fmt.Errorf("failed to read migration %s: %w", migration.version, err)
			}

			if _, err := db.Exec(string(content)); err != nil {
				return fmt.Errorf("failed to rollback migration %s: %w", migration.version, err)
			}

			if err := removeMigrationRecord(db, migration.version); err != nil {
				return err
			}

			log.Printf("Rolled back migration: %s", migration.version)
			return nil
		}
	}

	return fmt.Errorf("migration %s not found", lastVersion)
}

type migration struct {
	version  string
	upPath   string
	downPath string
}

func getMigrationFiles() ([]migration, error) {
	files, err := filepath.Glob("migrations/*.up.sql")
	if err != nil {
		return nil, err
	}

	var migrations []migration
	for _, upPath := range files {
		version := strings.TrimSuffix(filepath.Base(upPath), ".up.sql")
		downPath := fmt.Sprintf("migrations/%s.down.sql", version)

		migrations = append(migrations, migration{
			version:  version,
			upPath:   upPath,
			downPath: downPath,
		})
	}

	sort.Slice(migrations, func(i, j int) bool {
		return migrations[i].version < migrations[j].version
	})

	return migrations, nil
}

func isMigrationApplied(db *sql.DB, version string) (bool, error) {
	var exists bool
	err := db.QueryRow("SELECT EXISTS(SELECT 1 FROM schema_migrations WHERE version = $1)", version).Scan(&exists)
	return exists, err
}

func markMigrationApplied(db *sql.DB, version string) error {
	_, err := db.Exec("INSERT INTO schema_migrations (version) VALUES ($1)", version)
	return err
}

func getLastAppliedMigration(db *sql.DB) (string, error) {
	var version string
	err := db.QueryRow("SELECT version FROM schema_migrations ORDER BY version DESC LIMIT 1").Scan(&version)
	if errors.Is(err, sql.ErrNoRows) {
		return "", nil
	}
	return version, err
}

func removeMigrationRecord(db *sql.DB, version string) error {
	_, err := db.Exec("DELETE FROM schema_migrations WHERE version = $1", version)
	return err
}

func createMigration(name string) error {
	// Get next version number
	files, err := filepath.Glob("migrations/*.up.sql")
	if err != nil {
		return err
	}

	nextVersion := fmt.Sprintf("%03d_%s", len(files)+1, name)

	upPath := fmt.Sprintf("migrations/%s.up.sql", nextVersion)
	downPath := fmt.Sprintf("migrations/%s.down.sql", nextVersion)

	if err := os.WriteFile(upPath, []byte("-- Add up migration here\n"), 0644); err != nil {
		return err
	}

	if err := os.WriteFile(downPath, []byte("-- Add down migration here\n"), 0644); err != nil {
		return err
	}

	log.Printf("Created migration files:\n  %s\n  %s", upPath, downPath)
	return nil
}
