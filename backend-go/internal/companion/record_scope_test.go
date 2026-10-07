package companion

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

// CB-REC-03 guard: a companion link reaches the health record only through meds and appointments. Adding a section
// that maps to record documents (or to the record itself) must be a deliberate change that breaks this test.
func TestRecordScope_NeverDocuments(t *testing.T) {
	assert.Equal(t, []Section{SectionMeds, SectionAppointments}, RecordSections)
	for _, s := range []Section{"documents", "record", "health_record", "record_documents", "files", "emergency_card"} {
		assert.False(t, s.Valid(), s)
		for _, ty := range Types {
			assert.False(t, s.AllowedFor(ty), "%s on %s", s, ty)
		}
		assert.Error(t, Grants{s: LevelView}.ValidateFor(TypeSpouse), s)
	}
	for _, s := range append(append([]Section{}, Sections...), TeenSections...) {
		assert.NotContains(t, string(s), "document", s)
		assert.NotContains(t, string(s), "record", s)
	}
}
