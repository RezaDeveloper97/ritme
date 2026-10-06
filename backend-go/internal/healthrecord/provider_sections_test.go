package healthrecord_test

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/ritme/backend-go/internal/healthrecord"
	"github.com/ritme/backend-go/internal/platform/civildate"
	"github.com/ritme/backend-go/internal/platform/validation/phpval"
)

func TestReportSections_IncludeProviderKeys(t *testing.T) {
	keys := healthrecord.ReportSections()
	assert.Equal(t, healthrecord.Sections, keys[:len(healthrecord.Sections)], "built-in sections first, in screen order")
	assert.Equal(t, healthrecord.SectionMenopause, keys[len(keys)-1])

	now := time.Date(2026, 10, 1, 10, 0, 0, 0, civildate.Tehran)
	data := phpval.NewMap()
	data.Set("range", "3m")
	data.Set("sections", "menopause,basics,menopause")
	req, err := healthrecord.ParseReportRequest(data, "en", now, false)
	require.NoError(t, err)
	assert.Equal(t, []string{healthrecord.SectionBasics, healthrecord.SectionMenopause}, req.Sections)

	data.Set("sections", "menopause,unknown")
	_, err = healthrecord.ParseReportRequest(data, "en", now, false)
	assert.Error(t, err)
}
