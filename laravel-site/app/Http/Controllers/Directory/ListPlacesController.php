<?php

declare(strict_types=1);

namespace App\Http\Controllers\Directory;

use App\Domain\Directory\Contracts\PlaceRepository;
use App\Domain\Directory\Contracts\TaxonomyRepository;
use App\Domain\Directory\Data\AmenityData;
use App\Domain\Directory\Data\CategoryData;
use App\Domain\Directory\Data\CityData;
use App\Domain\Directory\Data\DistrictData;
use App\Domain\Directory\Data\LandingComboData;
use App\Domain\Directory\Data\PlaceCardData;
use App\Domain\Directory\Data\PlacePage;
use App\Domain\Directory\Data\PlaceSearchCriteria;
use App\Domain\Directory\Enums\PlaceSort;
use App\Domain\Directory\Support\DirectoryUrls;
use App\Domain\Directory\Support\LandingCopy;
use App\Domain\Directory\Support\PlaceListIndexing;
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
use Carbon\CarbonImmutable;
use Illuminate\Contracts\Config\Repository as Config;
use Illuminate\Contracts\View\View;
use Illuminate\Http\RedirectResponse;
use Illuminate\Http\Request;
use Illuminate\Support\Str;

/**
 * Directory listing (L5-02, design/html/directory.html): `/directory`, `/directory/{city}` and
 * `/directory/{city}/{category}`. The two landing shapes are the curated, indexable pages (copy from LandingCopy, editable
 * per combo in the admin); every other filter (text, district, child age, open now, amenities, sort, a category without
 * a city) is a plain query parameter → noindex,follow (SeoManager) with the canonical on the nearest landing.
 *
 * Filters are GET links / forms. A request whose query is not in its canonical shape (form fields `where`, `city`,
 * `category` that belong in the path, empty or unknown parameters, `?page=1`, unsorted amenities) is 301-redirected
 * once to the canonical URL, so every filter state has exactly one address. Pagination `?page=n` is self-canonical; an
 * invalid or out-of-range page is a 404. Lists that show no real (non-demo) place are noindex.
 *
 * All reads go through the cached Directory repositories; the view gets arrays of scalars / DTOs only.
 */
final class ListPlacesController
{
    public const PER_PAGE = PlaceListIndexing::PER_PAGE;

    /** Child-age choices of the search form, in months. */
    public const AGES = [3, 6, 9, 12, 18, 24, 36, 48, 60, 72];

    /** Amenities offered as one-tap toggles next to «فیلترها» (the design's row); the rest live in the panel. */
    public const QUICK_AMENITIES = 3;

    /** Cover illustration per category slug while a place has no photo (AUDIT §4.2: demo covers, one fallback). */
    private const COVERS = [
        'pool' => 'place-cover-pool',
        'playhouse' => 'place-cover-playhouse',
        'mother-child-class' => 'place-cover-movement',
        'music-art' => 'place-cover-music',
        'postpartum-exercise' => 'place-cover-yoga',
        'baby-massage' => 'place-cover-massage',
    ];

    private const FALLBACK_COVER = 'place-cover-playhouse';

    /** Query parameters this page understands, in canonical order. */
    private const PARAMS = ['category', 'district', 'q', 'age', 'open', 'amenity', 'sort', 'page'];

    public function __construct(
        private readonly PlaceRepository $places,
        private readonly TaxonomyRepository $taxonomy,
        private readonly SeoMetaRepository $seoMeta,
        private readonly DirectoryUrls $urls,
        private readonly Config $config,
    ) {}

    /** SeoManager and SchemaGraph are request-scoped: injected per call, not into the (route-cached) controller. */
    public function __invoke(Request $request, SeoManager $seo, SchemaGraph $graph, ?string $city = null, ?string $category = null): View|RedirectResponse
    {
        $cityData = $city === null ? null : ($this->taxonomy->findCity($city) ?? abort(404));
        $categoryData = $category === null ? null : ($this->taxonomy->findCategory($category) ?? abort(404));

        $state = $this->state($request, $cityData, $categoryData);
        $target = $this->url($state, keepTracking: $request->query->all());
        if ($this->needsRedirect($request, $state)) {
            return new RedirectResponse($target, 301);
        }

        $list = $this->places->search(new PlaceSearchCriteria(
            cityId: $state['city']?->id,
            districtId: $state['district']?->id,
            categoryId: $state['category']?->id,
            ageMonths: $state['age'],
            amenityIds: array_map(static fn (AmenityData $a): int => $a->id, $state['amenities']),
            openAt: $state['open'] ? CarbonImmutable::now() : null,
            text: $state['q'],
            sort: $state['sort'],
            page: $state['page'],
            perPage: self::PER_PAGE,
        ));
        if ($list->page > 1 && $list->isOutOfRange()) {
            abort(404);
        }

        $landing = $this->landing($state);
        $trail = $this->trail($state);
        $this->applySeo($seo, $state, $landing, $list);
        $this->schema($graph, $request, $trail, $list->items, $landing['h1']);

        return view('pages.directory.index', [
            'intro' => [
                'eyebrow' => __('directory.eyebrow'),
                'title' => $landing['h1'],
                'lead' => $landing['lead'],
            ],
            'breadcrumbs' => $state['city'] === null ? null : $trail,
            'search' => $this->searchForm($state),
            'chips' => $this->chips($state),
            'filters' => $this->filters($state),
            'sorts' => $this->sorts($state),
            'summary' => $this->summary($state, $list),
            'cards' => $this->cards($list),
            'pagination' => $this->pagination($state, $list),
            'resetUrl' => $this->url(['city' => $state['city'], 'category' => $state['category']] + $this->blank()),
            'areas' => $this->areas($state),
            'navRoute' => 'directory.index',
        ]);
    }

    /**
     * Parsed, validated filter state. Unknown slugs and out-of-range values are dropped (the redirect then removes
     * them from the URL).
     *
     * @return array{city: ?CityData, category: ?CategoryData, district: ?DistrictData, q: ?string, age: ?int, open: bool, amenities: list<AmenityData>, sort: PlaceSort, page: int}
     */
    private function state(Request $request, ?CityData $city, ?CategoryData $category): array
    {
        $query = $request->query->all();
        $string = static fn (string $key): ?string => isset($query[$key]) && is_string($query[$key]) && trim($query[$key]) !== '' ? trim($query[$key]) : null;

        // Search form: «کجا؟» submits `where` = city or city/district; links may carry `city` / `category`.
        $districtSlug = $string('district');
        if ($city === null && ($where = $string('where') ?? $string('city')) !== null) {
            $parts = explode('/', $where, 2);
            $whereDistrict = $parts[1] ?? null;
            $city = $this->taxonomy->findCity($parts[0]);
            $districtSlug = $whereDistrict ?? $districtSlug;
        }
        if ($category === null && ($slug = $string('category')) !== null) {
            $category = $this->taxonomy->findCategory($slug);
        }

        $district = null;
        if ($districtSlug !== null) {
            foreach ($this->cityWithDistricts($city)->districts ?? [] as $candidate) {
                if ($candidate->slug === $districtSlug) {
                    $district = $candidate;
                }
            }
        }

        $age = $string('age');
        $age = $age !== null && ctype_digit($age) && in_array((int) $age, self::AGES, true) ? (int) $age : null;

        $amenitySlugs = $query['amenity'] ?? [];
        $amenitySlugs = is_array($amenitySlugs) ? array_filter($amenitySlugs, is_string(...)) : [];
        $amenities = array_values(array_filter($this->taxonomy->amenities(), static fn (AmenityData $a): bool => in_array($a->slug, $amenitySlugs, true)));

        $sort = PlaceSort::tryFrom((string) $string('sort')) ?? PlaceSort::Recommended;
        if ($sort === PlaceSort::Nearest) {
            $sort = PlaceSort::Recommended; // needs a reference point; the site asks for no location
        }

        $q = $string('q');

        return [
            'city' => $city,
            'category' => $category,
            'district' => $district,
            'q' => $q === null ? null : mb_substr($q, 0, 80),
            'age' => $age,
            'open' => $string('open') === '1',
            'amenities' => $amenities,
            'sort' => $sort,
            'page' => $this->page($query['page'] ?? null),
        ];
    }

    private function page(mixed $raw): int
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
     * True when the request URL differs from the canonical URL of its state (path or query).
     *
     * @param  array<string, mixed>  $state
     */
    private function needsRedirect(Request $request, array $state): bool
    {
        if (rawurldecode(trim($request->getBaseUrl().$request->getPathInfo(), '/')) !== rawurldecode(trim((string) parse_url($this->url($state), PHP_URL_PATH), '/'))) {
            return true;
        }

        $actual = array_filter($request->query->all(), static fn (int|string $key): bool => ! Str::is(SeoManager::TRACKING_PARAMS, (string) $key), ARRAY_FILTER_USE_KEY);

        return $actual !== $this->query($state);
    }

    /**
     * @param  array<string, mixed>  $state
     * @param  array<string, mixed>  $keepTracking  request query whose tracking parameters survive the redirect
     */
    private function url(array $state, array $keepTracking = []): string
    {
        /** @var CityData|null $city */
        $city = $state['city'] ?? null;
        /** @var CategoryData|null $category */
        $category = $state['category'] ?? null;

        $path = match (true) {
            $city !== null && $category !== null => route('directory.category', [$city->slug, $category->slug]),
            $city !== null => route('directory.city', [$city->slug]),
            default => route('directory.index'),
        };

        $query = $this->query($state);
        foreach ($keepTracking as $key => $value) {
            if (Str::is(SeoManager::TRACKING_PARAMS, (string) $key)) {
                $query[(string) $key] = $value;
            }
        }

        return $query === [] ? $path : $path.'?'.(string) preg_replace('/%5B\d+%5D=/', '%5B%5D=', http_build_query($query));
    }

    /**
     * Canonical query of a state: only set filters, fixed order, amenity slugs sorted, page only when > 1.
     *
     * @param  array<string, mixed>  $state
     * @return array<string, string|list<string>>
     */
    private function query(array $state): array
    {
        $values = [
            'category' => ($state['city'] ?? null) === null ? ($state['category'] ?? null)?->slug : null,
            'district' => ($state['district'] ?? null)?->slug,
            'q' => $state['q'] ?? null,
            'age' => isset($state['age']) ? (string) $state['age'] : null,
            'open' => ($state['open'] ?? false) ? '1' : null,
            'amenity' => null,
            'sort' => isset($state['sort']) && $state['sort'] !== PlaceSort::Recommended ? $state['sort']->value : null,
            'page' => ($state['page'] ?? 1) > 1 ? (string) $state['page'] : null,
        ];

        $slugs = array_map(static fn (AmenityData $a): string => $a->slug, $state['amenities'] ?? []);
        sort($slugs);
        if ($slugs !== []) {
            $values['amenity'] = $slugs;
        }

        $query = [];
        foreach (self::PARAMS as $key) {
            if ($values[$key] !== null) {
                $query[$key] = $values[$key];
            }
        }

        return $query;
    }

    /**
     * A state with every filter cleared (city / category are added by the caller).
     *
     * @return array{district: null, q: null, age: null, open: false, amenities: list<AmenityData>, sort: PlaceSort, page: int}
     */
    private function blank(): array
    {
        return ['district' => null, 'q' => null, 'age' => null, 'open' => false, 'amenities' => [], 'sort' => PlaceSort::Recommended, 'page' => 1];
    }

    /**
     * @param  array<string, mixed>  $state
     */
    private function isFiltered(array $state): bool
    {
        return $this->query(['page' => 1, 'city' => $state['city'], 'category' => $state['category']] + $state) !== [];
    }

    /**
     * h1, title, description and lead of the page: lang strings on /directory, LandingCopy (admin row or template)
     * on the landings.
     *
     * @param  array<string, mixed>  $state
     * @return array{h1: string, title: string, description: string, lead: string, meta: bool}
     */
    private function landing(array $state): array
    {
        /** @var CityData|null $city */
        $city = $state['city'];
        /** @var CategoryData|null $category */
        $category = $state['category'];

        if ($city === null) {
            $meta = $this->seoMeta->forRoute('directory.index');

            return [
                'h1' => (string) __('directory.index.title'),
                'title' => $meta->title ?? (string) __('directory.index.seo_title'),
                'description' => $meta->description ?? (string) __('directory.index.seo_description'),
                'lead' => (string) __('directory.index.lead'),
                'meta' => $meta?->description !== null,
            ];
        }

        $copy = LandingCopy::resolve($city, $category, $this->taxonomy->landing($city->id, $category?->id));
        $lead = $copy->intro ?? ($category === null
            ? (string) __('directory.city.lead', ['city' => $city->name])
            : ($category->description ?? (string) __('directory.category.lead', ['city' => $city->name, 'category' => $category->name])));

        return [
            'h1' => (string) $copy->h1,
            'title' => (string) $copy->title,
            'description' => (string) $copy->description,
            'lead' => $lead,
            'meta' => true,
        ];
    }

    /**
     * @param  array<string, mixed>  $state
     * @param  array{h1: string, title: string, description: string, lead: string, meta: bool}  $landing
     */
    private function applySeo(SeoManager $seo, array $state, array $landing, PlacePage $list): void
    {
        $page = (int) $state['page'];
        if ($page > 1) {
            $suffix = (string) __('directory.page_suffix', ['page' => fa_digits($page)]);
            $seo->title($landing['title'].' — '.$suffix)->description(DescriptionText::fromExcerpt($suffix.' — '.$landing['description']));
        } else {
            $seo->title($landing['title'])->description($landing['meta'] ? $landing['description'] : DescriptionText::fromExcerpt($landing['description']));
        }

        if ($this->isFiltered($state)) {
            // Filter combinations are not landings: noindex,follow, canonical on the city / category landing.
            $seo->filtered()->canonical($this->url(['city' => $state['city'], 'category' => $state['city'] === null ? null : $state['category']]));
        }

        if (! PlaceListIndexing::showsRealPlace($list->items)) {
            $seo->noindex(); // empty or demo-only lists are thin content (PagesSitemapProvider skips them too)
        }
    }

    /**
     * @param  list<BreadcrumbItem>  $trail
     * @param  list<PlaceCardData>  $items
     */
    private function schema(SchemaGraph $graph, Request $request, array $trail, array $items, string $name): void
    {
        $graph->breadcrumbs(...$trail)->pageType(WebPageType::CollectionPage)->pageName($name);
        if ($items === []) {
            return;
        }

        $pageUrl = CanonicalUrl::normalize($request->fullUrl(), (string) $this->config->get('app.url'));
        $graph->add(ItemListNode::make($pageUrl, array_map(
            fn (PlaceCardData $card): ListEntry => new ListEntry($this->urls->place($card->slug), $card->name),
            $items,
        ), $name));
        $graph->add(['@id' => SchemaIds::webPage($pageUrl), 'mainEntity' => Node::ref(SchemaIds::itemList($pageUrl))]);
    }

    /**
     * Home → directory → city → category.
     *
     * @param  array<string, mixed>  $state
     * @return list<BreadcrumbItem>
     */
    private function trail(array $state): array
    {
        $trail = [
            new BreadcrumbItem(PageGraph::HOME_LABEL, SchemaIds::root((string) $this->config->get('app.url'))),
            new BreadcrumbItem((string) __('directory.name'), $this->urls->absolute(route('directory.index'))),
        ];
        if ($state['city'] !== null) {
            $trail[] = new BreadcrumbItem($state['city']->name, $this->urls->city($state['city']->slug));
            if ($state['category'] !== null) {
                $trail[] = new BreadcrumbItem($state['category']->name, $this->urls->category($state['city']->slug, $state['category']->slug));
            }
        }

        return $trail;
    }

    /**
     * Search form fields (GET → /directory; the redirect moves city / category into the path).
     *
     * @param  array<string, mixed>  $state
     * @return array{action: string, q: ?string, where: string, whereOptions: array<string, string>, age: string, ageOptions: array<int, string>, hidden: array<string, string|list<string>>}
     */
    private function searchForm(array $state): array
    {
        $where = [];
        foreach ($this->taxonomy->cities() as $city) {
            $where[$city->slug] = $city->name;
            foreach ($city->districts as $district) {
                $where[$city->slug.'/'.$district->slug] = $city->name.' · '.$district->name;
            }
        }

        $ages = [];
        foreach (self::AGES as $months) {
            $ages[$months] = $months >= 24
                ? (string) __('directory.search.years', ['n' => fa_digits(intdiv($months, 12))])
                : (string) __('directory.search.months', ['n' => fa_digits($months)]);
        }

        $hidden = $this->query(['page' => 1, 'city' => null, 'category' => $state['category']] + $state);
        unset($hidden['district'], $hidden['q'], $hidden['age']);

        return [
            'action' => route('directory.index'),
            'q' => is_string($state['q']) ? $state['q'] : null,
            'where' => $state['city'] === null ? '' : $state['city']->slug.($state['district'] === null ? '' : '/'.$state['district']->slug),
            'whereOptions' => $where,
            'age' => $state['age'] === null ? '' : (string) $state['age'],
            'ageOptions' => $ages,
            'hidden' => $hidden,
        ];
    }

    /**
     * Category chips: «همه» + categories, keeping the city (→ the indexable city × category landings) and the other
     * filters.
     *
     * @param  array<string, mixed>  $state
     * @return list<array{label: string, href: string, active: bool, icon: ?string}>
     */
    private function chips(array $state): array
    {
        $base = ['page' => 1] + $state;
        $chips = [['label' => (string) __('directory.all'), 'href' => $this->url(['category' => null] + $base), 'active' => $state['category'] === null, 'icon' => 'grid']];
        foreach ($this->taxonomy->categories() as $category) {
            $chips[] = [
                'label' => $category->name,
                'href' => $this->url(['category' => $category] + $base),
                'active' => $state['category']?->id === $category->id,
                'icon' => $category->icon,
            ];
        }

        return $chips;
    }

    /**
     * Toggle links (open now, quick amenities, cheapest first) and the full amenity list for the «فیلترها» panel.
     *
     * @param  array<string, mixed>  $state
     * @return array{toggles: list<array{label: string, href: string, active: bool}>, amenities: list<array{slug: string, label: string, checked: bool}>, hidden: array<string, string|list<string>>, action: string, count: int}
     */
    private function filters(array $state): array
    {
        $base = ['page' => 1] + $state;
        $selected = array_map(static fn (AmenityData $a): int => $a->id, $state['amenities']);
        $all = $this->taxonomy->amenities();

        $toggles = [[
            'label' => (string) __('directory.filters.open'),
            'href' => $this->url(['open' => ! $state['open']] + $base),
            'active' => $state['open'],
        ]];
        foreach (array_slice($all, 0, self::QUICK_AMENITIES) as $amenity) {
            $on = in_array($amenity->id, $selected, true);
            $toggles[] = [
                'label' => $amenity->name,
                'href' => $this->url(['amenities' => $on
                    ? array_values(array_filter($state['amenities'], static fn (AmenityData $a): bool => $a->id !== $amenity->id))
                    : [...$state['amenities'], $amenity]] + $base),
                'active' => $on,
            ];
        }
        $cheapest = $state['sort'] === PlaceSort::Price;
        $toggles[] = [
            'label' => (string) __('directory.filters.price'),
            'href' => $this->url(['sort' => $cheapest ? PlaceSort::Recommended : PlaceSort::Price] + $base),
            'active' => $cheapest,
        ];

        $hidden = $this->query(['page' => 1, 'city' => $state['city'], 'category' => $state['category']] + $state);
        unset($hidden['amenity']);

        return [
            'toggles' => $toggles,
            'amenities' => array_map(static fn (AmenityData $a): array => ['slug' => $a->slug, 'label' => $a->name, 'checked' => in_array($a->id, $selected, true)], $all),
            'hidden' => $hidden,
            'action' => $this->url(['city' => $state['city'], 'category' => $state['city'] === null ? null : $state['category']]),
            'count' => count($selected),
        ];
    }

    /**
     * @param  array<string, mixed>  $state
     * @return array{label: string, items: list<array{label: string, href: string, active: bool}>}
     */
    private function sorts(array $state): array
    {
        $items = [];
        foreach (PlaceSort::cases() as $sort) {
            if ($sort !== PlaceSort::Nearest) {
                $items[] = ['label' => $sort->label(), 'href' => $this->url(['sort' => $sort, 'page' => 1] + $state), 'active' => $state['sort'] === $sort];
            }
        }

        return ['label' => $state['sort']->label(), 'items' => $items];
    }

    /**
     * @param  array<string, mixed>  $state
     * @return array{count: string, where: ?string}
     */
    private function summary(array $state, PlacePage $list): array
    {
        $where = $state['district']->name ?? $state['city']->name ?? null;

        return [
            'count' => (string) __('directory.results.count', ['count' => fa_digits($list->total)]),
            'where' => $where === null ? null : (string) __('directory.results.where', ['where' => $where]),
        ];
    }

    /**
     * Props for <x-cards.place>. «باز است» shows only when the place stays open for the whole page-cache lifetime, so a
     * cached page never claims a closed place is open.
     *
     * @return list<array<string, mixed>>
     */
    private function cards(PlacePage $list): array
    {
        $now = CarbonImmutable::now();
        $later = $now->addSeconds($this->pageTtl());

        return array_map(fn (PlaceCardData $card): array => [
            'href' => $this->urls->place($card->slug),
            'name' => $card->name,
            'rating' => $card->rating()?->ratingValue,
            'reviews' => $card->rating()?->ratingCount,
            'category' => $card->category->name,
            'district' => $card->district?->name,
            'distance' => $card->distanceKm === null ? null : (string) __('directory.card.distance', ['km' => fa_digits(number_format($card->distanceKm, 1, '٫', ''))]),
            'ages' => ($label = $card->ageRange()->label()) === null ? null : (string) __('directory.card.ages', ['ages' => $label]),
            'priceFrom' => $card->priceFrom,
            'slots' => $card->isOpenAt($now) && $card->isOpenAt($later) ? [(string) __('directory.card.open')] : [],
            'verified' => $card->isVerified,
            'media' => $card->coverMediaId,
            'illustration' => $card->coverMediaId === null ? (self::COVERS[$card->category->slug] ?? self::FALLBACK_COVER) : null,
        ], $list->items);
    }

    private function pageTtl(): int
    {
        $ttl = $this->config->get('pagecache.ttl') ?? $this->config->get('cacheaside.namespaces.pages');

        return is_numeric($ttl) ? max(0, (int) $ttl) : 3600;
    }

    /**
     * @param  array<string, mixed>  $state
     * @return array{label: string, previous: ?string, next: ?string, pages: list<array{number: string, href: ?string, current: bool}>}|null
     */
    private function pagination(array $state, PlacePage $list): ?array
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
            $pages[] = ['number' => fa_digits($n), 'href' => $this->url(['page' => $n] + $state), 'current' => $n === $list->page];
            $previous = $n;
        }

        return [
            'label' => (string) __('directory.pagination.label'),
            'previous' => $list->page > 1 ? $this->url(['page' => $list->page - 1] + $state) : null,
            'next' => $list->page < $last ? $this->url(['page' => $list->page + 1] + $state) : null,
            'pages' => $pages,
        ];
    }

    /**
     * Crawlable links of the side panel that replaces the design's map (no maps, AUDIT §8): the city landings, the
     * districts of the current city and the city × category landings that have places.
     *
     * @param  array<string, mixed>  $state
     * @return array{cities: list<array{label: string, href: string, active: bool}>, districts: list<array{label: string, href: string, active: bool}>, landings: list<array{label: string, href: string, active: bool}>, cityName: ?string}
     */
    private function areas(array $state): array
    {
        $combos = $this->taxonomy->combos();
        /** @var CityData|null $current */
        $current = $state['city'];

        $cities = [];
        $landings = [];
        foreach ($combos as $combo) {
            if ($combo->categorySlug === null) {
                $cities[] = ['label' => $combo->cityName, 'href' => $this->urls->city($combo->citySlug), 'active' => $current?->slug === $combo->citySlug];
            } elseif ($current?->slug === $combo->citySlug || ($current === null && count(array_filter($combos, static fn (LandingComboData $c): bool => $c->categorySlug === null)) === 1)) {
                $landings[] = [
                    'label' => (string) __('directory.areas.landing', ['category' => (string) $combo->categoryName, 'city' => $combo->cityName]),
                    'href' => $this->urls->category($combo->citySlug, (string) $combo->categorySlug),
                    'active' => $current?->slug === $combo->citySlug && $state['category']?->slug === $combo->categorySlug,
                ];
            }
        }

        $districts = [];
        if ($current !== null) {
            foreach ($this->cityWithDistricts($current)->districts ?? [] as $district) {
                $districts[] = [
                    'label' => $district->name,
                    'href' => $this->url(['city' => $current, 'category' => $state['category'], 'district' => $district] + $this->blank()),
                    'active' => $state['district']?->id === $district->id,
                ];
            }
        }

        return ['cities' => $cities, 'districts' => $districts, 'landings' => $landings, 'cityName' => $current?->name];
    }

    /**
     * findCity() returns the city without districts; the cities() list carries them.
     */
    private function cityWithDistricts(?CityData $city): ?CityData
    {
        if ($city === null) {
            return null;
        }
        foreach ($this->taxonomy->cities() as $candidate) {
            if ($candidate->id === $city->id) {
                return $candidate;
            }
        }

        return $city;
    }
}
