package facts

import (
	"context"
	"strings"

	"github.com/ritme/backend-go/internal/enums"
	"github.com/ritme/backend-go/internal/platform/civildate"
)

// Account facts: plan, signup age, age, goal, locale, stored life mode, segments and push opt-in.

// Account is the account row of the user.
type Account struct {
	Name             string
	SignupDay        civildate.Date // zero = unknown
	Birthday         civildate.Date // zero = unknown
	UserGoal         string         // "" = no profile
	SubscriptionType string
	LifeMode         string // the stored life-stage mode, "" = none
	Plan             string // free | trial | plus
}

// AccountSource reads the user's account row (nil = no such user).
type AccountSource interface {
	Account(ctx context.Context) (*Account, error)
}

// Groups of this file.
const (
	GroupAccount = "account"
	// GroupGoal is user_goal: the stored goal, read as non_ttc when the engine suppresses fertility (a pill or teen
	// user with goal ttc gets no TTC targeting, T-M9-38). It needs the engine only for a ttc goal.
	GroupGoal = "goal"
)

// Plans are the plan values (plus.Tier).
var Plans = []string{"free", "trial", "plus"}

func init() {
	a := GroupAccount
	Register(
		Def{Name: "plan", Type: TypeEnum, Enum: Plans, Group: a,
			Description: desc("پلن کاربر در این لحظه", "The user's plan now.")},
		Def{Name: "is_premium", Type: TypeBool, Group: a,
			Description: desc("قدیمی: subscription_type = premium؛ به\u200cجایش plan", "Legacy: subscription_type = premium; use plan.")},
		Def{Name: "days_since_signup", Type: TypeInt, Unit: "day", Group: a,
			Description: desc("روز از ثبت\u200cنام", "Days since signup.")},
		Def{Name: "age", Type: TypeInt, Unit: "year", Group: a,
			Description: desc("سن (بدون تاریخ تولد وجود ندارد)", "Age in years (absent without a birthday).")},
		Def{Name: "locale", Type: TypeEnum, Group: a,
			Description: desc("زبان درخواست (از جدول زبان\u200cها)", "The request language (from the languages table).")},
		Def{Name: "life_mode", Type: TypeEnum, Enum: enums.LifeModeValues(), Group: a, MemberSafe: true,
			Description: desc("حالت زندگی ذخیره\u200cشده (بدون ردیف وجود ندارد)", "The stored life-stage mode (absent without one).")},
		Def{Name: "segment", Type: TypeSet, Group: GroupSegment,
			Description: desc("شناسه سگمنت\u200cهای ذخیره\u200cشده پنل که کاربر در آن\u200cهاست (از ردیف شبانه رصد کاربران)",
				"Ids of the saved admin segments the user is in (from the nightly user-directory row).")},
		Def{Name: "push_opt_in", Type: TypeBool, Group: a, Pending: "M-09",
			Description: desc("اجازه پوش؛ تا پوش وب واقعی false (M-09)", "Push opt-in; false until real web push (M-09).")},
		Def{Name: "name", Type: TypeEnum, Group: a, Variable: true,
			Description: desc("نام کوچک برای متن پیام (متغیر، نه شرط)", "First name for the message text (a variable, not a condition).")},
		Def{Name: "user_goal", Type: TypeEnum, Enum: enums.UserGoalValues(), Group: GroupGoal,
			Description: desc("هدف کاربر؛ زیر سرکوب باروری non_ttc", "The user's goal; non_ttc under fertility suppression.")},
	)
	RegisterGroup(a, accountFacts)
	RegisterGroup(GroupGoal, goalFacts)
	RegisterGroup(GroupSegment, segmentFacts)
}

// GroupSegment is the `segment` fact (T-M10-32): its own group, so only a rule that names a segment pays its queries.
const GroupSegment = "segment"

// SegmentSource lists the ids of the saved admin segments whose filters match the user's directory projection row
// (admin/segments.Resolver). An AccountSource that also implements it fills the `segment` fact; without one the fact
// is the empty set.
type SegmentSource interface {
	Segments(ctx context.Context) ([]string, error)
}

func segmentFacts(ctx context.Context, f *Facts) (Values, error) {
	src, ok := f.src.Account.(SegmentSource)
	if !ok || src == nil {
		return Values{"segment": []string{}}, nil
	}
	ids, err := src.Segments(ctx)
	if err != nil {
		return nil, err
	}
	if ids == nil {
		ids = []string{}
	}
	return Values{"segment": ids}, nil
}

// account reads the account row once per request (the account and goal groups share it).
func (f *Facts) account(ctx context.Context) (*Account, error) {
	if f.accountLoaded || f.src.Account == nil {
		return f.accountRow, nil
	}
	a, err := f.src.Account.Account(ctx)
	if err != nil {
		return nil, err
	}
	f.accountRow, f.accountLoaded = a, true
	return a, nil
}

func accountFacts(ctx context.Context, f *Facts) (Values, error) {
	v := Values{"push_opt_in": false, "is_premium": false}
	if f.locale != "" {
		v["locale"] = f.locale
	}
	a, err := f.account(ctx)
	if err != nil || a == nil {
		return v, err
	}
	v["is_premium"] = a.SubscriptionType == "premium"
	if a.Plan != "" {
		v["plan"] = a.Plan
	}
	if !a.SignupDay.IsZero() && !a.SignupDay.After(f.today) {
		v["days_since_signup"] = float64(a.SignupDay.DiffDays(f.today))
	}
	if !a.Birthday.IsZero() && !a.Birthday.After(f.today) {
		v["age"] = float64(a.Birthday.AgeOn(f.today))
	}
	if a.LifeMode != "" {
		v["life_mode"] = a.LifeMode
	}
	if first, _, _ := strings.Cut(strings.TrimSpace(a.Name), " "); first != "" {
		v["name"] = first
	}
	return v, nil
}

func goalFacts(ctx context.Context, f *Facts) (Values, error) {
	a, err := f.account(ctx)
	if err != nil || a == nil || a.UserGoal == "" {
		return nil, err
	}
	goal := a.UserGoal
	if goal == string(enums.UserGoalTtc) {
		if err := f.group(ctx, GroupEngine); err != nil {
			return nil, err
		}
		if s, _ := f.value("fertility_suppressed"); s == true {
			goal = string(enums.UserGoalNonTtc)
		}
	}
	return Values{"user_goal": goal}, nil
}
