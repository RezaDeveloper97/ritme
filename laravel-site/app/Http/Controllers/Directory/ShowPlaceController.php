<?php

declare(strict_types=1);

namespace App\Http\Controllers\Directory;

use App\Domain\Directory\Actions\SubmitPlaceReview;
use App\Domain\Directory\Contracts\PlaceRepository;
use App\Domain\Directory\Data\PlaceData;
use App\Domain\Directory\Data\PlaceServiceData;
use App\Domain\Directory\Data\RatingSummaryData;
use App\Domain\Directory\Data\ReviewData;
use App\Domain\Directory\Data\ReviewPage;
use App\Domain\Directory\Data\ReviewSubmission;
use App\Domain\Directory\Enums\ReviewAspect;
use App\Domain\Directory\Enums\Weekday;
use App\Domain\Directory\Models\Place;
use App\Domain\Directory\Support\DirectoryUrls;
use App\Domain\Directory\Support\OpeningHours;
use App\Domain\Seo\Contracts\OgImageResolver;
use App\Domain\Seo\Schema\Data\BreadcrumbItem;
use App\Domain\Seo\Schema\Enums\WebPageType;
use App\Domain\Seo\Schema\Node;
use App\Domain\Seo\Schema\Nodes\LocalBusinessNode;
use App\Domain\Seo\Schema\PageGraph;
use App\Domain\Seo\Schema\SchemaGraph;
use App\Domain\Seo\Schema\SchemaIds;
use App\Domain\Seo\SeoManager;
use App\Domain\Seo\Support\CanonicalUrl;
use App\Domain\Seo\Support\DescriptionText;
use App\View\Components\Icon;
use App\View\Components\Picture;
use Carbon\CarbonImmutable;
use Illuminate\Contracts\Config\Repository as Config;
use Illuminate\Contracts\Validation\Factory as ValidatorFactory;
use Illuminate\Contracts\View\View;
use Illuminate\Database\Eloquent\ModelNotFoundException;
use Illuminate\Http\RedirectResponse;
use Illuminate\Http\Request;
use InvalidArgumentException;

/**
 * `/directory/place/{slug}` — the place page (L5-03, design/html/directory-place.html) and its review form POST.
 *
 * Unknown slug: an old slug from the slug history 301s to the current URL (query kept), anything else 404s. Reviews
 * (approved only, newest first) are paginated with `?page=n` (self-canonical, `?page=1` 301s to the clean URL, out of
 * range 404s). Demo places (`PlaceData::$isDemo`) are noindex; demo reviews are labelled «نمونه» and never counted.
 *
 * SEO: admin seo_meta of the place wins, else name + category + area. JSON-LD: ItemPage, LocalBusiness subtype of the
 * category (PostalAddress, geo, OpeningHoursSpecification, priceRange, image) with `aggregateRating` + `review` only
 * from real approved reviews, BreadcrumbList (registered by <x-ui.breadcrumbs> in the view).
 *
 * Page cache (L1-07, 1 h TTL): time-dependent bits must not go stale in cached HTML — the "open now" pill is shown
 * only when the place stays open for the whole TTL, today's row is highlighted only when the day does not change
 * within the TTL. The review form keeps CSRF working on cache HITs (PageCache swaps the token per visitor); the POST
 * is rate limited (route), has a honeypot (`company_url`, answered like a real submit) and stores the review as
 * pending (SubmitPlaceReview). Success / errors come back as flash data, which bypasses the page cache.
 */
final class ShowPlaceController
{
    public const REVIEWS_PER_PAGE = 6;

    /** Review POSTs per IP per 10 minutes (route middleware). */
    public const REVIEWS_PER_10_MINUTES = 3;

    /** Honeypot field of the review form: humans never see or fill it. */
    public const HONEYPOT = 'company_url';

    /** Cover illustration per category slug while a place has no photo (same set as the listing, AUDIT §4.2). */
    private const COVERS = [
        'pool' => 'place-cover-pool',
        'playhouse' => 'place-cover-playhouse',
        'mother-child-class' => 'place-cover-movement',
        'music-art' => 'place-cover-music',
        'postpartum-exercise' => 'place-cover-yoga',
        'baby-massage' => 'place-cover-massage',
    ];

    private const FALLBACK_COVER = 'place-cover-playhouse';

    /** Order of the remaining placeholder tiles (as in the design's mosaic). */
    private const MOSAIC = ['place-cover-pool', 'place-cover-massage', 'place-cover-movement', 'place-cover-yoga', 'place-cover-playhouse', 'place-cover-music'];

    /** Category slugs that show a health note next to the hours (L5-01: the pool note lives in the view). */
    private const NOTES = ['pool'];

    public function __construct(
        private readonly PlaceRepository $places,
        private readonly OgImageResolver $ogImages,
        private readonly DirectoryUrls $urls,
        private readonly Config $config,
    ) {}

    public function show(Request $request, string $slug, SeoManager $seo, SchemaGraph $graph): View|RedirectResponse
    {
        $place = $this->places->findPublishedBySlug($slug);
        if ($place === null) {
            $current = $this->places->currentSlugFor($slug);
            if ($current === null || $current === $slug) {
                abort(404);
            }
            $query = $request->getQueryString();

            return new RedirectResponse(route('directory.place', [$current]).($query !== null && $query !== '' ? '?'.$query : ''), 301);
        }

        $rawPage = $request->query('page');
        if ($rawPage !== null) {
            $page = is_string($rawPage) && preg_match('/^[1-9]\d{0,5}$/', $rawPage) === 1 ? (int) $rawPage : 0;
            if ($page === 1) {
                return new RedirectResponse(route('directory.place', [$place->slug]), 301);
            }
            if ($page < 1) {
                abort(404);
            }
        } else {
            $page = 1;
        }

        $reviews = $this->places->reviews($place->id, $page, self::REVIEWS_PER_PAGE);
        if ($page > $reviews->lastPage()) {
            abort(404);
        }
        $rating = $this->places->ratingSummary($place->id);

        $placeUrl = CanonicalUrl::normalize($this->urls->place($place->slug), (string) $this->config->get('app.url'));
        $images = $this->images($place);
        $trail = $this->trail($place, $placeUrl);
        $this->describe($seo, $graph, $place, $page, $placeUrl, $images, $reviews);

        $now = CarbonImmutable::now();
        $later = $now->addSeconds($this->pageTtl());

        return view('pages.directory.show', [
            'place' => $this->header($place, $rating, $now, $later),
            'breadcrumbs' => $trail,
            'gallery' => [
                'images' => array_column($images, 'id'),
                'full' => $images,
                'alt' => $place->name,
                'illustrations' => $images === [] ? $this->illustrations($place) : [],
            ],
            'about' => $place->description ?? $place->summary,
            'amenities' => $this->amenities($place),
            'services' => array_map($this->service(...), $place->services),
            'hours' => $this->hours($place->openingHours(), $now, $later),
            'note' => in_array($place->category->slug, self::NOTES, true) ? (string) __('directory.place.notes.'.$place->category->slug) : null,
            'rating' => $this->ratingBlock($rating),
            'reviews' => array_map($this->review(...), $reviews->items),
            'reviewPagination' => $this->reviewPagination($place, $reviews),
            'reviewForm' => [
                'action' => route('directory.place.review', [$place->slug]),
                'aspects' => array_map(static fn (ReviewAspect $a): array => ['key' => $a->value, 'label' => $a->label()], ReviewAspect::cases()),
                'honeypot' => self::HONEYPOT,
            ],
            'address' => $this->address($place),
            'rules' => [...$place->ruleLines(), ...array_filter([trim((string) $place->cancellationPolicy)], static fn (string $l): bool => $l !== '')],
            'booking' => [
                'priceFrom' => $place->priceFrom,
                'priceUnit' => $this->priceUnit($place),
                'phone' => $place->phones[0] ?? null,
            ],
            'navRoute' => 'directory.index',
        ]);
    }

    public function storeReview(Request $request, string $slug, ValidatorFactory $validator, SubmitPlaceReview $submit): RedirectResponse
    {
        $place = $this->places->findPublishedBySlug($slug) ?? abort(404);
        $back = route('directory.place', [$place->slug]).'#review-form';

        // Honeypot: bots get the same answer as people, nothing is stored.
        if (trim((string) $request->input(self::HONEYPOT, '')) !== '') {
            return redirect()->to($back)->with('review_submitted', true);
        }

        $aspectKeys = array_map(static fn (ReviewAspect $a): string => $a->value, ReviewAspect::cases());
        $data = $validator->make($request->all(), [
            'author_name' => ['required', 'string', 'min:2', 'max:'.SubmitPlaceReview::MAX_NAME],
            'rating' => ['required', 'integer', 'between:1,5'],
            'body' => ['required', 'string', 'min:10', 'max:'.SubmitPlaceReview::MAX_BODY],
            'aspects' => ['nullable', 'array:'.implode(',', $aspectKeys)],
            'aspects.*' => ['nullable', 'integer', 'between:1,5'],
        ], [
            'author_name.required' => __('directory.place.review.errors.name'),
            'author_name.*' => __('directory.place.review.errors.name'),
            'rating.*' => __('directory.place.review.errors.rating'),
            'body.required' => __('directory.place.review.errors.body'),
            'body.min' => __('directory.place.review.errors.body_short'),
            'body.*' => __('directory.place.review.errors.body'),
            'aspects*' => __('directory.place.review.errors.aspects'),
        ]);

        if ($data->fails()) {
            return redirect()->to($back)->withErrors($data)->withInput($request->except(self::HONEYPOT));
        }

        /** @var array{author_name: string, rating: int|string, body: string, aspects?: array<string, int|string|null>|null} $valid */
        $valid = $data->validated();
        $aspects = [];
        foreach ((array) ($valid['aspects'] ?? []) as $key => $score) {
            if ($score !== null && $score !== '') {
                $aspects[(string) $key] = (int) $score;
            }
        }

        try {
            $submit->handle(new ReviewSubmission(
                placeSlug: $place->slug,
                authorName: (string) $valid['author_name'],
                rating: (int) $valid['rating'],
                body: (string) $valid['body'],
                aspects: $aspects,
                ip: $request->ip(),
            ));
        } catch (ModelNotFoundException) {
            abort(404);
        } catch (InvalidArgumentException) {
            return redirect()->to($back)->withErrors(['body' => __('directory.place.review.errors.body')])->withInput($request->except(self::HONEYPOT));
        }

        return redirect()->to($back)->with('review_submitted', true);
    }

    /**
     * Title, description, robots, OG image and the JSON-LD of the page. SeoManager / SchemaGraph are request-scoped,
     * so they are method-injected (a controller instance can outlive one request, e.g. in tests or Octane).
     *
     * @param  list<array{id: int, url: string}>  $images
     */
    private function describe(SeoManager $seo, SchemaGraph $graph, PlaceData $place, int $page, string $placeUrl, array $images, ReviewPage $reviews): void
    {
        $area = implode('، ', array_filter([$place->district->name ?? null, $place->city->name]));
        $title = str_contains($place->name, $place->category->name)
            ? $place->name.' در '.$area
            : $place->name.' — '.$place->category->name.' در '.$area;
        if ($page > 1) {
            $title .= ' — '.__('directory.place.reviews.page', ['page' => fa_digits($page)]);
        }

        $seo->for((new Place)->forceFill(['id' => $place->id]));
        $seo->title($title);
        $summary = trim((string) ($place->summary ?? $place->description));
        $seo->description(DescriptionText::fromExcerpt($summary !== ''
            ? $summary.' '.__('directory.place.seo_suffix', ['area' => $area])
            : __('directory.place.seo_fallback', ['name' => $place->name, 'category' => $place->category->name, 'area' => $area])));
        if ($place->isDemo) {
            $seo->noindex(); // placeholder business (L5-01): never indexed, never in the sitemap
        }

        $first = $images[0]['id'] ?? null;
        $og = $first === null ? null : $this->ogImages->resolve($first, $place->name);
        if ($og !== null) {
            $seo->image($og);
        }

        $graph->pageType(WebPageType::ItemPage)->pageName($place->name)
            ->dates(null, $place->updatedAt->setTimezone(self::tz())->toIso8601String());

        $node = LocalBusinessNode::make($place->toLocalBusiness($placeUrl, array_column($images, 'url')));
        $real = array_values(array_filter($reviews->items, static fn (ReviewData $r): bool => ! $r->isDemo));
        if ($place->rating() !== null && $real !== []) {
            $node['review'] = array_map(static fn (ReviewData $r): array => [
                '@type' => 'Review',
                'author' => ['@type' => 'Person', 'name' => $r->authorName],
                'datePublished' => $r->createdAt->setTimezone(self::tz())->toDateString(),
                'reviewBody' => $r->body,
                'reviewRating' => ['@type' => 'Rating', 'ratingValue' => $r->rating, 'bestRating' => 5, 'worstRating' => 1],
            ], array_slice($real, 0, 5));
        }
        $graph->add($node);
        $graph->add(['@id' => SchemaIds::webPage($placeUrl), 'mainEntity' => Node::ref(SchemaIds::place($placeUrl))]);
    }

    /**
     * Cover + gallery that resolve to a real media file: id + absolute URL (schema `image`, lightbox full size).
     *
     * @return list<array{id: int, url: string}>
     */
    private function images(PlaceData $place): array
    {
        $images = [];
        foreach ($place->imageMediaIds() as $id) {
            $url = Picture::mediaUrl($id, 'desktop_1920');
            if ($url !== null) {
                $images[] = ['id' => $id, 'url' => CanonicalUrl::normalize($url, (string) $this->config->get('app.url'))];
            }
        }

        return $images;
    }

    /**
     * Five decorative tiles for a place without photos: the category cover first, then the other covers.
     *
     * @return list<string>
     */
    private function illustrations(PlaceData $place): array
    {
        $first = self::COVERS[$place->category->slug] ?? self::FALLBACK_COVER;
        $rest = array_values(array_diff(self::MOSAIC, [$first]));

        return array_slice([$first, ...$rest], 0, 5);
    }

    /**
     * Home → directory → city → city × category → place.
     *
     * @return list<BreadcrumbItem>
     */
    private function trail(PlaceData $place, string $placeUrl): array
    {
        return [
            new BreadcrumbItem(PageGraph::HOME_LABEL, SchemaIds::root((string) $this->config->get('app.url'))),
            new BreadcrumbItem((string) __('directory.name'), $this->urls->absolute(route('directory.index'))),
            new BreadcrumbItem($place->city->name, $this->urls->city($place->city->slug)),
            new BreadcrumbItem($place->category->name, $this->urls->category($place->city->slug, $place->category->slug)),
            new BreadcrumbItem($place->name, $placeUrl),
        ];
    }

    /**
     * @return array<string, mixed>
     */
    private function header(PlaceData $place, RatingSummaryData $rating, CarbonImmutable $now, CarbonImmutable $later): array
    {
        $hours = $place->openingHours();
        $closes = $hours->closesAt($now);
        // Only when still open at the end of the cache TTL, so a cached page never claims "open" after closing time.
        $open = $closes !== null && $hours->isOpenAt($later) ? (string) __('directory.place.open_until', ['time' => fa_digits($closes === '23:59' ? '24:00' : $closes)]) : null;

        return [
            'name' => $place->name,
            'category' => $place->category->name,
            'verified' => $place->isVerified,
            'demo' => $place->isDemo,
            'area' => implode('، ', array_filter([$place->district->name ?? null, $place->city->name])),
            'rating' => $rating->isEmpty() ? null : ['value' => $rating->average, 'count' => $rating->count],
            'open' => $open,
            'phone' => $place->phones[0] ?? null,
            'website' => $place->website,
            'url' => $this->urls->place($place->slug),
        ];
    }

    /**
     * Age range first (as in the design), then the amenities with their sprite icon.
     *
     * @return list<array{icon: string, label: string}>
     */
    private function amenities(PlaceData $place): array
    {
        $items = [];
        $ages = $place->ageRange()->label();
        if ($ages !== null) {
            $items[] = ['icon' => 'person', 'label' => $ages];
        }
        foreach ($place->amenities as $amenity) {
            $items[] = ['icon' => $amenity->icon !== null && Icon::exists($amenity->icon) ? $amenity->icon : 'check', 'label' => $amenity->name];
        }

        return $items;
    }

    /**
     * @return array{name: string, meta: string|null, price: int|null, unit: string|null}
     */
    private function service(PlaceServiceData $service): array
    {
        $meta = array_filter([
            $service->durationMinutes === null ? null : (string) __('directory.place.services.minutes', ['n' => fa_digits($service->durationMinutes)]),
            $service->ageRange()->label(),
            $service->details === null || trim($service->details) === '' ? null : trim($service->details),
        ]);

        return [
            'name' => $service->name,
            'meta' => $meta === [] ? null : implode(' · ', $meta),
            'price' => $service->price !== null && $service->price > 0 ? $service->price : null,
            'unit' => $service->priceUnit,
        ];
    }

    /**
     * Weekly table; today is flagged only when the weekday stays the same for the whole cache TTL.
     *
     * @return list<array{label: string, hours: string, closed: bool, today: bool}>
     */
    private function hours(OpeningHours $hours, CarbonImmutable $now, CarbonImmutable $later): array
    {
        if ($hours->isEmpty()) {
            return [];
        }
        $tz = self::tz();
        $stable = Weekday::fromDate($now->setTimezone($tz)) === Weekday::fromDate($later->setTimezone($tz));

        return array_map(static fn (array $row): array => [
            'label' => $row['label'],
            'hours' => $row['hours'] ?? (string) __('directory.place.hours.unknown'),
            'closed' => $row['closed'],
            'today' => $stable && $row['today'],
        ], $hours->rows($stable ? $now : null));
    }

    /**
     * @return array{average: float, count: int, aspects: list<array{label: string, value: float, percent: int}>}|null
     */
    private function ratingBlock(RatingSummaryData $rating): ?array
    {
        if ($rating->isEmpty()) {
            return null;
        }

        $aspects = [];
        foreach (ReviewAspect::cases() as $aspect) {
            $value = $rating->aspects[$aspect->value] ?? null;
            if ($value !== null) {
                $aspects[] = ['label' => $aspect->label(), 'value' => $value, 'percent' => (int) (round($value / 5 * 20) * 5)];
            }
        }

        return ['average' => $rating->average, 'count' => $rating->count, 'aspects' => $aspects];
    }

    /**
     * @return array{name: string, rating: int, meta: string, text: string, demo: bool}
     */
    private function review(ReviewData $review): array
    {
        $meta = [jdate($review->createdAt, 'j F Y')];
        if ($review->isDemo) {
            $meta[] = (string) __('directory.place.reviews.demo');
        }

        return [
            'name' => $review->authorName,
            'rating' => $review->rating,
            'meta' => implode(' · ', $meta),
            'text' => $review->body,
            'demo' => $review->isDemo,
        ];
    }

    /**
     * @return array{previous: string|null, next: string|null, label: string}|null
     */
    private function reviewPagination(PlaceData $place, ReviewPage $reviews): ?array
    {
        $last = $reviews->lastPage();
        if ($last < 2) {
            return null;
        }
        $url = fn (int $page): string => route('directory.place', $page > 1 ? [$place->slug, 'page' => $page] : [$place->slug]).'#reviews';

        return [
            'previous' => $reviews->page > 1 ? $url($reviews->page - 1) : null,
            'next' => $reviews->page < $last ? $url($reviews->page + 1) : null,
            'label' => (string) __('directory.place.reviews.page_of', ['page' => fa_digits($reviews->page), 'last' => fa_digits($last)]),
        ];
    }

    /**
     * @return array{line: string|null, links: list<array{label: string, href: string, external: bool}>, pin: string}
     */
    private function address(PlaceData $place): array
    {
        $line = implode(' · ', array_filter([
            trim((string) $place->address) === '' ? null : trim((string) $place->address),
            implode('، ', array_filter([$place->district->name ?? null, $place->city->name])),
        ]));
        $maps = $place->mapLinks();

        return [
            'line' => $line === '' ? null : $line,
            'links' => $maps === null ? [] : [
                ['label' => (string) __('directory.place.address.geo'), 'href' => $maps->geo, 'external' => false],
                ['label' => (string) __('directory.place.address.neshan'), 'href' => $maps->neshan, 'external' => true],
                ['label' => (string) __('directory.place.address.balad'), 'href' => $maps->balad, 'external' => true],
                ['label' => (string) __('directory.place.address.google'), 'href' => $maps->google, 'external' => true],
            ],
            'pin' => $place->name,
        ];
    }

    /**
     * The unit of the cheapest priced service («هر جلسه»), for the booking panel's «از … تومان».
     */
    private function priceUnit(PlaceData $place): ?string
    {
        $priced = array_values(array_filter($place->services, static fn (PlaceServiceData $s): bool => $s->price !== null && $s->price > 0));
        usort($priced, static fn (PlaceServiceData $a, PlaceServiceData $b): int => $a->price <=> $b->price);

        return $priced[0]->priceUnit ?? null;
    }

    private function pageTtl(): int
    {
        $ttl = $this->config->get('pagecache.ttl') ?? $this->config->get('cacheaside.namespaces.pages');

        return is_numeric($ttl) ? max(0, (int) $ttl) : 3600;
    }

    private static function tz(): string
    {
        $tz = config('app.timezone');

        return is_string($tz) && $tz !== '' ? $tz : 'Asia/Tehran';
    }
}
