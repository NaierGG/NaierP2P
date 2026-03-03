package main

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strconv"

	"github.com/golang-migrate/migrate/v4"
	_ "github.com/golang-migrate/migrate/v4/database/postgres"
	_ "github.com/golang-migrate/migrate/v4/source/file"
	"github.com/naier/backend/internal/config"
)

func main() {
	if len(os.Args) < 2 {
		fatalf("usage: migrate [up|down|status] [steps]")
	}

	cfg, err := config.LoadConfig()
	if err != nil {
		fatalf("load config: %v", err)
	}

	migrationsPath, err := findMigrationsDir()
	if err != nil {
		fatalf("locate migrations: %v", err)
	}

	m, err := migrate.New("file://"+filepath.ToSlash(migrationsPath), cfg.Database.PostgresDSN)
	if err != nil {
		fatalf("create migrate client: %v", err)
	}
	defer func() {
		sourceErr, dbErr := m.Close()
		if sourceErr != nil {
			fmt.Fprintf(os.Stderr, "close source: %v\n", sourceErr)
		}
		if dbErr != nil {
			fmt.Fprintf(os.Stderr, "close database: %v\n", dbErr)
		}
	}()

	command := os.Args[1]
	switch command {
	case "up":
		if err := m.Up(); err != nil && !errors.Is(err, migrate.ErrNoChange) {
			fatalf("migrate up: %v", err)
		}
		fmt.Println("migrations applied")
	case "down":
		steps := 1
		if len(os.Args) >= 3 {
			parsed, err := strconv.Atoi(os.Args[2])
			if err != nil || parsed <= 0 {
				fatalf("invalid down steps %q", os.Args[2])
			}
			steps = parsed
		}

		if err := m.Steps(-steps); err != nil && !errors.Is(err, migrate.ErrNoChange) {
			fatalf("migrate down: %v", err)
		}
		fmt.Printf("rolled back %d migration(s)\n", steps)
	case "status":
		version, dirty, err := m.Version()
		if errors.Is(err, migrate.ErrNilVersion) {
			fmt.Println("version: none")
			fmt.Println("dirty: false")
			return
		}
		if err != nil {
			fatalf("migration status: %v", err)
		}

		fmt.Printf("version: %d\n", version)
		fmt.Printf("dirty: %t\n", dirty)
	default:
		fatalf("unknown command %q", command)
	}
}

func findMigrationsDir() (string, error) {
	candidates := []string{
		filepath.Join(".", "migrations"),
		filepath.Join("..", "..", "migrations"),
	}

	if exePath, err := os.Executable(); err == nil {
		exeDir := filepath.Dir(exePath)
		candidates = append(candidates,
			filepath.Join(exeDir, "migrations"),
			filepath.Join(exeDir, "..", "..", "migrations"),
		)
	}

	for _, candidate := range candidates {
		absPath, err := filepath.Abs(candidate)
		if err != nil {
			continue
		}

		info, err := os.Stat(absPath)
		if err == nil && info.IsDir() {
			return absPath, nil
		}
	}

	return "", errors.New("migrations directory not found")
}

func fatalf(format string, args ...any) {
	fmt.Fprintf(os.Stderr, format+"\n", args...)
	os.Exit(1)
}
