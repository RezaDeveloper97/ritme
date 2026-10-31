package content

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"slices"
	"strconv"
	"strings"

	"github.com/gofiber/fiber/v3"

	"github.com/ritme/backend-go/db"
	adminch "github.com/ritme/backend-go/internal/admin/challenges"
	"github.com/ritme/backend-go/internal/admin/content/form"
	"github.com/ritme/backend-go/internal/admin/httpadmin"
	"github.com/ritme/backend-go/internal/admin/store"
	"github.com/ritme/backend-go/internal/challenges"
	chstore "github.com/ritme/backend-go/internal/challenges/store"
	"github.com/ritme/backend-go/internal/enums"
	"github.com/ritme/backend-go/internal/platform/civildate"
	"github.com/ritme/backend-go/internal/platform/jsonx"
	"github.com/ritme/backend-go/internal/platform/validation"
	"github.com/ritme/backend-go/internal/platform/validation/phpval"
)

// Challenges v2 (bloom B-N9-07, D-87): a challenge with a program row is a multi-day run — duration, start rule,
// reminder time, eligibility (life mode / cycle phase include + exclude), per-day tasks, reward (badge + Plus days)
// and a completion rule. The legacy fields and endpoints are unchanged; every v2 field is additive and optional.

const (
	maxChallengeDays  = 366
	maxChallengeTasks = 100
)

// challengeV2Rules are the rules of the v2 fields (all optional; `duration_days` switches the program on).
func challengeV2Rules(c fiber.Ctx) validation.Rules {
	rules := validation.Rules{
		validation.F("duration_days", "nullable|integer|min:1|max:"+strconv.Itoa(maxChallengeDays)),
		validation.F("starts_on", "nullable|date_format:Y-m-d"),
		validation.F("reminder_time", "nullable|date_format:H:i"),
		validation.F("reward_badge_code", "nullable|string|max:64"),
		validation.F("reward_plus_days", "nullable|integer|min:0|max:365"),
		validation.F("completion_min_days", "nullable|integer|min:1|max:"+strconv.Itoa(maxChallengeDays)),
		validation.F("tasks", "nullable|array|max:"+strconv.Itoa(maxChallengeTasks)),
		validation.F("tasks.*.id", "nullable|integer|min:1"),
		validation.F("tasks.*.day_from", "required|integer|min:1|max:"+strconv.Itoa(maxChallengeDays)),
		validation.F("tasks.*.day_to", "required|integer|min:1|max:"+strconv.Itoa(maxChallengeDays)),
		validation.F("tasks.*.title", "required|array"),
		validation.F("tasks.*.sort_order", "nullable|integer"),
	}
	for _, f := range []struct {
		name   string
		values []string
	}{
		{"modes_include", enums.LifeModeValues()}, {"modes_exclude", enums.LifeModeValues()},
		{"phases_include", enums.MainPhaseValues()}, {"phases_exclude", enums.MainPhaseValues()},
	} {
		rules = append(rules, validation.F(f.name, "nullable|array"), validation.F(f.name+".*", validation.In(f.values...)))
	}
	rules = append(rules, adminch.Rules()...) // T-M10-10: start_rule / start_phase / eligibility / contraindications
	return append(rules, form.Translatable(c, "reward_badge_title", false)...)
}

// checkChallengeV2 are the cross-field rules: tasks fit the duration, a program needs tasks, the minimum fits too.
func checkChallengeV2(c fiber.Ctx) form.Check {
	v6 := adminch.Check(c) // T-M10-10: the rule objects and start_phase
	return func(in phpval.Map, add form.Add) error {
		if err := v6(in, add); err != nil {
			return err
		}
		dv, _ := in.Get("duration_days")
		if dv == nil || !phpval.IsNumeric(dv) {
			return nil
		}
		duration := int(phpval.ToFloat(dv))
		_, tasks := phpval.Entries(mustGet(in, "tasks"))
		if len(tasks) == 0 {
			add("tasks", form.Msg(c, "validation.required", "tasks"))
		}
		for i, t := range tasks {
			m, ok := t.(phpval.Map)
			if !ok {
				continue
			}
			from, _ := m.Get("day_from")
			to, _ := m.Get("day_to")
			field := fmt.Sprintf("tasks.%d.day_to", i)
			if phpval.IsNumeric(from) && phpval.IsNumeric(to) && phpval.ToFloat(to) < phpval.ToFloat(from) {
				add(field, form.Msg(c, "validation.gte.numeric", field, "value", phpval.ToString(from)))
			} else if phpval.IsNumeric(to) && int(phpval.ToFloat(to)) > duration {
				add(field, form.Msg(c, "validation.lte.numeric", field, "value", strconv.Itoa(duration)))
			}
			title, _ := m.Get("title")
			if _, vals := phpval.Entries(title); !slices.ContainsFunc(vals, func(v any) bool {
				s, isStr := v.(string)
				return isStr && strings.TrimSpace(s) != ""
			}) {
				add(fmt.Sprintf("tasks.%d.title", i), form.Msg(c, "validation.required", fmt.Sprintf("tasks.%d.title", i)))
			}
		}
		if mv, _ := in.Get("completion_min_days"); mv != nil && phpval.IsNumeric(mv) && int(phpval.ToFloat(mv)) > duration {
			add("completion_min_days", form.Msg(c, "validation.lte.numeric", "completion_min_days", "value", strconv.Itoa(duration)))
		}
		if rule, _ := in.Get("start_rule"); rule == StartFixedDate {
			if s, _ := in.Get("starts_on"); s == nil || s == "" {
				add("starts_on", form.Msg(c, "validation.required", "starts_on"))
			}
		}
		return nil
	}
}

// StartFixedDate is the start_rule that needs starts_on.
const StartFixedDate = "fixed_date"

func mustGet(in phpval.Map, key string) any {
	v, _ := in.Get(key)
	return v
}

func stringList(v any) []string {
	_, vals := phpval.Entries(v)
	out := make([]string, 0, len(vals))
	for _, x := range vals {
		if s := phpval.ToString(x); !slices.Contains(out, s) {
			out = append(out, s)
		}
	}
	return out
}

// saveChallengeProgram writes the program and the tasks of a validated request in one transaction (nothing when the
// request carries no `duration_days`: a legacy challenge stays legacy, and an edit that leaves it out keeps the run).
func (h *Handlers) saveChallengeProgram(ctx context.Context, id uint64, data phpval.Map, now sql.NullTime) error {
	dv, _ := data.Get("duration_days")
	if dv == nil {
		return nil
	}
	elig := jsonx.Obj(
		"modes_include", stringList(mustGet(data, "modes_include")), "modes_exclude", stringList(mustGet(data, "modes_exclude")),
		"phases_include", stringList(mustGet(data, "phases_include")), "phases_exclude", stringList(mustGet(data, "phases_exclude")),
	)
	tx, err := h.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback() }()
	q := chstore.New(tx)

	startRule := httpadmin.String(data, "start_rule")
	if startRule == "" {
		startRule = "anytime"
	}
	var startsOn civildateNull
	if s := httpadmin.String(data, "starts_on"); s != "" && startRule == StartFixedDate {
		d, err := civildate.Parse(s)
		if err != nil {
			return err
		}
		startsOn = civildateNull{Date: d, Valid: true}
	}
	if err := q.UpsertProgram(ctx, chstore.UpsertProgramParams{
		ChallengeID: id, DurationDays: uint16(form.Int(data, "duration_days", 1)), StartRule: startRule, //nolint:gosec // G115: 1…366
		StartsOn: startsOn, ReminderTime: form.Str(data, "reminder_time"), Eligibility: db.NullRawJSON{V: form.JSON(elig), Valid: true},
		RewardBadgeCode: form.Str(data, "reward_badge_code"), RewardBadgeTitle: form.Clean(data, "reward_badge_title"),
		RewardPlusDays: uint16(form.Int(data, "reward_plus_days", 0)), CompletionMinDays: form.NullInt16(data, "completion_min_days"), //nolint:gosec // G115: ≤ 365
		Now: now,
	}); err != nil {
		return err
	}
	// T-M10-10 (D-167): the v6 rule columns — eligibility in the {all_of, any_of, exclude} shape (the legacy lists
	// converted), contraindications (kept when the request has none), start rule + phase.
	if err := adminch.Save(ctx, q, id, data, now); err != nil {
		return err
	}
	if _, has := data.Get("tasks"); has {
		_, tasks := phpval.Entries(mustGet(data, "tasks"))
		existing, err := q.ListTasks(ctx, id)
		if err != nil {
			return err
		}
		known := map[uint64]bool{}
		for _, t := range existing {
			known[t.ID] = true
		}
		keep := []uint64{0}
		for i, raw := range tasks {
			m, ok := raw.(phpval.Map)
			if !ok {
				continue
			}
			title := form.ReqJSON(m, "title")
			from, to := uint16(form.Int(m, "day_from", 1)), uint16(form.Int(m, "day_to", 1)) //nolint:gosec // G115: 1…366
			order := int32(form.Int(m, "sort_order", int64(i)))                              //nolint:gosec // G115: bounded
			if tid := uint64(form.Int(m, "id", 0)); tid > 0 && known[tid] {                  //nolint:gosec // G115: ≥ 1
				if err := q.UpdateTask(ctx, chstore.UpdateTaskParams{DayFrom: from, DayTo: to, Title: title, SortOrder: order,
					UpdatedAt: now, ID: tid, ChallengeID: id}); err != nil {
					return err
				}
				keep = append(keep, tid)
				continue
			}
			res, err := tx.ExecContext(ctx, "INSERT INTO `challenge_tasks` (challenge_id, day_from, day_to, title, sort_order, created_at, updated_at) VALUES (?, ?, ?, ?, ?, ?, ?)",
				id, from, to, title, order, now, now)
			if err != nil {
				return err
			}
			nid, _ := res.LastInsertId()
			keep = append(keep, uint64(nid)) //nolint:gosec // G115: auto-increment
		}
		if err := q.DeleteTasksExcept(ctx, chstore.DeleteTasksExceptParams{ChallengeID: id, KeepIds: keep}); err != nil {
			return err
		}
	}
	return tx.Commit()
}

type civildateNull = civildate.NullDate

func rawOrNil(raw db.NullRawJSON) any {
	if !raw.Valid {
		return nil
	}
	return form.Raw(raw.V)
}

// challengeV2JSON is the additive v2 part of a challenge: program (null for a legacy challenge), tasks and stats.
func (h *Handlers) challengeV2JSON(ctx context.Context, id uint64) (program, tasks, stats any, err error) {
	prog, err := h.cq.GetProgram(ctx, id)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, jsonx.List([]*jsonx.OrderedMap{}), nil, nil
	}
	if err != nil {
		return nil, nil, nil, err
	}
	rows, err := h.cq.ListTasks(ctx, id)
	if err != nil {
		return nil, nil, nil, err
	}
	list := make([]*jsonx.OrderedMap, 0, len(rows))
	for _, t := range rows {
		list = append(list, jsonx.Obj("id", t.ID, "day_from", t.DayFrom, "day_to", t.DayTo, "title", form.Raw(t.Title), "sort_order", t.SortOrder))
	}
	st, err := h.cq.ProgramStats(ctx, []uint64{id})
	if err != nil {
		return nil, nil, nil, err
	}
	var participants, completed int64
	if len(st) > 0 {
		participants, completed = st[0].Participants, st[0].Completed
	}
	return programJSON(prog), jsonx.List(list), statsJSON(participants, completed), nil
}

func programJSON(p chstore.ChallengeProgram) *jsonx.OrderedMap {
	// The legacy lists the current editor reads, from either stored shape (empty when the rule needs the v6 shape:
	// the editor then reads `eligibility`, T-M10-10).
	var e struct {
		ModesInclude, ModesExclude, PhasesInclude, PhasesExclude []string
	}
	e.ModesInclude, e.ModesExclude, e.PhasesInclude, e.PhasesExclude, _ = challenges.ToLegacy(challenges.ParseRule(p.Eligibility))
	nz := func(s []string) []string {
		if s == nil {
			return []string{}
		}
		return s
	}
	var startsOn any
	if p.StartsOn.Valid {
		startsOn = p.StartsOn.Date.String()
	}
	var minDays any
	if p.CompletionMinDays.Valid {
		minDays = p.CompletionMinDays.Int16
	}
	m := jsonx.Obj(
		"duration_days", p.DurationDays, "start_rule", p.StartRule, "starts_on", startsOn,
		"reminder_time", httpadmin.NullString(p.ReminderTime),
		"modes_include", nz(e.ModesInclude), "modes_exclude", nz(e.ModesExclude),
		"phases_include", nz(e.PhasesInclude), "phases_exclude", nz(e.PhasesExclude),
		"reward_badge_code", httpadmin.NullString(p.RewardBadgeCode), "reward_badge_title", rawOrNil(p.RewardBadgeTitle),
		"reward_plus_days", p.RewardPlusDays, "completion_min_days", minDays,
	)
	adminch.SetProgramJSON(m, p)
	return m
}

func statsJSON(participants, completed int64) *jsonx.OrderedMap {
	rate := 0
	if participants > 0 {
		rate = int((completed*100 + participants/2) / participants)
	}
	return jsonx.Obj("participants", participants, "completed", completed, "completion_rate", rate)
}

// fullChallengeJSON is challengeJSON plus the v2 keys (program, tasks, stats).
func (h *Handlers) fullChallengeJSON(ctx context.Context, ch *store.Challenge) (*jsonx.OrderedMap, error) {
	m := challengeJSON(ch)
	program, tasks, stats, err := h.challengeV2JSON(ctx, ch.ID)
	if err != nil {
		return nil, err
	}
	m.Set("program", program)
	m.Set("tasks", tasks)
	m.Set("stats", stats)
	return m, nil
}

// respondChallenge reloads the row and answers {challenge} (200 or 201) with the v2 keys.
func (h *Handlers) respondChallenge(c fiber.Ctx, id uint64, created bool, msg string) error {
	row, err := h.q.GetChallenge(c.Context(), id)
	if err != nil {
		return err
	}
	m, err := h.fullChallengeJSON(c.Context(), &row)
	if err != nil {
		return err
	}
	body := jsonx.Obj("challenge", m)
	if created {
		return httpadmin.Created(c, body, msg)
	}
	return httpadmin.OK(c, body, msg)
}

// ChallengeStats is GET /challenges/:id/stats: participants, completion, rewards granted and the users active per day.
func (h *Handlers) ChallengeStats(c fiber.Ctx) error {
	ch, err := find(c, "Challenge", h.q.GetChallenge)
	if err != nil {
		return err
	}
	ctx := c.Context()
	st, err := h.cq.ProgramStats(ctx, []uint64{ch.ID})
	if err != nil {
		return err
	}
	var participants, completed int64
	if len(st) > 0 {
		participants, completed = st[0].Participants, st[0].Completed
	}
	rew, err := h.cq.ProgramRewardStats(ctx, ch.ID)
	if err != nil {
		return err
	}
	days, err := h.cq.ProgramDayStats(ctx, ch.ID)
	if err != nil {
		return err
	}
	perDay := make([]*jsonx.OrderedMap, 0, len(days))
	for _, d := range days {
		perDay = append(perDay, jsonx.Obj("day", d.DayNumber, "users", d.Users))
	}
	prog, err := h.cq.GetProgram(ctx, ch.ID)
	duration := 0
	switch {
	case err == nil:
		duration = int(prog.DurationDays)
	case !errors.Is(err, sql.ErrNoRows):
		return err
	}
	dropOff, err := adminch.DropOff(ctx, h.cq, ch.ID, duration, civildate.InTehran(httpadmin.Now(c)))
	if err != nil {
		return err
	}
	s := statsJSON(participants, completed)
	s.Set("active", participants-completed)
	s.Set("rewards_granted", rew.Rewards)
	s.Set("plus_days_granted", rew.PlusDays)
	s.Set("per_day", jsonx.List(perDay))
	s.Set("drop_off", jsonx.List(dropOff)) // T-M10-10: share of the runs that reached day n still going / dropped on it
	return httpadmin.OK(c, jsonx.Obj("challenge_id", ch.ID, "stats", s))
}

// challengeListExtras adds program + stats to the list rows (one query each for the whole page).
func (h *Handlers) challengeListExtras(ctx context.Context, rows []store.Challenge, items []*jsonx.OrderedMap) error {
	if len(rows) == 0 {
		return nil
	}
	ids := make([]uint64, len(rows))
	for i := range rows {
		ids[i] = rows[i].ID
	}
	progs, err := h.cq.ListPrograms(ctx, ids)
	if err != nil {
		return err
	}
	stats, err := h.cq.ProgramStats(ctx, ids)
	if err != nil {
		return err
	}
	pm := map[uint64]chstore.ChallengeProgram{}
	for _, p := range progs {
		pm[p.ChallengeID] = p
	}
	sm := map[uint64]chstore.ProgramStatsRow{}
	for _, s := range stats {
		sm[s.ChallengeID] = s
	}
	for i := range rows {
		if p, ok := pm[rows[i].ID]; ok {
			items[i].Set("program", programJSON(p))
			items[i].Set("stats", statsJSON(sm[rows[i].ID].Participants, sm[rows[i].ID].Completed))
		} else {
			items[i].Set("program", nil)
			items[i].Set("stats", nil)
		}
	}
	return nil
}
