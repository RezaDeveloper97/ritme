package facts

import "context"

// Challenge facts (T-M10-10, SPEC §5.5, D-167): challenge_id is not a fact of the user. The daily_challenge slot
// (messages/feed.PickChallenge) selects the slot once per candidate challenge with challenge_id set to it, so a rule
// that tests challenge_id promotes that challenge. Everywhere else the fact is absent (the producer has nothing), so a
// rule on it matches only through ne / nin.

// GroupChallenge is the challenge producer group.
const GroupChallenge = "challenge"

func init() {
	Register(Def{Name: "challenge_id", Type: TypeInt, Group: GroupChallenge,
		Description: desc("شناسه چالش نامزد در اسلات «چالش امروز» (فقط برای جلو انداختن یک چالش)",
			"The candidate challenge's id in the «today's challenge» slot (only to promote a challenge).")})
	RegisterGroup(GroupChallenge, func(context.Context, *Facts) (Values, error) { return nil, nil })
}
