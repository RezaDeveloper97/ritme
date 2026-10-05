<?php

declare(strict_types=1);

namespace App\Domain\Seo\Audit;

use App\Domain\Seo\Analysis\AnalysisInput;
use App\Domain\Seo\Analysis\ContentDocument;
use App\Domain\Seo\Analysis\ContentType;
use App\Domain\Seo\Analysis\SeoAnalyzer;
use App\Domain\Seo\Audit\Contracts\PageFetcher;
use App\Domain\Seo\Audit\Crawl\ContentTargets;
use App\Domain\Seo\Audit\Crawl\FetchResult;
use App\Domain\Seo\Audit\Crawl\PageFacts;
use App\Domain\Seo\Audit\Crawl\SitemapPaths;
use App\Domain\Seo\Audit\Crawl\SiteUrls;
use Illuminate\Contracts\Config\Repository as Config;
use Illuminate\Routing\Route;
use Illuminate\Routing\Router;
use Illuminate\Support\Str;
use Throwable;

/**
 * The SEO audit crawler (L7-05). Renders pages in-process through PageFetcher (no network), seeds from every
 * parameterless GET route + every sitemap URL, follows internal links within the run's bounds, and checks:
 *
 *  - per page: SeoAuditor (title / description / h1 / canonical / OG / img attributes / href="#") and PageInspector
 *    (canonical host + self-reference, JSON-LD parse + required props, LCP / lazy images, thin content, HTML weight,
 *    request count, third-party requests, og:image file, slow render);
 *  - across pages: broken internal links, missing #fragment targets, links to redirects, redirect chains / loops,
 *    canonical pointing at a broken page, sitemap URLs that are noindex / redirect / broken / not canonical,
 *    indexable pages missing from the sitemap, orphan pages (no inbound internal link), duplicate titles and
 *    descriptions among all indexable pages (static + content), and the content analyser score of posts / products /
 *    places (focus keyword from their SEO tab).
 */
final class SeoAuditEngine
{
    /** URI patterns (Str::is) that are not public pages. */
    public const EXCLUDED_URIS = [
        'up', 'admin', 'admin/*', 'livewire/*', 'filament/*', 'storage/*', '_ignition/*', 'sanctum/*', '_components',
        '_preview/*', 'sitemap.xml', 'sitemaps/*', 'robots.txt', 'manifest.webmanifest', 'sw.js', 'build/*',
    ];

    public const MAX_REDIRECT_HOPS = 5;

    /** Analyser score under which a content page "needs work". */
    public const ANALYSIS_THRESHOLD = 60;

    /** Fragments browsers resolve without an element id. */
    private const IMPLICIT_FRAGMENTS = ['', 'top'];

    /** @var array<string, FetchResult> */
    private array $fetched = [];

    /** @var array<string, array{0: string, 1: string|null}> path => focus keyword + body HTML of its admin record */
    private array $focus = [];

    private int $fetches = 0;

    private int $maxFetches = 0;

    public function __construct(
        private readonly PageFetcher $fetcher,
        private readonly SeoAuditor $auditor,
        private readonly SitemapPaths $sitemap,
        private readonly SiteUrls $urls,
        private readonly ContentTargets $targets,
        private readonly SeoAnalyzer $analyzer,
        private readonly Router $router,
        private readonly Config $config,
    ) {}

    public function run(AuditOptions $options): AuditResult
    {
        $started = hrtime(true);
        $this->fetched = [];
        $this->focus = [];
        $this->fetches = 0;
        $this->maxFetches = max(50, $options->maxPages * 4);

        $environment = $this->config->get('app.env');
        if ($options->assumeProduction) {
            $this->config->set('app.env', 'production');
        }

        try {
            return $this->crawl($options, $started);
        } finally {
            $this->config->set('app.env', $environment);
            $this->fetcher->restore();
            $this->fetched = [];
            $this->focus = [];
        }
    }

    private function crawl(AuditOptions $options, int $started): AuditResult
    {
        $inspector = new PageInspector($this->urls, rtrim(public_path(), '/'));
        $deadline = $started + max(1, $options->timeLimit) * 1_000_000_000;
        $full = $options->isFullCrawl();

        $sitemap = [];
        if ($full && $options->includeSitemap) {
            try {
                $sitemap = array_fill_keys($this->sitemap->all(), true);
            } catch (Throwable) {
                $sitemap = [];
            }
        }

        // Seeds: explicit paths, or routes + sitemap; the previous run's failing / new pages first.
        $queue = [];
        if ($full) {
            foreach ($this->routePaths() as $path) {
                $queue[$path] = AuditedPage::SOURCE_ROUTE;
            }
            foreach (array_keys($sitemap) as $path) {
                $queue[$path] ??= AuditedPage::SOURCE_SITEMAP;
            }
        } else {
            foreach ($options->paths as $path) {
                $queue[SiteUrls::decode('/'.ltrim($path, '/'))] = AuditedPage::SOURCE_PATH;
            }
        }
        if ($options->priority !== []) {
            $queue = array_intersect_key($queue, array_flip($options->priority)) + $queue;
        }

        /** @var array<string, AuditedPage> $pages */
        $pages = [];
        /** @var array<string, PageFacts> $facts */
        $facts = [];
        /** @var array<string, list<array{path: string, query: string, fragment: string}>> $links */
        $links = [];
        $skipped = [];
        $seen = $queue;
        $truncatedBy = null;

        while ($queue !== []) {
            if (count($pages) >= $options->maxPages) {
                $truncatedBy = 'max-pages';
                break;
            }
            if (hrtime(true) > $deadline || $this->fetches >= $this->maxFetches) {
                $truncatedBy = 'time';
                break;
            }

            $path = (string) array_key_first($queue);
            $source = $queue[$path];
            unset($queue[$path]);

            [$chain, $final, $loop] = $this->follow($path);
            if ($final->isHtml() && count($chain) === 1) {
                $pageFacts = PageFacts::fromHtml($final->html);
                $target = $this->targets->for($final->routeName, $final->routeParameters);
                $audit = $this->auditor->auditPage($path, $final->html);
                $words = $target->contentHtml === null ? null : ContentDocument::parse($target->contentHtml)->wordCount;
                $audit = $audit->with(...$inspector->inspect($final, $pageFacts, $target->type, $words, $options->assumeProduction));
                $pages[$path] = new AuditedPage(
                    audit: $audit,
                    status: $final->status,
                    source: $source,
                    inSitemap: isset($sitemap[$path]),
                    milliseconds: $final->milliseconds,
                    htmlBytes: $pageFacts->htmlBytes,
                    requests: count($pageFacts->resources),
                    routeName: $final->routeName,
                    contentType: $target->type,
                    editUrl: $target->editUrl,
                    contentHash: sha1($pageFacts->mainHtml),
                );
                $facts[$path] = $pageFacts;
                $this->focus[$path] = [$target->focusKeyword, $target->contentHtml];
                $links[$path] = $this->internalLinks($pageFacts, $path);

                if ($full && $options->followLinks) {
                    foreach ($links[$path] as $link) {
                        $key = self::key($link['path'], $link['query']);
                        if (! isset($seen[$key]) && $this->crawlable($link['path'], $link['query'])) {
                            $seen[$key] = AuditedPage::SOURCE_LINK;
                            $queue[$key] = AuditedPage::SOURCE_LINK;
                        }
                    }
                }

                continue;
            }

            if ($source === AuditedPage::SOURCE_LINK || $source === AuditedPage::SOURCE_SITEMAP) {
                continue; // reported on the linking page / by the sitemap check
            }
            if ($loop || $final->isBroken()) {
                $reason = $final->error ?? 'HTTP '.$final->status;
                $pages[$path] = new AuditedPage(
                    audit: new PageAudit($path, null, null, false, [
                        $loop ? AuditIssue::error('redirect.loop', 'حلقه ریدایرکت: '.implode(' ← ', $chain)) : AuditIssue::error('render', "صفحه باز نشد ({$reason})."),
                    ]),
                    status: $final->status,
                    source: $source,
                    inSitemap: isset($sitemap[$path]),
                    milliseconds: $final->milliseconds,
                );

                continue;
            }
            $skipped[] = ['path' => $path, 'status' => $this->fetched[$path]->status, 'type' => $this->fetched[$path]->contentType];
        }

        $pages = $this->checkLinks($pages, $facts, $links);
        $pages = $this->checkCanonicalTargets($pages, $facts);
        if ($sitemap !== []) {
            $pages = $this->checkSitemap($pages, $facts, $sitemap);
        }
        if ($full && $truncatedBy === null) {
            $pages = $this->checkOrphans($pages);
        }
        $pages = $this->checkDuplicates($pages);
        $pages = $this->analyse($pages, $facts);

        return new AuditResult(
            pages: array_values($pages),
            skipped: $skipped,
            truncated: $truncatedBy !== null,
            truncatedBy: $truncatedBy,
            milliseconds: (int) round((hrtime(true) - $started) / 1_000_000),
            fetches: $this->fetches,
        );
    }

    /**
     * Fix link for one finding: the page's admin edit form, the redirect manager for broken pages, or the indexing
     * controls for sitemap findings without an editable page.
     */
    public function fixUrl(string $code, ?string $editUrl): ?string
    {
        return match (true) {
            $code === 'render', $code === 'redirect.loop' => $this->targets->redirectsUrl(),
            str_starts_with($code, 'sitemap.') => $editUrl ?? $this->targets->indexingUrl(),
            default => $editUrl,
        };
    }

    /**
     * @return array{0: list<string>, 1: FetchResult, 2: bool} redirect chain (first = requested), final response, loop
     */
    private function follow(string $key): array
    {
        $chain = [$key];
        $current = $key;
        for ($hop = 0; $hop <= self::MAX_REDIRECT_HOPS; $hop++) {
            $result = $this->fetch($current);
            if (! $result->isRedirect()) {
                return [$chain, $result, false];
            }
            $next = $this->urls->resolve((string) $result->location, $current);
            if ($next === null) {
                return [$chain, $result, false]; // external redirect: not followed
            }
            $current = self::key($next['path'], $next['query']);
            if (in_array($current, $chain, true)) {
                return [[...$chain, $current], $result, true];
            }
            $chain[] = $current;
        }

        return [$chain, $this->fetch($current), true];
    }

    private function fetch(string $key): FetchResult
    {
        if (isset($this->fetched[$key])) {
            return $this->fetched[$key];
        }
        $this->fetches++;
        [$path, $query] = array_pad(explode('?', $key, 2), 2, '');

        $result = $this->fetcher->fetch(SiteUrls::encode($path, $query));

        return $this->fetched[$key] = $result;
    }

    /**
     * @param  array<string, AuditedPage>  $pages
     * @param  array<string, PageFacts>  $facts
     * @param  array<string, list<array{path: string, query: string, fragment: string}>>  $links
     * @return array<string, AuditedPage>
     */
    private function checkLinks(array $pages, array $facts, array $links): array
    {
        $inbound = [];
        $public = rtrim(public_path(), '/');

        foreach ($links as $source => $targets) {
            $issues = [];
            $reported = [];
            foreach ($targets as $link) {
                $key = self::key($link['path'], $link['query']);
                $isSelf = $link['path'] === $source || $key === $source;

                if (! $isSelf && $this->isStaticFile($link['path'], $public)) {
                    continue;
                }
                if (! $isSelf && ! isset($this->fetched[$key]) && $this->fetches >= $this->maxFetches) {
                    continue; // over budget: unchecked
                }

                [$chain, $final, $loop] = $isSelf ? [[$source], null, false] : $this->follow($key);
                $finalKey = (string) end($chain);
                if (isset($reported[$key])) {
                    continue;
                }

                if ($loop) {
                    $issues[] = AuditIssue::error('redirect.loop', "پیوند به {$key} در حلقه ریدایرکت می‌افتد.");
                    $reported[$key] = true;

                    continue;
                }
                if ($final !== null && $final->isBroken()) {
                    $issues[] = AuditIssue::error('link.broken', "پیوند به {$key} خراب است (HTTP {$final->status}).");
                    $reported[$key] = true;

                    continue;
                }
                if (count($chain) > 2) {
                    $issues[] = AuditIssue::error('redirect.chain', "پیوند به {$key} از ".(count($chain) - 1).' ریدایرکت پشت‌سرهم می‌گذرد: '.implode(' → ', $chain));
                    $reported[$key] = true;
                } elseif (count($chain) === 2) {
                    $issues[] = AuditIssue::warning('link.redirect', "پیوند به {$key} ریدایرکت می‌شود؛ مستقیم به {$finalKey} پیوند دهید.");
                    $reported[$key] = true;
                }

                if (! $isSelf && $finalKey !== $source) {
                    $inbound[$finalKey][$source] = true;
                }

                $fragment = $link['fragment'];
                if (in_array($fragment, self::IMPLICIT_FRAGMENTS, true) || str_starts_with($fragment, ':~:')) {
                    continue;
                }
                $targetFacts = $facts[$isSelf ? $source : $finalKey] ?? null;
                if ($targetFacts !== null && ! isset($targetFacts->ids[$fragment])) {
                    $where = $isSelf ? 'همین صفحه' : $finalKey;
                    $issues[] = AuditIssue::error('link.fragment', "لنگر #{$fragment} در {$where} پیدا نشد.");
                    $reported[$key.'#'.$fragment] = true;
                }
            }

            if (isset($pages[$source]) && $issues !== []) {
                $pages[$source] = $pages[$source]->with(...$issues);
            }
        }

        foreach ($pages as $path => $page) {
            $pages[$path] = $page->withInbound(count($inbound[$path] ?? []));
        }

        return $pages;
    }

    /**
     * @param  array<string, AuditedPage>  $pages
     * @param  array<string, PageFacts>  $facts
     * @return array<string, AuditedPage>
     */
    private function checkCanonicalTargets(array $pages, array $facts): array
    {
        foreach ($facts as $path => $pageFacts) {
            if (count($pageFacts->canonicals) !== 1) {
                continue;
            }
            $resolved = $this->urls->resolve($pageFacts->canonicals[0], $path);
            if ($resolved === null) {
                continue;
            }
            $key = self::key($resolved['path'], self::pageQuery($resolved['query']));
            if ($key === $path || ! isset($this->fetched[$key]) && $this->fetches >= $this->maxFetches) {
                continue;
            }
            [$chain, $final, $loop] = $this->follow($key);
            if ($loop || $final->isBroken() || count($chain) > 1) {
                $pages[$path] = $pages[$path]->with(AuditIssue::error('canonical.target', "canonical به {$key} اشاره می‌کند که ".($final->isBroken() ? "خراب است (HTTP {$final->status})." : 'ریدایرکت می‌شود.')));
            }
        }

        return $pages;
    }

    /**
     * @param  array<string, AuditedPage>  $pages
     * @param  array<string, PageFacts>  $facts
     * @param  array<string, true>  $sitemap
     * @return array<string, AuditedPage>
     */
    private function checkSitemap(array $pages, array $facts, array $sitemap): array
    {
        foreach (array_keys($sitemap) as $path) {
            if (! isset($this->fetched[$path])) {
                continue; // not reached within the run's bounds
            }
            [$chain, $final, $loop] = $this->follow($path);
            $issue = null;
            if ($loop || $final->isBroken()) {
                $issue = AuditIssue::error('sitemap.status', 'این نشانی در نقشه سایت هست ولی باز نمی‌شود (HTTP '.$final->status.').');
            } elseif (count($chain) > 1) {
                $issue = AuditIssue::error('sitemap.redirect', 'این نشانی در نقشه سایت هست ولی به '.end($chain).' ریدایرکت می‌شود.');
            } elseif (isset($facts[$path]) && ! $facts[$path]->indexable()) {
                $issue = AuditIssue::error('sitemap.noindex', 'صفحه noindex است ولی در نقشه سایت آمده.');
            } elseif (isset($facts[$path]) && count($facts[$path]->canonicals) === 1) {
                $resolved = $this->urls->resolve($facts[$path]->canonicals[0], $path);
                if ($resolved !== null && self::key($resolved['path'], self::pageQuery($resolved['query'])) !== $path) {
                    $issue = AuditIssue::error('sitemap.canonical', "نقشه سایت این نشانی را دارد ولی canonical صفحه {$resolved['path']} است.");
                }
            }
            if ($issue === null) {
                continue;
            }
            $pages[$path] = isset($pages[$path])
                ? $pages[$path]->with($issue)
                : new AuditedPage(new PageAudit($path, null, null, false, [$issue]), $final->status, AuditedPage::SOURCE_SITEMAP, true);
        }

        foreach ($pages as $path => $page) {
            if ($page->audit->indexable && ! $page->inSitemap && $page->source !== AuditedPage::SOURCE_PATH && ! str_contains($path, '?')) {
                $pages[$path] = $page->with(AuditIssue::notice('sitemap.missing', 'صفحه قابل نمایه است ولی در نقشه سایت نیست.'));
            }
        }

        return $pages;
    }

    /**
     * @param  array<string, AuditedPage>  $pages
     * @return array<string, AuditedPage>
     */
    private function checkOrphans(array $pages): array
    {
        foreach ($pages as $path => $page) {
            if ($path !== '/' && $page->audit->indexable && $page->status === 200 && $page->inbound === 0) {
                $pages[$path] = $page->with(AuditIssue::warning('page.orphan', 'هیچ صفحه دیگری به این صفحه پیوند نمی‌دهد.'));
            }
        }

        return $pages;
    }

    /**
     * @param  array<string, AuditedPage>  $pages
     * @return array<string, AuditedPage>
     */
    private function checkDuplicates(array $pages): array
    {
        $keys = array_keys($pages);
        $audits = $this->auditor->crossCheck(array_values(array_map(static fn (AuditedPage $page): PageAudit => $page->audit, $pages)));
        foreach ($audits as $index => $audit) {
            $pages[$keys[$index]] = $pages[$keys[$index]]->withAudit($audit);
        }

        return $pages;
    }

    /**
     * @param  array<string, AuditedPage>  $pages
     * @param  array<string, PageFacts>  $facts
     * @return array<string, AuditedPage>
     */
    private function analyse(array $pages, array $facts): array
    {
        $host = $this->urls->host();
        foreach ($pages as $path => $page) {
            $type = $page->contentType;
            if (! isset($facts[$path]) || ! in_array($type, [ContentType::Post, ContentType::Product, ContentType::Place], true)) {
                continue;
            }
            $pageFacts = $facts[$path];
            $segments = explode('/', trim($path, '/'));
            $analysis = $this->analyzer->analyze(new AnalysisInput(
                type: $type,
                title: (string) $pageFacts->title,
                heading: (string) $pageFacts->heading,
                description: (string) $pageFacts->description,
                slug: (string) end($segments),
                focusKeyword: $this->focus[$path][0] ?? '',
                contentHtml: $this->focus[$path][1] ?? $pageFacts->mainHtml,
                ownHosts: [$host],
            ));
            $page = $page->withAnalysisScore($analysis->score);
            if ($analysis->score < self::ANALYSIS_THRESHOLD) {
                $page = $page->with(AuditIssue::notice('content.score', "امتیاز تحلیل سئوی محتوا {$analysis->score} از ۱۰۰ است (کمتر از ".self::ANALYSIS_THRESHOLD.').'));
            }
            $pages[$path] = $page;
        }

        return $pages;
    }

    /**
     * @return list<array{path: string, query: string, fragment: string}>
     */
    private function internalLinks(PageFacts $facts, string $path): array
    {
        $links = [];
        foreach ($facts->links as $href) {
            if ($href === '#' || trim($href) === '') {
                continue; // SeoAuditor: link.hash
            }
            $resolved = $this->urls->resolve($href, $path);
            if ($resolved === null || Str::is(self::EXCLUDED_URIS, ltrim($resolved['path'], '/'))) {
                continue;
            }
            $resolved['query'] = self::pageQuery($resolved['query']);
            $links[] = $resolved;
        }

        return $links;
    }

    /** Only extension-less paths are crawled as pages; query strings other than ?page=n are never followed. */
    private function crawlable(string $path, string $query): bool
    {
        return ($query === '' || str_starts_with($query, 'page='))
            && pathinfo($path, PATHINFO_EXTENSION) === ''
            && ! Str::is(self::EXCLUDED_URIS, ltrim($path, '/'));
    }

    private function isStaticFile(string $path, string $public): bool
    {
        return pathinfo($path, PATHINFO_EXTENSION) !== '' && ! str_contains($path, '..') && is_file($public.$path);
    }

    /**
     * @return list<string>
     */
    private function routePaths(): array
    {
        $paths = [];
        foreach ($this->router->getRoutes()->getRoutes() as $route) {
            if ($this->auditable($route)) {
                $paths[] = SiteUrls::decode('/'.ltrim($route->uri(), '/'));
            }
        }

        return array_values(array_unique($paths));
    }

    private function auditable(Route $route): bool
    {
        if (! in_array('GET', $route->methods(), true) || Str::is(self::EXCLUDED_URIS, $route->uri())) {
            return false;
        }

        // Only routes without required parameters ({id} but not {id?}).
        return preg_match('/\{[^}?]+\}/', $route->uri()) !== 1;
    }

    private static function key(string $path, string $query): string
    {
        return $path.($query === '' ? '' : '?'.$query);
    }

    /** Keeps only `page=n` (n > 1) — the one query parameter that makes a different indexable page. */
    private static function pageQuery(string $query): string
    {
        parse_str($query, $parameters);
        $page = $parameters['page'] ?? null;

        return is_string($page) && ctype_digit($page) && (int) $page > 1 ? 'page='.(int) $page : '';
    }
}
