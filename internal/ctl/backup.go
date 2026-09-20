package ctl

import (
	"archive/tar"
	"compress/gzip"
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"
)

// dataFiles are the small files next to the database that a rebuilt server needs
var dataFiles = []string{"secrets.key", "node-id", "node-api-key", "registration-token"}

func (a *App) backupCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "backup [destination-dir]",
		Short: "Back up the database, keys and settings into one private archive",
		Long: "Takes an online database snapshot (no downtime), adds the secrets key, node identity and settings file,\n" +
			"and writes one archive with mode 0600. It contains secrets: keep it somewhere private, off this machine.",
		Args: cobra.MaximumNArgs(1),
		RunE: func(c *cobra.Command, args []string) error {
			dest := "./backups"
			if len(args) == 1 {
				dest = args[0]
			}
			_, err := a.backup(c.Context(), dest)
			return err
		},
	}
}

func (a *App) dataDir(env EnvFile) string {
	return a.dirPath(firstNonEmpty(os.Getenv("DATA_DIR"), env.Get("DATA_DIR"), "./data"))
}

func (a *App) now() string {
	t := timeNow(a)
	return t.UTC().Format("20060102T150405Z")
}

func (a *App) backup(ctx context.Context, dest string) (string, error) {
	env := EnvFile{a.envPath(a.EnvFile)}
	data := a.dataDir(env)
	target, err := a.findBackend(ctx, true)
	if err != nil {
		return "", err
	}
	if err := os.MkdirAll(dest, 0o700); err != nil {
		return "", err
	}

	a.say("1/3 taking an online database snapshot inside %s", target.name)
	out, errOut, err := a.Run.Output(ctx, "docker", "exec", target.name, target.bin, "backup", "--dir", "/app/data/backups")
	if err != nil {
		return "", fmt.Errorf("the snapshot failed: %s", strings.TrimSpace(errOut+out))
	}
	lines := strings.Split(strings.TrimSpace(out), "\n")
	snapshot := strings.TrimSpace(lines[len(lines)-1])
	a.say("    %s", snapshot)

	a.say("2/3 collecting files")
	// the snapshot is the consistent copy; the live db and its WAL files are deliberately skipped
	files := map[string]string{"data/selfhostly.db": filepath.Join(data, "backups", filepath.Base(snapshot))}
	for _, f := range dataFiles {
		if _, err := os.Stat(filepath.Join(data, f)); err == nil {
			files["data/"+f] = filepath.Join(data, f)
		}
	}
	if env.Exists() {
		files["env"] = env.Path
	}

	a.say("3/3 writing archive")
	outPath := filepath.Join(dest, "selfhostly-backup-"+a.now()+".tar.gz")
	if err := writeArchive(outPath, files); err != nil {
		return "", err
	}
	a.say("done: %s", outPath)
	a.say("note: app compose files and app data live under APPS_DIR and are not included; back those up separately")
	return outPath, nil
}

func writeArchive(path string, files map[string]string) (err error) {
	f, err := os.OpenFile(path, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0o600)
	if err != nil {
		return err
	}
	defer func() {
		if cerr := f.Close(); err == nil {
			err = cerr
		}
	}()
	gz := gzip.NewWriter(f)
	tw := tar.NewWriter(gz)
	for name, src := range files {
		in, err := os.Open(src)
		if err != nil {
			return err
		}
		st, err := in.Stat()
		if err != nil {
			_ = in.Close()
			return err
		}
		hdr := &tar.Header{Name: name, Mode: 0o600, Size: st.Size(), ModTime: st.ModTime()}
		if err := tw.WriteHeader(hdr); err != nil {
			_ = in.Close()
			return err
		}
		_, err = io.Copy(tw, in)
		_ = in.Close()
		if err != nil {
			return err
		}
	}
	if err := tw.Close(); err != nil {
		return err
	}
	return gz.Close()
}
