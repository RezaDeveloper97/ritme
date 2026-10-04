# Search

Site search (L4-04): `/search?q=` over every registered `Contracts\SearchProvider`.

- `Support\SearchTerms` normalises input the same way for every provider: Arabic ي/ى/ك → Persian ی/ک, ZWNJ, tatweel
  and diacritics removed, Persian/Arabic digits → Latin, lower case, collapsed whitespace; ≤ 5 tokens of ≥ 2 letters.
- Providers: `PostSearchProvider` (LIKE via `App\Domain\Blog\Queries\SearchPosts`, works on SQLite + MySQL) and
  `FaqSearchProvider` (filters the cached FAQ repository in PHP). Other contexts tag theirs with
  `SearchRegistry::TAG` in their service provider (directory L5-01, shop L6-01).
- `Actions\SearchSite` merges, ranks (title hits first) and paginates; results are cached briefly in the `pages`
  namespace (bumped by every content change) keyed by the normalised query.
- `Support\SearchTermLog` keeps an anonymised daily tally (term, count, zero-result) in the cache for the admin
  «جستجوهای بی‌نتیجه» report (L7-05 may move it to a table).
