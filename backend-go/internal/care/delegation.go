package care

import (
	"context"
	"strconv"

	"github.com/gofiber/fiber/v3"

	"github.com/ritme/backend-go/internal/companion"
	"github.com/ritme/backend-go/internal/i18n"
	"github.com/ritme/backend-go/internal/platform/httpx"
	"github.com/ritme/backend-go/internal/platform/validation"
	"github.com/ritme/backend-go/internal/platform/validation/phpval"
)

// «ثبت برای …» (bloom B-N4-02, Hamdam_RecordFor): a companion with `edit` on the owner's meds / appointments may
// create and edit them in the owner's list, and with `view` (or edit) open one of them. The request names the owner
// in ForUserField (body for POST / PUT, query for GET); without it, or naming oneself, everything is as before.
// Deletes, intake ticks, cancel and the prep checklist stay owner-only (a delegated appointment update drops `prep`).
// A companion with edit may change every other field of the owner's record, including the `is_active` / `notify`
// switches (pausing or resuming her reminder) — that is what «دیدن و ویرایش» grants.
// Every delegated access is audited before the data is touched (fail closed); the owner's notice after a write is
// best effort.

// ForUserField is the request key naming the account the record belongs to.
const ForUserField = "for_user_id"

// ErrorCodeCompanionForbidden is the 403's error_code when the caller has no grant on that owner's section.
const ErrorCodeCompanionForbidden = "companion_forbidden"

// Delegation decides and records companion access (companion.Delegation implements it).
type Delegation interface {
	// Authorize reports whether actorID may read (write=false) or write the section of ownerID's data and, when
	// allowed, records the access in the audit trail first; an error means the request must not proceed.
	Authorize(ctx context.Context, ownerID, actorID uint64, section companion.Section, write bool) (bool, error)
	// Notify tells the owner about a successful write (created: a new record, else an edit). Best effort, no error.
	Notify(ctx context.Context, ownerID, actorID uint64, section companion.Section, created bool, languages []string)
}

// SetDelegation enables ForUserField. Without it a request naming another account is refused with the 403.
func (h *Handlers) SetDelegation(d Delegation) { h.delegation = d }

// subject is the request's target account: the caller, or the owner named in ForUserField when the caller holds
// the grant. A malformed id is a 422; a missing grant (or an unknown account) the same 403, so the answer does not
// reveal whether the account exists.
func (h *Handlers) subject(c fiber.Ctx, section companion.Section, write bool) (subjectID, actorID uint64, err error) {
	actorID, err = h.user(c)
	if err != nil {
		return 0, 0, err
	}
	src := validation.Input(c)
	if !write {
		src = validation.Query(c)
	}
	raw, ok := src.Get(ForUserField)
	if !ok || raw == nil || phpval.ToString(raw) == "" {
		return actorID, actorID, nil
	}
	locale := i18n.Locale(c)
	owner, perr := strconv.ParseUint(phpval.ToString(raw), 10, 64)
	if perr != nil || owner == 0 {
		return 0, 0, fieldError(locale, ForUserField, T("validation.for_user_invalid", locale))
	}
	if owner == actorID {
		return actorID, actorID, nil
	}
	if h.delegation == nil {
		return 0, 0, companionForbidden(locale)
	}
	allowed, err := h.delegation.Authorize(c, owner, actorID, section, write)
	if err != nil {
		return 0, 0, err
	}
	if !allowed {
		return 0, 0, companionForbidden(locale)
	}
	return owner, actorID, nil
}

// delegated notifies the owner of a successful write on her data (no-op for one's own data and for reads, which
// subject already audited).
func (h *Handlers) delegated(c fiber.Ctx, subjectID, actorID uint64, section companion.Section, write, created bool) error {
	if subjectID == actorID || h.delegation == nil || !write {
		return nil
	}
	h.delegation.Notify(c, subjectID, actorID, section, created, i18n.LanguagesOf(c).Codes())
	return nil
}

func companionForbidden(locale string) error {
	return httpx.Fail(fiber.StatusForbidden, T("messages.companion_forbidden", locale), "error_code", ErrorCodeCompanionForbidden)
}
