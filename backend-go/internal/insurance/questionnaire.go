package insurance

import (
	"encoding/json"

	"github.com/ritme/backend-go/internal/platform/civildate"
)

// Question is one yes/no question of the health questionnaire; the client labels it by code.
type Question struct{ Code, Section string }

// Sections are the questionnaire steps in order («بخش ۲ از ۴ · سوابق بیماری»); the last one is the review step.
var Sections = []string{"general", "history", "family", "review"}

// Questions are the questionnaire rows in board order.
var Questions = []Question{
	{"smoker", "general"},
	{"disability", "general"},
	{"hospitalized_5y", "history"},
	{"chronic_disease", "history"},
	{"pregnant", "history"},
	{"gyn_surgery", "history"},
	{"regular_medication", "history"},
	{"infertility_treatment", "history"},
	{"family_history", "family"},
}

// Answer is one stored answer: hers, the record's suggestion and the record items behind it.
type Answer struct {
	Code         string   `json:"code"`
	Answer       *string  `json:"answer"`
	RecordAnswer *string  `json:"record_answer"`
	Evidence     []string `json:"evidence,omitempty"` // sealed in evidence_enc at rest (00093); plain only in rows written before it
}

// FromRecord reports whether her answer is the one the record suggested («از پرونده سلامت»).
func (a Answer) FromRecord() bool {
	return a.Answer != nil && a.RecordAnswer != nil && *a.Answer == *a.RecordAnswer
}

// RecordFacts are the owner's record items the prefill reads (nil slices = never answered there).
type RecordFacts struct {
	HasLifeProfile bool
	LifeMode       string
	IVFOrIUI       bool
	OpenIVF        bool
	Chronic        []string
	Medications    []string
	MenoSurgical   bool
	Surgeries      []Surgery
	FamilyHistory  []string
	PregnancyMode  *bool
	Cesareans      []civildate.Date
	HospitalDocs   []string // titles (or "") of hospital documents in the last five years
}

// Surgery is one record «بستری و جراحی» entry.
type Surgery struct {
	Title string `json:"title"`
	Date  string `json:"date"`
}

func yes() *string { s := AnswerYes; return &s }
func no() *string  { s := AnswerNo; return &s }

// Prefill builds the draft answers from the record on `today`. A question the record says nothing about stays
// unanswered (nil): she answers it herself.
func Prefill(f RecordFacts, today civildate.Date) []Answer {
	since := today.AddDays(-5 * 365)
	out := make([]Answer, 0, len(Questions))
	for _, q := range Questions {
		a := Answer{Code: q.Code, Evidence: []string{}}
		switch q.Code {
		case "hospitalized_5y":
			for _, s := range f.Surgeries {
				if d, err := civildate.Parse(s.Date); err == nil && !d.Before(since) {
					a.Evidence = append(a.Evidence, joinDated(s.Title, s.Date))
				}
			}
			for _, d := range f.Cesareans {
				if !d.Before(since) {
					a.Evidence = append(a.Evidence, joinDated("cesarean", d.String()))
				}
			}
			for _, t := range f.HospitalDocs {
				if t != "" {
					a.Evidence = append(a.Evidence, t)
				}
			}
			switch {
			case len(a.Evidence) > 0:
				a.RecordAnswer = yes()
			case f.Surgeries != nil:
				a.RecordAnswer = no()
			}
		case "chronic_disease":
			a.RecordAnswer, a.Evidence = listAnswer(f.Chronic)
		case "regular_medication":
			a.RecordAnswer, a.Evidence = listAnswer(f.Medications)
		case "family_history":
			a.RecordAnswer, a.Evidence = listAnswer(f.FamilyHistory)
		case "pregnant":
			switch {
			case (f.PregnancyMode != nil && *f.PregnancyMode) || f.LifeMode == "pregnancy":
				a.RecordAnswer = yes()
			case f.PregnancyMode != nil || f.HasLifeProfile:
				a.RecordAnswer = no()
			}
		case "gyn_surgery":
			for _, d := range f.Cesareans {
				a.Evidence = append(a.Evidence, joinDated("cesarean", d.String()))
			}
			if f.MenoSurgical {
				a.Evidence = append(a.Evidence, "surgical_menopause")
			}
			if len(a.Evidence) > 0 {
				a.RecordAnswer = yes()
			}
		case "infertility_treatment":
			switch {
			case f.OpenIVF || f.IVFOrIUI:
				a.RecordAnswer = yes()
				a.Evidence = append(a.Evidence, "ivf_iui")
			case f.HasLifeProfile:
				a.RecordAnswer = no()
			}
		}
		if a.RecordAnswer != nil {
			v := *a.RecordAnswer
			a.Answer = &v
		}
		out = append(out, a)
	}
	return out
}

func joinDated(title, date string) string {
	if date == "" {
		return title
	}
	return title + " · " + date
}

func listAnswer(items []string) (*string, []string) {
	if items == nil {
		return nil, []string{}
	}
	if len(items) == 0 {
		return no(), []string{}
	}
	return yes(), items
}

// DecodeAnswers reads the stored answers, aligned to Questions (unknown codes dropped, missing ones unanswered).
func DecodeAnswers(raw []byte) []Answer {
	var stored []Answer
	_ = json.Unmarshal(raw, &stored)
	byCode := map[string]Answer{}
	for _, a := range stored {
		byCode[a.Code] = a
	}
	out := make([]Answer, 0, len(Questions))
	for _, q := range Questions {
		a, ok := byCode[q.Code]
		if !ok {
			a = Answer{Code: q.Code}
		}
		if a.Evidence == nil {
			a.Evidence = []string{}
		}
		out = append(out, a)
	}
	return out
}

// Complete reports whether every question has an answer.
func Complete(as []Answer) bool {
	for _, a := range as {
		if a.Answer == nil {
			return false
		}
	}
	return len(as) == len(Questions)
}

// SectionOf is the step a question code belongs to.
func SectionOf(code string) string {
	for _, q := range Questions {
		if q.Code == code {
			return q.Section
		}
	}
	return ""
}

// TimelineStep is one step of the questionnaire status she maintains after confirming.
type TimelineStep struct {
	Status string `json:"status"`
	On     string `json:"on"`
}

// DecodeTimeline reads the stored timeline.
func DecodeTimeline(raw []byte) []TimelineStep {
	out := []TimelineStep{}
	if len(raw) > 0 {
		_ = json.Unmarshal(raw, &out)
	}
	if out == nil {
		out = []TimelineStep{}
	}
	return out
}
