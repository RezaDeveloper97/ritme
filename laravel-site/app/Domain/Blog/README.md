# Blog

Magazine: posts, categories (tree-light), tags, authors/medical reviewers (L4-01); feed and search come in L4-04.

- Read side: `Contracts/*Repository` → `Cached*` (`blog` ns, arrays) → `Eloquent*` → `Queries/*` (LatestPosts,
  PostsByCategory, RelatedPosts, Sitemap*). DTOs in `Data/`.
- Write side: `PostObserver` sanitises body/sources (`App\Support\Html`), assigns slugs + history, counts reading
  time, normalises the schedule; observers bump `blog`, `sitemap` (+ `pages`). Tag pivot changes: `SyncPostTags`.
- Scheduler (provider): `blog:publish-scheduled` every minute, `blog:flush-views` every five minutes.
- Sitemaps: `Sitemap/PostSitemapProvider` (`posts`), `Sitemap/CategorySitemapProvider` (`blog-categories`).
- Demo content: `php artisan db:seed --class=BlogSeeder` (not in DatabaseSeeder).
