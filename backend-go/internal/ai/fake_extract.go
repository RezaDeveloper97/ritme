package ai

import (
	"bytes"
	"context"
	"fmt"
)

// Fake extraction (B-N6-05). The fixture is chosen by a "RITME-FAKE:<key>" marker inside the document bytes
// (tests craft such files), otherwise by the schema name ("lab_panel", "imaging"), otherwise "blank" (nothing
// read). Special keys: "error" fails with ErrUpstream, "blurry" is lab_panel with low confidence everywhere.
// Only the keys the caller's schema asks for come back (the Client drops the rest), so downstream schemas use the
// fixture keys below to get values:
//
//	lab_panel fields: date, lab_name                       items: marker, value, unit, ref_low, ref_high, ref_text
//	imaging   fields: kind, date, centre, doctor, ga_weeks, ga_days, edd, findings
//
// (B-N6-05b: "name" is an identity key the Client refuses in schemas, so the marker column is "marker".)
var FakeExtractions = map[string]Extraction{
	"lab_panel": {
		Fields: []ExtractedField{
			{Key: "date", Value: "2026-09-20", Confidence: 0.93},
			{Key: "lab_name", Value: "آزمایشگاه نمونه", Confidence: 0.71},
		},
		Items: [][]ExtractedField{
			labRow("Hemoglobin", 11.4, "g/dL", 12, 16, 0.95),
			labRow("Ferritin", 9, "ng/mL", 15, 150, 0.9),
			labRow("TSH", 2.1, "mIU/L", 0.4, 4.0, 0.92),
			labRow("Vitamin D (25-OH)", 14, "ng/mL", 30, 100, 0.88),
			labRow("FBS", 92, "mg/dL", 70, 100, 0.62),
		},
	},
	"imaging": {
		Fields: []ExtractedField{
			{Key: "kind", Value: "ultrasound", Confidence: 0.96},
			{Key: "date", Value: "2026-09-18", Confidence: 0.9},
			{Key: "centre", Value: "مرکز تصویربرداری نمونه", Confidence: 0.8},
			{Key: "doctor", Value: "دکتر نمونه", Confidence: 0.7},
			{Key: "ga_weeks", Value: 12.0, Confidence: 0.9},
			{Key: "ga_days", Value: 3.0, Confidence: 0.85},
			{Key: "edd", Value: "2027-04-05", Confidence: 0.88},
			{Key: "findings", Value: "جنین زنده با ضربان قلب طبیعی", Confidence: 0.75},
		},
	},
	"blank": {},
}

func labRow(marker string, value float64, unit string, low, high, conf float64) []ExtractedField {
	return []ExtractedField{
		{Key: "marker", Value: marker, Confidence: conf},
		{Key: "value", Value: value, Confidence: conf},
		{Key: "unit", Value: unit, Confidence: conf},
		{Key: "ref_low", Value: low, Confidence: conf},
		{Key: "ref_high", Value: high, Confidence: conf},
		{Key: "ref_text", Value: fmt.Sprintf("%g–%g", low, high), Confidence: conf},
	}
}

// FakeExtractModel is the model name the fake extraction reports.
const FakeExtractModel = "fixtures-extract-v1"

// Extract implements Extractor.
func (f *Fake) Extract(_ context.Context, req ExtractRequest) (Extraction, Usage, error) {
	u := Usage{Provider: "fake", Model: FakeExtractModel, InputTokens: 258 + len(req.Document.Data)/4096}
	key := req.Schema.Name
	if bytes.Contains(req.Document.Data, []byte(FakeMarker)) {
		if m := fakeKey.FindSubmatch(req.Document.Data); m != nil {
			key = string(m[1])
		}
	}
	if key == "error" {
		return Extraction{}, u, fmt.Errorf("%w: fake error fixture", ErrUpstream)
	}
	blurry := key == "blurry"
	if blurry {
		key = "lab_panel"
	}
	fx, ok := FakeExtractions[key]
	if !ok {
		fx = FakeExtractions["blank"]
	}
	out := Extraction{Fields: copyRow(fx.Fields, blurry), Items: make([][]ExtractedField, 0, len(fx.Items))}
	for _, r := range fx.Items {
		out.Items = append(out.Items, copyRow(r, blurry))
	}
	u.OutputTokens = 20 * (len(out.Fields) + 6*len(out.Items))
	return out, u, nil
}

func copyRow(r []ExtractedField, blurry bool) []ExtractedField {
	out := append([]ExtractedField(nil), r...)
	if blurry {
		for i := range out {
			out[i].Confidence = 0.3
		}
	}
	return out
}
