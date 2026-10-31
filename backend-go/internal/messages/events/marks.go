package events

import (
	"context"
	"errors"
	"strconv"
	"time"

	"github.com/redis/go-redis/v9"

	"github.com/ritme/backend-go/internal/platform/cache"
)

// MarkTTL keeps a once-a-day mark a little past the user-local day (M-19: 26 h covers any zone offset).
const MarkTTL = 26 * time.Hour

// GoalMarkTTL keeps a goal mark past the 7-day attribution window.
const GoalMarkTTL = 8 * 24 * time.Hour

// ErrNoMarks is returned when there is no Redis: the caller checks the database instead.
var ErrNoMarks = errors.New("messages: no redis for event marks")

// Marks are the Redis SET NX marks that make server-logged events once-only: `shown` and `eligible` once per (user,
// message, slot, user-local day) — `ritme-go:msg:shown:{uid}:{rule}:{slot}:{day}` — and a goal once per shown window.
type Marks struct{ c *cache.Client }

// NewMarks wraps c (nil = no Redis: every call answers ErrNoMarks).
func NewMarks(c *cache.Client) *Marks { return &Marks{c: c} }

// DayKey is the mark of a daily event (kind "shown" | "eligible") without the prefix.
func DayKey(kind string, userID, ruleID uint64, slot, day string) string {
	b := make([]byte, 0, 64)
	b = append(b, "msg:"...)
	b = append(b, kind...)
	b = append(b, ':')
	b = strconv.AppendUint(b, userID, 10)
	b = append(b, ':')
	b = strconv.AppendUint(b, ruleID, 10)
	b = append(b, ':')
	b = append(b, slot...)
	b = append(b, ':')
	b = append(b, day...)
	return string(b)
}

// GoalKey is the mark of a goal credited to the shown row at shownAt.
func GoalKey(userID, ruleID uint64, shownAt time.Time) string {
	return "msg:goal:" + strconv.FormatUint(userID, 10) + ":" + strconv.FormatUint(ruleID, 10) + ":" +
		strconv.FormatInt(shownAt.Unix(), 10)
}

// First sets every key with SET NX in one pipeline and reports, per key, whether this call set it (the event is the
// first of its window).
func (m *Marks) First(ctx context.Context, ttl time.Duration, keys ...string) ([]bool, error) {
	if m == nil || m.c == nil {
		return nil, ErrNoMarks
	}
	if len(keys) == 0 {
		return nil, nil
	}
	cmds := make([]*redis.BoolCmd, len(keys))
	_, err := m.c.Redis().Pipelined(ctx, func(p redis.Pipeliner) error {
		for i, k := range keys {
			cmds[i] = p.SetNX(ctx, m.c.Key(k), 1, ttl)
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	out := make([]bool, len(keys))
	for i, c := range cmds {
		out[i] = c.Val()
	}
	return out, nil
}

// Exists reports whether key is set.
func (m *Marks) Exists(ctx context.Context, key string) (bool, error) {
	if m == nil || m.c == nil {
		return false, ErrNoMarks
	}
	n, err := m.c.Redis().Exists(ctx, m.c.Key(key)).Result()
	return n > 0, err
}
