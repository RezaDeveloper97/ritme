package telemed_test

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/ritme/backend-go/internal/platform/httpx"
	"github.com/ritme/backend-go/internal/platform/validation/phpval"
	"github.com/ritme/backend-go/internal/telemed"
)

func input(kv ...any) phpval.Map {
	m := phpval.NewMap()
	for i := 0; i+1 < len(kv); i += 2 {
		m.Set(kv[i].(string), kv[i+1])
	}
	return m
}

func TestRatingAndSatisfaction(t *testing.T) {
	assert.Nil(t, telemed.Rating(0, 0))
	assert.InDelta(t, 4.9, telemed.Rating(97, 20), 1e-9, "4.85 rounds half up")
	assert.InDelta(t, 4.8, telemed.Rating(96, 20), 1e-9)
	assert.InDelta(t, 5.0, telemed.Rating(5, 1), 1e-9)
	assert.Nil(t, telemed.Satisfaction(0, 0))
	assert.EqualValues(t, 97, telemed.Satisfaction(97, 100))
	assert.EqualValues(t, 67, telemed.Satisfaction(2, 3), "66.67 → 67")
}

func TestInitial(t *testing.T) {
	assert.Equal(t, "س", telemed.Initial("  سارا احمدی"))
	assert.Equal(t, "L", telemed.Initial("Leila"))
	assert.Nil(t, telemed.Initial("   "))
}

func TestValidateFilter(t *testing.T) {
	f, err := telemed.ValidateFilter(input("mode", "video", "today", "1", "specialty", "gynecology", "insurance", "any",
		"q", " نیک ", "unknown", "x"), "en", mon)
	require.NoError(t, err)
	assert.Equal(t, telemed.Filter{Mode: "video", Specialty: "gynecology", Insurance: "any", Today: true, Q: "نیک"}, f)

	_, err = telemed.ValidateFilter(input("mode", "fax", "kind", "nurse"), "en", mon)
	var fe *httpx.FailError
	require.True(t, errors.As(err, &fe))
	assert.Equal(t, 422, fe.Status)
}

func TestValidateSlots(t *testing.T) {
	in, err := telemed.ValidateSlots(input(), "fa", mon)
	require.NoError(t, err)
	assert.Equal(t, "2026-10-05", in.From.String(), "defaults to today")
	assert.Equal(t, telemed.DefaultSlotDays, in.Days)

	in, err = telemed.ValidateSlots(input("from", "2026-09-01", "days", "3", "mode", "phone"), "fa", mon)
	require.NoError(t, err)
	assert.Equal(t, "2026-10-05", in.From.String(), "a past start is clamped to today")
	assert.Equal(t, 3, in.Days)
	assert.Equal(t, telemed.ModePhone, in.Mode)

	in, err = telemed.ValidateSlots(input("from", "2026-10-20"), "fa", mon)
	require.NoError(t, err)
	assert.Equal(t, "2026-10-20", in.From.String())

	_, err = telemed.ValidateSlots(input("days", "32"), "fa", mon)
	require.Error(t, err)
	_, err = telemed.ValidateSlots(input("from", "20-10-2026"), "fa", mon)
	require.Error(t, err)
}

func TestValidateReview(t *testing.T) {
	in, err := telemed.ValidateReview(input("rating", 5, "body", "  Clear and kind  "), "en", mon)
	require.NoError(t, err)
	assert.Equal(t, telemed.ReviewInput{Rating: 5, Body: "Clear and kind"}, in)
	for _, bad := range []phpval.Map{input(), input("rating", 0), input("rating", 6), input("rating", 4.5)} {
		_, err := telemed.ValidateReview(bad, "en", mon)
		assert.Error(t, err)
	}
}

func TestMinutes(t *testing.T) {
	assert.Equal(t, "09:05", telemed.Minutes(545))
	assert.Equal(t, "24:00", telemed.Minutes(1440))
}
