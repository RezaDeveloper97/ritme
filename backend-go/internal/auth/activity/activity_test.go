package activity_test

import (
	"context"
	"database/sql"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	"github.com/ritme/backend-go/internal/auth/activity"
	"github.com/ritme/backend-go/internal/metrics/store"
	"github.com/ritme/backend-go/internal/platform/clock"
)

type fakeStore struct {
	mu      sync.Mutex
	session []store.MarkUserSessionDayParams
	logged  []store.MarkUserLoggedDayParams
	zoneQ   int
	block   chan struct{}
}

func (f *fakeStore) MarkUserSessionDay(_ context.Context, a store.MarkUserSessionDayParams) error {
	if f.block != nil {
		<-f.block
	}
	f.mu.Lock()
	defer f.mu.Unlock()
	f.session = append(f.session, a)
	return nil
}

func (f *fakeStore) MarkUserLoggedDay(_ context.Context, a store.MarkUserLoggedDayParams) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.logged = append(f.logged, a)
	return nil
}

func (f *fakeStore) GetActivityUserTimezone(context.Context, uint64) (sql.NullString, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.zoneQ++
	return sql.NullString{}, sql.ErrNoRows
}

func at(s string) context.Context {
	t, _ := time.Parse(time.RFC3339, s)
	return clock.WithClock(context.Background(), clock.At(t))
}

func TestMarker_OneWritePerDayAndZoneReadOffRequest(t *testing.T) {
	st := &fakeStore{}
	m := activity.New(st, nil, nil)
	defer m.Close()
	for i := 0; i < 5; i++ {
		m.Mark(at("2026-10-20T09:00:00+03:30"), 7, activity.Session, nil)
		m.Flush()
	}
	m.Mark(at("2026-10-20T09:05:00+03:30"), 7, activity.Logged, nil)
	m.Mark(at("2026-10-21T09:00:00+03:30"), 7, activity.Session, nil)
	m.Flush()
	assert.Len(t, st.session, 2)
	assert.Equal(t, "2026-10-20", st.session[0].Day.String())
	assert.Equal(t, "2026-10-21", st.session[1].Day.String())
	assert.Len(t, st.logged, 1)
	assert.Equal(t, 2, st.zoneQ, "stored zone read by the writer once per 15 min (the next day is a re-read), never per mark")
}

func TestMarker_FullBufferDropsNeverBlocks(t *testing.T) {
	st := &fakeStore{block: make(chan struct{})}
	m := activity.New(st, nil, nil)
	done := make(chan struct{})
	go func() {
		for i := 0; i < activity.Buffer+100; i++ {
			m.Mark(at("2026-10-20T09:00:00+03:30"), uint64(i+1), activity.Session, time.UTC)
		}
		close(done)
	}()
	select {
	case <-done:
	case <-time.After(5 * time.Second):
		t.Fatal("Mark blocked on a full buffer")
	}
	close(st.block)
	m.Close()
	m.Mark(at("2026-10-20T09:00:00+03:30"), 1, activity.Session, nil) // after Close: dropped, no panic
}

func TestParseZone(t *testing.T) {
	assert.Equal(t, "Europe/Berlin", activity.ParseZone("Europe/Berlin").String())
	assert.Nil(t, activity.ParseZone(""))
	assert.Nil(t, activity.ParseZone("Local"))
	assert.Nil(t, activity.ParseZone("../etc/passwd"))
	assert.Nil(t, activity.ParseZone("Nowhere/City"))
}

func TestMarkLogged_NoMarkerIsNoop(t *testing.T) {
	activity.MarkLogged(context.Background(), 1) //nolint:staticcheck // no marker attached: must not panic
}
