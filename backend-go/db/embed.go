// Package db ships the SQL assets of the Go API: goose migrations (migrations/) and the
// sqlc query sources (queries/<domain>/, compiled into internal/<domain>/store).
//
// The Go runtime code that applies the migrations lives in internal/platform/db.
package db

import "embed"

// Migrations holds db/migrations/*.sql (goose format), rooted at "migrations".
//
//go:embed migrations/*.sql
var Migrations embed.FS
