// Package learning is the courses domain (bloom B-N8-01, D-68): instructors (admin-approved) publish courses
// (chapters → lessons: video / audio / pdf) and standalone content, put students in groups and open courses to them
// by mobile number. A grant for a number without an account stays pending until that number signs up with the OTP
// (SignupHook), then it is active for unlimited / 30 / 90 days or until a date. Students keep per-lesson progress.
//
// Two separate HTTP surfaces:
//   - /api/v1/learning/*        the student (auth:api): my courses, a course, a lesson, progress, an unlocked grant;
//   - /api/instructor/v1/*      the instructor (auth:api + approved instructor): courses, chapters, lessons, groups,
//     grants (instructor-web, B-N8-05..07).
//
// Every instructor query is scoped by the instructor id, every student query by the user id; a foreign id answers
// the same 404 as a missing one. Phone numbers are stored normalised (09xxxxxxxxx), shown only to the instructor who
// typed them and never logged in full. Media references (media_id / media_status) are placeholders for the B-N8-02
// media pipeline.
package learning

import (
	"math"
	"time"

	"github.com/ritme/backend-go/internal/companion"
	"github.com/ritme/backend-go/internal/platform/civildate"
)

// Instructor statuses.
const (
	InstructorPending  = "pending"
	InstructorApproved = "approved"
	InstructorRevoked  = "revoked"
)

// Course kinds.
const (
	KindCourse     = "course"
	KindStandalone = "standalone"
)

// CourseKinds are the accepted course kinds.
var CourseKinds = []string{KindCourse, KindStandalone}

// Lesson kinds.
const (
	LessonVideo = "video"
	LessonAudio = "audio"
	LessonPDF   = "pdf"
)

// LessonKinds are the accepted lesson kinds.
var LessonKinds = []string{LessonVideo, LessonAudio, LessonPDF}

// Publication statuses (courses and lessons).
const (
	StatusDraft     = "draft"
	StatusPublished = "published"
)

// Statuses are the accepted publication statuses.
var Statuses = []string{StatusDraft, StatusPublished}

// Grant statuses (stored) and the derived «expired» of an active grant past expires_at.
const (
	GrantPending = "pending"
	GrantActive  = "active"
	GrantRevoked = "revoked"
	GrantExpired = "expired"
)

// Grant scopes.
const (
	ScopeGroup  = "group"
	ScopeCourse = "course"
)

// Scopes are the accepted grant scopes.
var Scopes = []string{ScopeGroup, ScopeCourse}

// Stored grant durations.
const (
	DurationUnlimited = "unlimited"
	DurationDays      = "days"
	DurationUntil     = "until"
)

// DurationChoices are the request values of a grant duration (Ins_AddUser: نامحدود / ۳۰ روز / ۹۰ روز / تا تاریخ…).
var DurationChoices = []string{DurationUnlimited, "30", "90", DurationUntil}

// Limits (abuse guards, far above real use).
const (
	MaxCourses          = 500
	MaxChapters         = 50
	MaxLessons          = 300
	MaxGroups           = 200
	MaxPhonesPerRequest = 200
	MaxGrantsListed     = 2000
	MaxTitleLen         = 150
	MaxDescriptionLen   = 5000
	MaxUntilYears       = 5
	// CompletePercent: a lesson watched this far counts as complete.
	CompletePercent = 95
)

// NormalizePhone returns the 09xxxxxxxxx form (Persian / Arabic digits, +98 / 0098 prefixes accepted) or ok=false.
func NormalizePhone(in string) (string, bool) { return companion.NormalizeMobile(in) }

// Duration is a grant's validated duration.
type Duration struct {
	Kind  string         // DurationUnlimited | DurationDays | DurationUntil
	Days  int            // DurationDays
	Until civildate.Date // DurationUntil (inclusive)
}

// ExpiresAt is when a grant activated at `at` ends: zero = never. Days count from activation; «until» ends at the
// start of the day after Until (Tehran).
func (d Duration) ExpiresAt(at time.Time) time.Time {
	switch d.Kind {
	case DurationDays:
		return at.AddDate(0, 0, d.Days)
	case DurationUntil:
		return d.Until.AddDays(1).TehranMidnight()
	}
	return time.Time{}
}

// DaysLeft is the whole days until expiry, rounded up (0 once expired); -1 = unlimited.
func DaysLeft(expires, now time.Time) int {
	if expires.IsZero() {
		return -1
	}
	if !expires.After(now) {
		return 0
	}
	return int(math.Ceil(expires.Sub(now).Hours() / 24))
}

// Expired reports whether an expiry (zero = never) has passed.
func Expired(expires, now time.Time) bool { return !expires.IsZero() && !expires.After(now) }

// LessonPercent is a lesson's effective completion (a completed lesson counts 100).
func LessonPercent(percent int, completed bool) int {
	if completed {
		return 100
	}
	return min(max(percent, 0), 100)
}

// CoursePercent is the rounded mean of the lessons' completion over total lessons (0 when the course has none).
func CoursePercent(sum, total int) int {
	if total <= 0 {
		return 0
	}
	return int(math.Round(float64(sum) / float64(total)))
}
