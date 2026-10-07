package media

import (
	"context"
	"errors"
)

// ErrInfected is what a Scanner returns for a file it rejects.
var ErrInfected = errors.New("media: file rejected by the scanner")

// Scanner is the virus-scan hook run on a complete upload before it becomes ready. A real implementation (ClamAV
// clamd INSTREAM, or an external service behind an adapter) returns ErrInfected for a bad file; any other error
// fails the upload too (fail closed) and is logged without the path. Today only NoopScanner exists (D-81).
type Scanner interface {
	Scan(ctx context.Context, absPath, kind, mime string) error
}

// NoopScanner accepts every file.
type NoopScanner struct{}

// Scan implements Scanner.
func (NoopScanner) Scan(context.Context, string, string, string) error { return nil }
