// Package extract reads record documents with the AI adapter (canvas-build CB-REC-02, board nbl_Rec_Doc «اطلاعات
// خوانده‌شده از سند — اگر اشتباه است ویرایش کن»): a Plus user who accepted the ai_documents consent sends a document
// of CB-REC-01 (imaging, visit, prescription, hospital, other — lab sheets have internal/labs) to extraction; a
// DB-backed job reads its files through bloom's ai.Client.Extract with the kind's schema; the values land in
// record_documents.extracted as suggestions with a confidence (review_state needs_review) until the user confirms
// them (POST …/review, or CB-REC-01's PUT confirm). Free users fill the fields in by hand (PUT, no extraction).
//
// A confirmed imaging document with a gestational age or a due date, while the user is in pregnancy mode, offers a
// dating update (GET …/dating); the pregnancy is re-dated only by the explicit POST …/dating {confirm: true}, never
// silently, and the document is linked to the pregnancy («سن بارداری از همین سند به‌روز شد»).
//
// Health data and file bytes are never logged: logs carry codes and ids only.
package extract

import (
	"github.com/ritme/backend-go/internal/ai"
	"github.com/ritme/backend-go/internal/healthrecord"
)

// Imaging study kinds (the "kind" field of the imaging schema; the platform fixture's key).
var imagingStudies = []string{"ultrasound", "xray", "mri", "ct", "mammography", "bone_density", "other"}

var (
	fDate   = ai.FieldSpec{Key: "date", Type: ai.FieldDate, Description: "the date of the visit, scan, prescription or admission (Gregorian)"}
	fCentre = ai.FieldSpec{Key: "centre", Type: ai.FieldString, Description: "the clinic, hospital or imaging centre as printed (an institution)"}
	fDoctor = ai.FieldSpec{Key: "doctor", Type: ai.FieldString, Description: "the physician who signed or reported the document, as printed (never the patient)"}
	fDiag   = ai.FieldSpec{Key: "diagnosis", Type: ai.FieldString, Description: "the diagnosis as written"}
	fFind   = ai.FieldSpec{Key: "findings", Type: ai.FieldString, Description: "the findings, impression or advice, short"}
	fReason = ai.FieldSpec{Key: "reason", Type: ai.FieldString, Description: "the reason for the visit or admission"}
)

// Schemas are the extraction schemas per document kind (the schema name picks the fake provider's fixture: the kind,
// "other_document" for kind other).
// Keys never name the person (identity keys are refused by the platform, B-N6-05b).
var Schemas = map[string]ai.ExtractSchema{
	healthrecord.KindImaging: {Name: healthrecord.KindImaging, Fields: []ai.FieldSpec{
		{Key: "kind", Type: ai.FieldEnum, Values: imagingStudies, Description: "the imaging modality"},
		fDate, fCentre, fDoctor,
		{Key: "ga_weeks", Type: ai.FieldNumber, Description: "pregnancy ultrasound only: gestational age at the scan, completed weeks"},
		{Key: "ga_days", Type: ai.FieldNumber, Description: "pregnancy ultrasound only: the extra days of the gestational age (0-6)"},
		{Key: "edd", Type: ai.FieldDate, Description: "pregnancy ultrasound only: the estimated due date as printed (Gregorian)"},
		fFind,
	}},
	healthrecord.KindVisit: {Name: healthrecord.KindVisit, Fields: []ai.FieldSpec{
		fDate, fCentre, fDoctor,
		{Key: "specialty", Type: ai.FieldString, Description: "the physician's specialty"},
		fReason, fDiag, fFind,
		{Key: "next_visit", Type: ai.FieldDate, Description: "the follow-up date if one is written (Gregorian)"},
	}},
	healthrecord.KindPrescription: {Name: healthrecord.KindPrescription, Fields: []ai.FieldSpec{
		fDate, fCentre, fDoctor, fDiag,
	}, Items: []ai.FieldSpec{
		{Key: "medicine", Type: ai.FieldString, Description: "the medicine name and strength as printed"},
		{Key: "dose", Type: ai.FieldString, Description: "the dose per intake"},
		{Key: "frequency", Type: ai.FieldString, Description: "how often"},
		{Key: "duration", Type: ai.FieldString, Description: "for how long"},
	}, MaxItems: MaxItems},
	healthrecord.KindHospital: {Name: healthrecord.KindHospital, Fields: []ai.FieldSpec{
		{Key: "date", Type: ai.FieldDate, Description: "the admission date (Gregorian)"},
		{Key: "ended_on", Type: ai.FieldDate, Description: "the discharge date (Gregorian)"},
		fCentre, fDoctor, fReason, fDiag,
		{Key: "procedures", Type: ai.FieldString, Description: "the surgeries or procedures done"},
		fFind,
	}},
	healthrecord.KindOther: {Name: "other_document", Fields: []ai.FieldSpec{
		fDate, fCentre, fDoctor,
		{Key: "title", Type: ai.FieldString, Description: "what the document is (e.g. medical certificate)"},
		fFind,
	}},
}

// Limits.
const (
	// MaxItems caps the prescription rows kept.
	MaxItems = 30
	// MaxFiles is how many of a document's files (first by position) are sent to extraction.
	MaxFiles = 5
	// MaxText caps a reviewed free-text value (runes); dates / numbers are validated by type.
	MaxText = 500
)

// hint is the context line sent with a document of kind.
func hint(kind string) string {
	return "an Iranian medical document (" + kind + "); it may be in Persian or English; read only what is printed"
}
