package content

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestPhpTruthy(t *testing.T) {
	assert.False(t, phpTruthy(""))
	assert.False(t, phpTruthy("0"))
	assert.True(t, phpTruthy("00"))
	assert.True(t, phpTruthy("a"))
}
