package ai

// Fake extraction fixtures of the record documents (canvas-build CB-REC-02, internal/healthrecord/extract): one per
// document kind, chosen like every fixture by the schema name (the document kind; "other_document" for kind other) or a "RITME-FAKE:<key>" marker
// inside the file. The imaging fixture is the platform's own ("imaging", fake_extract.go); "imaging_plain" is a scan
// without gestational age (no pregnancy dating offer), "imaging_edd" one with only a printed due date.
//
//	visit         fields: date, centre, doctor, specialty, reason, diagnosis, findings, next_visit
//	prescription  fields: date, centre, doctor, diagnosis          items: medicine, dose, frequency, duration
//	hospital      fields: date, ended_on, centre, doctor, reason, diagnosis, procedures, findings
//	other_document fields: date, centre, doctor, title, findings
func init() {
	FakeExtractions["visit"] = Extraction{Fields: []ExtractedField{
		{Key: "date", Value: "2026-09-15", Confidence: 0.92},
		{Key: "centre", Value: "مطب نمونه", Confidence: 0.74},
		{Key: "doctor", Value: "دکتر نمونه", Confidence: 0.81},
		{Key: "specialty", Value: "زنان و زایمان", Confidence: 0.77},
		{Key: "reason", Value: "پیگیری دوره\u200cای", Confidence: 0.6},
		{Key: "diagnosis", Value: "سندرم تخمدان پلی\u200cکیستیک", Confidence: 0.58},
		{Key: "findings", Value: "ادامهٔ درمان و آزمایش سه ماه بعد", Confidence: 0.55},
		{Key: "next_visit", Value: "2026-12-15", Confidence: 0.66},
	}}
	FakeExtractions["prescription"] = Extraction{
		Fields: []ExtractedField{
			{Key: "date", Value: "2026-09-15", Confidence: 0.9},
			{Key: "centre", Value: "مطب نمونه", Confidence: 0.7},
			{Key: "doctor", Value: "دکتر نمونه", Confidence: 0.85},
			{Key: "diagnosis", Value: "کم\u200cخونی فقر آهن", Confidence: 0.5},
		},
		Items: [][]ExtractedField{
			{
				{Key: "medicine", Value: "Ferrous sulfate 50mg", Confidence: 0.88},
				{Key: "dose", Value: "۱ عدد", Confidence: 0.8},
				{Key: "frequency", Value: "روزی یک بار", Confidence: 0.76},
				{Key: "duration", Value: "۳ ماه", Confidence: 0.7},
			},
			{
				{Key: "medicine", Value: "Folic acid 1mg", Confidence: 0.9},
				{Key: "dose", Value: "۱ عدد", Confidence: 0.82},
				{Key: "frequency", Value: "روزی یک بار", Confidence: 0.8},
			},
		},
	}
	FakeExtractions["hospital"] = Extraction{Fields: []ExtractedField{
		{Key: "date", Value: "2026-08-02", Confidence: 0.9},
		{Key: "ended_on", Value: "2026-08-04", Confidence: 0.86},
		{Key: "centre", Value: "بیمارستان نمونه", Confidence: 0.83},
		{Key: "doctor", Value: "دکتر نمونه", Confidence: 0.7},
		{Key: "reason", Value: "درد شکم", Confidence: 0.66},
		{Key: "diagnosis", Value: "کیست تخمدان", Confidence: 0.61},
		{Key: "procedures", Value: "لاپاراسکوپی", Confidence: 0.64},
		{Key: "findings", Value: "ترخیص با حال عمومی خوب", Confidence: 0.57},
	}}
	FakeExtractions["other_document"] = Extraction{Fields: []ExtractedField{
		{Key: "date", Value: "2026-07-10", Confidence: 0.8},
		{Key: "centre", Value: "مرکز نمونه", Confidence: 0.6},
		{Key: "title", Value: "گواهی پزشکی", Confidence: 0.72},
		{Key: "findings", Value: "استراحت سه روزه", Confidence: 0.5},
	}}
	FakeExtractions["imaging_plain"] = Extraction{Fields: []ExtractedField{
		{Key: "kind", Value: "mammography", Confidence: 0.94},
		{Key: "date", Value: "2026-09-10", Confidence: 0.9},
		{Key: "centre", Value: "مرکز تصویربرداری نمونه", Confidence: 0.82},
		{Key: "doctor", Value: "دکتر نمونه", Confidence: 0.7},
		{Key: "findings", Value: "یافتهٔ غیرطبیعی دیده نشد", Confidence: 0.73},
	}}
	FakeExtractions["imaging_edd"] = Extraction{Fields: []ExtractedField{
		{Key: "kind", Value: "ultrasound", Confidence: 0.95},
		{Key: "date", Value: "2026-09-18", Confidence: 0.9},
		{Key: "edd", Value: "2027-03-30", Confidence: 0.86},
		{Key: "findings", Value: "جنین زنده", Confidence: 0.7},
	}}
}
