package emergency

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"database/sql"
	"encoding/base64"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"regexp"
	"strings"
	"time"

	"github.com/ritme/backend-go/internal/care"
	carestore "github.com/ritme/backend-go/internal/care/store"
	"github.com/ritme/backend-go/internal/emergency/store"
	"github.com/ritme/backend-go/internal/healthrecord"
	"github.com/ritme/backend-go/internal/platform/civildate"
	"github.com/ritme/backend-go/internal/platform/jsonx"
	"github.com/ritme/backend-go/internal/pregnancy"
	"github.com/ritme/backend-go/internal/pregnancy/calc"
	pregstore "github.com/ritme/backend-go/internal/pregnancy/store"
)

// ErrNotFound: an unknown, malformed or disabled public token.
var ErrNotFound = errors.New("emergency: card not found")

// Records builds the share-audience record sections the card reads (*healthrecord.Service).
type Records interface {
	Build(ctx context.Context, userID uint64, aud healthrecord.Audience, opts healthrecord.Options) (*healthrecord.Record, error)
}

// Service reads and edits the card. Every owner query is scoped by the user id it is given.
type Service struct {
	q       *store.Queries
	care    carestore.Querier
	preg    pregstore.Querier
	records Records
	rand    io.Reader
}

// NewService wires the service on db.
func NewService(db *sql.DB, records Records) *Service {
	return &Service{q: store.New(db), care: carestore.New(db), preg: pregstore.New(db), records: records, rand: rand.Reader}
}

// WithRand replaces the token source (tests).
func (s *Service) WithRand(r io.Reader) *Service { s.rand = r; return s }

func nstr(v string) sql.NullString { return sql.NullString{String: v, Valid: v != ""} }

func stamp(t time.Time) sql.NullTime { return sql.NullTime{Time: t, Valid: true} }

// settings is the stored row (zero values when the owner never saved the card).
func (s *Service) settings(ctx context.Context, userID uint64) (store.EmergencyCard, error) {
	row, err := s.q.GetEmergencyCard(ctx, userID)
	if errors.Is(err, sql.ErrNoRows) {
		return store.EmergencyCard{UserID: userID}, nil
	}
	if err != nil {
		return row, fmt.Errorf("emergency: card: %w", err)
	}
	return row, nil
}

func (s *Service) allergiesOnCard(ctx context.Context, userID uint64) (bool, error) {
	on, err := s.q.GetEmergencyAllergiesFlag(ctx, userID)
	if errors.Is(err, sql.ErrNoRows) {
		return true, nil // CB-REC-01 default: on
	}
	if err != nil {
		return false, fmt.Errorf("emergency: allergies flag: %w", err)
	}
	return on, nil
}

func sectionValue(rec *healthrecord.Record, key, field string) any {
	sec, ok := rec.Section(key)
	if !ok || sec.Data == nil {
		return nil
	}
	v, _ := sec.Data.Get(field)
	return v
}

func listOrEmpty(v any) any {
	if v == nil {
		return []string{}
	}
	return v
}

// MaskLast4 is the masked insurance number «•••• 4821».
func MaskLast4(last4 string) string { return "•••• " + last4 }

// Card is the built card: Public is what a public link shows, Owner adds the insurance and the settings.
type Card struct {
	Row       store.EmergencyCard
	Allergies bool
	Public    *jsonx.OrderedMap // the minimal card (public link): first name, no gyn conditions, no onboarding meds
	Owner     *jsonx.OrderedMap // the owner's card without insurance
}

// firstName is the first word of a name (nil stays nil): the public card does not carry a full name.
func firstName(v any) any {
	s, ok := v.(string)
	if !ok {
		return v
	}
	if f := strings.Fields(s); len(f) > 0 {
		return f[0]
	}
	return nil
}

// Build reads the card of userID for today.
func (s *Service) Build(ctx context.Context, userID uint64, today civildate.Date, locale, def string) (Card, error) {
	row, err := s.settings(ctx, userID)
	if err != nil {
		return Card{}, err
	}
	allergiesOn, err := s.allergiesOnCard(ctx, userID)
	if err != nil {
		return Card{}, err
	}
	rec, err := s.records.Build(ctx, userID, healthrecord.AudienceShare, healthrecord.Options{
		Today: today, Locale: locale, DefaultLocale: def,
		Sections: []string{healthrecord.SectionBasics, healthrecord.SectionConditions, healthrecord.SectionMedications,
			healthrecord.SectionAllergies},
	})
	if err != nil {
		return Card{}, err
	}
	var name any
	if rec.Person != nil {
		name, _ = rec.Person.Get("name")
	}
	var allergies any
	if allergiesOn {
		allergies = listOrEmpty(sectionValue(rec, healthrecord.SectionAllergies, "items"))
	}
	meds, err := s.permanentMedications(ctx, userID, locale)
	if err != nil {
		return Card{}, err
	}
	var preg any
	if row.ShowPregnancy {
		if preg, err = s.pregnancy(ctx, userID, today, locale); err != nil {
			return Card{}, err
		}
	}
	var contact any
	if row.ContactName.Valid || row.ContactPhone.Valid {
		contact = jsonx.Obj("name", nullOrStr(row.ContactName), "relation", nullOrStr(row.ContactRelation),
			"phone", nullOrStr(row.ContactPhone))
	}
	public := jsonx.Obj(
		"name", firstName(name),
		"blood_type", sectionValue(rec, healthrecord.SectionBasics, "blood_type"),
		"allergies", allergies,
		"conditions", jsonx.Obj(
			"chronic_illnesses", listOrEmpty(sectionValue(rec, healthrecord.SectionConditions, "chronic_illnesses")),
		),
		"medications", meds,
		"pregnancy", preg,
		"emergency_contact", contact,
	)
	// The owner's card adds what stays off the public / lock-screen card (security review M2): the full name, the
	// gynaecological conditions and the onboarding medication list (may hold contraception / hormone therapy).
	owner := jsonx.NewArray()
	for _, k := range public.Keys() {
		v, _ := public.Get(k)
		owner.Set(k, v)
	}
	owner.Set("name", name)
	owner.Set("conditions", jsonx.Obj(
		"chronic_illnesses", listOrEmpty(sectionValue(rec, healthrecord.SectionConditions, "chronic_illnesses")),
		"gyn_conditions", listOrEmpty(sectionValue(rec, healthrecord.SectionConditions, "gyn_conditions")),
	))
	owner.Set("profile_medications", listOrEmpty(sectionValue(rec, healthrecord.SectionMedications, "profile_medications")))
	return Card{Row: row, Allergies: allergiesOn, Public: public, Owner: owner}, nil
}

func nullOrStr(s sql.NullString) any {
	if !s.Valid || s.String == "" {
		return nil
	}
	return s.String
}

// permanentMedications are the active care medications taken with no end («ongoing»): title and dose only.
func (s *Service) permanentMedications(ctx context.Context, userID uint64, locale string) ([]*jsonx.OrderedMap, error) {
	rows, err := s.care.ListActiveMedications(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("emergency: medications: %w", err)
	}
	out := []*jsonx.OrderedMap{}
	for _, r := range rows {
		m := care.ParseMedication(r)
		if m.Meta.Duration != care.DurationOngoing {
			continue
		}
		var dose any
		if sub := m.DisplaySubtitle(locale); sub.Valid && sub.String != "" {
			dose = sub.String
		}
		out = append(out, jsonx.Obj("title", r.Title, "dose", dose))
	}
	return out, nil
}

// pregnancy is {week} while a pregnancy is active (nil otherwise — an ended pregnancy shows nothing).
func (s *Service) pregnancy(ctx context.Context, userID uint64, today civildate.Date, locale string) (any, error) {
	p, err := pregnancy.LoadProfile(ctx, s.preg, userID)
	if err != nil {
		return nil, err
	}
	if p == nil || !p.PregnancyMode {
		return nil, nil
	}
	return jsonx.Obj("week", calc.New(p, locale, today).CurrentWeek()), nil
}

// OwnerJSON is the owner's GET /health-record/emergency-card body.
func (c Card) OwnerJSON() *jsonx.OrderedMap {
	var insurance any
	if c.Row.InsuranceLast4.Valid {
		insurance = jsonx.Obj("label", nullOrStr(c.Row.InsuranceLabel), "last4", c.Row.InsuranceLast4.String,
			"masked", MaskLast4(c.Row.InsuranceLast4.String))
	}
	src := c.Owner
	if src == nil {
		src = c.Public
	}
	card := jsonx.NewArray()
	for _, k := range src.Keys() {
		v, _ := src.Get(k)
		card.Set(k, v)
	}
	card.Set("insurance", insurance)
	var enabledAt, lastViewed any
	if c.Row.PublicEnabledAt.Valid {
		enabledAt = jsonx.ISO8601(c.Row.PublicEnabledAt.Time)
	}
	if c.Row.PublicLastViewedAt.Valid {
		lastViewed = jsonx.ISO8601(c.Row.PublicLastViewedAt.Time)
	}
	return jsonx.Obj(
		"card", card,
		"settings", jsonx.Obj("show_on_lock_screen", c.Row.ShowOnLockScreen, "show_pregnancy", c.Row.ShowPregnancy,
			"allergies_on_card", c.Allergies),
		"public_link", jsonx.Obj("enabled", c.Row.PublicTokenHash.Valid, "enabled_at", enabledAt,
			"views", c.Row.PublicViewCount, "last_viewed_at", lastViewed),
	)
}

// Contact is the emergency contact; Insurance the masked placeholder.
type (
	Contact   struct{ Name, Relation, Phone string }
	Insurance struct{ Label, Last4 string }
)

// Input is a validated PUT: only the Set* parts change.
type Input struct {
	SetLockScreen, LockScreen bool
	SetPregnancy, Pregnancy   bool
	SetContact                bool
	Contact                   *Contact // nil = clear
	SetInsurance              bool
	Insurance                 *Insurance // nil = clear
}

// Save applies in to the owner's card (created on first save).
func (s *Service) Save(ctx context.Context, userID uint64, in Input, now time.Time) error {
	if err := s.q.EnsureEmergencyCard(ctx, store.EnsureEmergencyCardParams{UserID: userID, Now: stamp(now)}); err != nil {
		return fmt.Errorf("emergency: ensure: %w", err)
	}
	row, err := s.settings(ctx, userID)
	if err != nil {
		return err
	}
	p := store.UpdateEmergencyCardParams{
		UserID: userID, ShowOnLockScreen: row.ShowOnLockScreen, ShowPregnancy: row.ShowPregnancy,
		ContactName: row.ContactName, ContactRelation: row.ContactRelation, ContactPhone: row.ContactPhone,
		InsuranceLabel: row.InsuranceLabel, InsuranceLast4: row.InsuranceLast4, Now: stamp(now),
	}
	if in.SetLockScreen {
		p.ShowOnLockScreen = in.LockScreen
	}
	if in.SetPregnancy {
		p.ShowPregnancy = in.Pregnancy
	}
	if in.SetContact {
		p.ContactName, p.ContactRelation, p.ContactPhone = sql.NullString{}, sql.NullString{}, sql.NullString{}
		if c := in.Contact; c != nil {
			p.ContactName, p.ContactRelation, p.ContactPhone = nstr(c.Name), nstr(c.Relation), nstr(c.Phone)
		}
	}
	if in.SetInsurance {
		p.InsuranceLabel, p.InsuranceLast4 = sql.NullString{}, sql.NullString{}
		if i := in.Insurance; i != nil {
			p.InsuranceLabel, p.InsuranceLast4 = nstr(i.Label), nstr(i.Last4)
		}
	}
	if err := s.q.UpdateEmergencyCard(ctx, p); err != nil {
		return fmt.Errorf("emergency: save: %w", err)
	}
	return nil
}

// Token format: 32 random bytes, base64url without padding (43 characters).
var reToken = regexp.MustCompile(`^[A-Za-z0-9_-]{43}$`)

func hashToken(token string) string {
	sum := sha256.Sum256([]byte(token))
	return hex.EncodeToString(sum[:])
}

// EnablePublic turns the public card on with a fresh token (an older token stops working) and returns the token —
// shown once; only its SHA-256 is stored.
func (s *Service) EnablePublic(ctx context.Context, userID uint64, now time.Time) (string, error) {
	b := make([]byte, 32)
	if _, err := io.ReadFull(s.rand, b); err != nil {
		return "", fmt.Errorf("emergency: token: %w", err)
	}
	token := base64.RawURLEncoding.EncodeToString(b)
	if err := s.q.EnsureEmergencyCard(ctx, store.EnsureEmergencyCardParams{UserID: userID, Now: stamp(now)}); err != nil {
		return "", fmt.Errorf("emergency: ensure: %w", err)
	}
	if err := s.q.SetEmergencyPublicToken(ctx, store.SetEmergencyPublicTokenParams{
		UserID: userID, TokenHash: sql.NullString{String: hashToken(token), Valid: true}, EnabledAt: stamp(now), Now: stamp(now),
	}); err != nil {
		return "", fmt.Errorf("emergency: enable: %w", err)
	}
	return token, nil
}

// DisablePublic turns the public card off (idempotent).
func (s *Service) DisablePublic(ctx context.Context, userID uint64, now time.Time) error {
	if err := s.q.SetEmergencyPublicToken(ctx, store.SetEmergencyPublicTokenParams{UserID: userID, Now: stamp(now)}); err != nil {
		return fmt.Errorf("emergency: disable: %w", err)
	}
	return nil
}

// Public is the minimal card behind a public token (ErrNotFound for an unknown, malformed or disabled token); the
// view is counted.
func (s *Service) Public(ctx context.Context, token string, today civildate.Date, now time.Time, locale, def string,
) (*jsonx.OrderedMap, error) {
	if !reToken.MatchString(token) {
		return nil, ErrNotFound
	}
	row, err := s.q.GetEmergencyCardByToken(ctx, sql.NullString{String: hashToken(token), Valid: true})
	if errors.Is(err, sql.ErrNoRows) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("emergency: lookup: %w", err)
	}
	card, err := s.Build(ctx, row.UserID, today, locale, def)
	if err != nil {
		return nil, err
	}
	if err := s.q.CountEmergencyCardView(ctx, store.CountEmergencyCardViewParams{ID: row.ID, Now: stamp(now)}); err != nil {
		return nil, fmt.Errorf("emergency: count view: %w", err)
	}
	return card.Public, nil
}
