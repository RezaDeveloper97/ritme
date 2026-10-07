package learning

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"strconv"
	"time"

	rootdb "github.com/ritme/backend-go/db"
	"github.com/ritme/backend-go/internal/learning/store"
	"github.com/ritme/backend-go/internal/platform/civildate"
)

// NotificationType is user_notifications.type of the «دوره برایت باز شد» inbox rows.
const NotificationType = "learning"

// UnlockedPath is the web-app screen a notice opens (Learn_Unlocked), followed by the grant id.
const UnlockedPath = "/learn/unlocked/"

// GrantInput is a validated POST /grants: normalised, de-duplicated phones, one target and a duration.
type GrantInput struct {
	Phones   []string
	Scope    string
	TargetID uint64
	Duration Duration
}

// GrantResult is one phone's outcome.
type GrantResult struct {
	Grant      store.LearningGrant
	Registered bool // the phone has an account: the grant is active now
	Renewed    bool // the instructor had already given this phone the same target
}

func nullDate(d Duration) civildate.NullDate {
	if d.Kind != DurationUntil || d.Until.IsZero() {
		return civildate.NullDate{}
	}
	return civildate.NullDate{Date: d.Until, Valid: true}
}

func durationOf(g store.LearningGrant) Duration {
	d := Duration{Kind: g.Duration}
	if g.DurationDays.Valid {
		d.Days = int(g.DurationDays.Int16)
	}
	if g.UntilDate.Valid {
		d.Until = g.UntilDate.Date
	}
	return d
}

// GrantState is a grant's status as shown: pending | active | expired | revoked.
func GrantState(status string, expires sql.NullTime, now time.Time) string {
	if status == GrantActive && Expired(timeOf(expires), now) {
		return GrantExpired
	}
	return status
}

// target checks that the group / course belongs to the instructor.
func (s *Service) target(ctx context.Context, q *store.Queries, instructorID uint64, scope string, id uint64) (sql.NullInt64, sql.NullInt64, error) {
	switch scope {
	case ScopeGroup:
		if _, err := q.GetGroup(ctx, store.GetGroupParams{ID: id, InstructorID: instructorID}); err != nil {
			if errors.Is(err, sql.ErrNoRows) {
				return sql.NullInt64{}, sql.NullInt64{}, ErrTargetInvalid
			}
			return sql.NullInt64{}, sql.NullInt64{}, fmt.Errorf("learning: grant group: %w", err)
		}
		return nid(id), sql.NullInt64{}, nil
	default:
		if _, err := q.GetInstructorCourse(ctx, store.GetInstructorCourseParams{ID: id, InstructorID: instructorID}); err != nil {
			if errors.Is(err, sql.ErrNoRows) {
				return sql.NullInt64{}, sql.NullInt64{}, ErrTargetInvalid
			}
			return sql.NullInt64{}, sql.NullInt64{}, fmt.Errorf("learning: grant course: %w", err)
		}
		return sql.NullInt64{}, nid(id), nil
	}
}

// CreateGrants opens the target to every phone: an existing account gets an active grant (inbox notice now), an
// unknown number a pending one that activates at signup. Re-adding a phone to the same target renews that grant
// (new duration from now). A new or re-opened grant queues the «دوره برایت باز شد» SMS.
func (s *Service) CreateGrants(ctx context.Context, ins store.LearningInstructor, in GrantInput, now time.Time) ([]GrantResult, error) {
	langs := s.langs(ctx)
	var out []GrantResult
	err := s.tx(ctx, func(q *store.Queries) error {
		out = out[:0]
		groupID, courseID, err := s.target(ctx, q, ins.ID, in.Scope, in.TargetID)
		if err != nil {
			return err
		}
		courses, err := q.ListGrantCourses(ctx, store.ListGrantCoursesParams{CourseID: courseID, GroupID: groupID})
		if err != nil {
			return fmt.Errorf("learning: grant courses: %w", err)
		}
		dur := in.Duration
		for _, phone := range in.Phones {
			res, err := s.grantOne(ctx, q, ins, phone, groupID, courseID, dur, now)
			if err != nil {
				return err
			}
			fresh := !res.Renewed || res.reopened
			if fresh && res.Registered && len(courses) > 0 {
				if err := s.notice(ctx, q, ins, res.Grant, courses, langs, now); err != nil {
					return err
				}
			}
			if fresh && len(courses) > 0 {
				if err := s.queueSMS(ctx, q, res.Grant.ID, now); err != nil {
					return err
				}
			}
			out = append(out, res.GrantResult)
		}
		return nil
	})
	return out, err
}

type grantOutcome struct {
	GrantResult
	reopened bool // the renewed grant was revoked, expired or pending before
}

func (s *Service) grantOne(ctx context.Context, q *store.Queries, ins store.LearningInstructor, phone string,
	groupID, courseID sql.NullInt64, dur Duration, now time.Time,
) (grantOutcome, error) {
	var user sql.NullInt64
	id, err := q.GetUserIDByMobile(ctx, sql.NullString{String: phone, Valid: true})
	switch {
	case err == nil:
		user = nid(id)
	case !errors.Is(err, sql.ErrNoRows):
		return grantOutcome{}, fmt.Errorf("learning: grant user: %w", err)
	}
	status, activated, expires := GrantPending, sql.NullTime{}, sql.NullTime{}
	if user.Valid {
		status, activated, expires = GrantActive, nt(now), nt(dur.ExpiresAt(now))
	}
	days := sql.NullInt16{}
	if dur.Kind == DurationDays {
		days = n16(dur.Days)
	}
	untilDate := nullDate(dur)
	cur, err := q.FindGrantForTarget(ctx, store.FindGrantForTargetParams{InstructorID: ins.ID, Phone: phone, GroupID: groupID, CourseID: courseID})
	switch {
	case err == nil:
		reopened := GrantState(cur.Status, cur.ExpiresAt, now) != GrantActive || (status == GrantActive && cur.Status == GrantPending)
		if err := q.RenewGrant(ctx, store.RenewGrantParams{
			UserID: user, Duration: dur.Kind, DurationDays: days, UntilDate: untilDate, Status: status,
			ActivatedAt: activated, ExpiresAt: expires, Now: nt(now), ID: cur.ID, InstructorID: ins.ID,
		}); err != nil {
			return grantOutcome{}, fmt.Errorf("learning: renew grant: %w", err)
		}
		g, err := q.GetInstructorGrant(ctx, store.GetInstructorGrantParams{ID: cur.ID, InstructorID: ins.ID})
		if err != nil {
			return grantOutcome{}, fmt.Errorf("learning: renew grant: %w", err)
		}
		return grantOutcome{GrantResult: GrantResult{Grant: g, Registered: user.Valid, Renewed: true}, reopened: reopened}, nil
	case !errors.Is(err, sql.ErrNoRows):
		return grantOutcome{}, fmt.Errorf("learning: find grant: %w", err)
	}
	gid, err := lastID(q.InsertGrant(ctx, store.InsertGrantParams{
		InstructorID: ins.ID, Phone: phone, UserID: user, GroupID: groupID, CourseID: courseID, Duration: dur.Kind,
		DurationDays: days, UntilDate: untilDate, Status: status, ActivatedAt: activated, ExpiresAt: expires, Now: nt(now),
	}))
	if err != nil {
		return grantOutcome{}, fmt.Errorf("learning: insert grant: %w", err)
	}
	g, err := q.GetInstructorGrant(ctx, store.GetInstructorGrantParams{ID: gid, InstructorID: ins.ID})
	if err != nil {
		return grantOutcome{}, fmt.Errorf("learning: insert grant: %w", err)
	}
	return grantOutcome{GrantResult: GrantResult{Grant: g, Registered: user.Valid}}, nil
}

// queueSMS adds the grant's SMS to the outbox (once while one is pending); the dispatcher applies the policy.
func (s *Service) queueSMS(ctx context.Context, q *store.Queries, grantID uint64, now time.Time) error {
	n, err := q.CountPendingSMSForGrant(ctx, grantID)
	if err != nil {
		return fmt.Errorf("learning: pending sms: %w", err)
	}
	if n > 0 {
		return nil
	}
	if err := q.InsertSMSOutbox(ctx, store.InsertSMSOutboxParams{GrantID: grantID, DueAt: nt(now).Time, Now: nt(now)}); err != nil {
		return fmt.Errorf("learning: queue sms: %w", err)
	}
	return nil
}

// notice writes the student's inbox row «… برایت باز کرد» (one entry per active language).
func (s *Service) notice(ctx context.Context, q *store.Queries, ins store.LearningInstructor, g store.LearningGrant,
	courses []store.LearningCourse, langs []string, now time.Time,
) error {
	userID := uid(g.UserID)
	if userID == 0 || len(courses) == 0 {
		return nil
	}
	title, body := map[string]string{}, map[string]string{}
	for _, code := range langs {
		name := ins.DisplayName
		if name == "" {
			name = T("notice.someone", code)
		}
		if len(courses) == 1 {
			title[code] = T("notice.title_one", code, "instructor", name, "course", courses[0].Title)
		} else {
			title[code] = T("notice.title_many", code, "instructor", name, "count", strconv.Itoa(len(courses)))
		}
		body[code] = T("notice.body", code)
	}
	rawTitle, err := json.Marshal(title)
	if err != nil {
		return fmt.Errorf("learning: notice title: %w", err)
	}
	rawBody, err := json.Marshal(body)
	if err != nil {
		return fmt.Errorf("learning: notice body: %w", err)
	}
	ids := make([]uint64, 0, len(courses))
	for _, c := range courses {
		ids = append(ids, c.ID)
	}
	data, err := json.Marshal(map[string]any{"event": "course_unlocked", "grant_id": g.ID, "course_ids": ids})
	if err != nil {
		return fmt.Errorf("learning: notice data: %w", err)
	}
	if err := q.InsertUserNotification(ctx, store.InsertUserNotificationParams{
		UserID: userID, Type: NotificationType, Title: rawTitle, Body: rootdb.NullRawJSON{V: rawBody, Valid: true},
		ActionUrl: sql.NullString{String: UnlockedPath + strconv.FormatUint(g.ID, 10), Valid: true},
		Data:      rootdb.NullRawJSON{V: data, Valid: true}, Now: nt(now),
	}); err != nil {
		return fmt.Errorf("learning: notice: %w", err)
	}
	return nil
}

// ClaimPending activates every pending grant of mobile for the new account userID (signup hook, and lazily on the
// student endpoints for accounts created by any other path). Each grant flips once even under concurrent claims.
func (s *Service) ClaimPending(ctx context.Context, userID uint64, mobile string, now time.Time) (int, error) {
	phone, ok := NormalizePhone(mobile)
	if !ok || userID == 0 {
		return 0, nil
	}
	pending, err := s.q.ListPendingGrantsByPhone(ctx, phone)
	if err != nil {
		return 0, fmt.Errorf("learning: pending grants: %w", err)
	}
	if len(pending) == 0 {
		return 0, nil
	}
	langs := s.langs(ctx)
	claimed := 0
	for _, g := range pending {
		n, err := s.q.ActivateGrant(ctx, store.ActivateGrantParams{
			UserID: nid(userID), ActivatedAt: nt(now), ExpiresAt: nt(durationOf(g).ExpiresAt(now)), ID: g.ID, Phone: phone,
		})
		if err != nil {
			return claimed, fmt.Errorf("learning: activate grant: %w", err)
		}
		if n == 0 {
			continue
		}
		claimed++
		g.UserID, g.Status = nid(userID), GrantActive
		if err := s.noticeFor(ctx, g, langs, now); err != nil {
			// The grant is active; a missing inbox row must not undo it.
			s.logger.ErrorContext(ctx, "learning: unlock notice failed", slog.Uint64("grant_id", g.ID), slog.String("error", err.Error()))
		}
	}
	return claimed, nil
}

func (s *Service) noticeFor(ctx context.Context, g store.LearningGrant, langs []string, now time.Time) error {
	ins, err := s.q.GetInstructor(ctx, g.InstructorID)
	if err != nil {
		return fmt.Errorf("learning: notice instructor: %w", err)
	}
	if ins.Status != InstructorApproved {
		return nil
	}
	courses, err := s.q.ListGrantCourses(ctx, store.ListGrantCoursesParams{CourseID: g.CourseID, GroupID: g.GroupID})
	if err != nil {
		return fmt.Errorf("learning: notice courses: %w", err)
	}
	return s.notice(ctx, s.q, ins, g, courses, langs, now)
}

// RevokeGrant ends one of the instructor's grants at once.
func (s *Service) RevokeGrant(ctx context.Context, instructorID, id uint64, now time.Time) error {
	n, err := s.q.RevokeGrant(ctx, store.RevokeGrantParams{Now: nt(now), ID: id, InstructorID: instructorID})
	if err != nil {
		return fmt.Errorf("learning: revoke grant: %w", err)
	}
	if n == 0 {
		return ErrGrantNotFound
	}
	return nil
}

// Grants lists the instructor's non-revoked grants (groupID 0 = all).
func (s *Service) Grants(ctx context.Context, instructorID, groupID uint64) ([]store.ListInstructorGrantsRow, error) {
	rows, err := s.q.ListInstructorGrants(ctx, store.ListInstructorGrantsParams{
		InstructorID: instructorID, GroupID: nid(groupID), Limit: MaxGrantsListed,
	})
	if err != nil {
		return nil, fmt.Errorf("learning: grants: %w", err)
	}
	return rows, nil
}

// MemberProgress is each student's completion percent over the given courses' published lessons.
func (s *Service) MemberProgress(ctx context.Context, userIDs, courseIDs []uint64) (map[uint64]int, error) {
	out := map[uint64]int{}
	if len(userIDs) == 0 || len(courseIDs) == 0 {
		return out, nil
	}
	lessons, err := s.q.ListPublishedLessonsForCourses(ctx, courseIDs)
	if err != nil {
		return nil, fmt.Errorf("learning: member lessons: %w", err)
	}
	if len(lessons) == 0 {
		return out, nil
	}
	published := make(map[uint64]bool, len(lessons))
	for _, l := range lessons {
		published[l.ID] = true
	}
	rows, err := s.q.ListProgressForUsersCourses(ctx, store.ListProgressForUsersCoursesParams{UserIds: userIDs, CourseIds: courseIDs})
	if err != nil {
		return nil, fmt.Errorf("learning: member progress: %w", err)
	}
	sum := map[uint64]int{}
	for _, r := range rows {
		if published[r.LessonID] {
			sum[r.UserID] += LessonPercent(int(r.Percent), r.CompletedAt.Valid)
		}
	}
	for _, u := range userIDs {
		out[u] = CoursePercent(sum[u], len(lessons))
	}
	return out, nil
}
