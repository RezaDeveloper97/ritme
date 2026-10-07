package companion

import (
	"context"

	"github.com/ritme/backend-go/internal/companion/store"
)

// Family scope of the health record (canvas-build CB-REC-03, nbl_Rec_Share «همدم و خانواده — داروها و نوبت‌ها · بدون
// دسترسی به اسناد»). A companion link reaches the record only through the regular meds and appointments grants
// (internal/care for_user_id views, companion/shared section views) — never the record summary, its documents or
// their files: no companion section maps to them (RecordSections is closed, Section.Valid refuses anything else), and
// every /health-record/* route is owner-only (it reads the authenticated user's own rows and ignores for_user_id).
// The guard tests in record_scope_test.go and internal/http keep it that way.

// RecordSections are the only record parts a companion grant can cover, in board order.
var RecordSections = []Section{SectionMeds, SectionAppointments}

// RecordAccess is one link as the owner's «چه کسی پرونده‌ات را می‌بیند؟» lists it: the effective meds / appointments
// levels (none while the link is not active or teen-suspended) and, always, no documents.
type RecordAccess struct {
	CompanionID  uint64
	Type         Type
	Status       Status
	Name         string // the owner's display name for the link, else the companion account's name ("" = none)
	Meds         Level
	Appointments Level
}

// RecordScope lists ownerID's invited and active links with their record scope, oldest first.
func (s *Service) RecordScope(ctx context.Context, ownerID uint64) ([]RecordAccess, error) {
	links, err := s.ListForOwner(ctx, ownerID)
	if err != nil {
		return nil, err
	}
	ids := make([]uint64, 0, len(links))
	for _, l := range links {
		ids = append(ids, l.CompanionUserID)
	}
	names, err := s.Names(ctx, ids...)
	if err != nil {
		return nil, err
	}
	teen, err := ownerIsTeen(ctx, store.New(s.conn), ownerID)
	if err != nil {
		return nil, err
	}
	out := make([]RecordAccess, 0, len(links))
	for _, l := range links {
		if l.Status == StatusRevoked {
			continue
		}
		a := RecordAccess{CompanionID: l.ID, Type: l.Type, Status: l.Status, Name: l.DisplayName,
			Meds: LevelNone, Appointments: LevelNone}
		if a.Name == "" {
			a.Name = names[l.CompanionUserID]
		}
		suspended := teen && l.Type != TypeParent // B-N4-08b: a teen owner's partner / spouse link grants nothing
		effective := l.Status == StatusActive && !suspended
		if effective {
			for _, sec := range RecordSections {
				lv := l.Grants.Of(sec)
				if !sec.AllowedFor(l.Type) {
					lv = LevelNone
				}
				if sec == SectionMeds {
					a.Meds = lv
				} else {
					a.Appointments = lv
				}
			}
		}
		out = append(out, a)
	}
	return out, nil
}
