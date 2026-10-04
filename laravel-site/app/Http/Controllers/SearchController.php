<?php

declare(strict_types=1);

namespace App\Http\Controllers;

use App\Domain\Search\Actions\SearchSite;
use App\Domain\Search\Data\SearchResults;
use App\Domain\Seo\Schema\Data\BreadcrumbItem;
use App\Domain\Seo\Schema\Enums\WebPageType;
use App\Domain\Seo\Schema\PageGraph;
use App\Domain\Seo\Schema\SchemaGraph;
use App\Domain\Seo\Schema\SchemaIds;
use App\Domain\Seo\SeoManager;
use App\Support\Text\PersianDigits;
use Illuminate\Contracts\Config\Repository as Config;
use Illuminate\Http\Request;
use Illuminate\Http\Response;

/**
 * `/search?q=` (L4-04): every registered SearchProvider through the SearchSite action (results cached briefly).
 * Always `noindex,follow` (route name `search` is in SeoManager::NOINDEX_ROUTES, so the page cache skips it too) and
 * rate limited per IP (route middleware). A plain GET form — works without JS.
 */
final class SearchController
{
    public const PER_PAGE = 10;

    /** Searches per IP and minute (`throttle:` route middleware). */
    public const PER_MINUTE = 30;

    public function __construct(
        private readonly SearchSite $search,
        private readonly SeoManager $seo,
        private readonly SchemaGraph $graph,
        private readonly Config $config,
    ) {}

    public function __invoke(Request $request): Response
    {
        $raw = $request->query->all()['q'] ?? null;
        $results = $this->search->handle(is_string($raw) ? $raw : null, $this->page($request), self::PER_PAGE);

        $title = $results->searched() ? __('search.seo_title_results', ['query' => $results->query]) : __('search.seo_title');
        $this->seo->title($title)->description(__('search.seo_description'))->noindex();
        $this->graph
            ->breadcrumbs(
                new BreadcrumbItem(PageGraph::HOME_LABEL, SchemaIds::root((string) $this->config->get('app.url'))),
                new BreadcrumbItem(__('search.name'), route('search')),
            )
            ->pageType(WebPageType::SearchResultsPage)
            ->pageName(__('search.name'));

        return response()->view('pages.search', [
            'results' => $results,
            'heading' => $results->searched() ? __('search.title_results', ['query' => $results->query]) : __('search.title'),
            'tooShort' => ! $results->searched() && $results->query !== '',
            'count' => PersianDigits::number($results->total),
            'pagination' => $this->pagination($request, $results),
        ])->header('X-Robots-Tag', 'noindex, follow');
    }

    private function page(Request $request): int
    {
        $raw = $request->query->all()['page'] ?? null;

        return is_string($raw) && ctype_digit($raw) && strlen($raw) <= 4 ? max(1, (int) $raw) : 1;
    }

    /**
     * Same shape as the magazine pagination partial.
     *
     * @return array{label: string, previous: string|null, next: string|null, pages: list<array{number: string, href: string|null, current: bool}>}|null
     */
    private function pagination(Request $request, SearchResults $results): ?array
    {
        $last = $results->lastPage();
        if (! $results->searched() || $last < 2) {
            return null;
        }

        $url = static fn (int $page): string => $request->url().'?'.http_build_query(['q' => $results->query] + ($page > 1 ? ['page' => $page] : []));
        $pages = [];
        $previous = 0;
        foreach (range(1, $last) as $n) {
            if ($n !== 1 && $n !== $last && abs($n - $results->page) > 2) {
                continue;
            }
            if ($previous !== 0 && $n - $previous > 1) {
                $pages[] = ['number' => '…', 'href' => null, 'current' => false];
            }
            $pages[] = ['number' => PersianDigits::toPersian($n), 'href' => $url($n), 'current' => $n === $results->page];
            $previous = $n;
        }

        return [
            'label' => __('blog.pagination.label'),
            'previous' => $results->page > 1 ? $url($results->page - 1) : null,
            'next' => $results->page < $last ? $url($results->page + 1) : null,
            'pages' => $pages,
        ];
    }
}
