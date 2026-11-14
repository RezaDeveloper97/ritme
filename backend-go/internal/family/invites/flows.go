package invites

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"slices"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/ritme/backend-go/internal/companion/store"
	"github.com/ritme/backend-go/internal/enums"
	"github.com/ritme/backend-go/internal/family"
)

// Channels of an invite.
const (
	ChannelCode  = "code"
	ChannelPhone = "phone"
)

// childrenLimited is the carer's children level (feeding / diaper / sleep sub-scopes only; family.md §2).
const childrenLimited = "limited"

// MutualDomains are the member's own domains an owner may ask for in a mutual request (SPEC §1.7: «برای نوبت‌ها و
// داروهای خود او»).
var MutualDomains = []family.Domain{family.DomainAppointments, family.DomainMeds}

// Roles a v6 invite may name (the teen parent card stays on /companions, F-17).
var Roles = []family.Role{family.RolePartner, family.RoleOtherParent, family.RoleCarer, family.RoleFriend}

// Templates a v6 invite may name (SPEC §1.7).
var Templates = []family.Template{family.TemplateCycleBuddy, family.TemplateMedsAppointments, family.TemplateFullFamily,
	family.TemplateCustom}

// CreateInput is POST /family/members: relation, template, row edits, expiry, urgent switch, mutual request, phone or
// code, and the consent text version the owner saw.
type CreateInput struct {
	Role     family.Role
	Template family.Template
	// Edits override the template's rows (LevelOff removes a row).
	Edits map[family.Domain]family.Level
	// Children is the child-profile level the invite asks for: "" (template default), off or write (limited for a
	// carer). Guardianship itself is granted by T-M11-11; here it only decides the owner confirmation (F-11).
	Children string
	// ExpiresAt is the link's end (zero = none, a carer's default F-13).
	ExpiresAt time.Time
	// UrgentAlert overrides the urgent-alert row when set; UrgentSMS is the per-member neutral SMS switch (F-09).
	UrgentAlert *bool
	UrgentSMS   bool
	// Mutual are the member's own domains the owner asks her to share back (optional).
	Mutual         []family.Domain
	Phone          string
	DisplayName    string
	ConsentVersion int
}

// DomainLevel is one granted row.
type DomainLevel struct {
	Domain family.Domain
	Level  family.Level
}

// Created is the outcome of Create. Code is the only place the plain code exists.
type Created struct {
	LinkID        uint64
	OwnerID       uint64
	Role          family.Role
	Template      family.Template
	Status        string
	DisplayName   string
	LinkExpiresAt time.Time
	UrgentSMS     bool
	Grants        []DomainLevel
	Children      string
	Mutual        []family.Domain
	Code          string
	Phone         string
	ExpiresAt     time.Time
}

func orderedGrants(set map[family.Domain]family.Level) []DomainLevel {
	out := []DomainLevel{}
	for _, d := range family.GrantDomains {
		if l := set[d]; l > family.LevelOff {
			out = append(out, DomainLevel{Domain: d, Level: l})
		}
	}
	return out
}

func validateCreate(in *CreateInput, now time.Time) error {
	if !slices.Contains(Roles, in.Role) {
		return ErrInvalidRole
	}
	if !slices.Contains(Templates, in.Template) {
		return ErrInvalidTemplate
	}
	in.DisplayName = strings.TrimSpace(in.DisplayName)
	if utf8.RuneCountInString(in.DisplayName) > maxDisplayName {
		return ErrInvalidName
	}
	if in.Phone != "" {
		m, ok := NormalizeMobile(in.Phone)
		if !ok {
			return ErrInvalidPhone
		}
		in.Phone = m
	}
	if in.ConsentVersion != ConsentVersion {
		return ErrConsentVersion
	}
	if !in.ExpiresAt.IsZero() && !in.ExpiresAt.After(now) {
		return ErrInvalidExpiry
	}
	seen := map[family.Domain]bool{}
	mutual := in.Mutual[:0:0]
	for _, d := range in.Mutual {
		if !slices.Contains(MutualDomains, d) || family.MaxFor(d, in.Role, false) == family.LevelOff {
			return ErrMutualNotAllowed // the reverse link has the same role, so it must be able to hold the domain
		}
		if !seen[d] {
			seen[d] = true
			mutual = append(mutual, d)
		}
	}
	in.Mutual = mutual
	switch in.Children {
	case "":
		if in.Template == family.TemplateFullFamily && in.Role != family.RoleFriend {
			in.Children = family.LevelWrite.String()
		} else {
			in.Children = family.LevelOff.String()
		}
	case family.LevelOff.String(), family.LevelWrite.String():
	default:
		return ErrInvalidChildren
	}
	if in.Children == family.LevelWrite.String() {
		switch in.Role {
		case family.RoleFriend:
			return ErrInvalidChildren
		case family.RoleCarer:
			in.Children = childrenLimited
		}
	}
	return nil
}

// grantsFor is the template (custom: the role's defaults) with the row edits and the urgent switch applied.
func grantsFor(in CreateInput) map[family.Domain]family.Level {
	set := family.TemplateGrants(in.Template, in.Role)
	if in.Template == family.TemplateCustom {
		set = family.Defaults(in.Role)
	}
	for d, l := range in.Edits {
		if l == family.LevelOff {
			delete(set, d)
		} else {
			set[d] = l
		}
	}
	if in.UrgentAlert != nil {
		if *in.UrgentAlert {
			set[family.DomainUrgentAlert] = family.LevelView
		} else {
			delete(set, family.DomainUrgentAlert)
		}
	}
	return set
}

// Create makes an invited link with its role, template, grants (through family.Grants.Apply: registry caps, epoch,
// activity diff and the family_sharing consent event), expiry and one invite (code 24 h, phone 7 days). Nothing is
// visible to anyone until the member accepts.
func (s *Service) Create(ctx context.Context, ownerID uint64, in CreateInput) (Created, error) {
	now := s.now()
	if err := validateCreate(&in, now); err != nil {
		return Created{}, err
	}
	set := grantsFor(in)
	var out Created
	err := s.inTx(ctx, func(tx *sql.Tx, q *store.Queries) error {
		if err := lock(ctx, q, ownerID); err != nil {
			return err
		}
		mode, err := q.GetUserLifeMode(ctx, ownerID)
		if err != nil && !errors.Is(err, sql.ErrNoRows) {
			return fmt.Errorf("family invites: life mode: %w", err)
		}
		if mode.Valid && mode.String == string(enums.LifeModeTeen) {
			return ErrTeenParentOnly // a minor invites only a parent (/companions, CB-TEEN-01)
		}
		if in.Phone != "" {
			own, err := q.GetUserMobile(ctx, ownerID)
			if err != nil {
				return notFound(err)
			}
			if own.Valid && own.String == in.Phone {
				return ErrSelf
			}
		}
		open, err := q.CountOpenCompanions(ctx, store.CountOpenCompanionsParams{OwnerID: ownerID})
		if err != nil {
			return fmt.Errorf("family invites: count: %w", err)
		}
		if open >= int64(s.opt.MaxOpen) {
			return ErrTooMany
		}
		expires := in.ExpiresAt
		if expires.IsZero() && in.Role == family.RoleCarer {
			if expires, err = carerExpiry(ctx, tx, q, ownerID, now); err != nil {
				return err
			}
		}
		id, err := q.CreateFamilyLink(ctx, store.CreateFamilyLinkParams{
			OwnerID: ownerID, Type: linkType, Status: StatusInvited, Role: nstr(in.Role.String()),
			Template: nstr(string(in.Template)), DisplayName: nstr(in.DisplayName), ExpiresAt: nt(expires),
			UrgentSms: in.UrgentSMS, Now: nt(now),
		})
		if err != nil {
			return fmt.Errorf("family invites: create: %w", err)
		}
		linkID := uint64(id) //nolint:gosec // auto-increment id
		if _, err := (family.Grants{}).Apply(ctx, tx, family.Change{
			LinkID: linkID, OwnerID: ownerID, ActorID: ownerID, Role: in.Role, Set: set, ExpiresAt: expires,
			Reason: family.ChangeInvite, Now: now,
		}); err != nil {
			return err
		}
		channel := ChannelCode
		if in.Phone != "" {
			channel = ChannelPhone
		}
		mutual := make([]string, 0, len(in.Mutual))
		for _, d := range in.Mutual {
			mutual = append(mutual, d.String())
		}
		if err := activity(ctx, q, ownerID, ownerID, linkID, ActionInvited, intent{Mutual: mutual, Children: in.Children,
			Channel: channel, ConsentVersion: in.ConsentVersion}, now); err != nil {
			return err
		}
		code, invExpires, err := s.Issue(ctx, q, ownerID, linkID, in.Phone, now)
		if err != nil {
			return err
		}
		out = Created{LinkID: linkID, OwnerID: ownerID, Role: in.Role, Template: in.Template, Status: StatusInvited,
			DisplayName: in.DisplayName, LinkExpiresAt: expires, UrgentSMS: in.UrgentSMS, Grants: orderedGrants(set),
			Children: in.Children, Mutual: in.Mutual, Code: code, Phone: in.Phone, ExpiresAt: invExpires}
		return nil
	})
	return out, err
}

// carerExpiry is the carer default (F-13) from the owner's effective mode, EDD and delivery date.
func carerExpiry(ctx context.Context, tx *sql.Tx, q *store.Queries, ownerID uint64, now time.Time) (time.Time, error) {
	st, err := family.SyncMode(ctx, tx, ownerID, now)
	if err != nil {
		return time.Time{}, err
	}
	dates, err := q.FamilyOwnerDates(ctx, ownerID)
	if err != nil && !errors.Is(err, sql.ErrNoRows) {
		return time.Time{}, fmt.Errorf("family invites: owner dates: %w", err)
	}
	var edd, delivery time.Time
	if dates.Edd.Valid {
		edd = dates.Edd.Date.TehranMidnight()
	}
	if dates.Delivery.Valid {
		delivery = dates.Delivery.Date.TehranMidnight()
	}
	return family.CarerExpiry(st.Mode, edd, delivery, now), nil
}

// Preview is what a member sees before answering: the owner's first name, the role, the domains she would see (levels
// only), the children level, the mutual request and whether the owner still has to confirm. LinkID is set for a
// pending acceptance only (answered without the code).
type Preview struct {
	LinkID            uint64
	OwnerID           uint64
	OwnerFirstName    string
	Role              family.Role
	Template          string
	Domains           []DomainLevel
	Children          string
	Mutual            []family.Domain
	OwnerConfirmation bool
	ExpiresAt         time.Time
}

func firstName(name string) string {
	f := strings.Fields(name)
	if len(f) == 0 {
		return ""
	}
	return f[0]
}

func describe(ctx context.Context, q *store.Queries, inv store.CompanionInvite, c store.Companion) (Preview, error) {
	set, err := grantSet(ctx, q, c.ID)
	if err != nil {
		return Preview{}, err
	}
	in, err := readIntent(ctx, q, c.ID)
	if err != nil {
		return Preview{}, err
	}
	owner, err := q.GetFamilyUserProfile(ctx, c.OwnerID)
	if err != nil {
		return Preview{}, notFound(err)
	}
	children := in.Children
	if children == family.LevelOff.String() {
		children = ""
	}
	confirm, err := needsOwner(ctx, q, inv, c, set, in)
	if err != nil {
		return Preview{}, err
	}
	return Preview{
		OwnerID: c.OwnerID, OwnerFirstName: firstName(owner.Name.String), Role: family.ParseRole(c.Role.String, c.Type),
		Template: c.Template.String, Domains: orderedGrants(set), Children: children, Mutual: in.mutualDomains(),
		OwnerConfirmation: confirm,
		ExpiresAt:         inv.ExpiresAt.Time,
	}, nil
}

// Preview shows a code's invite to viewerID with the same refusals as Accept (another code oracle, FAM-L3): nothing is
// written except a phone mismatch's attempt count.
func (s *Service) Preview(ctx context.Context, viewerID uint64, typed string) (Preview, error) {
	defer s.floor(ctx, time.Now())
	inv, _, err := s.resolve(ctx, viewerID, typed, 0)
	if err != nil {
		return Preview{}, err
	}
	now := s.now()
	q := store.New(s.conn)
	if err := s.check(ctx, q, inv, viewerID, now, false); err != nil {
		if errors.Is(err, ErrPhoneMismatch) {
			if cerr := s.countMismatch(ctx, inv.ID, viewerID, now); cerr != nil {
				return Preview{}, cerr
			}
		}
		return Preview{}, err
	}
	c, err := q.GetCompanion(ctx, inv.CompanionID)
	if err != nil {
		return Preview{}, notFound(err)
	}
	if c.Status != StatusInvited {
		return Preview{}, ErrUsed
	}
	if err := teenRule(ctx, q, inv.OwnerID, c.Type); err != nil {
		return Preview{}, err
	}
	if n, err := q.CountOpenLinks(ctx, store.CountOpenLinksParams{OwnerID: inv.OwnerID, CompanionUserID: nid(viewerID)}); err != nil {
		return Preview{}, fmt.Errorf("family invites: links: %w", err)
	} else if n > 0 {
		return Preview{}, ErrAlreadyLinked
	}
	return describe(ctx, q, inv, c)
}

// Pending lists viewerID's pending acceptances (phone invites bound to the account), newest first.
func (s *Service) Pending(ctx context.Context, viewerID uint64) ([]Preview, error) {
	q, now := store.New(s.conn), s.now()
	rows, err := q.ListBoundInvites(ctx, store.ListBoundInvitesParams{UserID: nid(viewerID), Now: nt(now)})
	if err != nil {
		return nil, fmt.Errorf("family invites: pending: %w", err)
	}
	out := make([]Preview, 0, len(rows))
	for _, r := range rows {
		inv, err := q.GetInvite(ctx, r.ID)
		if err != nil {
			return nil, notFound(err)
		}
		c, err := q.GetCompanion(ctx, r.CompanionID)
		if err != nil {
			return nil, notFound(err)
		}
		p, err := describe(ctx, q, inv, c)
		if err != nil {
			return nil, err
		}
		p.LinkID = c.ID
		out = append(out, p)
	}
	return out, nil
}

// AcceptInput answers an invite: by Code, or (LinkID) a pending acceptance bound to the caller. Mutual is the answer
// to the owner's mutual request (domain → level, nil = none; accepting without mutual is always allowed).
type AcceptInput struct {
	Code   string
	LinkID uint64
	Mutual map[family.Domain]family.Level
}

// Accepted is the outcome of Accept.
type Accepted struct {
	LinkID       uint64
	OwnerID      uint64
	Type         string
	Status       string // active | awaiting_owner
	DisplayName  string
	MutualLinkID uint64
}

// Accept redeems an invite for viewerID. The link becomes active, or awaiting_owner when a code-only invite includes
// child write, the urgent alert or vitals (F-11; the member sees nothing until the owner confirms). The grants the owner
// chose take effect with their invite-time epochs (family.Grants.Apply, consent + activity). A mutual answer creates an
// independent reverse link viewer → owner with the viewer's grants and the viewer's consent. A bound phone invite
// redeemed by another account counts a failed attempt (once per account with a Mismatch gate).
func (s *Service) Accept(ctx context.Context, viewerID uint64, in AcceptInput) (Accepted, error) {
	if in.Code != "" || in.LinkID == 0 {
		defer s.floor(ctx, time.Now())
	}
	inv, pending, err := s.resolve(ctx, viewerID, in.Code, in.LinkID)
	if err != nil {
		return Accepted{}, err
	}
	now := s.now()
	var out Accepted
	err = s.inTx(ctx, func(tx *sql.Tx, q *store.Queries) error {
		ids := []uint64{inv.OwnerID}
		if in.Mutual != nil {
			ids = append(ids, viewerID)
		}
		if err := lock(ctx, q, ids...); err != nil {
			return err
		}
		inv, err := q.GetInvite(ctx, inv.ID) // re-read under the owner lock
		if err != nil {
			return notFound(err)
		}
		if err := s.check(ctx, q, inv, viewerID, now, pending); err != nil {
			return err
		}
		c, err := q.GetCompanion(ctx, inv.CompanionID)
		if err != nil {
			return notFound(err)
		}
		if c.Status != StatusInvited {
			return ErrUsed
		}
		if err := teenRule(ctx, q, inv.OwnerID, c.Type); err != nil {
			return err // the owner's mode changed since the invite (CB-TEEN-01); not a failed attempt
		}
		linked, err := q.CountOpenLinks(ctx, store.CountOpenLinksParams{OwnerID: inv.OwnerID, CompanionUserID: nid(viewerID)})
		if err != nil {
			return fmt.Errorf("family invites: links: %w", err)
		}
		if linked > 0 {
			return ErrAlreadyLinked
		}
		set, err := grantSet(ctx, q, c.ID)
		if err != nil {
			return err
		}
		ask, err := readIntent(ctx, q, c.ID)
		if err != nil {
			return err
		}
		if in.Mutual != nil {
			if err := checkMutual(ctx, q, ask, in.Mutual, viewerID, inv.OwnerID); err != nil {
				return err
			}
		}
		status := StatusActive
		if wait, err := needsOwner(ctx, q, inv, c, set, ask); err != nil {
			return err
		} else if wait {
			status = StatusAwaitingOwner
		}
		if n, err := q.MarkInviteUsed(ctx, store.MarkInviteUsedParams{ID: inv.ID, UsedByID: nid(viewerID), Now: nt(now)}); err != nil {
			return fmt.Errorf("family invites: use invite: %w", err)
		} else if n == 0 {
			return ErrUsed
		}
		if n, err := q.AcceptFamilyLink(ctx, store.AcceptFamilyLinkParams{ID: c.ID, CompanionUserID: nid(viewerID), Status: status, Now: nt(now)}); err != nil {
			return fmt.Errorf("family invites: accept: %w", err)
		} else if n == 0 {
			return ErrUsed
		}
		if err := q.RevokeOpenInvites(ctx, store.RevokeOpenInvitesParams{CompanionID: c.ID, Now: nt(now)}); err != nil {
			return fmt.Errorf("family invites: revoke invites: %w", err)
		}
		if c.Type == "spouse" {
			if err := q.SetFamilySpouse(ctx, store.SetFamilySpouseParams{CompanionID: c.ID, SpouseUserID: nid(viewerID), Now: nt(now)}); err != nil {
				return fmt.Errorf("family invites: family spouse: %w", err)
			}
		}
		if err := q.InsertAudit(ctx, store.InsertAuditParams{OwnerID: inv.OwnerID, ActorID: nid(viewerID),
			CompanionID: nid(c.ID), Action: ActionAccepted, Now: nt(now)}); err != nil {
			return fmt.Errorf("family invites: audit: %w", err)
		}
		// The grants the owner chose take effect now. Levels do not change, so Grants.Apply keeps every row's stored
		// epoch (a grant chosen in mode A and accepted after a switch stays paused until she re-approves it, M2).
		if _, err := (family.Grants{}).Apply(ctx, tx, family.Change{
			LinkID: c.ID, OwnerID: inv.OwnerID, MemberID: viewerID, ActorID: inv.OwnerID,
			Role: family.ParseRole(c.Role.String, c.Type), Set: set, ExpiresAt: c.ExpiresAt.Time,
			Reason: family.ChangeAccept, Now: now,
		}); err != nil {
			return err
		}
		out = Accepted{LinkID: c.ID, OwnerID: inv.OwnerID, Type: c.Type, Status: status, DisplayName: c.DisplayName.String}
		if in.Mutual != nil && hasLevels(in.Mutual) {
			out.MutualLinkID, err = s.createMutual(ctx, tx, q, c, viewerID, in.Mutual, now)
			return err
		}
		return nil
	})
	if errors.Is(err, ErrPhoneMismatch) {
		if cerr := s.countMismatch(ctx, inv.ID, viewerID, now); cerr != nil {
			return Accepted{}, cerr
		}
	}
	return out, err
}

func hasLevels(m map[family.Domain]family.Level) bool {
	for _, l := range m {
		if l > family.LevelOff {
			return true
		}
	}
	return false
}

// checkMutual: the answer stays within the owner's request, and the viewer does not already share with the owner (a
// second reverse link would not be independent of the first).
func checkMutual(ctx context.Context, q *store.Queries, ask intent, answer map[family.Domain]family.Level, viewerID, ownerID uint64) error {
	requested := ask.mutualDomains()
	for d, l := range answer {
		if l > family.LevelOff && !slices.Contains(requested, d) {
			return ErrMutualNotRequested
		}
	}
	if !hasLevels(answer) {
		return nil
	}
	n, err := q.CountOpenLinks(ctx, store.CountOpenLinksParams{OwnerID: viewerID, CompanionUserID: nid(ownerID)})
	if err != nil {
		return fmt.Errorf("family invites: reverse links: %w", err)
	}
	if n > 0 {
		return ErrMutualExists
	}
	return nil
}

// createMutual is the reverse link viewer → owner: active at once (the viewer is its owner and chose it now), same
// role, the viewer's grants through Grants.Apply (her own consent row and activity).
func (s *Service) createMutual(ctx context.Context, tx *sql.Tx, q *store.Queries, c store.Companion, viewerID uint64,
	answer map[family.Domain]family.Level, now time.Time) (uint64, error) {
	if err := ownerNotTeen(ctx, q, viewerID); err != nil {
		return 0, err // CB-TEEN-01: a teen owns only her parent link
	}
	role := family.ParseRole(c.Role.String, c.Type)
	set := map[family.Domain]family.Level{}
	for d, l := range answer {
		if l > family.LevelOff {
			set[d] = l
		}
	}
	id, err := q.CreateFamilyLink(ctx, store.CreateFamilyLinkParams{
		OwnerID: viewerID, CompanionUserID: nid(c.OwnerID), Type: linkType, Status: StatusActive,
		Role: nstr(role.String()), Template: nstr(string(family.TemplateCustom)), Now: nt(now), AcceptedAt: nt(now),
	})
	if err != nil {
		return 0, fmt.Errorf("family invites: mutual link: %w", err)
	}
	linkID := uint64(id) //nolint:gosec // auto-increment id
	if _, err := (family.Grants{}).Apply(ctx, tx, family.Change{
		LinkID: linkID, OwnerID: viewerID, MemberID: c.OwnerID, ActorID: viewerID, Role: role, Set: set,
		Reason: family.ChangeAccept, Now: now,
	}); err != nil {
		return 0, err
	}
	return linkID, activity(ctx, q, viewerID, viewerID, linkID, ActionLinkCreate, map[string]any{"mutual_of": c.ID}, now)
}

// Decline answers an invite with no: the link is revoked with revoked_by = declined (F-12), hidden from lists and kept
// for audit and KPIs. Same refusals as Accept.
func (s *Service) Decline(ctx context.Context, viewerID uint64, typed string, linkID uint64) error {
	if typed != "" || linkID == 0 {
		defer s.floor(ctx, time.Now())
	}
	inv, pending, err := s.resolve(ctx, viewerID, typed, linkID)
	if err != nil {
		return err
	}
	now := s.now()
	err = s.inTx(ctx, func(_ *sql.Tx, q *store.Queries) error {
		if err := lock(ctx, q, inv.OwnerID); err != nil {
			return err
		}
		inv, err := q.GetInvite(ctx, inv.ID)
		if err != nil {
			return notFound(err)
		}
		if err := s.check(ctx, q, inv, viewerID, now, pending); err != nil {
			return err
		}
		c, err := q.GetCompanion(ctx, inv.CompanionID)
		if err != nil {
			return notFound(err)
		}
		if c.Status != StatusInvited {
			return ErrUsed
		}
		if n, err := q.RevokeCompanion(ctx, store.RevokeCompanionParams{ID: c.ID, RevokedBy: nstr(RevokedByDeclined), Now: nt(now)}); err != nil {
			return fmt.Errorf("family invites: decline: %w", err)
		} else if n == 0 {
			return ErrUsed
		}
		if err := cleanupLink(ctx, q, c.ID, now); err != nil {
			return err
		}
		return activity(ctx, q, c.OwnerID, viewerID, c.ID, ActionDeclined, nil, now)
	})
	if errors.Is(err, ErrPhoneMismatch) {
		if cerr := s.countMismatch(ctx, inv.ID, viewerID, now); cerr != nil {
			return cerr
		}
	}
	return err
}

// Confirm is the owner's confirmation of an accepted high-impact code invite (F-11): awaiting_owner → active, within
// ConfirmTTL of the acceptance and before the link's end date (else ErrNotFound; the sweep then revokes it).
func (s *Service) Confirm(ctx context.Context, ownerID, linkID uint64) (store.Companion, error) {
	now := s.now()
	var out store.Companion
	err := s.inTx(ctx, func(_ *sql.Tx, q *store.Queries) error {
		if err := lock(ctx, q, ownerID); err != nil {
			return err
		}
		n, err := q.ConfirmFamilyLink(ctx, store.ConfirmFamilyLinkParams{ID: linkID, OwnerID: ownerID, Now: nt(now),
			StaleBefore: nt(now.Add(-s.opt.ConfirmTTL))})
		if err != nil {
			return fmt.Errorf("family invites: confirm: %w", err)
		}
		if n == 0 {
			return ErrNotFound
		}
		if err := activity(ctx, q, ownerID, ownerID, linkID, ActionConfirmed, nil, now); err != nil {
			return err
		}
		out, err = q.GetCompanion(ctx, linkID)
		return notFoundIf(err)
	})
	return out, err
}

func notFoundIf(err error) error {
	if err == nil {
		return nil
	}
	return notFound(err)
}

// Cancel is the owner withdrawing an invite at any time: an invited or awaiting_owner link of hers is revoked
// (revoked_by owner) with its open invites, grants and share requests. Active links are removed through /companions.
func (s *Service) Cancel(ctx context.Context, ownerID, linkID uint64) error {
	now := s.now()
	return s.inTx(ctx, func(_ *sql.Tx, q *store.Queries) error {
		if err := lock(ctx, q, ownerID); err != nil {
			return err
		}
		c, err := q.GetCompanion(ctx, linkID)
		if err != nil || c.OwnerID != ownerID || (c.Status != StatusInvited && c.Status != StatusAwaitingOwner) {
			return notFound(err)
		}
		if n, err := q.RevokeCompanion(ctx, store.RevokeCompanionParams{ID: c.ID, RevokedBy: nstr(RevokedByOwner), Now: nt(now)}); err != nil {
			return fmt.Errorf("family invites: cancel: %w", err)
		} else if n == 0 {
			return ErrNotFound
		}
		if err := cleanupLink(ctx, q, c.ID, now); err != nil {
			return err
		}
		return activity(ctx, q, ownerID, ownerID, c.ID, ActionRevoked, nil, now)
	})
}
