package main

import (
	"errors"
	"fmt"
	"net"
	"strings"

	"github.com/go-sql-driver/mysql"
)

// CheckDSN refuses anything that is not a local scratch database: the host must be loopback and the database
// name must contain "test" or "local". The shared integration databases are refused even though "ritme_test"
// qualifies by name. Stage, beta and production never pass (remote host, other names).
func CheckDSN(dsn string) (*mysql.Config, error) {
	cfg, err := mysql.ParseDSN(dsn)
	if err != nil {
		return nil, fmt.Errorf("parse dsn: %w", err)
	}
	host, _, err := net.SplitHostPort(cfg.Addr)
	if err != nil {
		return nil, fmt.Errorf("dsn address %q: %w", cfg.Addr, err)
	}
	if host != "127.0.0.1" && host != "localhost" && host != "::1" {
		return nil, fmt.Errorf("refusing host %q: the seed tool only runs against a local database", host)
	}
	name := strings.ToLower(cfg.DBName)
	if !strings.Contains(name, "test") && !strings.Contains(name, "local") {
		return nil, fmt.Errorf("refusing database %q: its name must contain \"test\" or \"local\"", cfg.DBName)
	}
	if name == "ritme_test" || strings.HasPrefix(name, "gt_") {
		return nil, errors.New("refusing the shared integration database " + cfg.DBName + ": use your own scratch database (make loadtest-db)")
	}
	return cfg, nil
}
