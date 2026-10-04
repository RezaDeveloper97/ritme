<?php

declare(strict_types=1);

namespace App\Http\Controllers\Shop;

use App\Domain\Media\Contracts\MediaRepository;
use App\Domain\Media\Data\MediaData;
use App\Domain\Seo\Contracts\OgImageResolver;
use App\Domain\Seo\Contracts\SeoMetaRepository;
use App\Domain\Seo\Schema\Data\BreadcrumbItem;
use App\Domain\Seo\Schema\Enums\WebPageType;
use App\Domain\Seo\Schema\Node;
use App\Domain\Seo\Schema\Nodes\ProductNode;
use App\Domain\Seo\Schema\PageGraph;
use App\Domain\Seo\Schema\SchemaGraph;
use App\Domain\Seo\Schema\SchemaIds;
use App\Domain\Seo\SeoManager;
use App\Domain\Seo\Support\CanonicalUrl;
use App\Domain\Seo\Support\DescriptionText;
use App\Domain\Shop\Catalog\Actions\SubmitProductReview;
use App\Domain\Shop\Catalog\Contracts\CatalogRepository;
use App\Domain\Shop\Catalog\Contracts\ProductRepository;
use App\Domain\Shop\Catalog\Data\CategoryData;
use App\Domain\Shop\Catalog\Data\CategoryTree;
use App\Domain\Shop\Catalog\Data\ProductCardData;
use App\Domain\Shop\Catalog\Data\ProductData;
use App\Domain\Shop\Catalog\Data\ReviewData;
use App\Domain\Shop\Catalog\Data\ReviewPage;
use App\Domain\Shop\Catalog\Data\VariantData;
use App\Domain\Shop\Catalog\Enums\StockStatus;
use App\Domain\Shop\Catalog\Models\Product;
use App\Domain\Shop\Catalog\Support\ShopUrls;
use App\Support\Money\Money;
use App\View\Components\Picture;
use Carbon\CarbonImmutable;
use Illuminate\Contracts\Config\Repository as Config;
use Illuminate\Contracts\Validation\Factory as ValidatorFactory;
use Illuminate\Contracts\View\View;
use Illuminate\Database\Eloquent\ModelNotFoundException;
use Illuminate\Http\RedirectResponse;
use Illuminate\Http\Request;
use Illuminate\Routing\Router;
use Illuminate\Validation\Rule;
use InvalidArgumentException;

/**
 * `/shop/product/{slug}` — the product page (L6-03, design/html/shop-product.html) and its review form POST.
 *
 * Unknown slug: an old slug from the slug history 301s to the current URL (query kept), anything else 404s. Approved
 * reviews (newest first) are paginated with `?page=n` (self-canonical, `?page=1` 301s to the clean URL, out of range
 * 404s). Demo products are noindex, demo reviews are labelled «نمونه» and never counted (L6-01).
 *
 * Variant picker: plain radio inputs (`color`, `size`) inside the add-to-cart form, so it works without JS; a size is
 * disabled only when it is sold out in every colour. The lazy `product` module refines it per colour (sold-out sizes,
 * price, stock, size hint, size-chart row) and drives the quantity stepper. The cart (L6-04) resolves the variant
 * from size + colour and re-checks the stock; until a `shop.cart.add` route exists the button is disabled and the
 * form carries a marked `data-cart-slot`.
 *
 * SEO: admin seo_meta of the product wins, else title / short description templates. og:type product with
 * `product:price:*` (pushed by the view). JSON-LD: ItemPage + Product (images, sku, brand, Offer — or AggregateOffer
 * with one Offer per variant — availability from stock, `aggregateRating` + `review` only from real approved reviews)
 * + BreadcrumbList (registered by <x-ui.breadcrumbs>). Reads go through the cached Shop repositories; the view gets
 * DTOs and scalar arrays. The review form keeps CSRF working on page-cache HITs; the POST is rate limited (route),
 * has a honeypot (`company_url`, answered like a real submit) and stores the review as pending (SubmitProductReview).
 */
final class ProductController
{
    public const REVIEWS_PER_PAGE = 6;

    /** Review POSTs per IP per 10 minutes (route middleware). */
    public const REVIEWS_PER_10_MINUTES = 3;

    /** Honeypot field of the review form: humans never see or fill it. */
    public const HONEYPOT = 'company_url';

    /** Upper bound of the quantity stepper (the cart re-checks the stock). */
    public const MAX_QUANTITY = 10;

    private const CROSS_SELLS = 5;

    /** Days an Offer's price is declared valid in the JSON-LD (`priceValidUntil`). */
    private const PRICE_VALID_DAYS = 30;

    public function __construct(
        private readonly ProductRepository $products,
        private readonly CatalogRepository $catalog,
        private readonly MediaRepository $media,
        private readonly SeoMetaRepository $meta,
        private readonly OgImageResolver $ogImages,
        private readonly ShopUrls $urls,
        private readonly Router $router,
        private readonly Config $config,
    ) {}

    public function show(Request $request, string $slug, SeoManager $seo, SchemaGraph $graph): View|RedirectResponse
    {
        $product = $this->products->findPublishedBySlug($slug);
        if ($product === null) {
            $current = $this->products->currentSlugFor($slug);
            if ($current === null || $current === $slug) {
                abort(404);
            }
            $query = $request->getQueryString();

            return new RedirectResponse(route('shop.product', [$current]).($query !== null && $query !== '' ? '?'.$query : ''), 301);
        }

        $rawPage = $request->query('page');
        $page = 1;
        if ($rawPage !== null) {
            $page = is_string($rawPage) && preg_match('/^[1-9]\d{0,5}$/', $rawPage) === 1 ? (int) $rawPage : 0;
            if ($page === 1) {
                return new RedirectResponse(route('shop.product', [$product->slug]), 301);
            }
            if ($page < 1) {
                abort(404);
            }
        }

        $reviews = $this->products->reviews($product->id, $page, self::REVIEWS_PER_PAGE);
        if ($page > $reviews->lastPage()) {
            abort(404);
        }

        $tree = $this->catalog->categoryTree();
        $productUrl = CanonicalUrl::normalize($this->urls->product($product->slug), (string) $this->config->get('app.url'));
        $images = $this->images($product);
        $crossSells = $this->products->frequentlyBoughtWith($product->id, self::CROSS_SELLS);
        $picker = $this->picker($product);
        $this->describe($seo, $graph, $product, $page, $productUrl, $images, $reviews);
        $rating = $product->rating();

        return view('pages.shop.product', [
            'product' => $product,
            'breadcrumbs' => $this->trail($tree, $product, $productUrl),
            'subnav' => $this->subnav($tree, $product),
            'gallery' => ['images' => $images, 'illustration' => $product->illustration, 'alt' => $product->title],
            'header' => [
                'brand' => $product->brand?->name,
                'rating' => $rating === null ? null : ['value' => $rating->ratingValue, 'count' => $rating->ratingCount],
                'sales' => $product->salesCount > 0 ? $product->salesCount : null,
            ],
            'picker' => $picker,
            'cart' => [
                'action' => $this->router->has('shop.cart.add') ? route('shop.cart.add') : null,
                'purchasable' => $product->isPurchasable() && ($product->variants === [] || $this->firstInStock($product) !== null),
            ],
            'og' => [
                'amount' => (string) $picker['price']['rial'],
                'currency' => Money::CURRENCY,
                'availability' => $product->isPurchasable() ? 'instock' : 'oos',
            ],
            'specs' => $product->specs,
            'description' => $product->description,
            'sizeChart' => $product->sizeChart,
            'reviews' => array_map($this->review(...), $reviews->items),
            'reviewsDemo' => array_filter($reviews->items, static fn (ReviewData $r): bool => $r->isDemo) !== [],
            'reviewPagination' => $this->reviewPagination($product, $reviews),
            'reviewForm' => [
                'action' => route('shop.product.review', [$product->slug]),
                'sizes' => $product->sizes(),
                'honeypot' => self::HONEYPOT,
            ],
            'crossSells' => $crossSells,
            'crossSellMedia' => $this->coverMedia($crossSells),
        ]);
    }

    public function storeReview(Request $request, string $slug, ValidatorFactory $validator, SubmitProductReview $submit): RedirectResponse
    {
        $product = $this->products->findPublishedBySlug($slug) ?? abort(404);
        $back = route('shop.product', [$product->slug]).'#review-form';

        // Honeypot: bots get the same answer as people, nothing is stored.
        if (trim((string) $request->input(self::HONEYPOT, '')) !== '') {
            return redirect()->to($back)->with('review_submitted', true);
        }

        $data = $validator->make($request->all(), [
            'author_name' => ['required', 'string', 'min:2', 'max:'.SubmitProductReview::MAX_NAME],
            'rating' => ['required', 'integer', 'between:1,5'],
            'body' => ['required', 'string', 'min:10', 'max:'.SubmitProductReview::MAX_BODY],
            'size' => ['nullable', 'string', Rule::in($product->sizes())],
        ], [
            'author_name.*' => __('shop.product.review.errors.name'),
            'rating.*' => __('shop.product.review.errors.rating'),
            'body.min' => __('shop.product.review.errors.body_short'),
            'body.*' => __('shop.product.review.errors.body'),
            'size.*' => __('shop.product.review.errors.size'),
        ]);

        if ($data->fails()) {
            return redirect()->to($back)->withErrors($data)->withInput($request->except(self::HONEYPOT));
        }

        /** @var array{author_name: string, rating: int|string, body: string, size?: string|null} $valid */
        $valid = $data->validated();
        $size = $valid['size'] ?? null;

        try {
            $submit->handle(
                productSlug: $product->slug,
                authorName: (string) $valid['author_name'],
                rating: (int) $valid['rating'],
                body: (string) $valid['body'],
                variantLabel: $size === null || $size === '' ? null : (string) __('shop.product.review.size_label', ['size' => $size]),
                ip: $request->ip(),
            );
        } catch (ModelNotFoundException) {
            abort(404);
        } catch (InvalidArgumentException) {
            return redirect()->to($back)->withErrors(['body' => __('shop.product.review.errors.body')])->withInput($request->except(self::HONEYPOT));
        }

        return redirect()->to($back)->with('review_submitted', true);
    }

    /**
     * Title, description, robots, OG image and the JSON-LD of the page. SeoManager / SchemaGraph are request-scoped,
     * so they are method-injected.
     *
     * @param  list<array{id: int, url: string}>  $images
     */
    private function describe(SeoManager $seo, SchemaGraph $graph, ProductData $product, int $page, string $productUrl, array $images, ReviewPage $reviews): void
    {
        $seo->for((new Product)->forceFill(['id' => $product->id]))->type('product');
        $meta = $this->meta->forModel((new Product)->getMorphClass(), $product->id);

        $title = $meta->title ?? (string) __('shop.product.seo_title', ['name' => $product->title]);
        $template = $product->brand === null
            ? (string) __('shop.product.seo_description_plain', ['name' => $product->title])
            : (string) __('shop.product.seo_description', ['name' => $product->title, 'brand' => $product->brand->name]);
        $short = trim((string) $product->shortDescription);
        $description = $meta->description ?? DescriptionText::fromExcerpt($short !== '' ? $short.' '.$template : $template);
        if ($page > 1) {
            $suffix = (string) __('shop.product.reviews.page', ['page' => fa_digits($page)]);
            $title .= ' — '.$suffix;
            $description = DescriptionText::fromExcerpt($suffix.' — '.$description);
        }
        $seo->title($title)->description($description);
        if ($product->isDemo) {
            $seo->noindex(); // sample catalogue (L6-01): never indexed, never in the sitemap
        }

        $first = $images[0]['id'] ?? null;
        $og = $first === null ? null : $this->ogImages->resolve($first, $product->title);
        if ($og !== null) {
            $seo->image($og);
        }

        $graph->pageType(WebPageType::ItemPage)->pageName($product->title)->dates(null, $product->updatedAt);

        $siteUrl = (string) $this->config->get('app.url');
        $node = ProductNode::make($product->toSchema($productUrl, array_column($images, 'url')));
        $node['offers'] = $product->offersNode(
            $productUrl,
            CarbonImmutable::now(self::tz())->addDays(self::PRICE_VALID_DAYS)->toDateString(),
            Node::ref(SchemaIds::organization($siteUrl)),
        );
        $real = array_values(array_filter($reviews->items, static fn (ReviewData $r): bool => ! $r->isDemo));
        if ($product->rating() !== null && $real !== []) {
            $node['review'] = array_map(static fn (ReviewData $r): array => Node::clean([
                '@type' => 'Review',
                'author' => ['@type' => 'Person', 'name' => $r->authorName],
                'datePublished' => $r->approvedAt === null ? null : CarbonImmutable::parse($r->approvedAt)->setTimezone(self::tz())->toDateString(),
                'reviewBody' => $r->body,
                'reviewRating' => ['@type' => 'Rating', 'ratingValue' => $r->rating, 'bestRating' => 5, 'worstRating' => 1],
            ]), array_slice($real, 0, 5));
        }
        $graph->add($node);
        $graph->add(['@id' => SchemaIds::webPage($productUrl), 'mainEntity' => Node::ref(SchemaIds::product($productUrl))]);
    }

    /**
     * Cover + gallery that resolve to a real media file: id + absolute URL of the large variant (schema `image`,
     * lightbox full size).
     *
     * @return list<array{id: int, url: string}>
     */
    private function images(ProductData $product): array
    {
        $images = [];
        foreach ($product->mediaIds() as $id) {
            $url = Picture::mediaUrl($id, 'desktop_1920');
            if ($url !== null) {
                $images[] = ['id' => $id, 'url' => CanonicalUrl::normalize($url, (string) $this->config->get('app.url'))];
            }
        }

        return $images;
    }

    /**
     * Variant picker state for the view and the `product` module. Default selection = the first variant in stock
     * (else the first one). A size is `disabled` only when it is sold out in every colour (the form must work without
     * JS for any colour); `soldout` marks sizes without stock in the default colour (struck through, still selectable).
     *
     * @return array{colors: list<array{name: string, hex: ?string, checked: bool}>, sizes: list<array{name: string, checked: bool, disabled: bool, soldout: bool}>, color: ?string, size: ?string, hint: ?string, hints: array<string, string>, price: array{amount: int, compare: ?int, rial: int}, stock: array{in: bool, label: string}, labels: array{in: string, out: string}, quantityMax: int, variants: list<array{id: int, size: ?string, color: ?string, price: string, compare: ?string, stock: bool, max: int}>}
     */
    private function picker(ProductData $product): array
    {
        $default = $this->firstInStock($product) ?? ($product->variants[0] ?? null);
        $color = $default?->color;
        $size = $default?->size;
        $hints = $this->sizeHints($product);
        $manual = $product->stockStatus === StockStatus::PreOrder || $product->stockStatus === StockStatus::BackOrder;
        $inStock = static fn (VariantData $v): bool => $manual || $v->isInStock();

        $sizes = [];
        foreach ($product->sizes() as $name) {
            $ofSize = array_filter($product->variants, static fn (VariantData $v): bool => $v->size === $name);
            $ofDefault = array_filter($ofSize, static fn (VariantData $v): bool => $v->color === $color);
            $sizes[] = [
                'name' => $name,
                'checked' => $name === $size,
                'disabled' => array_filter($ofSize, $inStock) === [],
                'soldout' => array_filter($ofDefault, $inStock) === [],
            ];
        }

        $colors = array_map(static fn (array $c): array => [
            'name' => $c['name'],
            'hex' => is_string($c['hex']) && preg_match('/^#[0-9A-Fa-f]{6}$/', $c['hex']) === 1 ? $c['hex'] : null,
            'checked' => $c['name'] === $color,
        ], $product->colors());

        $price = $default->price ?? $product->price;
        $compare = $default === null ? $product->compareAtPrice : $default->compareAtPrice;
        $available = $default === null ? $product->isPurchasable() : $inStock($default);
        $stockQty = $default === null ? $product->stockQty : $default->stockQty;

        return [
            'colors' => $colors,
            'sizes' => $sizes,
            'color' => $color,
            'size' => $size,
            'hint' => $size === null ? null : ($hints[$size] ?? null),
            'hints' => $hints,
            'price' => [
                'amount' => $price->toToman(),
                'compare' => $compare !== null && $compare->greaterThan($price) ? $compare->toToman() : null,
                'rial' => $price->rial,
            ],
            'stock' => ['in' => $available, 'label' => $available ? ($manual ? $product->stockStatus->label() : StockStatus::InStock->label()) : StockStatus::OutOfStock->label()],
            'labels' => ['in' => $manual ? $product->stockStatus->label() : StockStatus::InStock->label(), 'out' => StockStatus::OutOfStock->label()],
            'quantityMax' => $manual ? self::MAX_QUANTITY : max(1, min(self::MAX_QUANTITY, $stockQty)),
            'variants' => array_map(static fn (VariantData $v): array => [
                'id' => $v->id,
                'size' => $v->size,
                'color' => $v->color,
                'price' => $v->price->formatAmount(true),
                'compare' => $v->compareAtPrice !== null && $v->compareAtPrice->greaterThan($v->price) ? $v->compareAtPrice->formatAmount(true) : null,
                'stock' => $inStock($v),
                'max' => $manual ? self::MAX_QUANTITY : max(1, min(self::MAX_QUANTITY, $v->stockQty)),
            ], $product->variants),
        ];
    }

    private function firstInStock(ProductData $product): ?VariantData
    {
        if ($product->stockStatus === StockStatus::PreOrder || $product->stockStatus === StockStatus::BackOrder) {
            return $product->variants[0] ?? null;
        }
        foreach ($product->variants as $variant) {
            if ($variant->isInStock()) {
                return $variant;
            }
        }

        return null;
    }

    /**
     * «۳-۶ ماه: سن ۳ تا ۶ ماه · قد ۶۱ تا ۶۷ سانت · …» per size, from the size-chart row whose first cell is the size.
     * A unit in brackets in the column header («قد (سانت)») moves behind the value.
     *
     * @return array<string, string>
     */
    private function sizeHints(ProductData $product): array
    {
        $chart = $product->sizeChart;
        if ($chart === null || $chart['columns'] === []) {
            return [];
        }

        $hints = [];
        foreach ($chart['rows'] as $row) {
            $size = trim((string) ($row[0] ?? ''));
            if ($size === '' || ! in_array($size, $product->sizes(), true)) {
                continue;
            }
            $parts = [];
            foreach (array_slice($chart['columns'], 1, null, true) as $i => $column) {
                $value = trim((string) ($row[$i] ?? ''));
                if ($value === '') {
                    continue;
                }
                $parts[] = preg_match('/^(.+?)\s*\((.+)\)$/u', trim($column), $m) === 1 ? $m[1].' '.$value.' '.$m[2] : trim($column).' '.$value;
            }
            if ($parts !== []) {
                $hints[$size] = $size.': '.implode(' · ', $parts);
            }
        }

        return $hints;
    }

    /**
     * Home → فروشگاه → category path of the primary category → product.
     *
     * @return list<BreadcrumbItem>
     */
    private function trail(CategoryTree $tree, ProductData $product, string $productUrl): array
    {
        $trail = [
            new BreadcrumbItem(PageGraph::HOME_LABEL, SchemaIds::root((string) $this->config->get('app.url'))),
            new BreadcrumbItem((string) __('shop.name'), $this->urls->absolute(route('shop.index'))),
        ];
        foreach ($this->path($tree, $product) as $step) {
            $trail[] = new BreadcrumbItem($step->name, $this->urls->category($step->slug));
        }
        $trail[] = new BreadcrumbItem($product->title, $productUrl);

        return $trail;
    }

    /**
     * Visible category path of the product's primary category (else its first category).
     *
     * @return list<CategoryData>
     */
    private function path(CategoryTree $tree, ProductData $product): array
    {
        foreach ([$product->primaryCategory, ...$product->categories] as $category) {
            if ($category !== null && $tree->find($category->id) !== null) {
                return $tree->path($category->id);
            }
        }

        return [];
    }

    /**
     * Department switch (root categories) + the categories of the product's department (as on the listing).
     *
     * @return array{departments: list<array{label: string, href: string, active: bool, icon: string}>, links: list<array{label: string, href: string, active: bool}>, label: string}
     */
    private function subnav(CategoryTree $tree, ProductData $product): array
    {
        $path = $this->path($tree, $product);
        $root = $path[0] ?? ($tree->roots()[0] ?? null);
        $ids = array_map(static fn (CategoryData $c): int => $c->id, $path);

        $departments = array_map(fn (CategoryData $department): array => [
            'label' => $department->name,
            'href' => $this->urls->category($department->slug),
            'active' => $root !== null && $department->id === $root->id,
            'icon' => ShopHomeController::departmentIcon($department->slug),
        ], $tree->roots());

        $links = $root === null ? [] : array_map(fn (CategoryData $child): array => [
            'label' => $child->name,
            'href' => $this->urls->category($child->slug),
            'active' => in_array($child->id, $ids, true),
        ], $tree->children($root->id));

        return ['departments' => $departments, 'links' => $links, 'label' => (string) __('shop.subnav.categories', ['name' => $root->name ?? __('shop.name')])];
    }

    /**
     * @return array{name: string, rating: int, meta: ?string, text: string, demo: bool}
     */
    private function review(ReviewData $review): array
    {
        $meta = array_filter([
            $review->isVerifiedPurchase ? (string) __('shop.product.reviews.verified') : null,
            $review->isDemo || $review->approvedAt === null ? null : jdate(CarbonImmutable::parse($review->approvedAt), 'j F Y'),
            $review->variantLabel,
            $review->isDemo ? (string) __('shop.product.reviews.demo') : null,
        ], static fn (?string $part): bool => $part !== null && $part !== '');

        return [
            'name' => $review->authorName,
            'rating' => $review->rating,
            'meta' => $meta === [] ? null : implode(' · ', $meta),
            'text' => $review->body,
            'demo' => $review->isDemo,
        ];
    }

    /**
     * @return array{previous: ?string, next: ?string, label: string}|null
     */
    private function reviewPagination(ProductData $product, ReviewPage $reviews): ?array
    {
        $last = $reviews->lastPage();
        if ($last < 2) {
            return null;
        }
        $url = static fn (int $page): string => route('shop.product', $page > 1 ? [$product->slug, 'page' => $page] : [$product->slug]).'#reviews';

        return [
            'previous' => $reviews->page > 1 ? $url($reviews->page - 1) : null,
            'next' => $reviews->page < $last ? $url($reviews->page + 1) : null,
            'label' => (string) __('shop.product.reviews.page_of', ['page' => fa_digits($reviews->page), 'last' => fa_digits($last)]),
        ];
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

    private static function tz(): string
    {
        $tz = config('app.timezone');

        return is_string($tz) && $tz !== '' ? $tz : 'Asia/Tehran';
    }
}
