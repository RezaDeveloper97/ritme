package invites

import "errors"

// Refusals of a code (or of a pending acceptance). Every one of them is answered with the same generic 422
// invite_invalid (IsRefusal), so neither preview nor accept nor decline tells whether a code exists.
var (
	// ErrInvalid is a malformed or unknown code.
	ErrInvalid = errors.New("family invites: invalid invite code")
	// ErrExpired is an invite past its expiry.
	ErrExpired = errors.New("family invites: invite expired")
	// ErrUsed is a used, revoked or no longer pending invite.
	ErrUsed = errors.New("family invites: invite already used")
	// ErrLocked is a code invite locked after MaxAttempts wrong-account tries.
	ErrLocked = errors.New("family invites: invite locked")
	// ErrPhoneMismatch is a phone-bound invite redeemed by an account with another mobile.
	ErrPhoneMismatch = errors.New("family invites: invite is for another number")
	// ErrSelf is the owner's own code (or her own number on create).
	ErrSelf = errors.New("family invites: cannot invite yourself")
	// ErrAlreadyLinked is a member who already has an open link from this owner.
	ErrAlreadyLinked = errors.New("family invites: already linked")
	// ErrNotFound is an unknown link or one that is not the caller's.
	ErrNotFound = errors.New("family invites: not found")
)

// ErrNotAllowed: the owner's life mode no longer allows the link (CB-TEEN-01); not a failed attempt.
var ErrNotAllowed = errors.New("family invites: invite not allowed for the owner's mode")

// Create / answer errors (422 on a field or with an error_code).
var (
	ErrInvalidRole        = errors.New("family invites: invalid role")
	ErrInvalidTemplate    = errors.New("family invites: invalid template")
	ErrInvalidPhone       = errors.New("family invites: invalid phone")
	ErrInvalidName        = errors.New("family invites: invalid display name")
	ErrInvalidExpiry      = errors.New("family invites: expires_at must be in the future")
	ErrInvalidChildren    = errors.New("family invites: children level not allowed for the role")
	ErrMutualNotAllowed   = errors.New("family invites: mutual request domain not allowed")
	ErrMutualNotRequested = errors.New("family invites: mutual answer outside the request")
	ErrMutualExists       = errors.New("family invites: the member already shares with the owner")
	ErrConsentVersion     = errors.New("family invites: consent text version is not the current one")
	ErrTooMany            = errors.New("family invites: too many open links")
	ErrTeenParentOnly     = errors.New("family invites: a teen may only invite a parent")
)

// IsRefusal reports whether err is one of the generic code refusals.
func IsRefusal(err error) bool {
	for _, e := range []error{ErrInvalid, ErrExpired, ErrUsed, ErrLocked, ErrPhoneMismatch, ErrSelf, ErrAlreadyLinked,
		ErrNotFound} {
		if errors.Is(err, e) {
			return true
		}
	}
	return false
}
