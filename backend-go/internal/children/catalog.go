package children

import (
	"context"
	"encoding/json"
	"fmt"
	"slices"
	"sort"

	"github.com/ritme/backend-go/internal/catalog"
)

// Catalog groups (catalog_items, seeded by goose 00031, admin-edited through /api/admin/v1/catalog/{group}).
const (
	GroupVaccines   = "child_vaccines"
	GroupMilestones = "child_milestones"
	GroupActivities = "child_milestone_activities"
	GroupDoctorNote = "child_milestone_notes"
	GroupAgeNotes   = "child_age_notes"
	GroupLearn      = "child_learn"
)

// Groups are every child catalog group (B-N5-09 lists them in the admin).
var Groups = []string{GroupVaccines, GroupMilestones, GroupActivities, GroupDoctorNote, GroupAgeNotes, GroupLearn}

// Learn topics, in tab order (nbl_v16_Learn).
var Topics = []string{"sleep", "feeding", "play", "health", "mother"}

// ItemReader reads a catalog group (catalog.Reader).
type ItemReader interface {
	Items(ctx context.Context, group string) ([]catalog.Item, error)
}

// Entry is a catalog item with its typed meta (title/body stay translatable JSON, picked per request).
type Entry struct {
	Code  string
	Title json.RawMessage
	Body  json.RawMessage
	// meta
	Visit      string
	AgeMonths  int
	Domain     string
	Topic      string
	FromMonths int
	ToMonths   int
	Minutes    int
	Featured   bool
	Article    string // article_slug
	Order      int    // position in the group (sort_order order)
	NeedsCheck bool   // needs_review
}

type entryMeta struct {
	Visit       string `json:"visit"`
	AgeMonths   *int   `json:"age_months"`
	Domain      string `json:"domain"`
	Topic       string `json:"topic"`
	FromMonths  *int   `json:"from_months"`
	ToMonths    *int   `json:"to_months"`
	Minutes     int    `json:"minutes"`
	Featured    bool   `json:"featured"`
	ArticleSlug string `json:"article_slug"`
}

// entries reads a group; items whose meta lacks what the group needs are skipped (an admin typo never breaks a screen).
func entries(ctx context.Context, r ItemReader, group string) ([]Entry, error) {
	items, err := r.Items(ctx, group)
	if err != nil {
		return nil, fmt.Errorf("children: catalog %s: %w", group, err)
	}
	out := make([]Entry, 0, len(items))
	for i, it := range items {
		var m entryMeta
		if len(it.Meta) > 0 {
			if json.Unmarshal(it.Meta, &m) != nil {
				continue
			}
		}
		e := Entry{
			Code: it.Code, Title: it.Title, Body: it.Body, Order: i, NeedsCheck: it.NeedsReview,
			Visit: m.Visit, Domain: m.Domain, Topic: m.Topic, Minutes: m.Minutes, Featured: m.Featured, Article: m.ArticleSlug,
		}
		switch group {
		case GroupLearn:
			if m.FromMonths == nil || m.ToMonths == nil || !slices.Contains(Topics, m.Topic) {
				continue
			}
			e.FromMonths, e.ToMonths = *m.FromMonths, *m.ToMonths
		default:
			if m.AgeMonths == nil || *m.AgeMonths < 0 {
				continue
			}
			e.AgeMonths = *m.AgeMonths
			if group == GroupVaccines && e.Visit == "" {
				continue
			}
		}
		out = append(out, e)
	}
	return out, nil
}

// bands are the distinct age_months of entries, ascending.
func bands(es []Entry) []int {
	var out []int
	for _, e := range es {
		if !slices.Contains(out, e.AgeMonths) {
			out = append(out, e.AgeMonths)
		}
	}
	sort.Ints(out)
	return out
}

// bandFor is the latest band ≤ months (the first band when the child is younger than all of them; -1 when empty).
func bandFor(bs []int, months int) int {
	if len(bs) == 0 {
		return -1
	}
	b := bs[0]
	for _, x := range bs {
		if x <= months {
			b = x
		}
	}
	return b
}

// inBand filters entries of one age band.
func inBand(es []Entry, months int) []Entry {
	var out []Entry
	for _, e := range es {
		if e.AgeMonths == months {
			out = append(out, e)
		}
	}
	return out
}
