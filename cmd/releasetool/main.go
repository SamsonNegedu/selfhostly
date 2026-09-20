// Command releasetool creates the signing key, builds release.json and signs it. CI runs it on every release;
// the private key only ever lives in the RELEASE_SIGNING_KEY secret.
package main

import (
	"crypto/ed25519"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/selfhostly/internal/update"
)

const (
	privateKeyFile = "private.key"
	publicKeyFile  = "public.key"
	keyFileMode    = 0o600
	usage          = `usage:
  releasetool keygen --out DIR
  releasetool manifest --version V --backend REF --gateway REF --frontend REF --compose-file FILE
                       [--notes TEXT] [--min-from V] [--setting KEY:kind:secret:description ...] --out FILE
  releasetool sign --key FILE|- --in FILE --out FILE
  releasetool pubkey --key FILE|-
  releasetool verify --pub KEY --image-prefix PREFIX --in FILE --sig FILE`
)

func main() {
	if len(os.Args) < 2 {
		fmt.Fprintln(os.Stderr, usage)
		os.Exit(2)
	}
	var err error
	switch os.Args[1] {
	case "keygen":
		err = keygen(os.Args[2:])
	case "manifest":
		err = manifest(os.Args[2:])
	case "sign":
		err = sign(os.Args[2:])
	case "pubkey":
		err = pubkey(os.Args[2:])
	case "verify":
		err = verify(os.Args[2:])
	default:
		fmt.Fprintln(os.Stderr, usage)
		os.Exit(2)
	}
	if err != nil {
		fmt.Fprintln(os.Stderr, "releasetool:", err)
		os.Exit(1)
	}
}

func keygen(args []string) error {
	fs := flag.NewFlagSet("keygen", flag.ExitOnError)
	out := fs.String("out", "", "directory to write private.key and public.key")
	_ = fs.Parse(args)
	if *out == "" {
		return fmt.Errorf("--out is required")
	}
	pub, priv, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		return err
	}
	if err := os.MkdirAll(*out, 0o700); err != nil {
		return err
	}
	files := map[string][]byte{
		privateKeyFile: []byte(base64.StdEncoding.EncodeToString(priv.Seed()) + "\n"),
		publicKeyFile:  []byte(base64.StdEncoding.EncodeToString(pub) + "\n"),
	}
	for name, content := range files {
		if err := os.WriteFile(filepath.Join(*out, name), content, keyFileMode); err != nil {
			return err
		}
	}
	fmt.Printf("wrote %s and %s\n", filepath.Join(*out, privateKeyFile), filepath.Join(*out, publicKeyFile))
	return nil
}

type settingFlags []update.Setting

func (s *settingFlags) String() string { return "" }

// Set reads KEY:kind:secret:description
func (s *settingFlags) Set(v string) error {
	parts := strings.SplitN(v, ":", 4)
	if len(parts) < 3 {
		return fmt.Errorf("--setting wants KEY:kind:secret[:description], got %q", v)
	}
	desc := ""
	if len(parts) == 4 {
		desc = parts[3]
	}
	*s = append(*s, update.Setting{Key: parts[0], Kind: parts[1], Secret: parts[2] == "true", Description: desc})
	return nil
}

func manifest(args []string) error {
	fs := flag.NewFlagSet("manifest", flag.ExitOnError)
	version := fs.String("version", "", "release version, like 1.4.0")
	backend := fs.String("backend", "", "backend image reference with digest")
	gateway := fs.String("gateway", "", "gateway image reference with digest")
	frontend := fs.String("frontend", "", "frontend image reference with digest")
	composeFile := fs.String("compose-file", "", "the release's docker-compose.prod.yml")
	notes := fs.String("notes", "", "release notes")
	minFrom := fs.String("min-from", "", "oldest version that can update to this one")
	inputs := fs.String("inputs", "", "JSON file with min_from_version and settings, kept in the repository")
	out := fs.String("out", "", "file to write")
	prefix := fs.String("image-prefix", update.DefaultImagePrefix, "trusted image repository prefix to validate against")
	var settings settingFlags
	fs.Var(&settings, "setting", "KEY:kind:secret:description (repeatable)")
	_ = fs.Parse(args)
	if *out == "" || *composeFile == "" {
		return fmt.Errorf("--out and --compose-file are required")
	}
	compose, err := os.ReadFile(*composeFile)
	if err != nil {
		return err
	}
	sum := sha256.Sum256(compose)
	m := update.Manifest{
		Schema:         update.SchemaVersion,
		Version:        strings.TrimPrefix(*version, "v"),
		PublishedAt:    time.Now().UTC().Format(time.RFC3339),
		Notes:          *notes,
		MinFromVersion: strings.TrimPrefix(*minFrom, "v"),
		Images:         update.Images{Backend: *backend, Gateway: *gateway, Frontend: *frontend},
		Compose:        update.ComposeInfo{SHA256: hex.EncodeToString(sum[:])},
		Settings:       settings,
	}
	if *inputs != "" {
		if err := applyInputs(&m, *inputs); err != nil {
			return err
		}
	}
	if m.Settings == nil {
		m.Settings = []update.Setting{}
	}
	if err := m.Validate(*prefix); err != nil {
		return err
	}
	b, err := json.MarshalIndent(m, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(*out, append(b, '\n'), 0o644)
}

func readKey(path string) (ed25519.PrivateKey, error) {
	var raw []byte
	var err error
	if path == "-" {
		raw, err = io.ReadAll(os.Stdin)
	} else {
		raw, err = os.ReadFile(path)
	}
	if err != nil {
		return nil, err
	}
	return update.ParsePrivateKey(string(raw))
}

func sign(args []string) error {
	fs := flag.NewFlagSet("sign", flag.ExitOnError)
	key := fs.String("key", "", "private key file, or - for stdin")
	in := fs.String("in", "", "file to sign")
	out := fs.String("out", "", "signature file to write")
	_ = fs.Parse(args)
	if *key == "" || *in == "" || *out == "" {
		return fmt.Errorf("--key, --in and --out are required")
	}
	priv, err := readKey(*key)
	if err != nil {
		return err
	}
	data, err := os.ReadFile(*in)
	if err != nil {
		return err
	}
	return os.WriteFile(*out, []byte(update.Sign(priv, data)+"\n"), 0o644)
}

func verify(args []string) error {
	fs := flag.NewFlagSet("verify", flag.ExitOnError)
	pubFlag := fs.String("pub", "", "base64 public key")
	prefix := fs.String("image-prefix", update.DefaultImagePrefix, "trusted image repository prefix")
	in := fs.String("in", "", "release.json")
	sig := fs.String("sig", "", "release.json.sig")
	_ = fs.Parse(args)
	pub, err := update.ParsePublicKey(*pubFlag)
	if err != nil {
		return err
	}
	data, err := os.ReadFile(*in)
	if err != nil {
		return err
	}
	sigData, err := os.ReadFile(*sig)
	if err != nil {
		return err
	}
	m, err := update.ParseVerified(pub, data, string(sigData), *prefix)
	if err != nil {
		return err
	}
	fmt.Printf("ok: %s\n", m.Version)
	return nil
}

// releaseInputs is the part of a manifest people maintain by hand between releases.
type releaseInputs struct {
	MinFromVersion string           `json:"min_from_version"`
	Settings       []update.Setting `json:"settings"`
}

// applyInputs adds what the repository keeps for every release: the oldest version that can update to it and the
// settings it introduces. Flags win over the file.
func applyInputs(m *update.Manifest, path string) error {
	b, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	var in releaseInputs
	if err := json.Unmarshal(b, &in); err != nil {
		return fmt.Errorf("%s: %w", path, err)
	}
	if m.MinFromVersion == "" {
		m.MinFromVersion = strings.TrimPrefix(in.MinFromVersion, "v")
	}
	m.Settings = append(in.Settings, m.Settings...)
	return nil
}

func pubkey(args []string) error {
	fs := flag.NewFlagSet("pubkey", flag.ExitOnError)
	key := fs.String("key", "", "private key file, or - for stdin")
	_ = fs.Parse(args)
	priv, err := readKey(*key)
	if err != nil {
		return err
	}
	fmt.Println(update.EncodeKey(priv.Public().(ed25519.PublicKey)))
	return nil
}
