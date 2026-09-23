package profile

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"github.com/ritme/backend-go/internal/enums"
	"github.com/ritme/backend-go/internal/platform/jsonx"
	"github.com/ritme/backend-go/internal/platform/phpround"
	"github.com/ritme/backend-go/internal/platform/validation/phpval"
	"github.com/ritme/backend-go/internal/profile/model"
	"github.com/ritme/backend-go/internal/profile/store"
)

// MessageContents is the read side of MessageContentRepository: the decoded payload of the
// live (active + approved) message_contents row, ok=false when there is none. T-M2-17 owns
// the full repository; any implementation of this interface can be plugged in.
type MessageContents interface {
	Payload(ctx context.Context, group, itemKey, locale string) (any, bool, error)
}

// StoreMessageContents reads message_contents through the profile store.
type StoreMessageContents struct{ Q store.Querier }

// Payload implements MessageContents (json_decode($payload, true); invalid JSON → null → fallback).
func (s StoreMessageContents) Payload(ctx context.Context, group, itemKey, locale string) (any, bool, error) {
	raw, err := s.Q.GetLiveMessagePayload(ctx, store.GetLiveMessagePayloadParams{Group: group, ItemKey: itemKey, Locale: locale})
	if errors.Is(err, sql.ErrNoRows) {
		return nil, false, nil
	}
	if err != nil {
		return nil, false, fmt.Errorf("profile: message content: %w", err)
	}
	// json_decode failure is null → the code fallback, as in Laravel (not an error).
	if v, decErr := phpval.Decode(raw); decErr == nil && v != nil {
		return v, true, nil
	}
	return nil, false, nil
}

// bmiGroup is BmiService::GROUP.
const bmiGroup = "bmi_message"

// Bmi is App\Services\BmiService.
type Bmi struct{ Content MessageContents }

// ForProfile is BmiService::forProfile: {value, category, category_label, message}, or nil
// when height/weight are missing or non-positive. defaultLocale is the product default
// language (the seeded messages' fallback).
func (b Bmi) ForProfile(ctx context.Context, p *model.UserProfile, locale, defaultLocale string) (any, error) {
	w, _ := model.ProfileWeight(p)
	h, _ := model.ProfileHeight(p)
	raw, ok := RawBmi(w, h)
	if !ok {
		return nil, nil
	}
	cat := BmiCategoryOf(raw)
	msg, err := b.message(ctx, cat, locale, defaultLocale)
	if err != nil {
		return nil, err
	}
	return jsonx.Obj(
		"value", jsonx.Float(phpround.Round(raw, 1)),
		"category", string(cat),
		"category_label", cat.Label(locale),
		"message", msg,
	), nil
}

// RawBmi is the unrounded kg/m² ratio; ok=false when a metric is missing or non-positive.
func RawBmi(weightKg float64, heightCm int) (float64, bool) {
	if weightKg <= 0 || heightCm <= 0 {
		return 0, false
	}
	m := float64(heightCm) / 100
	return weightKg / (m * m), true
}

// BmiCategoryOf is BmiCategory::fromBmi (a boundary value lands in the higher band).
func BmiCategoryOf(bmi float64) enums.BmiCategory {
	switch {
	case bmi < 18.5:
		return enums.BmiCategoryUnderweight
	case bmi < 25.0:
		return enums.BmiCategoryNormal
	case bmi < 30.0:
		return enums.BmiCategoryOverweight
	}
	return enums.BmiCategoryObese
}

// message is BmiService::messageFor: the DB payload's "message", else the seeded default.
func (b Bmi) message(ctx context.Context, cat enums.BmiCategory, locale, defaultLocale string) (any, error) {
	fallback, ok := bmiDefaults[cat][locale]
	if !ok {
		fallback = bmiDefaults[cat][defaultLocale]
	}
	if b.Content == nil {
		return fallback, nil
	}
	payload, found, err := b.Content.Payload(ctx, bmiGroup, string(cat), locale)
	if err != nil {
		return nil, err
	}
	if !found {
		return fallback, nil
	}
	if m, isMap := payload.(phpval.Map); isMap {
		if v, has := m.Get("message"); has && v != nil {
			return v, nil
		}
	}
	return fallback, nil
}

// bmiDefaults is BmiService::messageDefaults() (seed + runtime fallback), category → locale.
var bmiDefaults = map[enums.BmiCategory]map[string]string{
	enums.BmiCategoryUnderweight: {
		"fa": "بر اساس قد و وزن وارد شده، در محدوده کم‌وزن قرار می‌گیری. اگر این وضعیت برای مدت طولانی ادامه داشته باشد، می‌تواند روی انرژی، خلق و سیکل قاعدگی‌ات اثر بگذارد. اگر نگران هستی، بهتر است با پزشک یا کارشناس تغذیه صحبت کنی.", //nolint:staticcheck // ST1018: Persian text (ZWNJ) verbatim from BmiService
		"en": "Based on the height and weight you entered, you fall in the underweight range. If this continues for a long time, it can affect your energy, mood, and menstrual cycle. If you are concerned, it is best to talk to a doctor or a nutrition specialist.",
	},
	enums.BmiCategoryNormal: {
		"fa": "بر اساس قد و وزن وارد شده، در محدوده‌ی وزنی طبیعی قرار می‌گیری. این محدوده معمولاً برای سلامت عمومی و سیکل قاعدگی مناسب است. Ritme تلاش می‌کند به حفظ این وضعیت کمک کند.", //nolint:staticcheck // ST1018: Persian text (ZWNJ) verbatim from BmiService
		"en": "Based on the height and weight you entered, you are in the normal weight range. This range is usually good for overall health and your menstrual cycle. Ritme aims to help you maintain it.",
	},
	enums.BmiCategoryOverweight: {
		"fa": "بر اساس قد و وزن وارد شده، در محدوده‌ی اضافه‌وزن قرار می‌گیری. این وضعیت می‌تواند در بعضی افراد روی انرژی، خواب و سیکل قاعدگی (به‌خصوص در PCOS) اثر بگذارد. تغییرات کوچک و پایدار در فعالیت و تغذیه، بیشتر از رژیم‌های سخت به سلامتت کمک می‌کنند.", //nolint:staticcheck // ST1018: Persian text (ZWNJ) verbatim from BmiService
		"en": "Based on the height and weight you entered, you fall in the overweight range. In some people this can affect energy, sleep, and the menstrual cycle (especially with PCOS). Small, sustainable changes in activity and nutrition help your health more than strict diets.",
	},
	enums.BmiCategoryObese: {
		"fa": "بر اساس قد و وزن وارد شده، در محدوده‌ی چاقی قرار می‌گیری. این می‌تواند در طولانی‌مدت روی سلامت قلب، فشار خون، قند و سیکل قاعدگی اثر بگذارد. اگر امکانش را داری، صحبت با پزشک یا کارشناس تغذیه می‌تواند خیلی کمک‌کننده باشد. در Ritme سعی می‌کنیم با توصیه‌های کوچک قابل‌اجرا، به روند سلامتت کمک کنیم.", //nolint:staticcheck // ST1018: Persian text (ZWNJ) verbatim from BmiService
		"en": "Based on the height and weight you entered, you fall in the obesity range. Over the long term this can affect heart health, blood pressure, blood sugar, and the menstrual cycle. If possible, talking to a doctor or a nutrition specialist can be very helpful. At Ritme we try to support your health journey with small, doable recommendations.",
	},
}
