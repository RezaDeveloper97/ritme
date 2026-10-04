package children

import (
	"context"
	"slices"
)

// Band loads the milestone band for month (nil = the latest band not after the child's age in months) with the
// child's checks, the band's activities and doctor note, and every band (the month tabs). Band.Months is -1 when the
// catalog is empty.
func (s *Service) Band(ctx context.Context, childID uint64, ageMonths int, month *int) (MilestoneBand, []int, error) {
	items, err := entries(ctx, s.cat, GroupMilestones)
	if err != nil {
		return MilestoneBand{}, nil, err
	}
	bs := bands(items)
	m := bandFor(bs, ageMonths)
	if month != nil {
		if !slices.Contains(bs, *month) {
			return MilestoneBand{}, bs, ErrUnknownBand
		}
		m = *month
	}
	b := MilestoneBand{Months: m}
	if m < 0 {
		return b, bs, nil
	}
	b.Items = inBand(items, m)
	if b.Checked, err = s.Checks(ctx, childID); err != nil {
		return MilestoneBand{}, nil, err
	}
	acts, err := entries(ctx, s.cat, GroupActivities)
	if err != nil {
		return MilestoneBand{}, nil, err
	}
	b.Activities = inBand(acts, m)
	notes, err := entries(ctx, s.cat, GroupDoctorNote)
	if err != nil {
		return MilestoneBand{}, nil, err
	}
	if n := inBand(notes, m); len(n) > 0 {
		b.Note = &n[0]
	}
	return b, bs, nil
}

// MilestoneKnown reports whether code is an active milestone.
func (s *Service) MilestoneKnown(ctx context.Context, code string) (bool, error) {
	items, err := entries(ctx, s.cat, GroupMilestones)
	if err != nil {
		return false, err
	}
	for _, it := range items {
		if it.Code == code {
			return true, nil
		}
	}
	return false, nil
}

// AgeNote is the «این هفته …» note of the latest age band not after ageMonths (nil when none).
func (s *Service) AgeNote(ctx context.Context, ageMonths int) (*Entry, error) {
	notes, err := entries(ctx, s.cat, GroupAgeNotes)
	if err != nil {
		return nil, err
	}
	m := bandFor(bands(notes), ageMonths)
	if n := inBand(notes, m); len(n) > 0 && m <= ageMonths {
		return &n[0], nil
	}
	return nil, nil //nolint:nilnil // no note for this age
}

// LearnTips are the tips whose age range covers ageMonths (topic "" = all), featured first, each with its linked
// article when that article is published.
func (s *Service) LearnTips(ctx context.Context, ageMonths int, topic string) ([]LearnTip, error) {
	es, err := entries(ctx, s.cat, GroupLearn)
	if err != nil {
		return nil, err
	}
	var picked []Entry
	var slugs []string
	for _, e := range es {
		if ageMonths < e.FromMonths || ageMonths > e.ToMonths || (topic != "" && e.Topic != topic) {
			continue
		}
		picked = append(picked, e)
		if e.Article != "" {
			slugs = append(slugs, e.Article)
		}
	}
	slices.SortStableFunc(picked, func(a, b Entry) int {
		switch {
		case a.Featured && !b.Featured:
			return -1
		case !a.Featured && b.Featured:
			return 1
		}
		return 0
	})
	arts, err := s.LearnArticles(ctx, slugs)
	if err != nil {
		return nil, err
	}
	out := make([]LearnTip, 0, len(picked))
	for _, e := range picked {
		t := LearnTip{Entry: e}
		if a, ok := arts[e.Article]; ok {
			t.Article = &a
		}
		out = append(out, t)
	}
	return out, nil
}
