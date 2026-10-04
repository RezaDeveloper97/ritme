package ai

import (
	"context"
	"math"
	"regexp"
	"slices"
	"strconv"
	"strings"
	"unicode/utf8"
)

// Vision / document extraction (B-N6-05): an image or PDF in, structured fields with a confidence out. Lab
// analysis (B-N6-06) reads markers as repeated Items; CB-REC-02 reads imaging / prescription documents as Fields.
// The caller owns the schema (which keys, which types); the Client checks the document, sends it with the schema,
// and validates the answer against the schema (unknown keys dropped, values coerced or dropped, confidence
// clamped, strings redacted and capped). Results are suggestions: features keep them `needs_review` until the
// user confirms.

// Document is a file held in memory only. The owner wipes it (Wipe) once extracted.
type Document struct {
	Data []byte
	MIME string // sniffed server-side: image/jpeg | image/png | image/webp | application/pdf
}

// Wipe zeroes the document and drops the reference.
func (d *Document) Wipe() {
	clear(d.Data[:cap(d.Data)])
	d.Data = nil
}

// Field types of an extraction schema.
const (
	FieldString = "string" // free text, ≤ MaxExtractedRunes
	FieldNumber = "number" // a decimal number (float64)
	FieldDate   = "date"   // YYYY-MM-DD (Gregorian; the prompt asks providers to convert Jalali dates)
	FieldEnum   = "enum"   // one of Values
	FieldBool   = "bool"
)

// FieldSpec is one field the caller wants.
type FieldSpec struct {
	Key         string // stable snake_case key ("date", "ga_weeks", "value")
	Type        string // FieldString | FieldNumber | FieldDate | FieldEnum | FieldBool
	Description string // what the field is, for the model ("gestational age, whole weeks")
	Values      []string
}

// ExtractSchema is what to read from a document: single Fields and/or repeated Items (one row per marker).
type ExtractSchema struct {
	Name     string // fixture key for the fake ("lab_panel", "imaging", …) and a label in prompts
	Fields   []FieldSpec
	Items    []FieldSpec
	MaxItems int // 0 → DefaultMaxItems
}

// ExtractRequest is one extraction call.
type ExtractRequest struct {
	Document Document
	Schema   ExtractSchema
	Language string // language of free-text answers
	// Hint is optional context for the model ("an Iranian lab sheet, CBC + thyroid"); redacted like any text.
	Hint string
}

// ExtractedField is one value read from the document.
type ExtractedField struct {
	Key        string
	Value      any // string | float64 | bool, per the field type (dates as "YYYY-MM-DD")
	Confidence float64
}

// Extraction is what was read.
type Extraction struct {
	Fields []ExtractedField
	Items  [][]ExtractedField
}

// Extractor reads documents.
type Extractor interface {
	Extract(ctx context.Context, req ExtractRequest) (Extraction, Usage, error)
}

// Extraction limits enforced by the Client (ErrInvalidRequest).
const (
	MaxDocumentBytes  = 10 << 20
	MaxSchemaFields   = 40
	DefaultMaxItems   = 80
	MaxExtractedRunes = 500
)

var documentMIME = map[string]bool{"image/jpeg": true, "image/png": true, "image/webp": true, "application/pdf": true}

var reKey = regexp.MustCompile(`^[a-z][a-z0-9_]{0,47}$`)

func validateSchema(s ExtractSchema) error {
	if len(s.Fields)+len(s.Items) == 0 || len(s.Fields)+len(s.Items) > MaxSchemaFields || s.MaxItems < 0 {
		return ErrInvalidRequest
	}
	for _, list := range [][]FieldSpec{s.Fields, s.Items} {
		seen := map[string]bool{}
		for _, f := range list {
			if !reKey.MatchString(f.Key) || seen[f.Key] {
				return ErrInvalidRequest
			}
			switch f.Type {
			case FieldString, FieldNumber, FieldDate, FieldBool:
			case FieldEnum:
				if len(f.Values) == 0 {
					return ErrInvalidRequest
				}
			default:
				return ErrInvalidRequest
			}
			seen[f.Key] = true
		}
	}
	return nil
}

// Extract reads req.Document for feature. The document is not wiped here (the caller owns it).
func (c *Client) Extract(ctx context.Context, feature Feature, req ExtractRequest) (Extraction, error) {
	if c == nil || c.extractor == nil {
		return Extraction{}, ErrUnavailable
	}
	if len(req.Document.Data) == 0 || len(req.Document.Data) > MaxDocumentBytes || !documentMIME[req.Document.MIME] {
		return Extraction{}, ErrInvalidRequest
	}
	if err := validateSchema(req.Schema); err != nil {
		return Extraction{}, err
	}
	if err := c.allow(ctx); err != nil {
		return Extraction{}, err
	}
	req.Hint = redactCtx(ctx, req.Hint)
	start := c.now()
	size := len(req.Document.Data)
	out, u, err := c.extractor.Extract(ctx, req)
	u.Feature, u.Op, u.ImageBytes, u.OK = feature, OpExtract, size, err == nil
	c.record(ctx, u, start)
	if err != nil {
		return Extraction{}, err
	}
	names := SubjectFrom(ctx).Names
	return sanitizeExtraction(out, req.Schema, names), nil
}

// sanitizeExtraction keeps only schema keys with values of the right type (first answer per key wins).
func sanitizeExtraction(in Extraction, s ExtractSchema, names []string) Extraction {
	out := Extraction{Fields: sanitizeRow(in.Fields, s.Fields, names), Items: [][]ExtractedField{}}
	maxItems := s.MaxItems
	if maxItems == 0 {
		maxItems = DefaultMaxItems
	}
	if len(s.Items) > 0 {
		for _, row := range in.Items {
			if len(out.Items) == maxItems {
				break
			}
			if r := sanitizeRow(row, s.Items, names); len(r) > 0 {
				out.Items = append(out.Items, r)
			}
		}
	}
	return out
}

func sanitizeRow(row []ExtractedField, specs []FieldSpec, names []string) []ExtractedField {
	out := []ExtractedField{}
	for _, spec := range specs {
		i := slices.IndexFunc(row, func(f ExtractedField) bool { return f.Key == spec.Key })
		if i < 0 {
			continue
		}
		v, ok := coerce(row[i].Value, spec, names)
		if !ok {
			continue
		}
		conf := row[i].Confidence
		if math.IsNaN(conf) {
			conf = 0
		}
		out = append(out, ExtractedField{Key: spec.Key, Value: v, Confidence: math.Max(0, math.Min(1, conf))})
	}
	return out
}

var reDate = regexp.MustCompile(`^\d{4}-\d{2}-\d{2}$`)

func coerce(v any, spec FieldSpec, names []string) (any, bool) {
	switch spec.Type {
	case FieldNumber:
		switch n := v.(type) {
		case float64:
			return n, !math.IsNaN(n) && !math.IsInf(n, 0)
		case int:
			return float64(n), true
		case string:
			f, err := strconv.ParseFloat(strings.TrimSpace(normalizeDigits(n)), 64)
			return f, err == nil && !math.IsNaN(f) && !math.IsInf(f, 0)
		}
	case FieldBool:
		b, ok := v.(bool)
		return b, ok
	case FieldDate:
		s, ok := v.(string)
		s = normalizeDigits(strings.TrimSpace(s))
		return s, ok && reDate.MatchString(s)
	case FieldEnum:
		s, ok := v.(string)
		return s, ok && slices.Contains(spec.Values, s)
	case FieldString:
		var s string
		switch x := v.(type) {
		case string:
			s = x
		case float64:
			s = strconv.FormatFloat(x, 'f', -1, 64)
		default:
			return nil, false
		}
		s = strings.TrimSpace(Redact(s, names))
		if s == "" {
			return nil, false
		}
		if utf8.RuneCountInString(s) > MaxExtractedRunes {
			s = string([]rune(s)[:MaxExtractedRunes])
		}
		return s, true
	}
	return nil, false
}

// normalizeDigits maps Persian / Arabic-Indic digits and the Persian decimal separator to ASCII.
func normalizeDigits(s string) string {
	return strings.Map(func(r rune) rune {
		switch {
		case r >= '۰' && r <= '۹':
			return '0' + (r - '۰')
		case r >= '٠' && r <= '٩':
			return '0' + (r - '٠')
		case r == '٫':
			return '.'
		}
		return r
	}, s)
}
