package media_test

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/ritme/backend-go/internal/media"
)

// The scheduled media:sweep handler run twice leaves the same state: the abandoned upload goes once, the finished one
// stays (T-M9-11).
func TestJobs_SweepIsIdempotent(t *testing.T) {
	e := setup(t, media.NewSigner(media.DevKey()))
	itok := e.instructor(t, "09120000001", true)
	_, lid := e.lesson(t, itok, "video")
	_, lid2 := e.lesson(t, itok, "video")
	abandoned := e.create(t, itok, lid, 1000)
	require.Equal(t, 200, e.chunk(t, itok, abandoned, "0", video(400)).status)
	done := e.create(t, itok, lid2, 300)
	require.Equal(t, 200, e.chunk(t, itok, done, "0", video(300)).status)

	sweep := e.svc.SweepJob(func() time.Time { return t0.Add(media.UploadTTL + time.Minute) })
	require.NoError(t, sweep(context.Background(), []byte("{}")))
	first := e.files(t)
	_, status := e.lessonMedia(t, lid)
	require.NoError(t, sweep(context.Background(), []byte("{}")))
	assert.Equal(t, first, e.files(t))
	_, status2 := e.lessonMedia(t, lid)
	assert.Equal(t, status, status2)
	assert.Len(t, first, 1)
	assert.Equal(t, "none", status)
}
