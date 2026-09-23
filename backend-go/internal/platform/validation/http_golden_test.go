package validation_test

import (
	"net/url"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/ritme/backend-go/internal/i18n/lang"
	"github.com/ritme/backend-go/internal/platform/jsonx"
	"github.com/ritme/backend-go/internal/platform/validation"
	"github.com/ritme/backend-go/internal/platform/validation/phpval"
)

// ruleSet rebuilds one controller's validation in Go and renders its 422 body.
type ruleSet struct {
	fromQuery bool // Validator::make($request->query(), …)
	rules     func(enums map[string][]string, tr *lang.Translator, locale string) (validation.Rules, []string)
	prepare   func(data phpval.Map)                                                 // FormRequest::prepareForValidation
	body      func(v *validation.Validator, tr *lang.Translator, locale string) any // the 422 body
}

func frameworkBody(v *validation.Validator, _ *lang.Translator, _ string) any {
	return v.Errors().Body()
}

func controllerBody(msg func(v *validation.Validator, tr *lang.Translator, locale string) string) func(*validation.Validator, *lang.Translator, string) any {
	return func(v *validation.Validator, tr *lang.Translator, locale string) any {
		return jsonx.Obj("success", false, "message", msg(v, tr, locale), "errors", v.ErrorBag())
	}
}

func faOr(fa, en string) func(*validation.Validator, *lang.Translator, string) string {
	return func(_ *validation.Validator, _ *lang.Translator, locale string) string {
		if locale == "fa" {
			return fa
		}
		return en
	}
}

func firstError(v *validation.Validator, _ *lang.Translator, _ string) string { return v.First() }

func reminderRules(required bool) func(map[string][]string, *lang.Translator, string) (validation.Rules, []string) {
	presence := "sometimes"
	if required {
		presence = "required"
	}
	return func(e map[string][]string, _ *lang.Translator, _ string) (validation.Rules, []string) {
		return validation.Rules{
			validation.F("type", presence, validation.In(e["ReminderType"]...)),
			validation.F("title", presence, "string", "max:255"),
			validation.F("subtitle", "nullable|string|max:255"),
			validation.F("notes", "nullable|string|max:2000"),
			validation.F("scheduled_at", "nullable|date"),
			validation.F("recurrence", "sometimes", validation.In("none", "daily", "weekly", "monthly")),
			validation.F("recurrence_time", "nullable|date_format:H:i"),
			validation.F("starts_on", "nullable|date"),
			validation.F("ends_on", "nullable|date|after_or_equal:starts_on"),
			validation.F("is_active", "sometimes|boolean"),
		}, nil
	}
}

func periodRules(withStart bool) func(map[string][]string, *lang.Translator, string) (validation.Rules, []string) {
	return func(map[string][]string, *lang.Translator, string) (validation.Rules, []string) {
		if !withStart {
			return validation.Rules{validation.F("date", "nullable|date|before_or_equal:today")}, nil
		}
		return validation.Rules{
			validation.F("start_date", "required|date|before_or_equal:today"),
			validation.F("end_date", "nullable|date|after_or_equal:start_date"),
		}, nil
	}
}

func profileRules(e map[string][]string, tr *lang.Translator, locale string) (validation.Rules, []string) {
	t := func(key string, params ...string) string {
		p := map[string]string{}
		for i := 0; i+1 < len(params); i += 2 {
			p[params[i]] = params[i+1]
		}
		return tr.Trans("profile.errors."+key, p, locale)
	}
	join := func(name string) string { return strings.Join(e[name], ", ") }
	return validation.Rules{
		validation.F("name", "nullable|string|max:255"),
		validation.F("birthday", "nullable|date|before:today"),
		validation.F("weight", "nullable|numeric|min:20|max:300"),
		validation.F("height", "nullable|integer|min:50|max:250"),
		validation.F("period_duration", "nullable|integer|min:1|max:15"),
		validation.F("cycle_duration", "nullable|integer|min:15|max:60"),
		validation.F("last_period_start", "nullable|date|before_or_equal:today"),
		validation.F("user_goal", "nullable", validation.In(e["UserGoal"]...)),
		validation.F("subscription_type", "nullable", validation.In(e["SubscriptionType"]...)),
		validation.F("pregnancy_intention", "nullable", validation.In(e["PregnancyIntention"]...)),
		validation.F("chronic_conditions", "nullable", "array"),
		validation.F("chronic_conditions.*", validation.In(e["ChronicCondition"]...)),
	}, []string{
		"name.string", t("name_string"),
		"name.max", t("name_max"),
		"birthday.date", t("birthday_date"),
		"birthday.before", t("birthday_before"),
		"weight.numeric", t("weight_numeric"),
		"weight.min", t("weight_min"),
		"weight.max", t("weight_max"),
		"height.integer", t("height_integer"),
		"height.min", t("height_min"),
		"height.max", t("height_max"),
		"period_duration.integer", t("period_duration_integer"),
		"period_duration.min", t("period_duration_min"),
		"period_duration.max", t("period_duration_max"),
		"cycle_duration.integer", t("cycle_duration_integer"),
		"cycle_duration.min", t("cycle_duration_min"),
		"cycle_duration.max", t("cycle_duration_max"),
		"last_period_start.date", t("last_period_start_date"),
		"last_period_start.before_or_equal", t("last_period_start_before_or_equal"),
		"user_goal.in", t("user_goal_in", "values", join("UserGoal")),
		"subscription_type.in", t("subscription_type_in", "values", join("SubscriptionType")),
		"pregnancy_intention.in", t("pregnancy_intention_in", "values", join("PregnancyIntention")),
		"chronic_conditions.array", t("chronic_conditions_array"),
		"chronic_conditions.*.in", t("chronic_conditions_in", "values", join("ChronicCondition")),
	}
}

func formRequest(set string, sets phpval.Map) func(map[string][]string, *lang.Translator, string) (validation.Rules, []string) {
	return func(map[string][]string, *lang.Translator, string) (validation.Rules, []string) {
		fr := get(get(sets, "form_requests").(phpval.Map), set).(phpval.Map)
		return rulesFrom(get(fr, "rules")), kvFrom(get(fr, "messages"))
	}
}

func buildSets(sets phpval.Map) map[string]ruleSet {
	otpMsg := controllerBody(faOr("Validation failed", "Validation failed"))
	m := map[string]ruleSet{
		"otp_send": {rules: func(map[string][]string, *lang.Translator, string) (validation.Rules, []string) {
			return validation.Rules{validation.F("mobile", "required", "string", "regex:/^09[0-9]{9}$/")},
				[]string{"mobile.regex", "Mobile number must be a valid Iranian mobile number (e.g., 09123456789)"}
		}, body: otpMsg},
		"otp_verify": {rules: func(map[string][]string, *lang.Translator, string) (validation.Rules, []string) {
			return validation.Rules{
				validation.F("mobile", "required", "string", "regex:/^09[0-9]{9}$/"),
				validation.F("code", "required|string|size:4"),
			}, nil
		}, body: otpMsg},
		"reminder_store":  {rules: reminderRules(true), body: controllerBody(faOr("اطلاعات واردشده نامعتبر است", "Validation failed"))},
		"reminder_update": {rules: reminderRules(false), body: controllerBody(faOr("اطلاعات واردشده نامعتبر است", "Validation failed"))},
		"profile_store": {rules: profileRules, body: controllerBody(func(v *validation.Validator, tr *lang.Translator, locale string) string {
			return tr.Trans("profile.validation_failed", map[string]string{"first": v.First()}, locale)
		})},
		"period_start":  {rules: periodRules(false), body: controllerBody(firstError)},
		"period_end":    {rules: periodRules(false), body: controllerBody(firstError)},
		"period_store":  {rules: periodRules(true), body: controllerBody(firstError)},
		"period_update": {rules: periodRules(true), body: controllerBody(firstError)},
		"messages_daily": {fromQuery: true, rules: func(e map[string][]string, _ *lang.Translator, _ string) (validation.Rules, []string) {
			return validation.Rules{
				validation.F("date", "nullable|date_format:Y-m-d"),
				validation.F("mode", "nullable|in:"+strings.Join(e["MessageMode"], ",")),
			}, nil
		}, body: controllerBody(faOr("پارامترهای ورودی نامعتبر است", "Invalid query parameters"))},
		"articles_index": {rules: func(map[string][]string, *lang.Translator, string) (validation.Rules, []string) {
			return validation.Rules{
				validation.F("page", "nullable", "integer", "min:1"),
				validation.F("per_page", "nullable", "integer", "min:1", "max:50"),
				validation.F("category", "nullable", "string", "max:255"),
				validation.F("q", "nullable", "string", "max:100"),
			}, nil
		}, body: frameworkBody},
		"home": {rules: func(map[string][]string, *lang.Translator, string) (validation.Rules, []string) {
			return validation.Rules{validation.F("date", "nullable|date")}, nil
		}, body: frameworkBody},
	}
	for _, set := range []string{"healthlog_store", "pregnancy_onboarding", "pregnancy_update",
		"pregnancy_symptoms", "pregnancy_weekly", "pregnancy_fetal"} {
		m[set] = ruleSet{rules: formRequest(set, sets), body: frameworkBody}
	}
	hl := m["healthlog_store"]
	hl.prepare = func(data phpval.Map) { // StoreDailyHealthLogRequest::prepareForValidation
		if s, ok := get(data, "exercise_type").(string); ok {
			data.Set("exercise_type", []any{s})
		}
	}
	m["healthlog_store"] = hl
	return m
}

// httpBuildQuery encodes a flat query map like PHP's http_build_query.
func httpBuildQuery(q phpval.Map) string {
	if q == nil {
		return ""
	}
	parts := make([]string, 0, q.Len())
	for _, k := range q.Keys() {
		parts = append(parts, url.QueryEscape(k)+"="+url.QueryEscape(phpval.ToString(get(q, k))))
	}
	return strings.Join(parts, "&")
}

func TestHTTP_ValidationBodiesMatchLaravel(t *testing.T) {
	cases := loadJSON(t, "http_cases.json")
	golden := loadJSON(t, "http_golden.json")
	sets := loadJSON(t, "rulesets.json")
	enums := map[string][]string{}
	for _, name := range get(sets, "enums").(phpval.Map).Keys() {
		for _, x := range get(get(sets, "enums").(phpval.Map), name).([]any) {
			enums[name] = append(enums[name], x.(string))
		}
	}
	tr := lang.Default()
	defs := buildSets(sets)

	for _, name := range cases.Keys() {
		c := get(cases, name).(phpval.Map)
		want := get(golden, name).(phpval.Map)
		def, ok := defs[get(c, "set").(string)]
		require.True(t, ok, "unknown set %v", get(c, "set"))
		t.Run(name, func(t *testing.T) {
			query := validation.ParseQuery(httpBuildQuery(asMap(get(c, "query"))))
			var data phpval.Map
			switch {
			case def.fromQuery || get(c, "method") == "GET":
				data = query
			default:
				data = validation.DecodeBody(mustJSON(t, get(c, "body")))
			}
			if def.prepare != nil {
				def.prepare(data)
			}
			// SetLocale: ?locale= then Accept-Language, else the default language (fa).
			locale := "fa"
			if l, ok := get(c, "lang").(string); ok {
				locale = l
			}
			if l, ok := get(query, "locale").(string); ok {
				locale = l
			}

			rules, msgs := def.rules(enums, tr, locale)
			v := validation.Make(tr, locale, data, rules, validation.Now(captureNow), validation.Messages(msgs...))
			require.True(t, v.Fails())

			got, err := jsonx.Marshal(def.body(v, tr, locale), 0)
			require.NoError(t, err)
			assert.Equal(t, int64(422), get(want, "status"))
			assert.Equal(t, get(want, "body"), string(got))
		})
	}
}

func asMap(v any) phpval.Map {
	m, _ := v.(phpval.Map)
	return m
}

func mustJSON(t *testing.T, v any) []byte {
	t.Helper()
	if v == nil {
		return []byte("{}")
	}
	b, err := jsonx.Marshal(v, jsonx.UnescapedUnicode|jsonx.UnescapedSlashes)
	require.NoError(t, err)
	return b
}
