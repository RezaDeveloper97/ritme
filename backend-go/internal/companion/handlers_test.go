package companion

import (
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"sort"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/ritme/backend-go/internal/platform/validation"
)

func TestParseGrants(t *testing.T) {
	body := validation.DecodeBody([]byte(`{"grants":{"meds":"edit","cycle":"view","symptoms":"none"}}`))
	g, err := parseGrants("en", body, TypePartner)
	require.NoError(t, err)
	assert.Equal(t, Grants{SectionMeds: LevelEdit, SectionCycle: LevelView}, g, "none is not stored")

	for _, raw := range []string{`{"grants":{"bank":"view"}}`, `{"grants":{"meds":"admin"}}`, `{"grants":{"meds":1}}`, `{"grants":["view"]}`} {
		_, err := parseGrants("en", validation.DecodeBody([]byte(raw)), TypePartner)
		assert.Error(t, err, raw)
	}
	g, err = parseGrants("en", validation.DecodeBody([]byte(`{}`)), TypePartner)
	require.NoError(t, err)
	assert.Empty(t, g)
}

// CB-TEEN-01: a parent link holds only the teen sections, view only; other links never hold them.
func TestParseGrants_ParentTeenSectionsOnly(t *testing.T) {
	g, err := parseGrants("en", validation.DecodeBody([]byte(`{"grants":{"teen_period_week":"view","teen_kit":"none"}}`)), TypeParent)
	require.NoError(t, err)
	assert.Equal(t, Grants{SectionTeenPeriodWeek: LevelView}, g)

	for _, raw := range []string{
		`{"grants":{"cycle":"view"}}`,            // a regular section on a parent link
		`{"grants":{"symptoms":"view"}}`,         // symptoms (could carry sexual-health items) never go to a parent
		`{"grants":{"teen_notes":"edit"}}`,       // a parent never writes
		`{"grants":{"teen_period_week":"edit"}}`, // idem
	} {
		_, err := parseGrants("en", validation.DecodeBody([]byte(raw)), TypeParent)
		assert.Error(t, err, raw)
	}
	_, err = parseGrants("en", validation.DecodeBody([]byte(`{"grants":{"teen_kit":"view"}}`)), TypePartner)
	assert.Error(t, err, "teen sections are not grantable to a partner")
}

func TestGrants_ValidateForAndSectionRules(t *testing.T) {
	assert.NoError(t, Grants{SectionTeenNotes: LevelView}.ValidateFor(TypeParent))
	assert.ErrorIs(t, Grants{SectionTeenNotes: LevelEdit}.ValidateFor(TypeParent), ErrInvalidLevel)
	assert.ErrorIs(t, Grants{SectionMeds: LevelView}.ValidateFor(TypeParent), ErrInvalidSection)
	assert.ErrorIs(t, Grants{SectionTeenKit: LevelView}.ValidateFor(TypeSpouse), ErrInvalidSection)
	assert.NoError(t, Grants{SectionMeds: LevelEdit}.ValidateFor(TypeSpouse))

	for _, s := range TeenSections {
		assert.True(t, s.Valid())
		assert.True(t, s.IsTeen())
		assert.Equal(t, LevelView, s.MaxLevel())
		assert.False(t, s.Permits(LevelEdit))
		assert.False(t, s.AllowedFor(TypePartner))
		assert.True(t, s.AllowedFor(TypeParent))
	}
	for _, s := range Sections {
		assert.False(t, s.AllowedFor(TypeParent), s)
		assert.True(t, s.Permits(LevelEdit))
	}
}

func TestIsInviteRefusal_OneGenericAnswer(t *testing.T) {
	for _, e := range []error{ErrInviteInvalid, ErrInviteExpired, ErrInviteUsed, ErrInviteLocked, ErrInvitePhoneMismatch,
		ErrSelfInvite, ErrAlreadyLinked, ErrNotFound} {
		assert.True(t, isInviteRefusal(fmt.Errorf("wrapped: %w", e)), e.Error())
	}
	assert.False(t, isInviteRefusal(errors.New("db down")), "infrastructure errors stay 500")
}

func TestWriteNotice(t *testing.T) {
	n, ok := writeNotice(SectionMeds, true)
	assert.True(t, ok)
	assert.Equal(t, NoticeRecordedMeds, n)
	n, _ = writeNotice(SectionAppointments, false)
	assert.Equal(t, NoticeUpdatedAppointment, n)
	_, ok = writeNotice(SectionCycle, true)
	assert.False(t, ok)
}

func TestMaskMobile(t *testing.T) {
	assert.Equal(t, "0912****567", MaskMobile("09121234567"))
	assert.Equal(t, "***", MaskMobile("0912"))
}

func TestGrantsJSON_EverySection(t *testing.T) {
	raw, err := json.Marshal(grantsJSON(Grants{SectionMeds: LevelEdit}, TypePartner))
	require.NoError(t, err)
	assert.JSONEq(t, `{"cycle":"none","symptoms":"none","meds":"edit","appointments":"none","pregnancy":"none"}`, string(raw))

	raw, err = json.Marshal(grantsJSON(Grants{SectionTeenKit: LevelView}, TypeParent))
	require.NoError(t, err)
	assert.JSONEq(t, `{"teen_period_week":"none","teen_kit":"view","teen_notes":"none"}`, string(raw))
}

// Every language file has the same keys, and every notice title takes :name.
func TestLangFiles_SameKeys(t *testing.T) {
	keys := map[string][]string{}
	err := fs.WalkDir(langFS, "lang", func(path string, d fs.DirEntry, err error) error {
		if err != nil || d.IsDir() {
			return err
		}
		raw, err := langFS.ReadFile(path)
		if err != nil {
			return err
		}
		var m map[string]map[string]string
		if err := json.Unmarshal(raw, &m); err != nil {
			return fmt.Errorf("%s: %w", path, err)
		}
		var ks []string
		for g, inner := range m {
			for k := range inner {
				ks = append(ks, g+"."+k)
			}
		}
		sort.Strings(ks)
		keys[path] = ks
		return nil
	})
	require.NoError(t, err)
	require.GreaterOrEqual(t, len(keys), 2)
	var first []string
	for _, ks := range keys {
		if first == nil {
			first = ks
		}
		assert.Equal(t, first, ks)
	}
	for _, n := range []Notice{NoticeAccepted, NoticeRecordedMeds, NoticeRecordedAppointment, NoticeUpdatedMeds, NoticeUpdatedAppointment} {
		assert.Contains(t, T("notices."+string(n)+"_title", "fa", "name", "علی"), "علی")
		assert.Contains(t, T("notices."+string(n)+"_title", "en", "name", "Ali"), "Ali")
	}
}
