package extract

import (
	"context"
	"database/sql"
	"encoding/json"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/ritme/backend-go/db"
	"github.com/ritme/backend-go/internal/ai"
	"github.com/ritme/backend-go/internal/healthrecord"
	"github.com/ritme/backend-go/internal/healthrecord/extract/store"
	"github.com/ritme/backend-go/internal/i18n/lang"
	"github.com/ritme/backend-go/internal/platform/civildate"
	"github.com/ritme/backend-go/internal/platform/config"
	pregstore "github.com/ritme/backend-go/internal/pregnancy/store"
)

func pdf(key string) []byte { return []byte("%PDF-1.4\n% " + ai.FakeMarker + key + "\n%%EOF") }

// Every kind's schema is accepted by the platform (no identity key) and reads its own fixture from the fake.
func TestSchemas_FakeFixturePerKind(t *testing.T) {
	fake := ai.NewFake()
	c := ai.NewClientWith(ai.Options{Provider: config.AIProviderFake, Extractor: fake})
	want := map[string][]string{
		healthrecord.KindImaging:      {"kind", "date", "centre", "doctor", "ga_weeks", "ga_days", "edd", "findings"},
		healthrecord.KindVisit:        {"date", "centre", "doctor", "specialty", "reason", "diagnosis", "findings", "next_visit"},
		healthrecord.KindPrescription: {"date", "centre", "doctor", "diagnosis"},
		healthrecord.KindHospital:     {"date", "ended_on", "centre", "doctor", "reason", "diagnosis", "procedures", "findings"},
		healthrecord.KindOther:        {"date", "centre", "title", "findings"},
	}
	require.Len(t, Schemas, len(healthrecord.DocumentKinds))
	for _, kind := range healthrecord.DocumentKinds {
		doc := ai.Document{Data: []byte("%PDF-1.4 plain scan\n%%EOF"), MIME: "application/pdf"}
		out, err := c.Extract(context.Background(), ai.FeatureDocExtract, ai.ExtractRequest{Document: doc, Schema: Schemas[kind], Hint: hint(kind)})
		require.NoError(t, err, kind)
		var keys []string
		for _, f := range out.Fields {
			keys = append(keys, f.Key)
		}
		assert.Equal(t, want[kind], keys, kind)
		if kind == healthrecord.KindPrescription {
			require.Len(t, out.Items, 2)
			assert.Equal(t, "Ferrous sulfate 50mg", out.Items[0][0].Value)
		} else {
			assert.Empty(t, out.Items, kind)
		}
	}
	// A marker picks another fixture (no gestational age on a mammography).
	out, err := c.Extract(context.Background(), ai.FeatureDocExtract, ai.ExtractRequest{
		Document: ai.Document{Data: pdf("imaging_plain"), MIME: "application/pdf"}, Schema: Schemas[healthrecord.KindImaging]})
	require.NoError(t, err)
	for _, f := range out.Fields {
		assert.NotContains(t, []string{"ga_weeks", "ga_days", "edd"}, f.Key)
	}
}

func TestLangFiles(t *testing.T) {
	for _, loc := range []string{"fa", "en"} {
		for _, k := range []string{"messages.extraction_started", "messages.dating_unavailable", "offer.offered", "offer.same_dating", "validation.no_files"} {
			assert.NotEqual(t, "extract."+k, T(k, loc), loc+" "+k)
		}
		assert.NotEmpty(t, attributes(loc))
	}
	assert.Equal(t, T("messages.review_confirmed", lang.FallbackLocale), T("messages.review_confirmed", "xx"))
}

func storedRaw(t *testing.T, s *Stored) db.NullRawJSON {
	t.Helper()
	raw, err := s.raw()
	require.NoError(t, err)
	return raw
}

func imagingDoc(t *testing.T, state string, fields map[string]any, reviewed map[string]any, dating *string) store.RecordDocument {
	t.Helper()
	st := &Stored{Schema: healthrecord.KindImaging, Status: StatusDone, Fields: map[string]Value{}, Reviewed: reviewed, Dating: dating}
	for k, v := range fields {
		st.Fields[k] = Value{Value: v, Confidence: 0.9}
	}
	return store.RecordDocument{ID: 7, UserID: 3, Kind: healthrecord.KindImaging, ReviewState: state, Extracted: storedRaw(t, st)}
}

func TestOfferFor(t *testing.T) {
	today := civildate.MustParse("2026-09-23")
	scan := map[string]any{"date": "2026-09-18", "ga_weeks": 12.0, "ga_days": 3.0}
	lmp := &pregstore.PregnancyProfile{ID: 11, PregnancyMode: true, AgeSource: sql.NullString{String: "lmp", Valid: true},
		LmpDate: civildate.NullDate{Date: civildate.MustParse("2026-06-25"), Valid: true}}

	o := offerFor(imagingDoc(t, healthrecord.ReviewConfirmed, scan, nil, nil), lmp, today)
	assert.Equal(t, OfferOffered, o.State, o.Reason)
	assert.Equal(t, 87, o.AtScan)
	assert.Equal(t, "2027-03-30", o.Due.String()) // 2026-09-18 + 280 − 87
	assert.Equal(t, "2027-04-01", o.CurrentDue.String())
	assert.Equal(t, uint64(11), o.ProfileID)

	cases := []struct {
		name   string
		doc    store.RecordDocument
		p      *pregstore.PregnancyProfile
		reason string
	}{
		{"needs review", imagingDoc(t, healthrecord.ReviewNeedsReview, scan, nil, nil), lmp, ReasonNotConfirmed},
		{"no dating", imagingDoc(t, healthrecord.ReviewConfirmed, map[string]any{"date": "2026-09-18"}, nil, nil), lmp, ReasonNoDating},
		{"future scan", imagingDoc(t, healthrecord.ReviewConfirmed, map[string]any{"date": "2026-10-18", "ga_weeks": 12.0}, nil, nil), lmp, ReasonOutOfRange},
		{"huge ga (no int overflow)", imagingDoc(t, healthrecord.ReviewConfirmed, map[string]any{"date": "2026-09-18", "ga_weeks": 1e300, "ga_days": -1e300}, nil, nil), lmp, ReasonOutOfRange},
		{"ga too small", imagingDoc(t, healthrecord.ReviewConfirmed, map[string]any{"date": "2026-09-18", "ga_weeks": 2.0}, nil, nil), lmp, ReasonOutOfRange},
		{"past pregnancy", imagingDoc(t, healthrecord.ReviewConfirmed, map[string]any{"date": "2024-01-10", "ga_weeks": 20.0}, nil, nil), lmp, ReasonOutOfRange},
		{"not pregnant", imagingDoc(t, healthrecord.ReviewConfirmed, scan, nil, nil), nil, ReasonNotPregnant},
		{"other pregnancy", imagingDoc(t, healthrecord.ReviewConfirmed, map[string]any{"date": "2026-09-18", "ga_weeks": 30.0}, nil, nil), lmp, ReasonOtherPregnancy},
		{"user cleared the GA", imagingDoc(t, healthrecord.ReviewConfirmed, scan, map[string]any{"date": "2026-09-18"}, nil), lmp, ReasonNoDating},
	}
	for _, c := range cases {
		o := offerFor(c.doc, c.p, today)
		assert.Equal(t, OfferUnavailable, o.State, c.name)
		assert.Equal(t, c.reason, o.Reason, c.name)
	}

	visit := imagingDoc(t, healthrecord.ReviewConfirmed, scan, nil, nil)
	visit.Kind = healthrecord.KindVisit
	assert.Equal(t, ReasonNotImaging, offerFor(visit, lmp, today).Reason)

	// Only a printed due date: the GA at the scan comes from it.
	o = offerFor(imagingDoc(t, healthrecord.ReviewConfirmed, map[string]any{"date": "2026-09-18", "edd": "2027-03-30"}, nil, nil), lmp, today)
	assert.Equal(t, OfferOffered, o.State)
	assert.Equal(t, 87, o.AtScan)

	// The user's corrected values win over what was read.
	o = offerFor(imagingDoc(t, healthrecord.ReviewConfirmed, scan, map[string]any{"date": "2026-09-18", "ga_weeks": 13.0, "ga_days": 0.0}, nil), lmp, today)
	assert.Equal(t, 91, o.AtScan)

	applied := DatingApplied
	assert.Equal(t, OfferApplied, offerFor(imagingDoc(t, healthrecord.ReviewConfirmed, scan, nil, &applied), nil, today).State)

	us := *lmp
	us.AgeSource = sql.NullString{String: "ultrasound", Valid: true}
	us.UltrasoundDate = civildate.NullDate{Date: civildate.MustParse("2026-09-18"), Valid: true}
	us.UltrasoundWeeks, us.UltrasoundDays = sql.NullInt32{Int32: 12, Valid: true}, sql.NullInt32{Int32: 3, Valid: true}
	assert.Equal(t, ReasonSameDating, offerFor(imagingDoc(t, healthrecord.ReviewConfirmed, scan, nil, nil), &us, today).Reason)
}

func TestOfferJSON(t *testing.T) {
	today := civildate.MustParse("2026-09-23")
	o := Offer{State: OfferOffered, ScanDate: civildate.MustParse("2026-09-18"), AtScan: 87, Due: civildate.MustParse("2027-03-30"),
		Today: today, CurrentSource: "lmp", CurrentDue: civildate.MustParse("2027-04-01")}
	b, err := json.Marshal(offerJSON(o, "en"))
	require.NoError(t, err)
	assert.JSONEq(t, `{"state":"offered","reason":null,
		"message":"This scan says 12 weeks 3 days on 2026-09-18. Update your pregnancy dating from it?",
		"scan_date":"2026-09-18","proposed":{"ga_weeks":12,"ga_days":3,"due_date":"2027-03-30","weeks_today":13,"days_today":1},
		"current":{"source":"lmp","due_date":"2027-04-01"},"difference_days":-2}`, string(b))
	b, err = json.Marshal(offerJSON(Offer{State: OfferUnavailable, Reason: ReasonNotImaging, Today: today}, "en"))
	require.NoError(t, err)
	assert.JSONEq(t, `{"state":"unavailable","reason":"not_imaging","message":"Only an imaging report can update the pregnancy dating",
		"scan_date":null,"proposed":null,"current":null,"difference_days":null}`, string(b))
}

func TestTyped(t *testing.T) {
	now := time.Date(2026, 9, 23, 10, 0, 0, 0, civildate.Tehran)
	assert.InDelta(t, 12.0, typed(ai.FieldSpec{Type: ai.FieldNumber}, "12", now), 0)
	assert.Equal(t, "2026-09-18", typed(ai.FieldSpec{Type: ai.FieldDate}, "2026-09-18", now))
	assert.Equal(t, "Dr. A", typed(ai.FieldSpec{Type: ai.FieldString}, "  Dr. A ", now))
	assert.Nil(t, typed(ai.FieldSpec{Type: ai.FieldString}, " ", now))
	assert.Nil(t, typed(ai.FieldSpec{Type: ai.FieldNumber}, nil, now))
}

func TestStoredJobIsHiddenFromTheDocumentView(t *testing.T) {
	st := &Stored{Schema: "visit", Status: StatusPending, RequestedAt: "2026-09-23T10:00:00+03:30",
		Job: &Job{State: jobQueued, Locale: "fa", ReservedAt: "2026-09-23T10:00:00+03:30"}}
	raw := storedRaw(t, st)
	assert.Contains(t, string(raw.V), `"job"`)
	back := parseStored(raw)
	require.NotNil(t, back)
	at, ok := back.reservedAt()
	assert.True(t, ok)
	assert.Equal(t, "2026-09-23T10:00:00+03:30", isoTime(at))
	assert.Nil(t, parseStored(db.NullRawJSON{}))
}
