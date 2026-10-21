package httpx_test

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/ritme/backend-go/internal/platform/httpx"
)

func TestIsLongRunning(t *testing.T) {
	cases := []struct {
		method, path, ct string
		want             bool
	}{
		{"POST", "/api/v1/assistant/conversations/12/messages", "application/json", true},
		{"POST", "/api/v1/assistant/conversations/12/summary", "application/json", true},
		{"POST", "/API/v1//assistant/conversations/12/messages/", "application/json", true},
		{"POST", "/api/v1/logs/voice", "application/json", true},
		{"POST", "/api/v1/health-record/documents/5/extract", "application/json", true},
		{"POST", "/api/v1/labs", "multipart/form-data; boundary=x", true},
		{"PUT", "/api/admin/v1/banners/3", "Multipart/Form-Data; boundary=x", true},
		{"PATCH", "/api/instructor/v1/media/9", "application/offset+octet-stream", true},
		{"GET", "/api/v1/profile/export", "", true},
		{"GET", "/api/admin/v1/users/directory/export", "", true},

		{"GET", "/api/v1/cycle/month", "", false},
		{"GET", "/api/v1/assistant/conversations/12", "", false},
		{"POST", "/api/v1/logs/voice/commit", "application/json", false},
		{"GET", "/api/v1/labs", "multipart/form-data", false},
		{"POST", "/api/v1/profile", "application/json", false},
	}
	for _, tc := range cases {
		assert.Equal(t, tc.want, httpx.IsLongRunning(tc.method, tc.path, tc.ct), "%s %s", tc.method, tc.path)
	}
}

func TestMediaTransfer(t *testing.T) {
	for uri, want := range map[string]bool{
		"/api/v1/media/7/stream?u=1&expires=2&signature=x": true,
		"/api/instructor/v1/media/7":                       true,
		"/api/instructor/v1/media":                         true,
		"/API/V1/Media/7/stream":                           true,
		"/api/v1/mediakit":                                 false,
		"/api/v1/learning/lessons/1/playback":              false,
		"/api/v1/cycle/month?view=full":                    false,
	} {
		assert.Equal(t, want, httpx.MediaTransfer(uri), uri)
	}
}
