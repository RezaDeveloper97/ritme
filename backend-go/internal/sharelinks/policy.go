package sharelinks

import (
	"context"
	"errors"
	"fmt"

	"github.com/gofiber/fiber/v3"

	"github.com/ritme/backend-go/internal/healthrecord"
	"github.com/ritme/backend-go/internal/i18n"
	"github.com/ritme/backend-go/internal/platform/httpx"
	"github.com/ritme/backend-go/internal/platform/jsonx"
	"github.com/ritme/backend-go/internal/platform/validation/phpval"
)

// Two user decisions of 2026-10-12 (docs/security/canvas-record-review.md, CB-REC-06 M2 and L5):
//
//   - Loss opt-in: a share (7-day report link, 24h doctor code) carries no pregnancy loss unless the owner switched on
//     «شامل سابقه‌ی از دست دادن بارداری» for that one share (`include_losses`, stored on the link). Even then a loss is
//     only a neutral «ended» row. The choice is stored with the link and the public read re-applies it to the
//     decrypted snapshot (stripLosses), so a snapshot can never show more than the owner chose. The emergency card,
//     the companion (family) scope and the assistant never carry losses.
//   - Teen sharing: a teen-mode account may make a doctor code or a report link only while she has an active parent
//     link (companion type `parent`, CB-TEEN-01). Without one, both creates are 403 teen_share_needs_parent. Her
//     emergency card is always available (internal/emergency is not gated).

// CodeTeenNeedsParent is the 403 error code of a doctor code / report link create by a teen without a parent link.
const CodeTeenNeedsParent = "teen_share_needs_parent"

// ErrTeenNeedsParent: a teen-mode owner without an active parent link may not create a doctor code or a report link.
var ErrTeenNeedsParent = errors.New("sharelinks: a teen needs an active parent link to share with a doctor")

// SharingPolicy is what the owner may share with a doctor right now.
type SharingPolicy struct {
	Teen         bool // her stored life mode is teen
	ParentLinked bool // she has an active parent link
}

// DoctorSharing reports whether doctor codes and report links are allowed (always for an adult; for a teen only with
// an active parent link).
func (p SharingPolicy) DoctorSharing() bool { return !p.Teen || p.ParentLinked }

// JSON is {teen, parent_linked, doctor_sharing, emergency_card} (the emergency card is always available).
func (p SharingPolicy) JSON() *jsonx.OrderedMap {
	return jsonx.Obj("teen", p.Teen, "parent_linked", p.ParentLinked, "doctor_sharing", p.DoctorSharing(),
		"emergency_card", true)
}

// SharingGate answers the teen sharing question (internal/teen.Policy through an adapter in internal/http).
type SharingGate interface {
	Sharing(ctx context.Context, userID uint64) (SharingPolicy, error)
}

// WithSharingGate wires the teen sharing policy (nil = every owner is treated as an adult; unit wiring only).
func (s *Service) WithSharingGate(g SharingGate) *Service { s.gate = g; return s }

// Policy is the sharing policy of userID.
func (s *Service) Policy(ctx context.Context, userID uint64) (SharingPolicy, error) {
	if s.gate == nil {
		return SharingPolicy{}, nil
	}
	p, err := s.gate.Sharing(ctx, userID)
	if err != nil {
		return SharingPolicy{}, fmt.Errorf("sharelinks: sharing policy: %w", err)
	}
	return p, nil
}

// requireDoctorSharing is ErrTeenNeedsParent for a teen without an active parent link.
func (s *Service) requireDoctorSharing(ctx context.Context, userID uint64) error {
	p, err := s.Policy(ctx, userID)
	if err != nil {
		return err
	}
	if !p.DoctorSharing() {
		return ErrTeenNeedsParent
	}
	return nil
}

// RequireDoctorSharing is route middleware for POST /health-record/share-links, mounted before the Plus gate: a teen
// without an active parent link gets the 403 explanation, never a Plus paywall (teens see no Plus upsell, CB-TEEN-01).
// The service checks again on create.
func (h *Handlers) RequireDoctorSharing(c fiber.Ctx) error {
	userID, err := user(c)
	if err != nil {
		return err
	}
	p, err := h.svc.Policy(c, userID)
	if err != nil {
		return err
	}
	if !p.DoctorSharing() {
		return teenNeedsParent(i18n.Locale(c))
	}
	return c.Next()
}

func teenNeedsParent(locale string) error {
	return httpx.Fail(fiber.StatusForbidden, T("messages.teen_needs_parent", locale), "error_code", CodeTeenNeedsParent)
}

// stripLosses drops the loss-derived rows (tracked «ended» pregnancies) from a decrypted snapshot whose link was made
// without the loss opt-in, and recounts the section. Manual «ended» rows the owner typed in herself stay.
func stripLosses(report phpval.Map) {
	rec, _ := report.Get("record")
	recMap, ok := rec.(phpval.Map)
	if !ok {
		return
	}
	secs, _ := recMap.Get("sections")
	list, ok := secs.([]any)
	if !ok {
		return
	}
	for _, x := range list {
		sec, ok := x.(phpval.Map)
		if !ok {
			continue
		}
		if key, _ := sec.Get("key"); key != healthrecord.SectionPregnancies {
			continue
		}
		raw, _ := sec.Get("data")
		data, ok := raw.(phpval.Map)
		if !ok {
			continue
		}
		rawItems, _ := data.Get("items")
		items, _ := rawItems.([]any)
		kept := make([]any, 0, len(items))
		for _, it := range items {
			if m, ok := it.(phpval.Map); ok {
				src, _ := m.Get("source")
				out, _ := m.Get("outcome")
				if src == healthrecord.SourceTracked && out == healthrecord.OutcomeEnded {
					continue
				}
			}
			kept = append(kept, it)
		}
		if len(kept) == len(items) {
			continue
		}
		data.Set("items", kept)
		data.Set("pregnancies_count", int64(len(kept)))
		sec.Set("empty", len(kept) == 0)
	}
}
