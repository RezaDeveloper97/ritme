// Package search is the global search of the Today header (CB-NAV-01, board nbd_Nav_Search): one query
// over the user's own logs and care reminders, care programs, education and services, answered as groups of
// hits that each carry the in-app route to open.
//
// Sources and privacy:
//   - mine       the requesting user's health_log_entries (counted, never echoed: days with the matched log
//     item in the current cycle / last 30 days, and its peak pain score) and her care reminders. Every read is
//     scoped by the authenticated user id; nothing of another user is ever read.
//   - programs   the care programs that exist (registry in lang/<code>/search.json; condition programs join
//     when CB-COND-01 ships them).
//   - education  published articles (courses join when bloom N8 ships them).
//   - services   the services that exist (checkups, reminders) and the checkup catalog (shared rows plus the
//     user's own custom checkups). City services / the directory join when B-N7 / CB-DIR ship them.
//
// The shop is never searched here (DECISIONS #15: it has its own search). The query text and the hits are
// never logged; the request log only carries the path.
package search

import (
	"context"
	"fmt"
	"sort"
	"strings"

	"github.com/ritme/backend-go/internal/platform/civildate"
	"github.com/ritme/backend-go/internal/platform/jsonx"
)

// Scopes (the board's chips; «پزشک» joins services until the directory exists).
const (
	ScopeAll       = "all"
	ScopeMine      = "mine"
	ScopeEducation = "education"
	ScopePrograms  = "programs"
	ScopeServices  = "services"
)

// Scopes are the accepted scope values.
var Scopes = []string{ScopeAll, ScopeMine, ScopeEducation, ScopePrograms, ScopeServices}

// groupOrder is the board's group order.
var groupOrder = []string{ScopeMine, ScopePrograms, ScopeEducation, ScopeServices}

// Per-group limits: a few hits per group on «همه», a full page inside one scope.
const (
	DefaultLimitAll   = 3
	DefaultLimitScope = 20
	MaxLimit          = 20
	MinQueryLength    = 2
	MaxQueryLength    = 100
)

// Query is one validated search request.
type Query struct {
	UserID  uint64
	Text    string
	Scope   string
	Limit   int // per group; 0 = default for the scope
	Locale  string
	Default string // the default language code
	Today   civildate.Date
	// Taxonomy is the `log-taxonomy` namespace in the request locale (TranslationStore.NamespaceMessages).
	Taxonomy any
}

// Hit is one search result.
type Hit struct {
	Type     string
	ID       string
	Title    string
	Subtitle string // "" = null
	Route    string
	Meta     *jsonx.OrderedMap

	rank  int // higher first
	order int // tie-break inside a group: lower first
}

// Group is the hits of one source group.
type Group struct {
	Key   string
	Total int
	Items []Hit
}

// Result is the search answer.
type Result struct {
	Query  string
	Scope  string
	Groups []Group
}

// Service runs searches over its sources.
type Service struct {
	logs      LogSource
	periods   PeriodSource
	articles  ArticleSource
	checkups  CheckupSource
	reminders ReminderSource
}

// NewService wires the sources.
func NewService(logs LogSource, periods PeriodSource, articles ArticleSource, checkups CheckupSource, reminders ReminderSource) *Service {
	return &Service{logs: logs, periods: periods, articles: articles, checkups: checkups, reminders: reminders}
}

// Search runs q over the groups of its scope. Only the sources of those groups are read.
func (s *Service) Search(ctx context.Context, q Query) (*Result, error) {
	m := NewMatcher(q.Text)
	limit := q.Limit
	if limit <= 0 {
		limit = DefaultLimitScope
		if q.Scope == ScopeAll {
			limit = DefaultLimitAll
		}
	}
	res := &Result{Query: strings.TrimSpace(q.Text), Scope: q.Scope, Groups: []Group{}}
	for _, key := range groupOrder {
		if q.Scope != ScopeAll && q.Scope != key {
			continue
		}
		var hits []Hit
		if !m.Empty() {
			var err error
			if hits, err = s.group(ctx, key, q, m); err != nil {
				return nil, fmt.Errorf("search: %s: %w", key, err)
			}
		}
		res.Groups = append(res.Groups, page(key, hits, limit))
	}
	return res, nil
}

func (s *Service) group(ctx context.Context, key string, q Query, m Matcher) ([]Hit, error) {
	switch key {
	case ScopeMine:
		return s.mine(ctx, q, m)
	case ScopePrograms:
		return programs(q, m), nil
	case ScopeEducation:
		return s.education(ctx, q, m)
	case ScopeServices:
		return s.services(ctx, q, m)
	}
	return nil, nil
}

// page sorts the hits (rank, then source order) and keeps the first limit.
func page(key string, hits []Hit, limit int) Group {
	sort.SliceStable(hits, func(i, j int) bool {
		if hits[i].rank != hits[j].rank {
			return hits[i].rank > hits[j].rank
		}
		return hits[i].order < hits[j].order
	})
	g := Group{Key: key, Total: len(hits), Items: hits}
	if len(hits) > limit {
		g.Items = hits[:limit]
	}
	if g.Items == nil {
		g.Items = []Hit{}
	}
	return g
}

// JSON renders the result as data of the envelope.
func (r *Result) JSON() *jsonx.OrderedMap {
	groups := make([]*jsonx.OrderedMap, 0, len(r.Groups))
	for _, g := range r.Groups {
		items := make([]*jsonx.OrderedMap, 0, len(g.Items))
		for _, h := range g.Items {
			items = append(items, h.JSON())
		}
		groups = append(groups, jsonx.Obj("key", g.Key, "total", g.Total, "items", items))
	}
	return jsonx.Obj("query", r.Query, "scope", r.Scope, "groups", groups)
}

// JSON renders one hit.
func (h Hit) JSON() *jsonx.OrderedMap {
	var sub any
	if h.Subtitle != "" {
		sub = h.Subtitle
	}
	meta := h.Meta
	if meta == nil {
		meta = jsonx.NewObject()
	}
	return jsonx.Obj("type", h.Type, "id", h.ID, "title", h.Title, "subtitle", sub, "route", h.Route, "meta", meta)
}
