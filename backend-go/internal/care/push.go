package care

import (
	"cmp"
	"context"
	"crypto/sha256"
	"database/sql"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"log/slog"
	nethttp "net/http"
	"net/url"
	"slices"
	"strings"
	"sync"
	"time"

	webpush "github.com/SherClockHolmes/webpush-go"
	"github.com/gofiber/fiber/v3"

	"github.com/ritme/backend-go/internal/care/store"
	"github.com/ritme/backend-go/internal/i18n"
	"github.com/ritme/backend-go/internal/i18n/lang"
	"github.com/ritme/backend-go/internal/notifications"
	"github.com/ritme/backend-go/internal/platform/civildate"
	"github.com/ritme/backend-go/internal/platform/config"
	"github.com/ritme/backend-go/internal/platform/httpx"
	"github.com/ritme/backend-go/internal/platform/jsonx"
	"github.com/ritme/backend-go/internal/platform/usertz"
	"github.com/ritme/backend-go/internal/platform/validation"
	"github.com/ritme/backend-go/internal/platform/validation/phpval"
)

// Medication reminders reach the user at the dose time on two channels, both fed by Occurrences and both gated by
// notifications.Decide / Render (category switch, quiet hours, «متن خنثی»):
//   - Web Push (browsers, installed PWA): Pusher.Loop sends every due dose to the user's push_subscriptions;
//   - the Android shell, whose WebView has no Push API: the page hands GET /care/upcoming to the native bridge,
//     which arms local alarms.
// A dose already ticked is not announced; a dose that falls in quiet hours is dropped, not announced late.

const (
	// PushEvery is how often the sender looks for due doses.
	PushEvery = 30 * time.Second
	// PushLookback is how late a dose may still be pushed (a restart, a slow tick); older ones are dropped.
	PushLookback = 10 * time.Minute
	// PushAttempts is how many times one process tries a dose or an inbox row that no browser accepted because of a
	// passing failure (timeout, 429, 5xx): once per tick, inside PushLookback / NoticeLookback. After that it stays
	// claimed and silent. A refusal for good (404 / 410 → the subscription is dropped; any other 4xx) is never retried.
	PushAttempts = 3
	// PushTTL is how long a push service keeps an undelivered reminder (seconds).
	PushTTL = 3600
	// PushLogKeep is how long reminder_pushes rows are kept.
	PushLogKeep = 7 * 24 * time.Hour
	// MaxPushSubscriptions caps the browsers one user subscribes (the oldest goes first).
	MaxPushSubscriptions = 10
	// UpcomingDefaultHours / UpcomingMaxHours bound GET /care/upcoming?hours=; UpcomingMaxItems its list.
	UpcomingDefaultHours = 48
	UpcomingMaxHours     = 168
	UpcomingMaxItems     = 64
	// UpcomingMedication / UpcomingAppointment are the `kind` of a GET /care/upcoming item.
	UpcomingMedication  = "medication"
	UpcomingAppointment = "appointment"
	// MaxRemindOffset is the longest remind_before lead time (RemindOffset "2d").
	MaxRemindOffset = 48 * time.Hour
)

// pushHosts are the push services a subscription endpoint may point at (the server POSTs to the endpoint, so an
// arbitrary URL would be a request forgery primitive): exact hosts or ".suffix" domains.
var pushHosts = []string{
	"fcm.googleapis.com", "android.googleapis.com", "updates.push.services.mozilla.com",
	".push.services.mozilla.com", ".notify.windows.com", ".push.apple.com",
}

// ValidPushEndpoint reports whether raw is an https URL of a known push service.
func ValidPushEndpoint(raw string) bool {
	u, err := url.Parse(raw)
	if err != nil || u.Scheme != "https" || u.User != nil || u.Port() != "" && u.Port() != "443" {
		return false
	}
	host := strings.ToLower(u.Hostname())
	for _, h := range pushHosts {
		if host == h || strings.HasPrefix(h, ".") && strings.HasSuffix(host, h) {
			return true
		}
	}
	return false
}

func validPushKey(s string, size int) bool {
	b, err := base64.RawURLEncoding.DecodeString(strings.TrimRight(s, "="))
	if err != nil {
		b, err = base64.RawStdEncoding.DecodeString(strings.TrimRight(s, "="))
	}
	return err == nil && len(b) == size
}

func endpointHash(endpoint string) string {
	sum := sha256.Sum256([]byte(endpoint))
	return hex.EncodeToString(sum[:])
}

// Occurrence is one dose at its wall-clock time.
type Occurrence struct {
	Med  Medication
	Day  civildate.Date // the dose day on the user's clock
	Slot string
	At   time.Time // the slot on Day in the user's zone (OccurrencesIn)
}

// ID is the stable key of the dose: "<reminder>:<Y-m-d>:<HH:MM>".
func (o Occurrence) ID() string {
	return fmt.Sprintf("%d:%s:%s", o.Med.Row.ID, o.Day.String(), o.Slot)
}

// Occurrences is OccurrencesIn on the Asia/Tehran clock (a user without a stored zone; senders that do not resolve
// one yet, e.g. internal/remindersms).
func Occurrences(rows []store.Reminder, from, to time.Time) []Occurrence {
	return OccurrencesIn(rows, from, to, civildate.Tehran)
}

// OccurrencesIn are the doses of the active, notifying medications in rows with from ≤ At < to, sorted by time. The
// stored slots are wall-clock times of the user's zone loc (T-M12-21c, like M-08): «۸ صبح» is 08:00 where she is,
// across daylight-saving changes too. Each (day, slot) is one instant (notifications.WallTime): a slot inside the
// hour skipped when clocks go forward fires once at the transition (02:30 → 03:00 local), a slot in the hour
// repeated when they go back fires once, at its first occurrence — never skipped, never doubled.
func OccurrencesIn(rows []store.Reminder, from, to time.Time, loc *time.Location) []Occurrence {
	if loc == nil {
		loc = civildate.Tehran
	}
	out := []Occurrence{}
	first, last := civildate.FromTime(from.In(loc)), civildate.FromTime(to.In(loc))
	for _, r := range rows {
		if !r.IsActive {
			continue
		}
		m := ParseMedication(r)
		if !m.Meta.Notify {
			continue
		}
		for day := first; !day.After(last); day = day.AddDays(1) {
			for _, slot := range m.SlotsOn(day) {
				minute, ok := slotMinutes(slot)
				if !ok {
					continue
				}
				at := notifications.WallTime(day, minute, loc)
				if !at.Before(from) && at.Before(to) {
					out = append(out, Occurrence{Med: m, Day: day, Slot: slot, At: at})
				}
			}
		}
	}
	slices.SortStableFunc(out, func(a, b Occurrence) int {
		return cmp.Or(a.At.Compare(b.At), cmp.Compare(a.Med.Row.ID, b.Med.Row.ID))
	})
	return out
}

// Message is what the reminder of the dose says before Render: «وقت داروی توست» + the name with the dose
// («آهن · ۱ قرص · بعد از غذا»); a tap opens the reminders screen.
func (o Occurrence) Message(locale string) notifications.Message {
	body := o.Med.Row.Title
	if spec := o.Med.Meta.DoseAt(o.Slot); spec != nil {
		body += " · " + DoseLine(*spec, locale)
	} else if sub := o.Med.DisplaySubtitle(locale); sub.Valid && sub.String != "" {
		body += " · " + sub.String
	}
	return notifications.Message{
		Category: notifications.Medications,
		Title:    T("push.medication_title", locale),
		Body:     body,
		URL:      "/" + locale + "/reminders",
	}
}

// Key is the dose's occurrence inside its reminder, "Y-m-d HH:MM" (reminder_sms.occurrence).
func (o Occurrence) Key() string { return o.Day.String() + " " + o.Slot }

// AppointmentDue is an appointment at its reminder moment (the visit − remind_before).
type AppointmentDue struct {
	A     Appointment
	At    time.Time // when the reminder fires (Tehran)
	Visit time.Time
}

// AppointmentsDue are the reminders of the active, scheduled appointments in rows with from ≤ At < to, soonest first.
func AppointmentsDue(rows []store.Reminder, from, to time.Time) []AppointmentDue {
	out := []AppointmentDue{}
	for _, r := range rows {
		if !r.IsActive || !r.ScheduledAt.Valid {
			continue
		}
		a := ParseAppointment(r)
		if a.Meta.Status != StatusScheduled {
			continue
		}
		visit := r.ScheduledAt.Time.In(civildate.Tehran)
		at := visit.Add(-RemindOffset(a.Meta.RemindBefore))
		if !at.Before(from) && at.Before(to) {
			out = append(out, AppointmentDue{A: a, At: at, Visit: visit})
		}
	}
	slices.SortStableFunc(out, func(a, b AppointmentDue) int {
		return cmp.Or(a.At.Compare(b.At), cmp.Compare(a.A.Row.ID, b.A.Row.ID))
	})
	return out
}

// Key is the occurrence inside the reminder: the visit time, so a rescheduled appointment reminds again.
func (d AppointmentDue) Key() string { return d.Visit.Format("2006-01-02 15:04") }

// When is the visit seen from the reminder moment: «امروز ساعت 10:30», «فردا ساعت 10:30», «2 روز دیگر ساعت 10:30».
func (d AppointmentDue) When(locale string) string {
	clock := d.Visit.Format("15:04")
	days := civildate.InTehran(d.At).DiffDays(civildate.InTehran(d.Visit))
	if days < 0 {
		days = -days
	}
	switch days {
	case 0:
		return translator().Trans("care.push.when_today", map[string]string{"time": clock}, locale)
	case 1:
		return translator().Trans("care.push.when_tomorrow", map[string]string{"time": clock}, locale)
	}
	return translator().Trans("care.push.when_days", map[string]string{"time": clock, "days": fmt.Sprint(days)}, locale)
}

// Message is what the appointment reminder says before Render: «یادآور نوبت پزشک» + «title · فردا ساعت 10:30».
func (d AppointmentDue) Message(locale string) notifications.Message {
	return notifications.Message{
		Category: notifications.Appointments,
		Title:    T("push.appointment_title", locale),
		Body:     d.A.Row.Title + " · " + d.When(locale),
		URL:      "/" + locale + "/reminders",
	}
}

// PrefsGetter loads a user's notification settings and stored time zone (the profile store).
type PrefsGetter interface {
	notifications.Getter
	usertz.Reader
}

// SetPush gives the handlers what the push endpoints need: the VAPID settings and the settings reader.
func (h *Handlers) SetPush(cfg config.Push, prefs PrefsGetter) {
	h.push, h.prefs = cfg, prefs
}

// PushConfig is GET /push/config: the VAPID public key a browser subscribes with (null = web push is off here).
func (h *Handlers) PushConfig(c fiber.Ctx) error {
	if _, err := h.user(c); err != nil {
		return err
	}
	var key any
	if h.push.Enabled() {
		key = h.push.VAPIDPublicKey
	}
	return httpx.OK(c, jsonx.Obj("public_key", key))
}

func pushRules(keys bool) validation.Rules {
	F := validation.F
	rules := validation.Rules{F("endpoint", "required", "string", "max:2000")}
	if keys {
		rules = append(rules, F("keys.p256dh", "required", "string", "max:255"), F("keys.auth", "required", "string", "max:255"))
	}
	return rules
}

// Subscribe is POST /push/subscriptions {endpoint, keys: {p256dh, auth}}: this browser receives the user's
// reminders (idempotent; a browser belongs to the last account that subscribed it).
func (h *Handlers) Subscribe(c fiber.Ctx) error {
	userID, err := h.user(c)
	if err != nil {
		return err
	}
	locale, now := i18n.Locale(c), h.now(c)
	v := validation.Make(lang.Default(), locale, validation.Input(c), pushRules(true),
		validation.Now(now), validation.Attributes(attributes(locale)...))
	if v.Fails() {
		return failValidation(locale, v.ErrorBag())
	}
	in := validation.Input(c)
	get := func(path string) string {
		x, _ := phpval.Get(in, path)
		return phpval.ToString(x)
	}
	endpoint, p256dh, auth := get("endpoint"), get("keys.p256dh"), get("keys.auth")
	if !ValidPushEndpoint(endpoint) {
		return fieldError(locale, "endpoint", T("validation.push_endpoint", locale))
	}
	if !validPushKey(p256dh, 65) || !validPushKey(auth, 16) {
		return fieldError(locale, "keys", T("validation.push_keys", locale))
	}
	if err := h.q.UpsertPushSubscription(c, store.UpsertPushSubscriptionParams{
		UserID: userID, Endpoint: endpoint, EndpointHash: endpointHash(endpoint), P256dh: p256dh, Auth: auth,
		Locale: locale, Now: sql.NullTime{Time: now, Valid: true},
	}); err != nil {
		return fmt.Errorf("care: save push subscription: %w", err)
	}
	for range MaxPushSubscriptions { // bounded: drop the oldest ones over the cap
		n, err := h.q.CountPushSubscriptions(c, userID)
		if err != nil {
			return fmt.Errorf("care: count push subscriptions: %w", err)
		}
		if n <= MaxPushSubscriptions {
			break
		}
		if err := h.q.DeleteOldestPushSubscription(c, userID); err != nil {
			return fmt.Errorf("care: trim push subscriptions: %w", err)
		}
	}
	return httpx.OK(c, jsonx.Obj("subscribed", true), T("messages.push_subscribed", locale))
}

// Unsubscribe is DELETE /push/subscriptions {endpoint} (body or query): idempotent.
func (h *Handlers) Unsubscribe(c fiber.Ctx) error {
	userID, err := h.user(c)
	if err != nil {
		return err
	}
	locale, now := i18n.Locale(c), h.now(c)
	in := validation.Input(c)
	if x, _ := in.Get("endpoint"); x == nil {
		in = validation.Query(c)
	}
	v := validation.Make(lang.Default(), locale, in, pushRules(false),
		validation.Now(now), validation.Attributes(attributes(locale)...))
	if v.Fails() {
		return failValidation(locale, v.ErrorBag())
	}
	x, _ := in.Get("endpoint")
	if _, err := h.q.DeletePushSubscription(c, store.DeletePushSubscriptionParams{
		UserID: userID, EndpointHash: endpointHash(phpval.ToString(x)),
	}); err != nil {
		return fmt.Errorf("care: delete push subscription: %w", err)
	}
	return httpx.OK(c, jsonx.Obj("subscribed", false), T("messages.push_unsubscribed", locale))
}

// Upcoming is GET /care/upcoming?hours=N (default 48, max 168): the reminders that will fire from now on, already
// rendered for the lock screen — what the Android shell arms as local alarms: medication doses and the reminders
// of scheduled appointments (the visit − remind_before). Ticked doses, reminders in quiet hours and a kind whose
// category («داروها» / «نوبت پزشک») is off are left out; at most UpcomingMaxItems, soonest first. Doses, quiet hours
// and the wall-clock strings are on the user's clock (the request's zone: X-Timezone → stored → Asia/Tehran,
// T-M12-21c); appointment visits stay the stored Tehran wall-clock instant.
func (h *Handlers) Upcoming(c fiber.Ctx) error {
	userID, err := h.user(c)
	if err != nil {
		return err
	}
	locale, now := i18n.Locale(c), h.now(c)
	v := validation.Make(lang.Default(), locale, validation.Query(c),
		validation.Rules{validation.F("hours", "nullable", "integer", "min:1", fmt.Sprintf("max:%d", UpcomingMaxHours))},
		validation.Now(now), validation.Attributes(attributes(locale)...))
	if v.Fails() {
		return failValidation(locale, v.ErrorBag())
	}
	hours := UpcomingDefaultHours
	if raw, _ := v.Validated().Get("hours"); raw != nil && phpval.ToString(raw) != "" {
		hours = int(phpval.ToFloat(raw))
	}
	loc := civildate.Zone(c)
	type entry struct {
		at   time.Time
		item *jsonx.OrderedMap
	}
	var list []entry
	out := func() error {
		slices.SortStableFunc(list, func(a, b entry) int { return a.at.Compare(b.at) })
		items := []*jsonx.OrderedMap{}
		for _, e := range list[:min(len(list), UpcomingMaxItems)] {
			items = append(items, e.item)
		}
		return httpx.OK(c, jsonx.Obj("generated_at", now.In(loc).Format(wallClock), "items", items))
	}
	if h.prefs == nil {
		return out()
	}
	prefs, err := notifications.Load(c, h.prefs, userID)
	if err != nil {
		return err
	}
	prefs = prefs.InZone(loc)
	from, to := now.Add(time.Second), now.Add(time.Duration(hours)*time.Hour)
	add := func(kind, id string, reminderID uint64, day civildate.Date, slot string, at time.Time, m notifications.Message) {
		p := notifications.Render(prefs, m, locale)
		list = append(list, entry{at: at, item: jsonx.Obj(
			"id", id,
			"kind", kind,
			"reminder_id", reminderID,
			"date", day.String(),
			"slot", slot,
			"at", at.In(loc).Format(wallClock),
			"at_ms", at.UnixMilli(),
			"title", p.Title,
			"body", p.Body,
			"url", p.URL,
		)})
	}
	if prefs.Enabled(notifications.Medications) {
		meds, err := h.q.ListActiveMedications(c, userID)
		if err != nil {
			return fmt.Errorf("care: upcoming medications: %w", err)
		}
		taken, err := h.q.ListIntakesBetween(c, store.ListIntakesBetweenParams{
			UserID: userID, FromDate: civildate.FromTime(now.In(loc)), ToDate: civildate.FromTime(to.In(loc)),
		})
		if err != nil {
			return fmt.Errorf("care: upcoming intakes: %w", err)
		}
		n := 0
		for _, o := range OccurrencesIn(meds, from, to, loc) {
			if n >= UpcomingMaxItems {
				break
			}
			if slices.ContainsFunc(taken, func(t store.ListIntakesBetweenRow) bool {
				return t.ReminderID == o.Med.Row.ID && t.IntakeDate == o.Day && t.Slot == o.Slot
			}) {
				continue
			}
			if !notifications.Decide(prefs, notifications.Medications, o.At, time.Time{}).Send {
				continue
			}
			add(UpcomingMedication, o.ID(), o.Med.Row.ID, o.Day, o.Slot, o.At, o.Message(locale))
			n++
		}
	}
	if prefs.Enabled(notifications.Appointments) {
		// A visit is announced remind_before ahead, so visits up to the longest lead time past the window count.
		appts, err := h.q.ListActiveAppointmentsOfUsers(c, store.ListActiveAppointmentsOfUsersParams{
			UserIds: []uint64{userID},
			FromAt:  sql.NullTime{Time: now, Valid: true},
			ToAt:    sql.NullTime{Time: to.Add(MaxRemindOffset), Valid: true},
		})
		if err != nil {
			return fmt.Errorf("care: upcoming appointments: %w", err)
		}
		for _, d := range AppointmentsDue(appts, from, to) {
			if !notifications.Decide(prefs, notifications.Appointments, d.At, time.Time{}).Send {
				continue
			}
			day, slot := civildate.InTehran(d.Visit), d.Visit.Format("15:04")
			add(UpcomingAppointment, fmt.Sprintf("%d:%s:%s", d.A.Row.ID, day.String(), slot), d.A.Row.ID, day, slot, d.At, d.Message(locale))
		}
	}
	return out()
}

// PushSender delivers one encrypted push; the default is the webpush library.
type PushSender func(ctx context.Context, payload []byte, sub store.PushSubscription) (status int, err error)

// Pusher sends the due medication reminders (Tick) and the new inbox rows (TickNotices, push_notices.go) by Web Push.
type Pusher struct {
	q      *store.Queries
	prefs  PrefsGetter
	zones  *usertz.Zones // each user's stored zone: doses and quiet hours are on her clock (T-M12-21c)
	logger *slog.Logger
	send   PushSender

	mu           sync.Mutex // guards the fields below: one Pusher may tick from several goroutines
	pruned       time.Time
	retryDoses   []pending[Occurrence]                         // claimed doses to try again (push_retry.go)
	retryNotices []pending[store.ListUnpushedNotificationsRow] // claimed inbox rows to try again
}

// NewPusher wires the sender; cfg must be Enabled.
func NewPusher(db store.DBTX, prefs PrefsGetter, cfg config.Push, logger *slog.Logger) *Pusher {
	client := &nethttp.Client{
		Timeout: 10 * time.Second,
		// A push service never redirects; following one would let an endpoint steer the request elsewhere.
		CheckRedirect: func(*nethttp.Request, []*nethttp.Request) error { return nethttp.ErrUseLastResponse },
	}
	send := func(ctx context.Context, payload []byte, sub store.PushSubscription) (int, error) {
		resp, err := webpush.SendNotificationWithContext(ctx, payload, &webpush.Subscription{
			Endpoint: sub.Endpoint, Keys: webpush.Keys{P256dh: sub.P256dh, Auth: sub.Auth},
		}, &webpush.Options{
			HTTPClient: client, Subscriber: cfg.Subject, TTL: PushTTL, Urgency: webpush.UrgencyHigh,
			VAPIDPublicKey: cfg.VAPIDPublicKey, VAPIDPrivateKey: cfg.VAPIDPrivateKey,
		})
		if err != nil {
			return 0, err
		}
		_ = resp.Body.Close()
		return resp.StatusCode, nil
	}
	return &Pusher{q: store.New(db), prefs: prefs, zones: usertz.NewZones(prefs), logger: logger, send: send}
}

// WithSender replaces the delivery (tests).
func (p *Pusher) WithSender(s PushSender) *Pusher { p.send = s; return p }

// Loop runs Tick (due doses) and TickNotices (new inbox rows) every PushEvery until ctx ends.
func (p *Pusher) Loop(ctx context.Context, now func() time.Time) {
	t := time.NewTicker(PushEvery)
	defer t.Stop()
	for {
		if _, err := p.Tick(ctx, now()); err != nil && ctx.Err() == nil {
			p.logger.ErrorContext(ctx, "care: reminder push", slog.String("error", err.Error()))
		}
		if _, err := p.TickNotices(ctx, now()); err != nil && ctx.Err() == nil {
			p.logger.ErrorContext(ctx, "care: inbox push", slog.String("error", err.Error()))
		}
		select {
		case <-ctx.Done():
			return
		case <-t.C:
		}
	}
}

// Tick pushes the doses that came due in the last PushLookback and returns how many reminders went out. Each dose
// is claimed in reminder_pushes first, so several processes (API + worker) never announce it twice; a dose that is
// already ticked, switched off or inside quiet hours stays claimed and silent. A dose no browser accepted because of
// a passing failure stays claimed by this process, which tries it again on its next ticks (retryDose, at most
// PushAttempts tries inside PushLookback). Doses and quiet hours are on each user's stored zone (usertz.Zones; none =
// Asia/Tehran), so «۸ صبح» reaches a user in Berlin at 08:00 Berlin. Safe for concurrent use.
func (p *Pusher) Tick(ctx context.Context, now time.Time) (int, error) {
	now = now.In(civildate.Tehran)
	if p.prunePushes(now) {
		if err := p.q.PruneReminderPushes(ctx, now.Add(-PushLogKeep)); err != nil {
			return 0, fmt.Errorf("care: prune pushes: %w", err)
		}
	}
	users := map[uint64]*doseUser{}
	sent := 0
	for _, r := range takeRetries(p, &p.retryDoses) { // the doses this process claimed and could not deliver yet
		if r.item.At.Before(now.Add(-PushLookback)) {
			continue // too late now: dropped like any other stale dose
		}
		ok, err := p.pushDose(ctx, users, r.item, r.tries)
		if err != nil {
			return sent, err
		}
		if ok {
			sent++
		}
	}
	meds, err := p.q.ListPushableMedications(ctx)
	if err != nil {
		return sent, fmt.Errorf("care: pushable medications: %w", err)
	}
	byUser := map[uint64][]store.Reminder{}
	for _, r := range meds {
		byUser[r.UserID] = append(byUser[r.UserID], r)
	}
	var due []Occurrence
	for userID, rows := range byUser {
		loc, err := p.zones.For(ctx, userID)
		if err != nil {
			return sent, fmt.Errorf("care: push zone: %w", err)
		}
		due = append(due, OccurrencesIn(rows, now.Add(-PushLookback), now.Add(time.Second), loc)...)
	}
	slices.SortStableFunc(due, func(a, b Occurrence) int {
		return cmp.Or(a.At.Compare(b.At), cmp.Compare(a.Med.Row.ID, b.Med.Row.ID))
	})
	for _, o := range due {
		n, err := p.q.ClaimReminderPush(ctx, store.ClaimReminderPushParams{
			UserID: o.Med.Row.UserID, ReminderID: o.Med.Row.ID, IntakeDate: o.Day, Slot: o.Slot, SentAt: now,
		})
		if err != nil {
			return sent, fmt.Errorf("care: claim push: %w", err)
		}
		if n == 0 {
			continue
		}
		ok, err := p.pushDose(ctx, users, o, 0)
		if err != nil {
			return sent, err
		}
		if ok {
			sent++
		}
	}
	return sent, nil
}

// doseUser is what one Tick reads once per user: her settings on her zone, her browsers, the doses she ticked.
type doseUser struct {
	prefs notifications.Preferences
	subs  []store.PushSubscription
	taken map[civildate.Date][]store.ListIntakesOnDateRow
}

// pushDose announces the claimed dose o (tries = the earlier tries of this process) unless it is switched off, in
// quiet hours or already ticked, and reports whether a browser accepted it.
func (p *Pusher) pushDose(ctx context.Context, users map[uint64]*doseUser, o Occurrence, tries int) (bool, error) {
	userID := o.Med.Row.UserID
	u := users[userID]
	if u == nil {
		u = &doseUser{taken: map[civildate.Date][]store.ListIntakesOnDateRow{}}
		var err error
		if u.prefs, err = notifications.Load(ctx, p.prefs, userID); err != nil {
			return false, err
		}
		loc, err := p.zones.For(ctx, userID)
		if err != nil {
			return false, fmt.Errorf("care: push zone: %w", err)
		}
		u.prefs = u.prefs.InZone(loc)
		if u.subs, err = p.q.ListPushSubscriptions(ctx, userID); err != nil {
			return false, fmt.Errorf("care: push subscriptions: %w", err)
		}
		users[userID] = u
	}
	if !notifications.Decide(u.prefs, notifications.Medications, o.At, time.Time{}).Send {
		return false, nil
	}
	taken, ok := u.taken[o.Day]
	if !ok {
		var err error
		if taken, err = p.q.ListIntakesOnDate(ctx, store.ListIntakesOnDateParams{UserID: userID, IntakeDate: o.Day}); err != nil {
			return false, fmt.Errorf("care: push intakes: %w", err)
		}
		u.taken[o.Day] = taken
	}
	if slices.ContainsFunc(taken, func(t store.ListIntakesOnDateRow) bool {
		return t.ReminderID == o.Med.Row.ID && t.Slot == o.Slot
	}) {
		return false, nil
	}
	res, err := p.deliver(ctx, u.subs, o.ID(), func(locale string) notifications.Push {
		return notifications.Render(u.prefs, o.Message(locale), locale)
	})
	if err != nil {
		return false, err
	}
	if res == retryLater {
		retry(p, &p.retryDoses, o, tries+1)
	}
	return res == delivered, nil
}

// deliveryResult is how one deliver went.
type deliveryResult int

const (
	// notDelivered: no browser accepted it for good (gone, refused, or no browser at all) — never retried.
	notDelivered deliveryResult = iota
	// delivered: at least one push service accepted it.
	delivered
	// retryLater: none accepted it and at least one failed for a passing reason (timeout, 429, 5xx).
	retryLater
)

// deliver sends one message to every subscription, rendered in the subscription's language. At least one push
// service accepting it is delivered (the others are not tried again: the browser that has it must not get it
// twice). A subscription the push service calls gone (404 / 410) is deleted; other refusals are logged without the
// text.
func (p *Pusher) deliver(ctx context.Context, subs []store.PushSubscription, tag string, render func(locale string) notifications.Push) (deliveryResult, error) {
	accepted, passing := false, false
	for _, sub := range subs {
		msg := render(sub.Locale)
		payload, err := json.Marshal(map[string]string{"title": msg.Title, "body": msg.Body, "url": msg.URL, "tag": tag})
		if err != nil {
			return notDelivered, fmt.Errorf("care: push payload: %w", err)
		}
		status, err := p.send(ctx, payload, sub)
		switch {
		case err != nil:
			passing = true
			p.logger.WarnContext(ctx, "care: push not delivered", slog.Uint64("subscription", sub.ID), slog.String("error", err.Error()))
		case status == nethttp.StatusNotFound || status == nethttp.StatusGone:
			if err := p.q.DeletePushSubscriptionByID(ctx, sub.ID); err != nil {
				return notDelivered, fmt.Errorf("care: drop push subscription: %w", err)
			}
		case status == nethttp.StatusTooManyRequests || status >= 500:
			passing = true
			p.logger.WarnContext(ctx, "care: push refused for now", slog.Uint64("subscription", sub.ID), slog.Int("status", status))
		case status >= 400:
			p.logger.WarnContext(ctx, "care: push refused", slog.Uint64("subscription", sub.ID), slog.Int("status", status))
		default:
			accepted = true
		}
	}
	switch {
	case accepted:
		return delivered, nil
	case passing:
		return retryLater, nil
	}
	return notDelivered, nil
}
