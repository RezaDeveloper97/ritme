package content

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

// D-16: an internal banner link is a same-origin app path.
func TestIsInternalPath(t *testing.T) {
	for _, ok := range []string{"/", "/articles/x", "/pregnancy?tab=week#top", "/a//b", "/a b"} {
		assert.True(t, isInternalPath(ok), ok)
	}
	for _, bad := range []string{
		"", "articles", "javascript:alert(1)", "https://evil.test/", "//evil.test", `/\evil.test`,
		" /x", "/\t/evil.test", "/x\n", "/x\x7f",
	} {
		assert.False(t, isInternalPath(bad), bad)
	}
}
