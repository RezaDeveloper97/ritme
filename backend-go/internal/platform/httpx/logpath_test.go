package httpx

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestLogPath(t *testing.T) {
	for in, want := range map[string]string{
		"/api/v1/loss":          "/api/v1/loss/*",
		"/api/v1/loss/note":     "/api/v1/loss/*",
		"/api/v1/loss/followup": "/api/v1/loss/*",
		"/api/v1/lossy":         "/api/v1/lossy",
		"/api/v1/shared-reports/abcDEF_-0123456789abcdefghijklmnopqrstuvwxyz": "/api/v1/shared-reports/*",
		"/api/v1/home": "/api/v1/home",
	} {
		assert.Equal(t, want, LogPath(in), in)
	}
}
