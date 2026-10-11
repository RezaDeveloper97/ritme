package partners

import (
	"context"
	"errors"
	"strings"

	"github.com/ritme/backend-go/internal/platform/civildate"
)

// Submission is the whole payload a partner receives: the allow-listed form fields and the request reference. It
// carries no user id, account mobile, e-mail, health data or profile. Adding a field here is a privacy decision (the
// allow-list test pins the names).
type Submission struct {
	Reference     string
	ServiceCode   string
	FullName      string
	Phone         string
	City          string
	Address       string
	PreferredDate civildate.Date
	Window        string
	Note          string
}

// ErrNoUpdate is Status's answer when the partner has nothing new.
var ErrNoUpdate = errors.New("partners: no status update")

// LabPartner is the adapter of one lab partner (PartnerCode of the catalog service). Real partners plug in here; their
// secrets come from env only.
type LabPartner interface {
	Name() string
	// Submit hands the request over and returns the partner's reference.
	Submit(ctx context.Context, s Submission) (partnerRef string, err error)
	// Status is the partner's current status (one of the Status* values), or ErrNoUpdate.
	Status(ctx context.Context, partnerRef string) (string, error)
	// Cancel withdraws a request the user cancelled.
	Cancel(ctx context.Context, partnerRef string) error
	// FetchResult is the result sheet (a PDF) of a request whose status is result_ready.
	FetchResult(ctx context.Context, partnerRef string) ([]byte, error)
}

// ManualStatus marks an adapter whose statuses an admin may advance by hand (the fake partner).
type ManualStatus interface{ ManualStatus() bool }

// StatusLookup reads what the (fake) partner currently reports for a partner reference.
type StatusLookup func(ctx context.Context, partnerRef string) (string, error)

// Fake is the default adapter for local / stage: it accepts everything, reports the status an admin set in the queue
// (the console plays the partner) and returns a canned PDF. It never contacts anything.
type Fake struct{ Lookup StatusLookup }

// FakeRefPrefix starts every fake partner reference.
const FakeRefPrefix = "FAKE-"

// Name implements LabPartner.
func (Fake) Name() string { return DefaultPartner }

// ManualStatus implements ManualStatus.
func (Fake) ManualStatus() bool { return true }

// Submit implements LabPartner.
func (Fake) Submit(_ context.Context, s Submission) (string, error) {
	return FakeRefPrefix + strings.ToUpper(s.Reference), nil
}

// Status implements LabPartner.
func (f Fake) Status(ctx context.Context, ref string) (string, error) {
	if f.Lookup == nil {
		return "", ErrNoUpdate
	}
	st, err := f.Lookup(ctx, ref)
	if err != nil {
		return "", err
	}
	if st == "" {
		return "", ErrNoUpdate
	}
	return st, nil
}

// Cancel implements LabPartner.
func (Fake) Cancel(context.Context, string) error { return nil }

// FetchResult implements LabPartner: a tiny canned PDF, never real data.
func (Fake) FetchResult(context.Context, string) ([]byte, error) { return []byte(CannedPDF), nil }

// CannedPDF is the fake partner's result sheet.
const CannedPDF = "%PDF-1.4\n1 0 obj<</Type/Catalog/Pages 2 0 R>>endobj\n2 0 obj<</Type/Pages/Kids[3 0 R]/Count 1>>endobj\n" +
	"3 0 obj<</Type/Page/Parent 2 0 R/MediaBox[0 0 200 100]>>endobj\ntrailer<</Root 1 0 R>>\n%%EOF\n"
