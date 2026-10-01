package search

import (
	"context"
	"encoding/json"
	"net/url"
	"strconv"
	"strings"

	"github.com/ritme/backend-go/internal/i18n"
	"github.com/ritme/backend-go/internal/platform/jsonx"
)

// pick is a translatable column in the request locale (default language underneath).
func pick(raw json.RawMessage, q Query) string {
	if len(raw) == 0 {
		return ""
	}
	return i18n.PickString(raw, q.Locale, q.Default)
}

// education is the education group: published articles by title (then excerpt).
func (s *Service) education(ctx context.Context, q Query, m Matcher) ([]Hit, error) {
	list, err := s.articles.PublishedArticles(ctx)
	if err != nil {
		return nil, err
	}
	var hits []Hit
	for i, a := range list {
		title := pick(a.Title, q)
		if title == "" {
			continue
		}
		rank := m.Best(title, pick(a.Excerpt, q))
		if rank == 0 {
			continue
		}
		var readTime, category any
		if a.ReadTime > 0 {
			readTime = a.ReadTime
		}
		if a.Category != "" {
			category = a.Category
		}
		hits = append(hits, Hit{
			Type: "article", ID: a.Slug, Title: title, Route: "/articles/" + url.PathEscape(a.Slug),
			Meta:  jsonx.Obj("kind", "article", "category", category, "read_time_minutes", readTime),
			rank:  rank,
			order: i,
		})
	}
	return hits, nil
}

// services is the services group: the service screens that exist, then the checkup catalog the user sees
// (shared rows plus her own custom checkups).
func (s *Service) services(ctx context.Context, q Query, m Matcher) ([]Hit, error) {
	hits := registryHits("services", "service", serviceRegistry, q, m, 0)
	list, err := s.checkups.VisibleCheckups(ctx, q.UserID)
	if err != nil {
		return nil, err
	}
	for i, c := range list {
		title := pick(c.Title, q)
		if title == "" {
			continue
		}
		sub := pick(c.Subtitle, q)
		rank := m.Best(title, sub)
		if rank == 0 {
			continue
		}
		hits = append(hits, Hit{
			Type: "checkup", ID: strconv.FormatUint(c.ID, 10), Title: title, Subtitle: sub, Route: checkupRoute(c),
			Meta:  jsonx.Obj("category", c.Category, "custom", c.Custom),
			rank:  rank,
			order: len(serviceRegistry) + i,
		})
	}
	return hits, nil
}

// checkupRoute mirrors the checkups card: self-exam types open the guide, everything else the detail page.
func checkupRoute(c Checkup) string {
	if strings.Contains(c.Key, "self_exam") {
		return "/checkups/self-exam"
	}
	return "/checkups/" + strconv.FormatUint(c.ID, 10)
}
