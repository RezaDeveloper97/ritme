package learning

import (
	"database/sql"
	"encoding/json"
	"io/fs"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/ritme/backend-go/internal/learning/store"
	"github.com/ritme/backend-go/internal/platform/civildate"
	"github.com/ritme/backend-go/internal/platform/httpx"
	"github.com/ritme/backend-go/internal/platform/validation/phpval"
)

var now = time.Date(2026, 10, 7, 10, 0, 0, 0, civildate.Tehran)

func body(t *testing.T, s string) phpval.Map {
	t.Helper()
	v, err := phpval.Decode([]byte(s))
	require.NoError(t, err)
	m, ok := v.(phpval.Map)
	require.True(t, ok)
	return m
}

func TestDuration_ExpiresAt(t *testing.T) {
	assert.True(t, Duration{Kind: DurationUnlimited}.ExpiresAt(now).IsZero())
	assert.Equal(t, now.AddDate(0, 0, 30), Duration{Kind: DurationDays, Days: 30}.ExpiresAt(now))
	until := civildate.MustParse("2026-12-20")
	assert.Equal(t, time.Date(2026, 12, 21, 0, 0, 0, 0, civildate.Tehran), Duration{Kind: DurationUntil, Until: until}.ExpiresAt(now))
}

func TestDaysLeftAndExpired(t *testing.T) {
	assert.Equal(t, -1, DaysLeft(time.Time{}, now))
	assert.Equal(t, 0, DaysLeft(now, now))
	assert.Equal(t, 1, DaysLeft(now.Add(time.Hour), now))
	assert.Equal(t, 85, DaysLeft(now.AddDate(0, 0, 85), now))
	assert.True(t, Expired(now, now))
	assert.False(t, Expired(time.Time{}, now))
	assert.False(t, Expired(now.Add(time.Second), now))
}

func TestPercents(t *testing.T) {
	assert.Equal(t, 100, LessonPercent(40, true))
	assert.Equal(t, 100, LessonPercent(140, false))
	assert.Equal(t, 0, CoursePercent(10, 0))
	assert.Equal(t, 35, CoursePercent(420, 12))
}

func TestNormalizePhone(t *testing.T) {
	for in, want := range map[string]string{
		"09123456789": "09123456789", "0912 345 6789": "09123456789", "+989123456789": "09123456789",
		"۰۹۱۲۳۴۵۶۷۸۹": "09123456789", "00989123456789": "09123456789", "9123456789": "09123456789",
	} {
		got, ok := NormalizePhone(in)
		assert.True(t, ok, in)
		assert.Equal(t, want, got, in)
	}
	for _, in := range []string{"", "0912345678", "08123456789", "0912abc6789"} {
		_, ok := NormalizePhone(in)
		assert.False(t, ok, in)
	}
}

func TestValidateGrants(t *testing.T) {
	in, err := ValidateGrants(body(t, `{"phones":["0912 345 6789","+989123456789","۰۹۳۵۱۱۱۲۲۳۳"],"scope":"group","target_id":7,"duration":"90"}`), "en", now)
	require.NoError(t, err)
	assert.Equal(t, []string{"09123456789", "09351112233"}, in.Phones)
	assert.Equal(t, Duration{Kind: DurationDays, Days: 90}, in.Duration)
	assert.Equal(t, uint64(7), in.TargetID)

	in, err = ValidateGrants(body(t, `{"phones":["09123456789"],"scope":"course","target_id":1,"duration":"until","until":"2026-12-20"}`), "en", now)
	require.NoError(t, err)
	assert.Equal(t, DurationUntil, in.Duration.Kind)

	_, err = ValidateGrants(body(t, `{"phones":["09123456789","123"],"scope":"course","target_id":1,"duration":"unlimited"}`), "en", now)
	var fe *httpx.FailError
	require.ErrorAs(t, err, &fe)
	raw, _ := json.Marshal(fe.Body())
	assert.Contains(t, string(raw), `"phones.1"`)

	_, err = ValidateGrants(body(t, `{"phones":["09123456789"],"scope":"course","target_id":1,"duration":"until","until":"2026-01-01"}`), "en", now)
	require.Error(t, err)

	_, err = ValidateGrants(body(t, `{"phones":[],"scope":"x","duration":"45"}`), "en", now)
	require.Error(t, err)
}

func TestValidateProgress(t *testing.T) {
	in, err := ValidateProgress(body(t, `{"position_seconds":120}`), "en", now)
	require.NoError(t, err)
	assert.Equal(t, ProgressInput{Position: 120, Percent: -1}, in)
	in, err = ValidateProgress(body(t, `{"position_seconds":0,"percent":40,"completed":true}`), "en", now)
	require.NoError(t, err)
	assert.Equal(t, ProgressInput{Position: 0, Percent: 40, CompleteIt: true}, in)
	_, err = ValidateProgress(body(t, `{"percent":140}`), "en", now)
	require.Error(t, err)
}

func lesson(id, chapter uint64, order uint16) store.LearningLesson {
	return store.LearningLesson{ID: id, ChapterID: sql.NullInt64{Int64: int64(chapter), Valid: chapter != 0}, SortOrder: order, Status: StatusPublished}
}

func TestOrderLessons(t *testing.T) {
	chapters := []store.LearningChapter{{ID: 20, SortOrder: 1}, {ID: 10, SortOrder: 2}}
	got := OrderLessons(chapters, []store.LearningLesson{lesson(1, 10, 1), lesson(2, 0, 1), lesson(3, 20, 2), lesson(4, 20, 1)})
	ids := []uint64{}
	for _, l := range got {
		ids = append(ids, l.ID)
	}
	assert.Equal(t, []uint64{4, 3, 1, 2}, ids)
}

func TestBetterGrant(t *testing.T) {
	live := store.ListUserCourseAccessRow{ExpiresAt: sql.NullTime{Time: now.AddDate(0, 0, 5), Valid: true}}
	later := store.ListUserCourseAccessRow{ExpiresAt: sql.NullTime{Time: now.AddDate(0, 0, 50), Valid: true}}
	unlimited := store.ListUserCourseAccessRow{}
	expired := store.ListUserCourseAccessRow{ExpiresAt: sql.NullTime{Time: now.AddDate(0, 0, -1), Valid: true}}
	assert.True(t, better(live, expired, now))
	assert.False(t, better(expired, live, now))
	assert.True(t, better(unlimited, later, now))
	assert.True(t, better(later, live, now))
}

func TestFindContinue_LatestUnfinishedOpenLesson(t *testing.T) {
	seen := func(min int) sql.NullTime {
		return sql.NullTime{Time: now.Add(-time.Duration(min) * time.Minute), Valid: true}
	}
	locked := store.LearningChapter{ID: 9, UnlockAt: sql.NullTime{Time: now.Add(time.Hour), Valid: true}}
	c := &StudentCourse{
		Chapters: []store.LearningChapter{locked},
		Lessons:  []store.LearningLesson{lesson(1, 0, 1), lesson(2, 0, 2), lesson(3, 9, 1)},
		Progress: map[uint64]store.LearningProgress{
			1: {LessonID: 1, Percent: 100, CompletedAt: seen(1), LastSeenAt: seen(1)},
			2: {LessonID: 2, Percent: 40, LastSeenAt: seen(30)},
			3: {LessonID: 3, Percent: 10, LastSeenAt: seen(0)},
		},
	}
	k := FindContinue([]*StudentCourse{c}, now)
	require.NotNil(t, k)
	assert.Equal(t, uint64(2), k.Lesson.ID)
	assert.Equal(t, 2, k.Number)

	c.Access.ExpiresAt = sql.NullTime{Time: now.Add(-time.Hour), Valid: true}
	assert.Nil(t, FindContinue([]*StudentCourse{c}, now))
}

func TestLangFiles_SameKeys(t *testing.T) {
	keys := func(code string) map[string]bool {
		raw, err := fs.ReadFile(langFS, "lang/"+code+"/learning.json")
		require.NoError(t, err)
		var m map[string]map[string]string
		require.NoError(t, json.Unmarshal(raw, &m))
		out := map[string]bool{}
		for g, kv := range m {
			for k := range kv {
				out[g+"."+k] = true
			}
		}
		return out
	}
	assert.Equal(t, keys("en"), keys("fa"))
	assert.Equal(t, "Your instructor opened 2 courses for you", T("notice.title_many", "en", "instructor", "Your instructor", "count", "2"))
}
