package main

import (
	"context"
	"crypto/rand"
	"database/sql"
	"encoding/hex"
	"errors"
	"fmt"
	"net"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"syscall"
	"time"

	"github.com/go-sql-driver/mysql"
	"github.com/redis/go-redis/v9"
)

// GoServer is cmd/api built and started by the harness against a private database
// (loaded from contract/fixtures/dump.sql) on a free port, so concurrent
// `make contract ROUTES=<group>` runs never share state.
type GoServer struct {
	Paths    Paths
	runID    string
	dbName   string
	admin    *sql.DB // server-level connection (CREATE/DROP DATABASE)
	conn     *sql.DB // connection to dbName (reset, OTP lookup)
	rdb      *redis.Client
	dump     []byte
	sessions map[string]Session
	base     string
	cmd      *exec.Cmd
	exited   chan struct{}
	workDir  string
	logPath  string
}

// Environment knobs (all optional):
//
//	CONTRACT_GO_DB_ADMIN_DSN  server DSN with CREATE/DROP DATABASE rights
//	                          (default TEST_DB_ADMIN_DSN, else root:root@127.0.0.1:13317 = docker-compose.test.yml)
//	CONTRACT_GO_REDIS_ADDR    Redis for the Go server (default TEST_REDIS_ADDR, else 127.0.0.1:16380)
const defaultAdminDSN = "root:root@tcp(127.0.0.1:13317)/"

// NewGoServer prepares (but does not start) the Go target.
func NewGoServer(p Paths) (*GoServer, error) {
	id := make([]byte, 4)
	if _, err := rand.Read(id); err != nil {
		return nil, err
	}
	runID := fmt.Sprintf("%d_%s", os.Getpid(), hex.EncodeToString(id))
	return &GoServer{Paths: p, runID: runID, dbName: "contract_" + runID, sessions: map[string]Session{}}, nil
}

// Name implements Target.
func (g *GoServer) Name() string { return "go" }

// BaseURL implements Target.
func (g *GoServer) BaseURL() string { return g.base }

func adminDSN() (*mysql.Config, bool, error) {
	dsn := firstString(os.Getenv("CONTRACT_GO_DB_ADMIN_DSN"), os.Getenv("TEST_DB_ADMIN_DSN"))
	isDefault := dsn == ""
	if isDefault {
		dsn = defaultAdminDSN
	}
	cfg, err := mysql.ParseDSN(dsn)
	if err != nil {
		return nil, false, fmt.Errorf("CONTRACT_GO_DB_ADMIN_DSN: %w", err)
	}
	cfg.DBName = ""
	cfg.MultiStatements = true
	cfg.ParseTime = true
	if cfg.Params == nil {
		cfg.Params = map[string]string{}
	}
	cfg.Params["charset"] = "utf8mb4"
	return cfg, isDefault, nil
}

// Start creates the database, loads the fixtures, mints persona sessions and starts
// the server.
func (g *GoServer) Start(ctx context.Context, dump []byte) error {
	g.dump = dump
	cfg, isDefault, err := adminDSN()
	if err != nil {
		return err
	}
	g.admin, err = sql.Open("mysql", cfg.FormatDSN())
	if err != nil {
		return err
	}
	if err := g.admin.PingContext(ctx); err != nil {
		if !isDefault {
			return fmt.Errorf("go db (%s): %w", cfg.Addr, err)
		}
		logf("go: test MariaDB not reachable, starting docker-compose.test.yml")
		cmd := exec.CommandContext(ctx, "docker", "compose", "-f", g.Paths.TestCompose, "up", "-d", "--wait") //nolint:gosec // G204: fixed args
		if out, err := cmd.CombinedOutput(); err != nil {
			return fmt.Errorf("docker compose test up: %w\n%s", err, clip(string(out)))
		}
		if err := g.admin.PingContext(ctx); err != nil {
			return fmt.Errorf("go db: %w", err)
		}
	}
	if _, err := g.admin.ExecContext(ctx, "CREATE DATABASE `"+g.dbName+"` CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci"); err != nil {
		return err
	}
	dbCfg := cfg.Clone()
	dbCfg.DBName = g.dbName
	g.conn, err = sql.Open("mysql", dbCfg.FormatDSN())
	if err != nil {
		return err
	}
	g.conn.SetMaxOpenConns(1)

	minter, err := NewMinter(filepath.Join(g.Paths.Keys, "oauth-private.key"), ContractClientID)
	if err != nil {
		return fmt.Errorf("contract keys (run make contract-record first): %w", err)
	}
	if err := g.mintSessions(minter); err != nil {
		return err
	}
	g.rdb = redis.NewClient(&redis.Options{
		Addr: firstString(os.Getenv("CONTRACT_GO_REDIS_ADDR"), os.Getenv("TEST_REDIS_ADDR"), "127.0.0.1:16380"),
	})
	if err := g.rdb.Ping(ctx).Err(); err != nil {
		return fmt.Errorf("go redis: %w", err)
	}
	if err := g.Reset(ctx); err != nil {
		return err
	}
	return g.startServer(ctx, cfg)
}

func (g *GoServer) mintSessions(m *Minter) error {
	now := time.Now()
	created, err := time.ParseInLocation(time.RFC3339, DefaultNow, m.tehran)
	if err != nil {
		return err
	}
	for _, p := range Personas {
		if p.Blocked {
			continue
		}
		tok, row, err := m.Mint(p.UserID, now, now.AddDate(1, 0, 0), created, false)
		if err != nil {
			return err
		}
		g.sessions[p.Key] = Session{Persona: p.Key, Token: tok, Row: row}
	}
	synth, err := m.MintSynthetic(now)
	if err != nil {
		return err
	}
	for _, s := range synth {
		g.sessions[s.Persona] = s
	}
	return nil
}

func (g *GoServer) startServer(ctx context.Context, cfg *mysql.Config) error {
	var err error
	g.workDir, err = os.MkdirTemp("", "ritme-contract-"+g.runID+"-")
	if err != nil {
		return err
	}
	bin := filepath.Join(g.workDir, "api")
	build := exec.CommandContext(ctx, "go", "build", "-o", bin, "./cmd/api") //nolint:gosec // G204: fixed args
	build.Dir = g.Paths.GoRoot
	if out, err := build.CombinedOutput(); err != nil {
		return fmt.Errorf("go build ./cmd/api: %w\n%s", err, clip(string(out)))
	}
	storage, err := g.storageDir()
	if err != nil {
		return err
	}
	port, err := freePort()
	if err != nil {
		return err
	}
	host, dbPort, err := net.SplitHostPort(cfg.Addr)
	if err != nil {
		return err
	}
	redisHost, redisPort, err := net.SplitHostPort(g.rdb.Options().Addr)
	if err != nil {
		return err
	}
	g.base = fmt.Sprintf("http://127.0.0.1:%d", port)
	env := map[string]string{
		"APP_ENV": "contract", "APP_DEBUG": "false", "APP_URL": CanonicalBaseURL, "APP_TIMEZONE": "Asia/Tehran",
		"TEST_CLOCK_ENABLED": "true", "HTTP_ADDR": fmt.Sprintf("127.0.0.1:%d", port),
		"DB_HOST": host, "DB_PORT": dbPort, "DB_DATABASE": g.dbName, "DB_USERNAME": cfg.User, "DB_PASSWORD": cfg.Passwd,
		"REDIS_HOST": redisHost, "REDIS_PORT": redisPort, "REDIS_PASSWORD": "", "REDIS_PREFIX": g.redisPrefix(),
		"STORAGE_PATH": storage, "SMS_PROVIDER": "log", "QUEUE_CONNECTION": "sync",
		"TELEGRAM_BOT_TOKEN": "", "TELEGRAM_CHAT_ID": "",
	}
	g.logPath = filepath.Join(g.workDir, "api.log")
	logFile, err := os.Create(g.logPath)
	if err != nil {
		return err
	}
	g.cmd = exec.Command(bin) //nolint:gosec,noctx // G204: our own binary; stopped by Stop
	g.cmd.Env = mergeEnv(os.Environ(), env)
	g.cmd.Stdout = logFile
	g.cmd.Stderr = logFile
	if err := g.cmd.Start(); err != nil {
		return err
	}
	g.exited = make(chan struct{})
	go func() { _ = g.cmd.Wait(); _ = logFile.Close(); close(g.exited) }()
	deadline := time.Now().Add(60 * time.Second)
	for time.Now().Before(deadline) {
		req, _ := http.NewRequestWithContext(ctx, http.MethodGet, g.base+"/up", nil)
		if resp, err := http.DefaultClient.Do(req); err == nil {
			_ = resp.Body.Close()
			return nil
		}
		select {
		case <-g.exited:
			return fmt.Errorf("go server exited during startup; log:\n%s", tailFile(g.logPath))
		case <-time.After(200 * time.Millisecond):
		}
	}
	return fmt.Errorf("go server did not come up on %s; log:\n%s", g.base, tailFile(g.logPath))
}

// storageDir assembles STORAGE_PATH: the contract key pair (0600) plus the repo's
// backend/storage/app (translations, public uploads) linked in.
func (g *GoServer) storageDir() (string, error) {
	dir := filepath.Join(g.workDir, "storage")
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return "", err
	}
	for _, f := range []string{"oauth-private.key", "oauth-public.key"} {
		raw, err := os.ReadFile(filepath.Join(g.Paths.Keys, f)) //nolint:gosec // G304: fixture keys
		if err != nil {
			return "", err
		}
		if err := os.WriteFile(filepath.Join(dir, f), raw, 0o600); err != nil { //nolint:gosec // G703: fixed file names in our temp dir
			return "", err
		}
	}
	app := filepath.Join(g.Paths.RepoRoot, "backend", "storage", "app")
	if fileExists(app) {
		if err := os.Symlink(app, filepath.Join(dir, "app")); err != nil {
			return "", err
		}
	}
	return dir, nil
}

// redisPrefix isolates this run's keys (rate limits, caches) from other runs.
func (g *GoServer) redisPrefix() string { return "ritme-go-contract-" + g.runID + ":" }

// Reset implements Target: fixture DB + session rows, and an empty Redis keyspace for
// this run (rate-limit counters must not leak between cases; Laravel's contract stack
// uses the per-request array cache).
func (g *GoServer) Reset(ctx context.Context) error {
	if _, err := g.conn.ExecContext(ctx, string(resetScript(g.dump, g.sessions))); err != nil {
		return err
	}
	return g.flushRedis(ctx)
}

func (g *GoServer) flushRedis(ctx context.Context) error {
	iter := g.rdb.Scan(ctx, 0, g.redisPrefix()+"*", 500).Iterator()
	var keys []string
	for iter.Next(ctx) {
		keys = append(keys, iter.Val())
	}
	if err := iter.Err(); err != nil {
		return err
	}
	if len(keys) == 0 {
		return nil
	}
	return g.rdb.Del(ctx, keys...).Err()
}

// LatestOTP implements Target.
func (g *GoServer) LatestOTP(ctx context.Context, mobile string) (string, error) {
	var code string
	err := g.conn.QueryRowContext(ctx,
		"SELECT code FROM otp_verifications WHERE mobile=? ORDER BY id DESC LIMIT 1", mobile).Scan(&code)
	if errors.Is(err, sql.ErrNoRows) {
		return "", errors.New("no OTP row")
	}
	return code, err
}

// Session implements Target.
func (g *GoServer) Session(persona string) (string, bool) {
	if persona == PersonaAnon {
		return "", true
	}
	s, ok := g.sessions[persona]
	return s.Token, ok
}

// Stop kills the server and drops the database. Safe to call on a partial Start.
func (g *GoServer) Stop() {
	if g.exited != nil {
		_ = g.cmd.Process.Signal(syscall.SIGTERM)
		select {
		case <-g.exited:
		case <-time.After(10 * time.Second):
			_ = g.cmd.Process.Kill()
		}
	}
	if g.conn != nil {
		_ = g.conn.Close()
	}
	if g.rdb != nil {
		_ = g.flushRedis(context.Background())
		_ = g.rdb.Close()
	}
	if g.admin != nil {
		_, _ = g.admin.ExecContext(context.Background(), "DROP DATABASE IF EXISTS `"+g.dbName+"`")
		_ = g.admin.Close()
	}
	if g.workDir != "" && os.Getenv("CONTRACT_KEEP_WORKDIR") == "" {
		_ = os.RemoveAll(g.workDir)
	}
}

// LogTail returns the end of the server log (printed on failures).
func (g *GoServer) LogTail() string { return tailFile(g.logPath) }

func freePort() (int, error) {
	l, err := net.Listen("tcp", "127.0.0.1:0") //nolint:noctx // short-lived probe
	if err != nil {
		return 0, err
	}
	defer func() { _ = l.Close() }()
	addr, ok := l.Addr().(*net.TCPAddr)
	if !ok {
		return 0, errors.New("unexpected listener address")
	}
	return addr.Port, nil
}

func mergeEnv(base []string, set map[string]string) []string {
	out := make([]string, 0, len(base)+len(set))
	for _, kv := range base {
		k, _, _ := strings.Cut(kv, "=")
		if _, ok := set[k]; !ok {
			out = append(out, kv)
		}
	}
	for _, k := range sortedKeys(set) {
		out = append(out, k+"="+set[k])
	}
	return out
}

func tailFile(p string) string {
	raw, err := os.ReadFile(p) //nolint:gosec // G304: our own log
	if err != nil {
		return ""
	}
	if len(raw) > 4000 {
		raw = raw[len(raw)-4000:]
	}
	return string(raw)
}
