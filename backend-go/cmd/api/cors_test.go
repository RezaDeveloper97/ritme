package main

import (
	"testing"

	"github.com/stretchr/testify/assert"

	apihttp "github.com/ritme/backend-go/internal/http"
)

// T-M9-26 (D-126): the browser preflights every API call once the frontend sends X-Timezone (and If-None-Match on
// the ETag'd routes). corsMiddleware echoes Access-Control-Request-Headers, so both pass, next to the usual ones.
func TestCORS_PreflightAllowsTimezoneAndIfNoneMatch(t *testing.T) {
	app := newApp(apihttp.NewRegistry(), testDeps())

	resp := do(t, app, preflight("https://web.ritme.app", "GET", "x-timezone, if-none-match"))
	assert.Equal(t, 204, resp.StatusCode)
	assert.Equal(t, "https://web.ritme.app", resp.Header.Get("Access-Control-Allow-Origin"))
	assert.Equal(t, "GET", resp.Header.Get("Access-Control-Allow-Methods"))
	assert.Equal(t, "x-timezone, if-none-match", resp.Header.Get("Access-Control-Allow-Headers"))

	resp = do(t, app, preflight("https://web.ritme.app", "POST",
		"accept-language,authorization,content-type,x-timezone"))
	assert.Equal(t, 204, resp.StatusCode)
	assert.Equal(t, "accept-language,authorization,content-type,x-timezone",
		resp.Header.Get("Access-Control-Allow-Headers"))
}
