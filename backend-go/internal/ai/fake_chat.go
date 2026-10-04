package ai

import (
	"context"
	"fmt"
	"strings"
	"unicode/utf8"
)

// Fake chat (B-N6-05). The reply is a fixture chosen by a "RITME-FAKE:<key>" marker in the last user message
// (tests and e2e type it), otherwise "default" — in the request language, English as fallback. The reply is
// streamed word by word, so SSE rendering can be seen locally. Special keys: "error" fails before streaming,
// "error_mid" fails after three words, "length" ends with FinishLength, "safety" with FinishSafety.
//
// FakeChatReplies are the fixtures by key and language.
var FakeChatReplies = map[string]map[string]string{
	"default": {
		"fa": "این یک پاسخ آزمایشی از دستیار ریتمی است. من پزشک نیستم و تشخیص نمی\u200cدهم؛ اگر علائمت شدید است یا بدتر می\u200cشود، با پزشک مشورت کن.",
		"en": "This is a test answer from the Ritme assistant. I am not a doctor and I don't diagnose; if your symptoms are severe or getting worse, please see a doctor.",
	},
	"self_care": {
		"fa": "درد خفیف قبل از پریود شایع است. گرما روی شکم، نوشیدن آب کافی و کمی پیاده\u200cروی معمولاً کمک می\u200cکند.",
		"en": "Mild pain before a period is common. Warmth on your belly, enough water and a short walk usually help.",
	},
	"emergency": {
		"fa": "این علائم ممکن است اورژانسی باشد. همین حالا با ۱۱۵ تماس بگیر یا به نزدیک\u200cترین اورژانس برو.",
		"en": "These symptoms may be an emergency. Call 115 now or go to the nearest emergency department.",
	},
	"image": {
		"fa": "تصویرت را دیدم. برای نظر دقیق\u200cتر بهتر است پزشک آن را از نزدیک ببیند.",
		"en": "I can see your photo. For a precise opinion it is best that a doctor looks at it in person.",
	},
	"summary": {
		"fa": "خلاصه گفت\u200cوگو: درد خفیف شکم از دو روز پیش، بدون تب. پیشنهاد: استراحت و پیگیری؛ در صورت تشدید مراجعه به پزشک.",
		"en": "Conversation summary: mild abdominal pain for two days, no fever. Suggestion: rest and follow up; see a doctor if it gets worse.",
	},
}

// FakeChatModel is the model name the fake chat reports.
const FakeChatModel = "fixtures-chat-v1"

func fakeChatKey(req ChatRequest) string {
	for i := len(req.Messages) - 1; i >= 0; i-- {
		m := req.Messages[i]
		if m.Role != RoleUser {
			continue
		}
		if k := fakeKey.FindStringSubmatch(m.Text); k != nil {
			return k[1]
		}
		if len(m.Images) > 0 {
			return "image"
		}
		break
	}
	return "default"
}

func fakeReply(key, lang string) string {
	fx, ok := FakeChatReplies[key]
	if !ok {
		fx = FakeChatReplies["default"]
	}
	if t, ok := fx[lang]; ok {
		return t
	}
	return fx["en"]
}

// ChatStream implements Chatter.
func (f *Fake) ChatStream(ctx context.Context, req ChatRequest) (<-chan ChatChunk, error) {
	key := fakeChatKey(req)
	in := utf8.RuneCountInString(req.System)
	for _, m := range req.Messages {
		in += utf8.RuneCountInString(m.Text) + 258*len(m.Images)
	}
	u := Usage{Provider: "fake", Model: FakeChatModel, InputTokens: in / 4}
	if key == "error" {
		return nil, fmt.Errorf("%w: fake error fixture", ErrUpstream)
	}
	words := strings.SplitAfter(fakeReply(key, req.Language), " ")
	out := make(chan ChatChunk)
	go func() {
		defer close(out)
		emit := func(c ChatChunk) bool {
			select {
			case out <- c:
				return true
			case <-ctx.Done():
				return false
			}
		}
		sent := 0
		for i, w := range words {
			if key == "error_mid" && i == 3 {
				u.OutputTokens = sent / 4
				emit(ChatChunk{Err: fmt.Errorf("%w: fake mid-stream error", ErrUpstream), Usage: u})
				return
			}
			if req.MaxOutputTokens > 0 && (sent+utf8.RuneCountInString(w))/4 > req.MaxOutputTokens {
				key = "length"
				break
			}
			if !emit(ChatChunk{Delta: w}) {
				return
			}
			sent += utf8.RuneCountInString(w)
		}
		u.OutputTokens = sent / 4
		finish := FinishStop
		switch key {
		case "length":
			finish = FinishLength
		case "safety":
			finish = FinishSafety
		}
		emit(ChatChunk{Done: true, Finish: finish, Usage: u})
	}()
	return out, nil
}
