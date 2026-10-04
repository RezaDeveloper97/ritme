package who

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestTablesLoad(t *testing.T) {
	for _, ind := range Indicators {
		for _, sex := range Sexes {
			tab := Table(ind, sex)
			require.Len(t, tab, MaxDay+1, "%s %s", ind, sex)
			for i, r := range tab {
				require.Equal(t, i, r.Day)
				require.Positive(t, r.M)
				require.Positive(t, r.S)
			}
			if ind != Weight {
				assert.InDelta(t, 1, tab[0].L, 0, "%s is normal (L = 1)", ind)
			}
		}
	}
	// First and last rows of wfa-girls-zscore-expanded-tables.xlsx.
	assert.Equal(t, LMS{Day: 0, L: 0.3809, M: 3.2322, S: 0.14171}, Table(Weight, Girl)[0])
	assert.Equal(t, LMS{Day: 1856, L: -0.3531, M: 18.389, S: 0.14892}, Table(Weight, Girl)[MaxDay])
	assert.Nil(t, Table("x", Girl))
	_, ok := At(Head, Boy, MaxDay+1)
	assert.False(t, ok)
}
