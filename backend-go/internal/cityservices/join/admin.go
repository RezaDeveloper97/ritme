package join

import (
	"slices"

	"github.com/ritme/backend-go/internal/cityservices/join/store"
)

// CB-DIR-05: the Ritme team's side of an application (admin API in internal/cityservices, admin_applications.go).
// Only the pieces the admin handlers need are exported here; the applicant service above stays owner-scoped.

// Decode is decodeRow for the admin review queue (JSON columns parsed, a malformed column reads as empty).
func Decode(r store.CityServiceApplication) Application { return decodeRow(r) }

// reviewMoves is the team's status machine. approved is reached through Approve (it creates the listing), never
// through a plain move; needs_info hands the application back to the applicant, who resubmits (→ submitted).
var reviewMoves = map[string][]string{
	StatusSubmitted: {StatusInReview, StatusNeedsInfo, StatusRejected},
	StatusInReview:  {StatusNeedsInfo, StatusRejected},
	StatusNeedsInfo: {StatusRejected},
}

// ReviewMoves lists the plain moves allowed from status (never nil).
func ReviewMoves(status string) []string {
	if m := reviewMoves[status]; m != nil {
		return m
	}
	return []string{}
}

// CanReview reports whether the team may move an application from one status to another (not approval).
func CanReview(from, to string) bool { return slices.Contains(reviewMoves[from], to) }

// CanApprove reports whether an application in status may be approved (published as a listing).
func CanApprove(status string) bool { return status == StatusSubmitted || status == StatusInReview }

// NoteRequired reports whether a move must carry a note for the applicant (what to fix, or why it was declined).
func NoteRequired(to string) bool { return to == StatusNeedsInfo || to == StatusRejected }
