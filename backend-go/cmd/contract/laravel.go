package main

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/ritme/backend-go/cmd/contract/internal/ojson"
)

// Laravel is the T-M2-01 contract stack (docker-compose.contract.yml at the repo root).
// It is shared: the recorder never tears it down and serialises itself with a lock.
type Laravel struct {
	Paths    Paths
	base     string
	dump     []byte
	sessions map[string]Session
}

// NewLaravel points at CONTRACT_BASE_URL or http://127.0.0.1:${CONTRACT_PORT:-8090}.
func NewLaravel(p Paths) *Laravel {
	base := os.Getenv("CONTRACT_BASE_URL")
	if base == "" {
		base = "http://127.0.0.1:" + firstString(os.Getenv("CONTRACT_PORT"), "8090")
	}
	return &Laravel{Paths: p, base: strings.TrimRight(base, "/"), sessions: map[string]Session{}}
}

// Name implements Target.
func (l *Laravel) Name() string { return "laravel" }

// BaseURL implements Target.
func (l *Laravel) BaseURL() string { return l.base }

func (l *Laravel) compose(ctx context.Context, stdin io.Reader, args ...string) ([]byte, error) {
	full := append([]string{"compose", "-f", l.Paths.ContractCompose}, args...)
	cmd := exec.CommandContext(ctx, "docker", full...) //nolint:gosec // G204: fixed binary, harness args
	cmd.Stdin = stdin
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		return nil, fmt.Errorf("docker %s: %w\n%s", strings.Join(args[:min(len(args), 3)], " "), err, clip(stderr.String()))
	}
	return stdout.Bytes(), nil
}

// sql runs statements in the contract DB and returns tab-separated rows.
func (l *Laravel) sql(ctx context.Context, stdin io.Reader, query string) ([]byte, error) {
	args := []string{"exec", "-T", "contract-mariadb", "mariadb", "-uritme", "-pcontract", "ritme_contract"}
	if query != "" {
		args = append(args, "-N", "-B", "-e", query)
	}
	return l.compose(ctx, stdin, args...)
}

func (l *Laravel) up(ctx context.Context) bool {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, l.base+"/up", nil)
	if err != nil {
		return false
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return false
	}
	_ = resp.Body.Close()
	return resp.StatusCode == http.StatusOK
}

// EnsureUp reuses a running stack or starts it (never rebuilds or stops it).
func (l *Laravel) EnsureUp(ctx context.Context) error {
	if l.up(ctx) {
		return nil
	}
	logf("laravel: %s/up not answering, starting docker-compose.contract.yml (CONTRACT_PORT=%s)",
		l.base, firstString(os.Getenv("CONTRACT_PORT"), "8090"))
	if _, err := l.compose(ctx, nil, "up", "-d", "--wait"); err != nil {
		return err
	}
	for range 60 {
		if l.up(ctx) {
			return nil
		}
		time.Sleep(2 * time.Second)
	}
	return fmt.Errorf("laravel: %s/up still not answering (port taken? set CONTRACT_PORT)", l.base)
}

// Reseed runs contract-reset (migrate:fresh + fixtures) and pins random/real-clock
// values, then exports the fixture dump.
func (l *Laravel) Reseed(ctx context.Context) error {
	logf("laravel: contract-reset (migrate:fresh + ContractFixtureSeeder)")
	if _, err := l.compose(ctx, nil, "run", "--rm", "contract-reset"); err != nil {
		return err
	}
	if _, err := l.sql(ctx, strings.NewReader(normaliseSQL), ""); err != nil {
		return fmt.Errorf("normalise: %w", err)
	}
	raw, err := l.compose(ctx, nil, "exec", "-T", "contract-mariadb", "mariadb-dump", "-uritme", "-pcontract",
		"--skip-dump-date", "--skip-comments", "--single-transaction", "--routines", "--triggers", "ritme_contract")
	if err != nil {
		return fmt.Errorf("dump: %w", err)
	}
	l.dump = cleanDump(raw)
	return nil
}

var sandboxLine = regexp.MustCompile(`(?m)^/\*M!999999\\- enable the sandbox mode \*/ ?\n`)

// cleanDump drops the mariadb-dump sandbox marker (not SQL a driver can send).
func cleanDump(raw []byte) []byte { return sandboxLine.ReplaceAll(raw, nil) }

// LoadDump uses an existing dump.sql instead of reseeding.
func (l *Laravel) LoadDump(path string) error {
	raw, err := os.ReadFile(path) //nolint:gosec // G304: fixture dump
	if err != nil {
		return err
	}
	l.dump = raw
	return nil
}

// Dump returns the fixture dump taken by Reseed.
func (l *Laravel) Dump() []byte { return l.dump }

const containerStorage = "/var/www/html/storage"

// SyncKeys makes contract/fixtures/keys the key pair Laravel signs with: an existing
// committed pair is installed into the container (so goldens and Go-side tokens stay
// valid across stack re-creations); otherwise the container's pair is copied out.
func (l *Laravel) SyncKeys(ctx context.Context) error {
	priv := filepath.Join(l.Paths.Keys, "oauth-private.key")
	pub := filepath.Join(l.Paths.Keys, "oauth-public.key")
	if fileExists(priv) && fileExists(pub) {
		for _, f := range []string{"oauth-private.key", "oauth-public.key"} {
			if _, err := l.compose(ctx, nil, "cp", filepath.Join(l.Paths.Keys, f), "contract-laravel:"+containerStorage+"/"+f); err != nil {
				return err
			}
		}
		_, err := l.compose(ctx, nil, "exec", "-T", "-u", "root", "contract-laravel", "sh", "-c",
			"cd "+containerStorage+" && chown www-data:www-data oauth-*.key && chmod 600 oauth-private.key && chmod 660 oauth-public.key")
		return err
	}
	if err := os.MkdirAll(l.Paths.Keys, 0o750); err != nil {
		return err
	}
	for _, f := range []string{"oauth-private.key", "oauth-public.key"} {
		if _, err := l.compose(ctx, nil, "cp", "contract-laravel:"+containerStorage+"/"+f, filepath.Join(l.Paths.Keys, f)); err != nil {
			return err
		}
		if err := os.Chmod(filepath.Join(l.Paths.Keys, f), 0o600); err != nil {
			return err
		}
	}
	return nil
}

// Login logs every persona in via send-otp → OTP from the DB → verify-otp, adds the
// synthetic 401 personas, and snapshots the token rows so Reset can restore them.
func (l *Laravel) Login(ctx context.Context, minter *Minter) error {
	if err := l.restore(ctx, nil); err != nil {
		return err
	}
	client := &http.Client{Timeout: 30 * time.Second}
	for _, p := range Personas {
		if p.Blocked {
			continue
		}
		tok, err := l.loginOne(ctx, client, p.Mobile)
		if err != nil {
			return fmt.Errorf("login %s: %w", p.Key, err)
		}
		l.sessions[p.Key] = Session{Persona: p.Key, Token: tok}
	}
	rows, err := l.sql(ctx, nil,
		"SELECT id,user_id,client_id,name,scopes,revoked,created_at,expires_at FROM oauth_access_tokens ORDER BY user_id")
	if err != nil {
		return err
	}
	byUser := map[int]*TokenRow{}
	for _, line := range strings.Split(strings.TrimSpace(string(rows)), "\n") {
		f := strings.Split(line, "\t")
		if len(f) != 8 {
			return fmt.Errorf("unexpected token row %q", line)
		}
		uid, _ := strconv.Atoi(f[1])
		byUser[uid] = &TokenRow{ID: f[0], UserID: uid, ClientID: f[2], Name: f[3], Scopes: f[4],
			Revoked: f[5] == "1", CreatedAt: f[6], ExpiresAt: f[7]}
	}
	for _, p := range Personas {
		if s, ok := l.sessions[p.Key]; ok {
			s.Row = byUser[p.UserID]
			if s.Row == nil {
				return fmt.Errorf("no token row for %s", p.Key)
			}
			l.sessions[p.Key] = s
		}
	}
	synth, err := minter.MintSynthetic(time.Now())
	if err != nil {
		return err
	}
	for _, s := range synth {
		l.sessions[s.Persona] = s
	}
	return nil
}

func (l *Laravel) loginOne(ctx context.Context, client *http.Client, mobile string) (string, error) {
	post := func(path, body string) (*ojson.Value, int, error) {
		req, err := http.NewRequestWithContext(ctx, http.MethodPost, l.base+"/api/v1"+path, strings.NewReader(body))
		if err != nil {
			return nil, 0, err
		}
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Accept", "application/json")
		req.Header.Set("X-Test-Now", DefaultNow)
		resp, err := client.Do(req)
		if err != nil {
			return nil, 0, err
		}
		defer func() { _ = resp.Body.Close() }()
		raw, err := io.ReadAll(resp.Body)
		if err != nil {
			return nil, 0, err
		}
		v, err := ojson.Parse(raw)
		if err != nil {
			return nil, resp.StatusCode, fmt.Errorf("%s: %d %s", path, resp.StatusCode, clip(string(raw)))
		}
		return v, resp.StatusCode, nil
	}
	if _, st, err := post("/auth/send-otp", `{"mobile":"`+mobile+`"}`); err != nil || st != http.StatusOK {
		return "", fmt.Errorf("send-otp: status %d: %w", st, err)
	}
	code, err := l.LatestOTP(ctx, mobile)
	if err != nil {
		return "", err
	}
	v, st, err := post("/auth/verify-otp", `{"mobile":"`+mobile+`","code":"`+code+`"}`)
	if err != nil || st != http.StatusOK {
		return "", fmt.Errorf("verify-otp: status %d: %w", st, err)
	}
	return lookup(v, "data.access_token")
}

var mobileRe = regexp.MustCompile(`^09\d{9}$`)

// LatestOTP implements Target.
func (l *Laravel) LatestOTP(ctx context.Context, mobile string) (string, error) {
	if !mobileRe.MatchString(mobile) {
		return "", fmt.Errorf("bad mobile %q", mobile)
	}
	out, err := l.sql(ctx, nil, "SELECT code FROM otp_verifications WHERE mobile='"+mobile+"' ORDER BY id DESC LIMIT 1")
	if err != nil {
		return "", err
	}
	code := strings.TrimSpace(string(out))
	if code == "" {
		return "", errors.New("no OTP row")
	}
	return code, nil
}

// Session implements Target.
func (l *Laravel) Session(persona string) (string, bool) {
	if persona == PersonaAnon {
		return "", true
	}
	s, ok := l.sessions[persona]
	return s.Token, ok
}

// Sessions returns every session (for the Go side's persona list).
func (l *Laravel) Sessions() map[string]Session { return l.sessions }

// Reset implements Target: fixture dump + session token rows.
func (l *Laravel) Reset(ctx context.Context) error { return l.restore(ctx, l.sessions) }

func (l *Laravel) restore(ctx context.Context, sessions map[string]Session) error {
	if len(l.dump) == 0 {
		return errors.New("no fixture dump (Reseed first)")
	}
	_, err := l.sql(ctx, bytes.NewReader(resetScript(l.dump, sessions)), "")
	return err
}

// resetScript is the dump followed by the session token rows (sorted by persona).
func resetScript(dump []byte, sessions map[string]Session) []byte {
	var b bytes.Buffer
	b.Write(dump)
	b.WriteString("\n")
	for _, p := range sortedKeys(sessions) {
		if r := sessions[p].Row; r != nil {
			b.WriteString(r.InsertSQL() + "\n")
		}
	}
	return b.Bytes()
}

func fileExists(p string) bool {
	_, err := os.Stat(p)
	return err == nil
}
