package main

import (
	"database/sql"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/joho/godotenv"
	"github.com/selfhostly/internal/config"
	"github.com/selfhostly/internal/constants"
	_ "modernc.org/sqlite"
)

// runBackup writes a consistent copy of the live database. VACUUM INTO produces a standalone file
// even while the server is running in WAL mode, so no downtime is needed.
func runBackup(args []string) int {
	fs := flag.NewFlagSet("backup", flag.ExitOnError)
	dir := fs.String("dir", "", "directory to write the backup to (default: <data dir>/backups)")
	_ = fs.Parse(args)

	envFile := os.Getenv("ENV_FILE")
	if envFile == "" {
		envFile = ".env"
	}
	_ = godotenv.Load(envFile)
	cfg, err := config.Load()
	if err != nil {
		fmt.Fprintf(os.Stderr, "configuration: %v\n", err)
		return 1
	}
	if _, err := os.Stat(cfg.DatabasePath); err != nil {
		fmt.Fprintf(os.Stderr, "no database at %s: %v\n", cfg.DatabasePath, err)
		return 1
	}
	target := *dir
	if target == "" {
		target = filepath.Join(filepath.Dir(cfg.DatabasePath), "backups")
	}
	if err := os.MkdirAll(target, 0o700); err != nil {
		fmt.Fprintf(os.Stderr, "create %s: %v\n", target, err)
		return 1
	}
	dest := filepath.Join(target, "selfhostly-"+time.Now().UTC().Format("20060102T150405Z")+".db")

	conn, err := sql.Open("sqlite", "file:"+cfg.DatabasePath+"?mode=ro")
	if err != nil {
		fmt.Fprintf(os.Stderr, "open database: %v\n", err)
		return 1
	}
	defer conn.Close()
	if _, err := conn.Exec(`VACUUM INTO ?`, dest); err != nil {
		fmt.Fprintf(os.Stderr, "backup failed: %v\n", err)
		return 1
	}
	if err := os.Chmod(dest, constants.SecretFileMode); err != nil {
		fmt.Fprintf(os.Stderr, "warning: could not restrict permissions on %s: %v\n", dest, err)
	}
	fmt.Println(dest)
	return 0
}
