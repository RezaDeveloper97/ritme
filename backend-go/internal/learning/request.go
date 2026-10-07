package learning

import (
	"slices"
	"strconv"
	"strings"
	"time"

	"github.com/gofiber/fiber/v3"

	"github.com/ritme/backend-go/internal/i18n/lang"
	"github.com/ritme/backend-go/internal/learning/store"
	"github.com/ritme/backend-go/internal/platform/civildate"
	"github.com/ritme/backend-go/internal/platform/httpx"
	"github.com/ritme/backend-go/internal/platform/jsonx"
	"github.com/ritme/backend-go/internal/platform/validation"
	"github.com/ritme/backend-go/internal/platform/validation/phpval"
)

// unlockLayout is the request format of a chapter's unlock time (Tehran wall-clock).
const unlockLayout = "2006-01-02 15:04"

// failValidation is the controller-style 422 {success:false, message, errors}.
func failValidation(locale string, errs any) error {
	return httpx.Fail(fiber.StatusUnprocessableEntity, T("messages.validation_failed", locale), "errors", errs)
}

func fieldFail(locale, field, key string) error {
	return failValidation(locale, jsonx.Obj(field, []string{T("validation."+key, locale)}))
}

func pick(body phpval.Map, keys []string) phpval.Map {
	data := phpval.NewMap()
	for _, k := range keys {
		if v, ok := body.Get(k); ok {
			data.Set(k, v)
		}
	}
	return data
}

func validate(data phpval.Map, rules validation.Rules, locale string, now time.Time) error {
	v := validation.Make(lang.Default(), locale, data, rules, validation.Now(now), validation.Attributes(attributes(locale)...))
	if v.Fails() {
		return failValidation(locale, v.ErrorBag())
	}
	return nil
}

func str(data phpval.Map, key string) string {
	v, ok := data.Get(key)
	if !ok || v == nil {
		return ""
	}
	return strings.TrimSpace(phpval.ToString(v))
}

func num(data phpval.Map, key string) int64 {
	v, ok := data.Get(key)
	if !ok || v == nil {
		return 0
	}
	n, _ := strconv.ParseInt(phpval.ToString(v), 10, 64)
	return n
}

func has(data phpval.Map, key string) bool {
	_, ok := data.Get(key)
	return ok
}

func truthy(data phpval.Map, key string) bool {
	v, _ := data.Get(key)
	return phpval.Truthy(v)
}

func elems(v any) []any {
	switch x := v.(type) {
	case []any:
		return x
	case phpval.Map:
		out := make([]any, 0, x.Len())
		for _, k := range x.Keys() {
			e, _ := x.Get(k)
			out = append(out, e)
		}
		return out
	}
	return nil
}

func presence(partial bool) []any {
	if partial {
		return []any{"sometimes", "required"}
	}
	return []any{"required"}
}

func rule(base []any, more ...any) []any { return append(append([]any(nil), base...), more...) }

func maxLen(n int) string { return "max:" + strconv.Itoa(n) }

// ValidateApply is POST /api/instructor/v1/apply {display_name, title?, bio?}.
func ValidateApply(body phpval.Map, locale string, now time.Time) (ApplyInput, error) {
	data := pick(body, []string{"display_name", "title", "bio"})
	if err := validate(data, validation.Rules{
		validation.F("display_name", "required", "string", "min:2", maxLen(80)),
		validation.F("title", "nullable", "string", maxLen(60)),
		validation.F("bio", "nullable", "string", maxLen(500)),
	}, locale, now); err != nil {
		return ApplyInput{}, err
	}
	return ApplyInput{DisplayName: str(data, "display_name"), Title: str(data, "title"), Bio: str(data, "bio")}, nil
}

// ValidateCourse is POST /courses {kind, title, description?, status?} and PUT /courses/{id} (every field optional;
// kind cannot change).
func ValidateCourse(body phpval.Map, cur *store.LearningCourse, locale string, now time.Time) (CourseInput, error) {
	data := pick(body, []string{"kind", "title", "description", "status"})
	partial := cur != nil
	rules := validation.Rules{
		validation.F("title", rule(presence(partial), "string", maxLen(MaxTitleLen))...),
		validation.F("description", "nullable", "string", maxLen(MaxDescriptionLen)),
		validation.F("status", "sometimes", "required", validation.In(Statuses...)),
	}
	if !partial {
		rules = append(rules, validation.F("kind", "required", validation.In(CourseKinds...)))
	}
	if err := validate(data, rules, locale, now); err != nil {
		return CourseInput{}, err
	}
	in := CourseInput{Status: StatusDraft}
	if cur != nil {
		in = CourseInput{Kind: cur.Kind, Title: cur.Title, Description: cur.Description.String, Status: cur.Status}
	} else {
		in.Kind = str(data, "kind")
	}
	if has(data, "title") {
		in.Title = str(data, "title")
	}
	if has(data, "description") {
		in.Description = str(data, "description")
	}
	if has(data, "status") {
		in.Status = str(data, "status")
	}
	return in, nil
}

// ValidateChapter is POST /courses/{id}/chapters {title, unlock_at?: "Y-m-d H:i", sort_order?} and PUT …/{chapter}.
func ValidateChapter(body phpval.Map, cur *store.LearningChapter, locale string, now time.Time) (ChapterInput, error) {
	data := pick(body, []string{"title", "unlock_at", "sort_order"})
	partial := cur != nil
	if err := validate(data, validation.Rules{
		validation.F("title", rule(presence(partial), "string", maxLen(MaxTitleLen))...),
		validation.F("unlock_at", "nullable", "date_format:Y-m-d H:i"),
		validation.F("sort_order", "sometimes", "required", "integer", "between:1,1000"),
	}, locale, now); err != nil {
		return ChapterInput{}, err
	}
	in := ChapterInput{}
	if cur != nil {
		in = ChapterInput{Title: cur.Title, SortOrder: int(cur.SortOrder), UnlockAt: timeOf(cur.UnlockAt)}
	}
	if has(data, "title") {
		in.Title = str(data, "title")
	}
	if has(data, "sort_order") {
		in.SortOrder = int(num(data, "sort_order"))
	}
	if has(data, "unlock_at") {
		in.UnlockAt = time.Time{}
		if s := str(data, "unlock_at"); s != "" {
			t, err := time.ParseInLocation(unlockLayout, s, civildate.Tehran)
			if err != nil {
				return ChapterInput{}, fieldFail(locale, "unlock_at", "until_out_of_range")
			}
			in.UnlockAt = t
		}
	}
	return in, nil
}

// ValidateLesson is POST /courses/{id}/lessons {kind, title, chapter_id?, description?, duration_seconds?,
// page_count?, size_bytes?, status?, sort_order?} and PUT …/{lesson} (every field optional).
func ValidateLesson(body phpval.Map, cur *store.LearningLesson, locale string, now time.Time) (LessonInput, error) {
	data := pick(body, []string{"kind", "title", "chapter_id", "description", "duration_seconds", "page_count", "size_bytes", "status", "sort_order"})
	partial := cur != nil
	if err := validate(data, validation.Rules{
		validation.F("kind", rule(presence(partial), validation.In(LessonKinds...))...),
		validation.F("title", rule(presence(partial), "string", maxLen(MaxTitleLen))...),
		validation.F("chapter_id", "nullable", "integer", "min:1"),
		validation.F("description", "nullable", "string", maxLen(MaxDescriptionLen)),
		validation.F("duration_seconds", "nullable", "integer", "between:0,86400"),
		validation.F("page_count", "nullable", "integer", "between:0,5000"),
		validation.F("size_bytes", "nullable", "integer", "between:0,2147483648"),
		validation.F("status", "sometimes", "required", validation.In(Statuses...)),
		validation.F("sort_order", "sometimes", "required", "integer", "between:1,1000"),
	}, locale, now); err != nil {
		return LessonInput{}, err
	}
	in := LessonInput{Status: StatusDraft}
	if cur != nil {
		in = LessonInput{
			ChapterID: uid(cur.ChapterID), Kind: cur.Kind, Title: cur.Title, Description: cur.Description.String,
			DurationSeconds: int(cur.DurationSeconds.Int32), PageCount: int(cur.PageCount.Int16), SizeBytes: cur.SizeBytes.Int64,
			Status: cur.Status, SortOrder: int(cur.SortOrder),
		}
	}
	if has(data, "kind") {
		in.Kind = str(data, "kind")
	}
	if has(data, "title") {
		in.Title = str(data, "title")
	}
	if has(data, "chapter_id") {
		in.ChapterID = uint64(max(num(data, "chapter_id"), 0)) //nolint:gosec // G115: validated min:1
	}
	if has(data, "description") {
		in.Description = str(data, "description")
	}
	if has(data, "duration_seconds") {
		in.DurationSeconds = int(num(data, "duration_seconds"))
	}
	if has(data, "page_count") {
		in.PageCount = int(num(data, "page_count"))
	}
	if has(data, "size_bytes") {
		in.SizeBytes = num(data, "size_bytes")
	}
	if has(data, "status") {
		in.Status = str(data, "status")
	}
	if has(data, "sort_order") {
		in.SortOrder = int(num(data, "sort_order"))
	}
	return in, nil
}

// ValidateGroup is POST /groups {name, course_ids?: [id…]} and PUT /groups/{id} (name optional; course_ids replaces
// the set when sent).
func ValidateGroup(body phpval.Map, cur *store.LearningGroup, locale string, now time.Time) (GroupInput, error) {
	data := pick(body, []string{"name", "course_ids"})
	partial := cur != nil
	if err := validate(data, validation.Rules{
		validation.F("name", rule(presence(partial), "string", maxLen(100))...),
		validation.F("course_ids", "nullable", "array", maxLen(MaxCourses)),
		validation.F("course_ids.*", "required", "integer", "min:1"),
	}, locale, now); err != nil {
		return GroupInput{}, err
	}
	in := GroupInput{}
	if cur != nil {
		in.Name = cur.Name
	}
	if has(data, "name") {
		in.Name = str(data, "name")
	}
	if has(data, "course_ids") {
		in.CourseIDsSet = true
		raw, _ := data.Get("course_ids")
		for _, v := range elems(raw) {
			id, err := strconv.ParseUint(phpval.ToString(v), 10, 64)
			if err == nil && id > 0 && !slices.Contains(in.CourseIDs, id) {
				in.CourseIDs = append(in.CourseIDs, id)
			}
		}
	}
	if !partial {
		in.CourseIDsSet = true
	}
	return in, nil
}

// ValidateGrants is POST /grants {phones: [..], scope: group|course, target_id, duration: unlimited|30|90|until,
// until?: Y-m-d}. Phones are normalised (Persian digits, +98 accepted) and de-duplicated; an invalid one fails the
// request with its index.
func ValidateGrants(body phpval.Map, locale string, now time.Time) (GrantInput, error) {
	data := pick(body, []string{"phones", "scope", "target_id", "duration", "until"})
	if err := validate(data, validation.Rules{
		validation.F("phones", "required", "array", "min:1", maxLen(MaxPhonesPerRequest)),
		validation.F("phones.*", "required", "string", maxLen(20)),
		validation.F("scope", "required", validation.In(Scopes...)),
		validation.F("target_id", "required", "integer", "min:1"),
		validation.F("duration", "required", validation.In(DurationChoices...)),
		validation.F("until", "required_if:duration,until", "nullable", "date_format:Y-m-d"),
	}, locale, now); err != nil {
		return GrantInput{}, err
	}
	in := GrantInput{Scope: str(data, "scope"), TargetID: uint64(max(num(data, "target_id"), 0))} //nolint:gosec // G115: validated min:1
	raw, _ := data.Get("phones")
	for i, v := range elems(raw) {
		p, ok := NormalizePhone(phpval.ToString(v))
		if !ok {
			return GrantInput{}, fieldFail(locale, "phones."+strconv.Itoa(i), "phone_invalid")
		}
		if !slices.Contains(in.Phones, p) {
			in.Phones = append(in.Phones, p)
		}
	}
	switch d := str(data, "duration"); d {
	case DurationUnlimited:
		in.Duration = Duration{Kind: DurationUnlimited}
	case DurationUntil:
		until, err := civildate.Parse(str(data, "until"))
		today := civildate.InTehran(now)
		if err != nil || until.Before(today) || until.After(today.AddDays(366*MaxUntilYears)) {
			return GrantInput{}, fieldFail(locale, "until", "until_out_of_range")
		}
		in.Duration = Duration{Kind: DurationUntil, Until: until}
	default:
		days, _ := strconv.Atoi(d)
		in.Duration = Duration{Kind: DurationDays, Days: days}
	}
	return in, nil
}

// ValidateProgress is PUT /learning/lessons/{id}/progress {position_seconds, percent?, completed?}.
func ValidateProgress(body phpval.Map, locale string, now time.Time) (ProgressInput, error) {
	data := pick(body, []string{"position_seconds", "percent", "completed"})
	if err := validate(data, validation.Rules{
		validation.F("position_seconds", "required", "integer", "between:0,86400"),
		validation.F("percent", "nullable", "integer", "between:0,100"),
		validation.F("completed", "nullable", "boolean"),
	}, locale, now); err != nil {
		return ProgressInput{}, err
	}
	in := ProgressInput{Position: int(num(data, "position_seconds")), Percent: -1}
	if v, ok := data.Get("percent"); ok && v != nil {
		in.Percent = int(num(data, "percent"))
	}
	in.CompleteIt = truthy(data, "completed")
	return in, nil
}
