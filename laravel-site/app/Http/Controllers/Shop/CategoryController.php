<?php

declare(strict_types=1);

namespace App\Http\Controllers\Shop;

use App\Domain\Media\Contracts\MediaRepository;
use App\Domain\Media\Data\MediaData;
use App\Domain\Seo\Contracts\SeoMetaRepository;
use App\Domain\Seo\Schema\Data\BreadcrumbItem;
use App\Domain\Seo\Schema\Data\ListEntry;
use App\Domain\Seo\Schema\Enums\WebPageType;
use App\Domain\Seo\Schema\Node;
use App\Domain\Seo\Schema\Nodes\ItemListNode;
use App\Domain\Seo\Schema\PageGraph;
use App\Domain\Seo\Schema\SchemaGraph;
use App\Domain\Seo\Schema\SchemaIds;
use App\Domain\Seo\SeoManager;
use App\Domain\Seo\Support\CanonicalUrl;
use App\Domain\Seo\Support\DescriptionText;
use App\Domain\Shop\Catalog\Contracts\CatalogRepository;
use App\Domain\Shop\Catalog\Contracts\ProductRepository;
use App\Domain\Shop\Catalog\Data\BrandData;
use App\Domain\Shop\Catalog\Data\CategoryData;
use App\Domain\Shop\Catalog\Data\CategoryTree;
use App\Domain\Shop\Catalog\Data\ProductCardData;
use App\Domain\Shop\Catalog\Data\ProductCriteria;
use App\Domain\Shop\Catalog\Data\ProductFacets;
use App\Domain\Shop\Catalog\Data\ProductPage;
use App\Domain\Shop\Catalog\Enums\ProductSort;
use App\Domain\Shop\Catalog\Models\Category;
use App\Domain\Shop\Catalog\Support\ProductListIndexing;
use App\Domain\Shop\Catalog\Support\ShopUrls;
use App\Support\Money\Money;
use App\Support\Text\PersianDigits;
use App\Support\Text\Toman;
use Illuminate\Contracts\Config\Repository as Config;
use Illuminate\Contracts\View\Factory as ViewFactory;
use Illuminate\Contracts\View\View;
use Illuminate\Http\RedirectResponse;
use Illuminate\Http\Request;
use Illuminate\Support\Str;

/**
 * Shop category listing (`/shop/category/{slug}`, design/html/shop-list.html) — pages/shop/category.blade.php.
 *
 * A category lists its whole visible subtree (ProductsInCategory). Filters are crawlable GET parameters: `brand[]`
 * (slugs), `size[]`, `color[]` (variant values), `min` / `max` (tomans), `stock=1`, plus `sort` and `page`. Every
 * request whose query is not in its canonical shape (unknown or empty parameters, invalid values, unsorted lists,
 * `?page=1`, Persian digits in prices, the default sort) is 301-redirected once to the canonical URL, so each filter
 * state has exactly one address. Indexable: the category and its pages (`?page=n` self-canonical, «صفحه n»); any
 * filter or non-default sort → `noindex,follow` with the canonical on the category. Out-of-range pages → 404. Lists
 * without a real (non-demo) product are noindex.
 *
 * Filter options come from the category's facets (computed before filters, so options never vanish while
 * filtering). JSON-LD: CollectionPage + ItemList of the cards + BreadcrumbList (via <x-ui.breadcrumbs>). Reads go
 * through the cached Shop repositories; the view gets DTOs and scalar arrays.
 */
final class CategoryController
{
    public const PER_PAGE = 24;

    /** Query parameters in canonical order. */
    private const PARAMS = ['brand', 'size', 'color', 'min', 'max', 'stock', 'sort', 'page'];

    /** Upper bound of a price filter (tomans) and of list lengths, against junk URLs. */
    private const MAX_PRICE = 1_000_000_000;

    private const MAX_VALUES = 20;

    public function __construct(
        private readonly CatalogRepository $catalog,
        private readonly ProductRepository $products,
        private readonly MediaRepository $media,
        private readonly SeoMetaRepository $meta,
        private readonly ShopUrls $urls,
        private readonly ViewFactory $views,
        private readonly Config $config,
    ) {}

    /** SeoManager and SchemaGraph are request-scoped: injected per call, not into the (route-cached) controller. */
    public function __invoke(Request $request, SeoManager $seo, SchemaGraph $graph, string $slug): View|RedirectResponse
    {
        $tree = $this->catalog->categoryTree();
        $category = $tree->findBySlug($slug) ?? abort(404);

        // Facets of the whole category = the unfiltered first page (one shared cache entry).
        $base = $this->products->list(new ProductCriteria(categoryId: $category->id, perPage: self::PER_PAGE));
        $state = $this->state($request, $base->facets);
        if ($this->needsRedirect($request, $state)) {
            return new RedirectResponse($this->url($category, $state, keepTracking: $request->query->all()), 301);
        }

        $list = $this->isDefault($state) ? $base : $this->products->list($this->criteria($category, $state));
        if ($state['page'] > 1 && $list->isOutOfRange()) {
            abort(404);
        }

        $trail = $this->trail($tree, $category);
        $this->applySeo($seo, $category, $state, $list);
        $this->schema($graph, $request, $category, $list->items);

        return $this->views->make('pages.shop.category', [
            'category' => $category,
            'breadcrumbs' => $trail,
            'subnav' => $this->subnav($tree, $category),
            'summary' => $this->summary($category, $state, $list),
            'sorts' => $this->sorts($category, $state),
            'filters' => $this->filters($tree, $category, $state, $base->facets),
            'active' => $this->activeFilters($category, $state),
            'cards' => $list->items,
            'media' => $this->coverMedia($list->items),
            'demo' => array_filter($list->items, static fn (ProductCardData $c): bool => $c->isDemo) !== [],
            'empty' => $base->total === 0 ? (string) __('shop.category.empty_category') : (string) __('shop.category.empty'),
            'resetUrl' => $this->url($category, $this->blank()),
            'pagination' => $this->pagination($category, $state, $list),
        ]);
    }

    /**
     * Parsed, validated filter state. Unknown brands / sizes / colours and invalid numbers are dropped (the redirect
     * then removes them from the URL).
     *
     * @return array{brands: list<BrandData>, sizes: list<string>, colors: list<string>, min: ?int, max: ?int, stock: bool, sort: ProductSort, page: int}
     */
    private function state(Request $request, ProductFacets $facets): array
    {
        $query = $request->query->all();

        $brandSlugs = self::values($query['brand'] ?? null);
        $brands = array_values(array_filter($this->catalog->brands(), static fn (BrandData $b): bool => in_array($b->slug, $brandSlugs, true)));
        usort($brands, static fn (BrandData $a, BrandData $b): int => strcmp($a->slug, $b->slug));

        $sizes = array_values(array_filter($facets->sizes, static fn (string $s): bool => in_array($s, self::values($query['size'] ?? null), true)));
        $colorNames = array_map(static fn (array $c): string => $c['name'], $facets->colors);
        $colors = array_values(array_filter($colorNames, static fn (string $c): bool => in_array($c, self::values($query['color'] ?? null), true)));
        sort($sizes);
        sort($colors);

        $min = self::price($query['min'] ?? null);
        $max = self::price($query['max'] ?? null);
        if ($min !== null && $max !== null && $min > $max) {
            [$min, $max] = [$max, $min];
        }

        $sort = is_string($query['sort'] ?? null) ? ProductSort::tryFrom($query['sort']) : null;

        return [
            'brands' => $brands,
            'sizes' => $sizes,
            'colors' => $colors,
            'min' => $min,
            'max' => $max,
            'stock' => ($query['stock'] ?? null) === '1',
            'sort' => $sort ?? ProductSort::default(),
            'page' => self::page($query['page'] ?? null),
        ];
    }

    /**
     * @return list<string>
     */
    private static function values(mixed $raw): array
    {
        $raw = is_string($raw) ? [$raw] : (is_array($raw) ? $raw : []);
        $values = [];
        foreach (array_slice($raw, 0, self::MAX_VALUES) as $value) {
            if (is_string($value) && ($value = trim($value)) !== '' && mb_strlen($value) <= 60) {
                $values[] = $value;
            }
        }

        return $values;
    }

    private static function price(mixed $raw): ?int
    {
        if (! is_string($raw)) {
            return null;
        }
        $digits = PersianDigits::toLatin(str_replace([',', '٬', ' '], '', trim($raw)));
        if ($digits === '' || ! ctype_digit($digits) || strlen($digits) > 10) {
            return null;
        }
        $value = (int) $digits;

        return $value > 0 && $value <= self::MAX_PRICE ? $value : null;
    }

    private static function page(mixed $raw): int
    {
        if ($raw === null) {
            return 1;
        }
        if (! is_string($raw) || ! ctype_digit($raw) || (int) $raw < 1 || strlen($raw) > 6) {
            abort(404);
        }

        return (int) $raw;
    }

    /**
     * @param  array<string, mixed>  $state
     */
    private function needsRedirect(Request $request, array $state): bool
    {
        $actual = array_filter($request->query->all(), static fn (int|string $key): bool => ! Str::is(SeoManager::TRACKING_PARAMS, (string) $key), ARRAY_FILTER_USE_KEY);

        return $actual !== $this->query($state);
    }

    /**
     * @param  array<string, mixed>  $state
     * @param  array<string, mixed>  $keepTracking  request query whose tracking parameters survive the redirect
     */
    private function url(CategoryData $category, array $state, array $keepTracking = []): string
    {
        $query = $this->query($state);
        foreach ($keepTracking as $key => $value) {
            if (Str::is(SeoManager::TRACKING_PARAMS, (string) $key)) {
                $query[(string) $key] = $value;
            }
        }

        $path = route('shop.category', [$category->slug]);

        return $query === [] ? $path : $path.'?'.(string) preg_replace('/%5B\d+%5D=/', '%5B%5D=', http_build_query($query));
    }

    /**
     * Canonical query of a state: set filters only, fixed order, sorted lists, default sort and page 1 omitted.
     *
     * @param  array<string, mixed>  $state
     * @return array<string, string|list<string>>
     */
    private function query(array $state): array
    {
        /** @var list<BrandData> $brands */
        $brands = $state['brands'] ?? [];
        $values = [
            'brand' => array_map(static fn (BrandData $b): string => $b->slug, $brands),
            'size' => $state['sizes'] ?? [],
            'color' => $state['colors'] ?? [],
            'min' => isset($state['min']) ? (string) $state['min'] : null,
            'max' => isset($state['max']) ? (string) $state['max'] : null,
            'stock' => ($state['stock'] ?? false) ? '1' : null,
            'sort' => isset($state['sort']) && $state['sort'] !== ProductSort::default() ? $state['sort']->value : null,
            'page' => ($state['page'] ?? 1) > 1 ? (string) $state['page'] : null,
        ];

        $query = [];
        foreach (self::PARAMS as $key) {
            if ($values[$key] !== null && $values[$key] !== []) {
                $query[$key] = $values[$key];
            }
        }

        return $query;
    }

    /**
     * @return array{brands: list<BrandData>, sizes: list<string>, colors: list<string>, min: null, max: null, stock: false, sort: ProductSort, page: int}
     */
    private function blank(): array
    {
        return ['brands' => [], 'sizes' => [], 'colors' => [], 'min' => null, 'max' => null, 'stock' => false, 'sort' => ProductSort::default(), 'page' => 1];
    }

    /**
     * @param  array<string, mixed>  $state
     */
    private function isDefault(array $state): bool
    {
        return $this->query($state) === [];
    }

    /**
     * Anything beyond the page number (filters or a non-default sort) — such URLs are not landings.
     *
     * @param  array<string, mixed>  $state
     */
    private function isFiltered(array $state): bool
    {
        return $this->query(['page' => 1] + $state) !== [];
    }

    /**
     * @param  array<string, mixed>  $state
     */
    private function criteria(CategoryData $category, array $state): ProductCriteria
    {
        /** @var list<BrandData> $brands */
        $brands = $state['brands'];

        return new ProductCriteria(
            categoryId: $category->id,
            brandIds: array_map(static fn (BrandData $b): int => $b->id, $brands),
            sizes: $state['sizes'],
            colors: $state['colors'],
            minPrice: $state['min'] === null ? null : Money::fromToman($state['min']),
            maxPrice: $state['max'] === null ? null : Money::fromToman($state['max']),
            inStockOnly: $state['stock'],
            sort: $state['sort'],
            page: $state['page'],
            perPage: self::PER_PAGE,
        );
    }

    /**
     * Admin seo_meta of the category wins over the lang templates; «صفحه n» on later pages; filter states noindex
     * with the canonical on the category; demo-only / empty lists noindex.
     *
     * @param  array<string, mixed>  $state
     */
    private function applySeo(SeoManager $seo, CategoryData $category, array $state, ProductPage $list): void
    {
        $seo->for((new Category)->forceFill(['id' => $category->id]));
        $meta = $this->meta->forModel((new Category)->getMorphClass(), $category->id);

        $title = $meta->title ?? (string) __('shop.category.seo_title', ['name' => $category->name]);
        $description = $meta->description ?? ($category->intro !== null && $category->intro !== ''
            ? $category->name.': '.$category->intro
            : (string) __('shop.category.seo_description', ['name' => $category->name]));

        $page = (int) $state['page'];
        if ($page > 1) {
            $suffix = (string) __('shop.page_suffix', ['page' => fa_digits($page)]);
            $seo->title($title.' — '.$suffix)->description(DescriptionText::fromExcerpt($suffix.' — '.$description));
        } else {
            $seo->title($title)->description($meta?->description !== null ? $description : DescriptionText::fromExcerpt($description));
        }

        if ($this->isFiltered($state)) {
            $seo->filtered()->canonical(route('shop.category', [$category->slug]));
        }
        if (! ProductListIndexing::showsRealProduct($list->items)) {
            $seo->noindex();
        }
    }

    /**
     * @param  list<ProductCardData>  $items
     */
    private function schema(SchemaGraph $graph, Request $request, CategoryData $category, array $items): void
    {
        // BreadcrumbList: registered by <x-ui.breadcrumbs> in the view (same items as the visible trail).
        $graph->pageType(WebPageType::CollectionPage)->pageName($category->name);
        if ($items === []) {
            return;
        }

        $pageUrl = CanonicalUrl::normalize($request->fullUrl(), (string) $this->config->get('app.url'));
        $graph->add(ItemListNode::make($pageUrl, array_map(
            fn (ProductCardData $card): ListEntry => new ListEntry($this->urls->product($card->slug), $card->title),
            $items,
        ), $category->name));
        $graph->add(['@id' => SchemaIds::webPage($pageUrl), 'mainEntity' => Node::ref(SchemaIds::itemList($pageUrl))]);
    }

    /**
     * Home → فروشگاه → ancestors → category.
     *
     * @return list<BreadcrumbItem>
     */
    private function trail(CategoryTree $tree, CategoryData $category): array
    {
        $trail = [
            new BreadcrumbItem(PageGraph::HOME_LABEL, SchemaIds::root((string) $this->config->get('app.url'))),
            new BreadcrumbItem((string) __('shop.name'), route('shop.index')),
        ];
        foreach ($tree->path($category->id) as $step) {
            $trail[] = new BreadcrumbItem($step->name, $this->urls->category($step->slug));
        }

        return $trail;
    }

    /**
     * Department switch (root categories) + the categories of the current department.
     *
     * @return array{departments: list<array{label: string, href: string, active: bool, icon: string}>, links: list<array{label: string, href: string, active: bool}>, label: string}
     */
    private function subnav(CategoryTree $tree, CategoryData $category): array
    {
        $root = $tree->path($category->id)[0] ?? $category;
        $path = array_map(static fn (CategoryData $c): int => $c->id, $tree->path($category->id));

        $departments = [];
        foreach ($tree->roots() as $department) {
            $departments[] = [
                'label' => $department->name,
                'href' => $this->urls->category($department->slug),
                'active' => $department->id === $root->id,
                'icon' => ShopHomeController::departmentIcon($department->slug),
            ];
        }

        $links = [];
        foreach ($tree->children($root->id) as $child) {
            $links[] = ['label' => $child->name, 'href' => $this->urls->category($child->slug), 'active' => in_array($child->id, $path, true)];
        }

        return ['departments' => $departments, 'links' => $links, 'label' => (string) __('shop.subnav.categories', ['name' => $root->name])];
    }

    /**
     * @param  array<string, mixed>  $state
     */
    private function summary(CategoryData $category, array $state, ProductPage $list): string
    {
        $count = (string) __($this->hasFilters($state) ? 'shop.category.count_filtered' : 'shop.category.count', ['count' => fa_digits($list->total)]);

        return $category->intro !== null && $category->intro !== '' ? $count.' · '.$category->intro : $count;
    }

    /**
     * @param  array<string, mixed>  $state
     */
    private function hasFilters(array $state): bool
    {
        return $this->query(['page' => 1, 'sort' => ProductSort::default()] + $state) !== [];
    }

    /**
     * @param  array<string, mixed>  $state
     * @return list<array{label: string, href: string, active: bool}>
     */
    private function sorts(CategoryData $category, array $state): array
    {
        return array_map(fn (ProductSort $sort): array => [
            'label' => $sort->label(),
            'href' => $this->url($category, ['sort' => $sort, 'page' => 1] + $state),
            'active' => $state['sort'] === $sort,
        ], ProductSort::cases());
    }

    /**
     * Side panel: category links (subcategories with counts, or the siblings of a leaf) and the GET form fields.
     *
     * @param  array<string, mixed>  $state
     * @return array{categories: list<array{label: string, href: string, count: ?string, active: bool}>, brands: list<array{value: string, label: string, count: string, checked: bool}>, sizes: list<array{value: string, checked: bool}>, colors: list<array{value: string, label: string, hex: ?string, checked: bool}>, price: array{min: ?int, max: ?int, from: ?string, to: ?string}, stock: bool, action: string, hidden: array<string, string>, filtered: bool}
     */
    private function filters(CategoryTree $tree, CategoryData $category, array $state, ProductFacets $facets): array
    {
        $categories = [];
        $children = $tree->children($category->id);
        if ($children !== []) {
            foreach ($children as $child) {
                $count = $facets->categoryCounts[$child->id] ?? 0;
                if ($count > 0) {
                    $categories[] = ['label' => $child->name, 'href' => $this->urls->category($child->slug), 'count' => fa_digits($count), 'active' => false];
                }
            }
        } elseif ($category->parentId !== null) {
            foreach ($tree->children($category->parentId) as $sibling) {
                $categories[] = ['label' => $sibling->name, 'href' => $this->urls->category($sibling->slug), 'count' => null, 'active' => $sibling->id === $category->id];
            }
        }

        /** @var list<BrandData> $selectedBrands */
        $selectedBrands = $state['brands'];
        $selected = array_map(static fn (BrandData $b): string => $b->slug, $selectedBrands);

        return [
            'categories' => $categories,
            'brands' => array_map(static fn (array $b): array => [
                'value' => $b['slug'], 'label' => $b['name'], 'count' => fa_digits($b['count']), 'checked' => in_array($b['slug'], $selected, true),
            ], $facets->brands),
            'sizes' => array_map(static fn (string $s): array => ['value' => $s, 'checked' => in_array($s, $state['sizes'], true)], $facets->sizes),
            'colors' => array_map(static fn (array $c): array => [
                'value' => $c['name'],
                'label' => (string) __('shop.filters.color', ['name' => $c['name']]),
                'hex' => is_string($c['hex']) && preg_match('/^#[0-9A-Fa-f]{6}$/', $c['hex']) === 1 ? $c['hex'] : null,
                'checked' => in_array($c['name'], $state['colors'], true),
            ], $facets->colors),
            'price' => [
                'min' => $state['min'],
                'max' => $state['max'],
                'from' => $facets->minPrice === null ? null : Toman::format($facets->minPrice->toToman()),
                'to' => $facets->maxPrice === null ? null : Toman::format($facets->maxPrice->toToman()),
            ],
            'stock' => $state['stock'],
            'action' => route('shop.category', [$category->slug]),
            'hidden' => $state['sort'] === ProductSort::default() ? [] : ['sort' => $state['sort']->value],
            'filtered' => $this->hasFilters($state),
        ];
    }

    /**
     * Removable chips of the active filters (each link drops one value).
     *
     * @param  array<string, mixed>  $state
     * @return list<array{label: string, href: string}>
     */
    private function activeFilters(CategoryData $category, array $state): array
    {
        $base = ['page' => 1] + $state;
        $chips = [];
        /** @var list<BrandData> $brands */
        $brands = $state['brands'];
        foreach ($brands as $brand) {
            $chips[] = ['label' => $brand->name, 'href' => $this->url($category, ['brands' => array_values(array_filter($brands, static fn (BrandData $b): bool => $b->id !== $brand->id))] + $base)];
        }
        foreach (['sizes', 'colors'] as $key) {
            /** @var list<string> $values */
            $values = $state[$key];
            foreach ($values as $value) {
                $chips[] = ['label' => $value, 'href' => $this->url($category, [$key => array_values(array_diff($values, [$value]))] + $base)];
            }
        }
        if ($state['min'] !== null) {
            $chips[] = ['label' => (string) __('shop.filters.price_from', ['amount' => Toman::format($state['min'])]), 'href' => $this->url($category, ['min' => null] + $base)];
        }
        if ($state['max'] !== null) {
            $chips[] = ['label' => (string) __('shop.filters.price_to', ['amount' => Toman::format($state['max'])]), 'href' => $this->url($category, ['max' => null] + $base)];
        }
        if ($state['stock']) {
            $chips[] = ['label' => (string) __('shop.filters.in_stock'), 'href' => $this->url($category, ['stock' => false] + $base)];
        }

        return $chips;
    }

    /**
     * @param  list<ProductCardData>  $cards
     * @return array<int, MediaData>
     */
    private function coverMedia(array $cards): array
    {
        $ids = array_values(array_unique(array_filter(array_map(static fn (ProductCardData $c): ?int => $c->coverMediaId, $cards))));

        return $ids === [] ? [] : $this->media->findMany($ids);
    }

    /**
     * @param  array<string, mixed>  $state
     * @return array{label: string, previous: ?string, next: ?string, pages: list<array{number: string, href: ?string, current: bool}>}|null
     */
    private function pagination(CategoryData $category, array $state, ProductPage $list): ?array
    {
        $last = $list->lastPage();
        if ($last < 2) {
            return null;
        }

        $pages = [];
        $previous = 0;
        foreach (range(1, $last) as $n) {
            if ($n !== 1 && $n !== $last && abs($n - $list->page) > 2) {
                continue;
            }
            if ($previous !== 0 && $n - $previous > 1) {
                $pages[] = ['number' => '…', 'href' => null, 'current' => false];
            }
            $pages[] = ['number' => fa_digits($n), 'href' => $this->url($category, ['page' => $n] + $state), 'current' => $n === $list->page];
            $previous = $n;
        }

        return [
            'label' => (string) __('shop.pagination.label'),
            'previous' => $list->page > 1 ? $this->url($category, ['page' => $list->page - 1] + $state) : null,
            'next' => $list->page < $last ? $this->url($category, ['page' => $list->page + 1] + $state) : null,
            'pages' => $pages,
        ];
    }
}
