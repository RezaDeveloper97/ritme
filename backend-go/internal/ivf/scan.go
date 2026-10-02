package ivf

import (
	"github.com/ritme/backend-go/internal/ivf/store"
	"github.com/ritme/backend-go/internal/platform/civildate"
)

// Ovary is the follicle count of one ovary per size bin (Bins order).
type Ovary [4]int

// Total is every follicle of the ovary.
func (o Ovary) Total() int { return o[0] + o[1] + o[2] + o[3] }

// Plus is the bin-wise sum of two ovaries.
func (o Ovary) Plus(p Ovary) Ovary {
	return Ovary{o[0] + p[0], o[1] + p[1], o[2] + p[2], o[3] + p[3]}
}

// Scan is one ultrasound. Decimal fields keep the DB text ("8.5"); "" = not recorded.
type Scan struct {
	Date          civildate.Date
	Right, Left   Ovary
	EndometriumMM string
	E2            string
	E2Unit        string
	Notes         string
}

func scanFromRow(r store.IvfScan) Scan {
	return Scan{
		Date:          r.ScanDate,
		Right:         Ovary{int(r.RightLt10), int(r.Right1014), int(r.Right1517), int(r.Right18Plus)},
		Left:          Ovary{int(r.LeftLt10), int(r.Left1014), int(r.Left1517), int(r.Left18Plus)},
		EndometriumMM: r.EndometriumMm.String, E2: r.E2.String, E2Unit: r.E2Unit.String, Notes: r.Notes.String,
	}
}

// Growth is one point of the follicle growth chart (two series: 10–14 mm and ≥ 15 mm, both ovaries).
type Growth struct {
	Date   civildate.Date
	Mid    int // 10–14 mm
	Mature int // 15 mm and more
}

// GrowthOf is the scan's chart point.
func (s Scan) GrowthOf() Growth {
	t := s.Right.Plus(s.Left)
	return Growth{Date: s.Date, Mid: t[1], Mature: t[2] + t[3]}
}
