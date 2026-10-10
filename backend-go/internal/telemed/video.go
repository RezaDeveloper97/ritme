package telemed

import (
	"context"
	"strings"

	"github.com/ritme/backend-go/internal/platform/config"
	"github.com/ritme/backend-go/internal/telemed/store"
)

// VideoProvider creates (or names) the video room of a booking (bloom B-N7-04). Real providers plug in here behind
// TELEMED_VIDEO_PROVIDER; their secrets come from env only.
type VideoProvider interface {
	Name() string
	Room(ctx context.Context, bk store.TelemedBooking) (string, error)
}

// FakeVideo returns a stable fake room URL per booking (local / stage only; refused in production by config).
type FakeVideo struct{}

// FakeVideoBase is the fake room origin (never a real service).
const FakeVideoBase = "https://video.fake.ritme.invalid/room/"

// Name implements VideoProvider.
func (FakeVideo) Name() string { return config.VideoProviderFake }

// Room implements VideoProvider: the room is named after the unguessable booking reference.
func (FakeVideo) Room(_ context.Context, bk store.TelemedBooking) (string, error) {
	return FakeVideoBase + strings.ToLower(bk.Reference), nil
}

// NewVideo is the provider of TELEMED_VIDEO_PROVIDER (nil for none: the link answers 503 while active).
func NewVideo(provider string) VideoProvider {
	if provider == config.VideoProviderFake {
		return FakeVideo{}
	}
	return nil
}
