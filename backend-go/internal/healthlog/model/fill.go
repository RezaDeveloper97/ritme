package model

import (
	"database/sql"
	"fmt"
	"math"
	"strconv"
	"strings"

	rootdb "github.com/ritme/backend-go/db"
	"github.com/ritme/backend-go/internal/enums"
	"github.com/ritme/backend-go/internal/platform/jsonx"
	"github.com/ritme/backend-go/internal/platform/validation/phpval"
)

// Set is $log->setAttribute($name, $value) for a data column, storing the value the way the
// DB will hold it: booleans as (bool), integers as (int), decimals rounded HALF_UP to the
// column scale (what MariaDB does to the bound string), arrays json_encode()d with PHP's
// default flags. null clears the column. v is a request value (see phpval).
func (l *DailyHealthLog) Set(name string, v any) error {
	c, ok := ColumnByName(name)
	if !ok {
		return fmt.Errorf("healthlog: unknown column %q", name)
	}
	switch f := c.Field(&l.Row).(type) {
	case *sql.NullString:
		if v == nil {
			*f = sql.NullString{}
			return nil
		}
		s := phpval.ToString(v)
		if c.Kind == Decimal {
			d, err := jsonx.Decimal(decimalSource(v), c.Scale)
			if err != nil {
				return fmt.Errorf("healthlog: %s: %w", name, err)
			}
			s = d.String()
		}
		*f = sql.NullString{String: s, Valid: true}
	case *sql.NullBool:
		if v == nil {
			*f = sql.NullBool{}
			return nil
		}
		*f = sql.NullBool{Bool: phpval.Truthy(v), Valid: true}
	case *sql.NullInt16:
		if v == nil {
			*f = sql.NullInt16{}
			return nil
		}
		n, err := phpInt(v)
		if err != nil || n < math.MinInt16 || n > math.MaxInt16 {
			return fmt.Errorf("healthlog: %s: %v is not a smallint", name, v)
		}
		*f = sql.NullInt16{Int16: int16(n), Valid: true}
	case *rootdb.NullRawJSON:
		if v == nil {
			*f = rootdb.NullRawJSON{}
			return nil
		}
		b, err := jsonx.Marshal(v, 0)
		if err != nil {
			return fmt.Errorf("healthlog: %s: %w", name, err)
		}
		*f = rootdb.NullRawJSON{V: b, Valid: true}
	default:
		return fmt.Errorf("healthlog: column %q has an unsupported field type %T", name, f)
	}
	return nil
}

// decimalSource passes numbers through as jsonx.Decimal expects them (int64 / float64 /
// string); anything else goes through PHP's string conversion.
func decimalSource(v any) any {
	switch v.(type) {
	case int64, float64, string:
		return v
	}
	return phpval.ToString(v)
}

// phpInt is PHP's (int) cast for the values the `integer` rule lets through
// (int, integral float, or a decimal-integer string).
func phpInt(v any) (int64, error) {
	switch x := v.(type) {
	case int64:
		return x, nil
	case int:
		return int64(x), nil
	case float64:
		return int64(x), nil
	case bool:
		if x {
			return 1, nil
		}
		return 0, nil
	case string:
		s := strings.TrimSpace(x)
		if n, err := strconv.ParseInt(s, 10, 64); err == nil {
			return n, nil
		}
		f, err := strconv.ParseFloat(s, 64)
		if err != nil {
			return 0, fmt.Errorf("healthlog: %q is not numeric", x)
		}
		return int64(f), nil
	}
	return 0, fmt.Errorf("healthlog: cannot cast %T to int", v)
}

// EnumValues is DailyHealthLog::getEnumValues() (GET /health-logs/enums), in its key order.
func EnumValues() *jsonx.OrderedMap {
	return jsonx.Obj(
		"bleeding_intensity", enums.IntensityValues(),
		"blood_color", enums.BloodColorValues(),
		"bleeding_smell", enums.SmellValues(),
		"pain_intensity", enums.PainIntensityValues(),
		"nausea_intensity", enums.PainIntensityValues(),
		"bloating_intensity", enums.PainIntensityValues(),
		"clots_amount", enums.ClotsAmountValues(),
		// Ordered least → most, the way the form reads them.
		"appetite_change", []string{"loss", "normal", "gain"},
		"urination_change", []string{"decrease", "increase"},
		"moods", enums.MoodValues(),
		"sleep_duration", enums.SleepDurationValues(),
		"sleep_quality", enums.SleepQualityValues(),
		"exercise_type", enums.ExerciseTypeValues(),
		"exercise_intensity", enums.ExerciseIntensityValues(),
		"sexual_desire", enums.SexualDesireValues(),
		"intercourse_type", enums.IntercourseTypeValues(),
		// Only the "during / after intercourse" experiences.
		"sexual_activities", []string{
			string(enums.SexualActivityDryness),
			string(enums.SexualActivityBurning),
			string(enums.SexualActivityPainDuringIntercourse),
			string(enums.SexualActivityBleedingAfterIntercourse),
			string(enums.SexualActivityLubricantUse),
		},
		"discharge_texture", enums.DischargeTextureValues(),
		"discharge_amount", enums.AmountValues(),
		"discharge_color", enums.DischargeColorValues(),
		"discharge_smell", enums.SmellValues(),
	)
}
