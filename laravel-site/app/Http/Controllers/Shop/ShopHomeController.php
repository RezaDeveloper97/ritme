<?php

declare(strict_types=1);

namespace App\Http\Controllers\Shop;

use App\Domain\Content\Enums\StaticPage;
use App\Domain\Content\SiteNavigation;
use App\Domain\Media\Contracts\MediaRepository;
use App\Domain\Media\Data\MediaData;
use App\Domain\Seo\Contracts\SeoMetaRepository;
use App\Domain\Seo\Schema\Data\ListEntry;
use App\Domain\Seo\Schema\Enums\WebPageType;
use App\Domain\Seo\Schema\Node;
use App\Domain\Seo\Schema\Nodes\ItemListNode;
use App\Domain\Seo\Schema\SchemaGraph;
use App\Domain\Seo\Schema\SchemaIds;
use App\Domain\Seo\SeoManager;
use App\Domain\Seo\Support\CanonicalUrl;
use App\Domain\Settings\Contracts\SettingsRepository;
use App\Domain\Shop\Catalog\Contracts\CatalogRepository;
use App\Domain\Shop\Catalog\Data\CategoryData;
use App\Domain\Shop\Catalog\Data\CategoryTree;
use App\Domain\Shop\Catalog\Data\ProductCardData;
use App\Domain\Shop\Catalog\Support\ProductListIndexing;
use App\Domain\Shop\Catalog\Support\ShopUrls;
use Illuminate\Contracts\Config\Repository as Config;
use Illuminate\Contracts\View\Factory as ViewFactory;
use Illuminate\Contracts\View\View;
use Illuminate\Http\Request;

/**
 * Shop home (`/shop`, design/html/shop.html) — pages/shop/index.blade.php.
 *
 * One block per department (root category of the visible tree, admin order): the promo tile, the department's
 * category tiles and its BestSellers row (purchasable products; editor picks lead until orders feed `sales_count`).
 * The design's «لیست سیسمونی» and «یادآور خرید قبل از پریود» are app features → app CTA cards (no fake progress or
 * switch). JSON-LD: CollectionPage + ItemList of the products shown; BreadcrumbList Home → فروشگاه comes from the
 * page registry. A shop that shows no real (non-demo) product is noindex. Reads: cached Shop repositories only.
 */
final class ShopHomeController
{
    /** Product cards per department row (design: 5). */
    public const PRODUCTS_PER_DEPARTMENT = ProductListIndexing::PRODUCTS_PER_DEPARTMENT;

    /** Category tiles per department (design: 8). */
    public const TILES_PER_DEPARTMENT = 8;

    /** Promo tile per department slug: tone (x-ui.promo-split) + illustration; the rest cycle through FALLBACK_PROMOS. */
    private const PROMOS = [
        'baby' => ['tone' => 'lavender', 'illustration' => 'product-sleepsuit'],
        'beauty' => ['tone' => 'blush', 'illustration' => 'product-serum'],
    ];

    private const FALLBACK_PROMOS = [
        ['tone' => 'lavender', 'illustration' => 'product-bodysuit'],
        ['tone' => 'blush', 'illustration' => 'product-shampoo'],
    ];

    /** Department switch icon per root slug (shop-list subnav); other roots get FALLBACK_ICON. */
    private const DEPARTMENT_ICONS = ['baby' => 'person', 'beauty' => 'sparkle'];

    private const FALLBACK_ICON = 'store';

    /**
     * Category tile art while a category has neither a cover photo nor its own illustration (the seeded demo tree has
     * three): the closest product illustration of the design, by category slug.
     */
    private const TILE_ILLUSTRATIONS = [
        'baby-clothes' => 'product-bodysuit',
        'sleepwear' => 'product-sleepsuit',
        'feeding' => 'product-bottle',
        'blankets' => 'product-blanket',
        'socks-hats' => 'product-socks',
        'feminine-hygiene' => 'product-pad',
        'skincare' => 'product-serum',
        'sunscreen' => 'product-sunscreen',
        'makeup' => 'product-lipstick',
        'hair' => 'product-shampoo',
        'body-bath' => 'category-bath',
        'menstrual-cups' => 'product-cup',
        'breastfeeding' => 'product-bottle',
    ];

    public function __construct(
        private readonly CatalogRepository $catalog,
        private readonly ProductListIndexing $indexing,
        private readonly MediaRepository $media,
        private readonly SeoManager $seo,
        private readonly SeoMetaRepository $meta,
        private readonly SchemaGraph $graph,
        private readonly SettingsRepository $settings,
        private readonly SiteNavigation $navigation,
        private readonly ShopUrls $urls,
        private readonly ViewFactory $views,
        private readonly Config $config,
    ) {}

    public function __invoke(Request $request): View
    {
        $tree = $this->catalog->categoryTree();

        $departments = [];
        $cards = [];
        foreach ($this->indexing->homeDepartments() as $i => [$root, $products]) {
            $departments[] = $this->department($tree, $root, $i, $products);
            array_push($cards, ...$products);
        }

        $this->describe($request, $cards);

        $appLinks = $this->settings->all()->appLinks;

        return $this->views->make('pages.shop.index', [
            'intro' => [
                'eyebrow' => (string) __('shop.eyebrow'),
                'title' => (string) __('shop.home.title'),
                'lead' => (string) __('shop.home.lead'),
            ],
            'promos' => array_map(static fn (array $d): array => $d['promo'], $departments),
            'departments' => $departments,
            'media' => $this->coverMedia($cards),
            'demo' => array_filter($cards, static fn (ProductCardData $c): bool => $c->isDemo) !== [],
            'appHref' => $appLinks->webApp ?? $this->navigation->url(StaticPage::Home, 'download'),
            'trust' => self::trust(),
        ]);
    }

    /**
     * @param  list<ProductCardData>  $products
     * @return array{slug: string, name: string, promo: array<string, string|null>, tiles: list<array{href: string, name: string, illustration: ?string, media: ?int}>, products: list<ProductCardData>, productsTitle: string, productsLead: ?string, href: string}
     */
    private function department(CategoryTree $tree, CategoryData $root, int $position, array $products): array
    {
        $copy = __("shop.departments.{$root->slug}");
        $copy = is_array($copy) ? $copy : [];
        $promo = self::PROMOS[$root->slug] ?? self::FALLBACK_PROMOS[$position % count(self::FALLBACK_PROMOS)];
        $href = $this->urls->category($root->slug);

        $tiles = [];
        foreach (array_slice($tree->children($root->id), 0, self::TILES_PER_DEPARTMENT) as $child) {
            $tiles[] = [
                'href' => $this->urls->category($child->slug),
                'name' => $child->name,
                'illustration' => $child->illustration ?? self::TILE_ILLUSTRATIONS[$child->slug] ?? null,
                'media' => $child->coverMediaId,
            ];
        }

        return [
            'slug' => $root->slug,
            'name' => $root->name,
            'href' => $href,
            'promo' => [
                'href' => $href,
                'tone' => $promo['tone'],
                'eyebrow' => $root->name,
                'title' => is_string($copy['title'] ?? null) ? $copy['title'] : $root->name,
                'text' => $root->intro,
                'cta' => (string) __('shop.department.cta'),
                'illustration' => $promo['illustration'],
            ],
            'tiles' => $tiles,
            'products' => $products,
            'productsTitle' => is_string($copy['products_title'] ?? null) ? $copy['products_title'] : (string) __('shop.home.products_title', ['name' => $root->name]),
            'productsLead' => is_string($copy['products_lead'] ?? null) ? $copy['products_lead'] : null,
        ];
    }

    /**
     * Title / description (admin seo_meta of `shop.index` wins), CollectionPage + ItemList, noindex while no real
     * product is shown.
     *
     * @param  list<ProductCardData>  $cards
     */
    private function describe(Request $request, array $cards): void
    {
        $meta = $this->meta->forRoute(StaticPage::Shop->routeName());
        if (($meta->title ?? '') === '') {
            $this->seo->rawTitle((string) __('shop.home.seo_title'));
        }
        if (($meta->description ?? '') === '') {
            $this->seo->description((string) __('shop.home.seo_description'));
        }

        $name = StaticPage::Shop->label();
        $this->graph->pageType(WebPageType::CollectionPage)->pageName($name);
        if (! ProductListIndexing::showsRealProduct($cards)) {
            $this->seo->noindex(); // empty or demo-only shop: nothing real to index yet
        }
        if ($cards === []) {
            return;
        }

        $pageUrl = CanonicalUrl::normalize($request->fullUrl(), (string) $this->config->get('app.url'));
        $this->graph->add(ItemListNode::make($pageUrl, array_map(
            fn (ProductCardData $card): ListEntry => new ListEntry($this->urls->product($card->slug), $card->title),
            $cards,
        ), $name));
        $this->graph->add(['@id' => SchemaIds::webPage($pageUrl), 'mainEntity' => Node::ref(SchemaIds::itemList($pageUrl))]);
    }

    public static function departmentIcon(string $slug): string
    {
        return self::DEPARTMENT_ICONS[$slug] ?? self::FALLBACK_ICON;
    }

    /**
     * Cover images of all cards in one cached lookup (x-picture gets MediaData, no per-card query).
     *
     * @param  list<ProductCardData>  $cards
     * @return array<int, MediaData>
     */
    private function coverMedia(array $cards): array
    {
        $ids = array_values(array_unique(array_filter(array_map(static fn (ProductCardData $c): ?int => $c->coverMediaId, $cards))));

        return $ids === [] ? [] : $this->media->findMany($ids);
    }

    /**
     * @return list<array{icon: string, title: string, text: string}>
     */
    private static function trust(): array
    {
        $items = __('shop.trust.items');

        return array_values(array_filter(is_array($items) ? $items : [], static fn (mixed $item): bool => is_array($item)));
    }
}
