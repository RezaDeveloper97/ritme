// Package usertz resolves the user's time zone for a request (engine v6 §2.3, X-07, deviation D-126).
//
// Precedence: a valid X-Timezone header (the device zone, sent by the frontend apiClient) → the stored
// user_profiles.timezone → Asia/Tehran. A request without the header and without a stored zone behaves exactly like
// before (Tehran), so the contract runs, which send no header, stay byte-identical.
//
// When an authenticated request carries a valid zone that differs from the stored one, it is stored with one
// conditional UPDATE that runs after the response, in the background: it never blocks or fails the request. There is
// no revision bump: the zone moves "today" (part of the ETag), not the engine state.
//
// Handlers read the zone through civildate.LocalToday / LocalNow. DB timestamps stay Asia/Tehran wall-clock; only
// the user's calendar day follows the zone. Background jobs, which have no request, use ZoneFor(profile).
package usertz

import (
	"context"
	"database/sql"
	"errors"
	"log/slog"
	"strings"
	"sync"
	"sync/atomic"
	"time"
	_ "time/tzdata" // every IANA zone must load, whatever the runtime image ships

	"github.com/gofiber/fiber/v3"

	"github.com/ritme/backend-go/internal/auth"
	"github.com/ritme/backend-go/internal/platform/civildate"
	profilestore "github.com/ritme/backend-go/internal/profile/store"
)

// Header carries the device's IANA zone (Intl.DateTimeFormat().resolvedOptions().timeZone).
const Header = "X-Timezone"

// maxNameLen is the user_profiles.timezone column width (varchar(64)).
const maxNameLen = 64

// locations caches the parsed valid zones (name → *time.Location). Only valid names enter, so its size is bounded by
// the tz database.
var locations sync.Map

// Parse validates an IANA zone name with time.LoadLocation (tzdata embedded) and returns its location. "" , "Local",
// over-long names and names with characters no IANA name uses are rejected.
func Parse(name string) (*time.Location, bool) {
	name = strings.TrimSpace(name)
	if name == "" || len(name) > maxNameLen || name == "Local" || !validChars(name) {
		return nil, false
	}
	if loc, ok := locations.Load(name); ok {
		return loc.(*time.Location), true
	}
	loc, err := time.LoadLocation(name)
	if err != nil {
		return nil, false
	}
	locations.Store(name, loc)
	return loc, true
}

func validChars(s string) bool {
	for _, r := range s {
		switch {
		case r >= 'a' && r <= 'z', r >= 'A' && r <= 'Z', r >= '0' && r <= '9':
		case r == '/', r == '_', r == '-', r == '+':
		default:
			return false
		}
	}
	return !strings.Contains(s, "..")
}

// ZoneOf is the location of a stored zone column: Asia/Tehran when it is NULL or no longer valid.
func ZoneOf(name sql.NullString) *time.Location {
	if name.Valid {
		if loc, ok := Parse(name.String); ok {
			return loc
		}
	}
	return civildate.Tehran
}

// ZoneFor is the stored zone of a profile, for background jobs (no request, no header).
func ZoneFor(p profilestore.UserProfile) *time.Location { return ZoneOf(p.Timezone) }

// Store reads and writes user_profiles.timezone (satisfied by *profilestore.Queries).
type Store interface {
	GetUserTimezone(ctx context.Context, userID uint64) (sql.NullString, error)
	StoreUserTimezone(ctx context.Context, arg profilestore.StoreUserTimezoneParams) (int64, error)
}

const (
	// knownTTL bounds how long a remembered stored zone is trusted (another instance may have changed it).
	knownTTL = 15 * time.Minute
	// knownMax bounds the remembered zones; the map is dropped when it fills up.
	knownMax = 50_000
	// storeTimeout bounds the background UPDATE.
	storeTimeout = 5 * time.Second
)

type known struct {
	zone string // "" = no stored zone (Tehran)
	at   time.Time
}

// Middleware attaches the request's zone resolver and stores changed zones.
type Middleware struct {
	store  Store
	logger *slog.Logger
	now    func() time.Time
	userID func(fiber.Ctx) (uint64, bool) // auth.CurrentUserID; tests swap it

	mu    sync.Mutex
	known map[uint64]known

	wg sync.WaitGroup
}

// New returns the middleware over store.
func New(store Store, logger *slog.Logger) *Middleware {
	if logger == nil {
		logger = slog.Default()
	}
	return &Middleware{store: store, logger: logger, now: time.Now, userID: auth.CurrentUserID, known: map[uint64]known{}}
}

// Wait blocks until the background stores started so far have finished (tests, shutdown).
func (m *Middleware) Wait() { m.wg.Wait() }

// Handler is the fiber middleware. Mount it globally; the user is read after the route's auth ran.
func (m *Middleware) Handler() fiber.Handler {
	return func(c fiber.Ctx) error {
		header, _ := Parse(c.Get(Header))

		var (
			mu       sync.Mutex
			resolved *time.Location
			done     atomic.Bool
		)
		resolver := civildate.ZoneResolver(func(ctx context.Context) *time.Location {
			mu.Lock()
			defer mu.Unlock()
			if resolved != nil {
				return resolved
			}
			if header != nil {
				resolved = header
				return resolved
			}
			if done.Load() { // the fiber.Ctx is recycled once the request ended
				return civildate.Tehran
			}
			uid, ok := m.userID(c)
			if !ok {
				return civildate.Tehran // the user may not be known yet; ask again later
			}
			resolved = m.stored(ctx, uid)
			return resolved
		})
		c.Locals(civildate.ZoneKey(), resolver)
		c.SetContext(civildate.WithZone(c.Context(), resolver))

		err := c.Next()

		if header != nil {
			if uid, ok := m.userID(c); ok {
				m.storeAfter(uid, header.String())
			}
		}
		done.Store(true)
		return err
	}
}

// stored is the user's stored zone (remembered, else one SELECT). Errors fall back to Tehran: the zone never fails a
// request.
func (m *Middleware) stored(ctx context.Context, uid uint64) *time.Location {
	if k, ok := m.lookup(uid); ok {
		return zoneOrTehran(k)
	}
	name, err := m.store.GetUserTimezone(ctx, uid)
	if err != nil && !errors.Is(err, sql.ErrNoRows) {
		m.logger.WarnContext(ctx, "usertz: read stored zone", slog.Uint64("user_id", uid), slog.String("error", err.Error()))
		return civildate.Tehran
	}
	zone := ""
	if name.Valid {
		if _, ok := Parse(name.String); ok {
			zone = name.String
		}
	}
	m.remember(uid, zone)
	return zoneOrTehran(zone)
}

func zoneOrTehran(zone string) *time.Location {
	if loc, ok := Parse(zone); ok {
		return loc
	}
	return civildate.Tehran
}

// storeAfter stores zone for uid in the background unless it is already known to be stored. The zone is remembered
// before the UPDATE runs, so the burst of parallel requests an app start makes issues one UPDATE, not one each.
func (m *Middleware) storeAfter(uid uint64, zone string) {
	if k, ok := m.lookup(uid); ok && k == zone {
		return
	}
	m.remember(uid, zone)
	m.wg.Add(1)
	go func() {
		defer m.wg.Done()
		ctx, cancel := context.WithTimeout(context.Background(), storeTimeout)
		defer cancel()
		_, err := m.store.StoreUserTimezone(ctx, profilestore.StoreUserTimezoneParams{
			Timezone: sql.NullString{String: zone, Valid: true},
			UserID:   uid,
		})
		if err != nil {
			m.forget(uid)
			m.logger.Warn("usertz: store zone", slog.Uint64("user_id", uid), slog.String("error", err.Error()))
		}
	}()
}

func (m *Middleware) lookup(uid uint64) (string, bool) {
	m.mu.Lock()
	defer m.mu.Unlock()
	k, ok := m.known[uid]
	if !ok || m.now().Sub(k.at) > knownTTL {
		return "", false
	}
	return k.zone, true
}

func (m *Middleware) remember(uid uint64, zone string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if len(m.known) >= knownMax {
		m.known = map[uint64]known{}
	}
	m.known[uid] = known{zone: zone, at: m.now()}
}

func (m *Middleware) forget(uid uint64) {
	m.mu.Lock()
	defer m.mu.Unlock()
	delete(m.known, uid)
}

// Reader reads the stored zone (satisfied by *profilestore.Queries).
type Reader interface {
	GetUserTimezone(ctx context.Context, userID uint64) (sql.NullString, error)
}

// Zones is the stored zone per user for background jobs (push and reminder senders, T-M12-21c), which have no
// request and so no header: one SELECT per user, remembered for knownTTL. A missing profile, a NULL or an invalid
// zone is Asia/Tehran; a read error is returned (the caller decides) and nothing is remembered.
type Zones struct {
	r   Reader
	now func() time.Time

	mu    sync.Mutex
	known map[uint64]known
}

// NewZones returns the job-side zone lookup over r.
func NewZones(r Reader) *Zones {
	return &Zones{r: r, now: time.Now, known: map[uint64]known{}}
}

// For is userID's stored zone (Asia/Tehran when none).
func (z *Zones) For(ctx context.Context, userID uint64) (*time.Location, error) {
	z.mu.Lock()
	k, ok := z.known[userID]
	z.mu.Unlock()
	if ok && z.now().Sub(k.at) <= knownTTL {
		return zoneOrTehran(k.zone), nil
	}
	name, err := z.r.GetUserTimezone(ctx, userID)
	if err != nil && !errors.Is(err, sql.ErrNoRows) {
		return civildate.Tehran, err
	}
	zone := ""
	if name.Valid {
		if _, ok := Parse(name.String); ok {
			zone = name.String
		}
	}
	z.mu.Lock()
	if len(z.known) >= knownMax {
		z.known = map[uint64]known{}
	}
	z.known[userID] = known{zone: zone, at: z.now()}
	z.mu.Unlock()
	return zoneOrTehran(zone), nil
}
