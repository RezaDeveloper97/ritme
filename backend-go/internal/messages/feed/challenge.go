package feed

import (
	"context"
	"database/sql"
	"encoding/json"
	"maps"
	"slices"
	"sync/atomic"
	"time"

	"github.com/ritme/backend-go/internal/messages/compiled"
	"github.com/ritme/backend-go/internal/messages/rules"
	"github.com/ritme/backend-go/internal/platform/civildate"
)

// The daily_challenge slot (T-M10-10, SPEC §5.5, D-167): "today's challenge" comes through the message engine.
//
//  1. An ongoing joined challenge (an active run inside its duration that is not contraindicated today) wins until it
//     is completed or abandoned (its run ended).
//  2. Otherwise the eligible, not-joined, active challenges are product candidates in admin sort order — the first one
//     wins, unless an admin rule of the slot that tests `challenge_id` promotes one: the slot is selected once per
//     candidate with the fact challenge_id = that candidate, and the highest-priority winning rule that reads
//     challenge_id names the challenge (ties: catalogue order). Its message is the pick's copy.
//  3. No pick → the legacy home pick (home.ChallengePayloadOr) fills the card.
//
// The slot is product-kind: everything in it is blocked for rules.PostSafetyBlock after a safety message was shown
// and while the engine needs the user's input (SPEC §5.4.1), ongoing challenges included — the slot stays empty then.
// PickChallenge is pure: the caller passes the candidates it already has (no query here).

// FactChallengeID is the fact a daily_challenge rule tests to promote a challenge.
const FactChallengeID = "challenge_id"

// SlotDailyChallenge is the slot name.
const SlotDailyChallenge = "daily_challenge"

// Pick reasons.
const (
	ChallengeOngoing  = "ongoing"
	ChallengePromoted = "promoted"
	ChallengeProduct  = "product"
)

// ChallengeCandidate is one challenge the user may see, in admin sort order.
type ChallengeCandidate struct {
	ID uint64
	// Ongoing is an active run of the user inside its duration (not completed, not ended).
	Ongoing bool
	// Contraindicated: the challenge's contraindications hold for the user today (never suggested, never continued).
	Contraindicated bool
	// Eligible: eligibility allows and the start rule allows joining today (product candidates only).
	Eligible bool
}

// ChallengeInput is one slot decision.
type ChallengeInput struct {
	Candidates []ChallengeCandidate
	Facts      rules.Facts
	// Blocked is ChallengeBlocked of the request.
	Blocked bool
	// Select runs the slot's selection for facts (nil = no admin rules); used for the challenge_id promotion.
	Select func(f rules.Facts) []rules.Choice
}

// ChallengePick is the slot's challenge.
type ChallengePick struct {
	ID     uint64
	Reason string
	// Choice is the promoting rule (Reason = promoted), nil otherwise.
	Choice *rules.Choice
}

// ChallengeBlocked reports whether the slot is blocked: a safety message shown less than PostSafetyBlock ago, or the
// engine needs the user's input.
func ChallengeBlocked(h *rules.UserHistory, f rules.Facts, now time.Time) bool {
	if v, _ := f["requires_user_input"].(bool); v {
		return true
	}
	return h != nil && !h.LastSafetyShown.IsZero() && now.Sub(h.LastSafetyShown) < rules.PostSafetyBlock
}

// TestsChallenge reports whether a rule's conditions or exceptions read challenge_id.
func TestsChallenge(r *rules.Rule) bool {
	if r == nil {
		return false
	}
	for _, g := range r.Groups {
		if slices.ContainsFunc(g.Items, func(c rules.Condition) bool { return c.Fact == FactChallengeID }) {
			return true
		}
	}
	return slices.ContainsFunc(r.Exceptions, func(c rules.Condition) bool { return c.Fact == FactChallengeID })
}

// PickChallenge decides the daily_challenge slot (ok = false: nothing, the legacy pick may fill the card).
func PickChallenge(in ChallengeInput) (ChallengePick, bool) {
	if in.Blocked {
		return ChallengePick{}, false
	}
	for _, c := range in.Candidates {
		if c.Ongoing && !c.Contraindicated {
			return ChallengePick{ID: c.ID, Reason: ChallengeOngoing}, true
		}
	}
	var product []ChallengeCandidate
	for _, c := range in.Candidates {
		if !c.Ongoing && c.Eligible && !c.Contraindicated {
			product = append(product, c)
		}
	}
	if len(product) == 0 {
		return ChallengePick{}, false
	}
	if in.Select != nil {
		var best *rules.Choice
		var bestID uint64
		for _, c := range product {
			f := maps.Clone(in.Facts)
			if f == nil {
				f = rules.Facts{}
			}
			f[FactChallengeID] = float64(c.ID)
			for _, ch := range in.Select(f) {
				if !TestsChallenge(ch.Rule) {
					continue
				}
				if best == nil || ch.Rule.Priority > best.Rule.Priority {
					chc := ch
					best, bestID = &chc, c.ID
				}
				break // the slot's top challenge rule for this candidate
			}
		}
		if best != nil {
			return ChallengePick{ID: bestID, Reason: ChallengePromoted, Choice: best}, true
		}
	}
	return ChallengePick{ID: product[0].ID, Reason: ChallengeProduct}, true
}

// ---------------------------------------------------------------------------
// The feed's daily_challenge slot (wired in Build).

// ChallengeRun is one of the user's challenge runs, read off the fact profile row (GetMessageFactProfile
// challenge_runs: no query of its own).
type ChallengeRun struct {
	ChallengeID uint64 `json:"id"`
	StartedOn   string `json:"started_on"` // Y-m-d
	Status      string `json:"status"`
}

// ParseChallengeRuns reads the challenge_runs column (a JSON list; NULL / invalid = none).
func ParseChallengeRuns(v any) []ChallengeRun {
	var b []byte
	switch t := v.(type) {
	case []byte:
		b = t
	case string:
		b = []byte(t)
	case sql.NullString:
		b = []byte(t.String)
	default:
		return nil
	}
	var runs []ChallengeRun
	if json.Unmarshal(b, &runs) != nil {
		return nil
	}
	return runs
}

// ChallengeCard is the slot's challenge as the feed renders it (catalogue data only, no query).
type ChallengeCard struct {
	ID           uint64
	Title        json.RawMessage // translatable
	Description  json.RawMessage // translatable or nil
	Category     string
	DurationDays int
	Day          int // 1-based day of an ongoing run (0 = not joined)
}

// ChallengeSource is the challenges catalogue the slot reads (challenges.Service). Every call is answered from the
// process catalogue snapshot and the runs it is given: no query on a warm feed.
type ChallengeSource interface {
	// NeedFacts are the facts the active challenges' eligibility and contraindications read.
	NeedFacts(ctx context.Context) ([]string, error)
	// FeedCandidates are the candidates in admin sort order, decided on f and runs.
	FeedCandidates(ctx context.Context, today civildate.Date, f rules.Facts, runs []ChallengeRun) ([]ChallengeCandidate, error)
	// FeedCard is the card of a picked challenge (ok = false: no longer in the catalogue).
	FeedCard(ctx context.Context, id uint64, today civildate.Date, runs []ChallengeRun) (ChallengeCard, bool, error)
}

var installedChallenges atomic.Pointer[challengeBox]

type challengeBox struct{ src ChallengeSource }

// InstallChallenges is the challenges catalogue every feed service reads (the route mount wires it once, as
// InstallInbox); nil uninstalls.
func InstallChallenges(src ChallengeSource) {
	if src == nil {
		installedChallenges.Store(nil)
		return
	}
	installedChallenges.Store(&challengeBox{src: src})
}

func challengeSource() ChallengeSource {
	if b := installedChallenges.Load(); b != nil {
		return b.src
	}
	return nil
}

// ChallengeResult is the slot's challenge of a built feed (nil Card = none).
type ChallengeResult struct {
	Card   *ChallengeCard
	Reason string
}

// challengeOn reports whether the request gets the challenge pick: the slot requested, a catalogue installed and the
// runs answered by the request (req.ChallengeRuns, off the fact profile row).
func challengeOn(req Request) (ChallengeSource, bool) {
	src := challengeSource()
	return src, src != nil && req.ChallengeRuns != nil && slices.Contains(req.Slots, SlotDailyChallenge)
}

// challengeNeed are the facts the challenge rules read (empty when the pick is off).
func challengeNeed(ctx context.Context, req Request) []string {
	src, on := challengeOn(req)
	if !on {
		return nil
	}
	need, err := src.NeedFacts(ctx)
	if err != nil {
		return nil
	}
	return need
}

// pickChallenge runs PickChallenge for the request. A promoting rule replaces the slot's winners (so its text is
// rendered and its shown is logged like any winner); otherwise the rule winners stay as they are.
func (s *Service) pickChallenge(ctx context.Context, req Request, comp *compiled.Compiled, f rules.Facts,
	h *rules.UserHistory,
) (*ChallengeResult, *rules.Choice, error) {
	src, on := challengeOn(req)
	if !on {
		return nil, nil, nil
	}
	runs, ok, err := req.ChallengeRuns(ctx)
	if err != nil || !ok {
		return nil, nil, err
	}
	today := civildate.FromTime(req.Now)
	cands, err := src.FeedCandidates(ctx, today, f, runs)
	if err != nil {
		return nil, nil, err
	}
	pick, ok := PickChallenge(ChallengeInput{Candidates: cands, Facts: f, Blocked: ChallengeBlocked(h, f, req.Now),
		Select: func(ff rules.Facts) []rules.Choice {
			return comp.Select(SlotDailyChallenge, ff, h, req.UserID, req.Now)
		}})
	res := &ChallengeResult{}
	if !ok {
		return res, nil, nil
	}
	card, found, err := src.FeedCard(ctx, pick.ID, today, runs)
	if err != nil || !found {
		return res, nil, err
	}
	res.Card, res.Reason = &card, pick.Reason
	return res, pick.Choice, nil
}
