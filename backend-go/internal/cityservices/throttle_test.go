package cityservices

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

// CB-DIR-13: the public directory reads' throttle answers in the request language, not the framework's English line.
func TestTooManyReads_Localized(t *testing.T) {
	headers := map[string]string{"Retry-After": "30"}
	fa := TooManyReads("fa", 30, headers)
	en := TooManyReads("en", 30, headers)
	assert.Equal(t, 429, fa.Status)
	assert.NotEqual(t, fa.Msg, en.Msg)
	assert.Contains(t, fa.Msg, "صبر")
	assert.Equal(t, "Too many requests; wait a few seconds and try again.", en.Msg)
	assert.Equal(t, "30", fa.Headers["Retry-After"])
	assert.Equal(t, en.Msg, TooManyReads("xx", 1, nil).Msg, "an unknown language falls back to English")
}
