<?php

declare(strict_types=1);

namespace App\Domain\Seo\Actions;

use App\Domain\Blog\Models\Category as BlogCategory;
use App\Domain\Blog\Models\Post;
use App\Domain\Content\Enums\StaticPage;
use App\Domain\Directory\Models\Place;
use App\Domain\Seo\Analysis\AnalysisInput;
use App\Domain\Seo\Analysis\ContentType;
use App\Domain\Seo\Analysis\Queries\FindDuplicateSeoMeta;
use App\Domain\Seo\Models\SeoMeta;
use App\Domain\Seo\Support\DescriptionText;
use App\Domain\Settings\Data\SeoDefaults;
use App\Domain\Shop\Catalog\Models\Category as ShopCategory;
use App\Domain\Shop\Catalog\Models\Product;
use Illuminate\Contracts\Config\Repository as Config;
use Illuminate\Contracts\Routing\UrlGenerator;
use Illuminate\Database\Eloquent\Model;
use Illuminate\Routing\Router;

/**
 * The pages the bulk SEO editor (L7-06) works on — posts, products, places, blog + shop categories and the indexable
 * static pages — behind one string key per page: `{morph alias}-{id}` for models (`blog_post-12`), `page-{route}`
 * for static pages (dots of the route name become `--`, so the key is a safe Livewire path segment).
 *
 * handle() resolves a key to the page's `seo_meta` row (new unsaved one when missing); analysisInput() builds the
 * content analyser input. Used by ListBulkSeoRows, BulkSaveSeoMeta, ResetSeoMeta, RecomputeSeoScores and the CSV import.
 */
final class ResolveSeoTarget
{
    public const PAGE = 'page';

    /**
     * Model page types: morph alias => model, label, name column, excerpt / body columns, public route, analyser type.
     *
     * @var array<string, array{model: class-string<Model>, label: string, name: string, excerpt: string, body: string|null, route: string, type: ContentType}>
     */
    public const TYPES = [
        'blog_post' => ['model' => Post::class, 'label' => 'مقاله', 'name' => 'title', 'excerpt' => 'excerpt', 'body' => 'body', 'route' => 'blog.show', 'type' => ContentType::Post],
        'shop_product' => ['model' => Product::class, 'label' => 'محصول', 'name' => 'title', 'excerpt' => 'short_description', 'body' => 'description', 'route' => 'shop.product', 'type' => ContentType::Product],
        'directory_place' => ['model' => Place::class, 'label' => 'مرکز', 'name' => 'name', 'excerpt' => 'summary', 'body' => 'description', 'route' => 'directory.place', 'type' => ContentType::Place],
        'blog_category' => ['model' => BlogCategory::class, 'label' => 'دسته مجله', 'name' => 'name', 'excerpt' => 'description', 'body' => null, 'route' => 'blog.category', 'type' => ContentType::Archive],
        'shop_category' => ['model' => ShopCategory::class, 'label' => 'دسته فروشگاه', 'name' => 'name', 'excerpt' => 'intro', 'body' => null, 'route' => 'shop.category', 'type' => ContentType::Archive],
    ];

    public function __construct(
        private readonly ListStaticPageSeo $staticPages,
        private readonly FindDuplicateSeoMeta $duplicates,
        private readonly UrlGenerator $url,
        private readonly Router $router,
        private readonly Config $config,
    ) {}

    /**
     * @return array<string, string> type => Persian label (static pages included), for filters
     */
    public static function typeOptions(): array
    {
        $options = [];
        foreach (self::TYPES as $type => $config) {
            $options[$type] = $config['label'];
        }

        return $options + [self::PAGE => 'صفحه ثابت'];
    }

    public static function key(string $type, int|string $id): string
    {
        return $type.'-'.$id;
    }

    public static function pageKey(StaticPage $page): string
    {
        return self::PAGE.'-'.str_replace('.', '--', $page->routeName());
    }

    /**
     * @return array{type: string, id: int|null, page: StaticPage|null}|null
     */
    public static function parse(string $key): ?array
    {
        [$type, $id] = array_pad(explode('-', $key, 2), 2, '');

        if ($type === self::PAGE) {
            foreach (ListStaticPageSeo::pages() as $page) {
                if (self::pageKey($page) === $key) {
                    return ['type' => self::PAGE, 'id' => null, 'page' => $page];
                }
            }

            return null;
        }

        return isset(self::TYPES[$type]) && ctype_digit($id) && (int) $id > 0
            ? ['type' => $type, 'id' => (int) $id, 'page' => null]
            : null;
    }

    /**
     * The page's seo_meta row; a new unsaved one (keyed by morph / route name) when it has none yet.
     * Returns null for an unknown key or a deleted model.
     */
    public function handle(string $key): ?SeoMeta
    {
        $target = self::parse($key);
        if ($target === null) {
            return null;
        }

        if ($target['page'] !== null) {
            return SeoMeta::query()->firstOrNew(['route_name' => $target['page']->routeName()]);
        }

        $model = self::TYPES[$target['type']]['model'];
        if (! $model::query()->whereKey($target['id'])->exists()) {
            return null;
        }

        return SeoMeta::query()->firstOrNew(['seoable_type' => $target['type'], 'seoable_id' => $target['id']]);
    }

    public function url(string $type, string $slug): string
    {
        $route = self::TYPES[$type]['route'] ?? null;

        return $route !== null && $this->router->has($route) ? $this->url->route($route, ['slug' => $slug]) : $this->url->to('/');
    }

    /**
     * Effective meta description of a model page: admin override, else the excerpt / body start, else the default.
     */
    public static function description(?string $override, ?string $excerpt, ?string $body, SeoDefaults $seo): string
    {
        $override = trim((string) $override);
        if ($override !== '') {
            return $override;
        }

        foreach ([$excerpt, $body] as $text) {
            $text = DescriptionText::fromExcerpt((string) $text);
            if ($text !== '') {
                return $text;
            }
        }

        return $seo->defaultDescription;
    }

    /**
     * What the content analyser sees for a page, built like the SEO tab does (SeoFields). Null for unknown keys.
     */
    public function analysisInput(string $key, SeoMeta $meta, SeoDefaults $seo): ?AnalysisInput
    {
        $target = self::parse($key);
        if ($target === null) {
            return null;
        }

        $duplicates = $this->duplicates->handle($meta->title, $meta->description, $meta->exists ? $meta->id : null);
        $hosts = array_values(array_unique(array_filter([(string) parse_url((string) $this->config->get('app.url'), PHP_URL_HOST)])));

        if ($target['page'] !== null) {
            $row = $this->staticPages->find($target['page']);
            $segments = explode('/', trim($target['page']->path(), '/'));

            return new AnalysisInput(
                type: ContentType::Page,
                title: $row->title,
                description: $row->description,
                slug: (string) end($segments),
                focusKeyword: trim((string) $meta->focus_keyword),
                ownHosts: $hosts,
                cornerstone: $meta->cornerstone,
                duplicateTitle: $duplicates['title'],
                duplicateDescription: $duplicates['description'],
            );
        }

        $config = self::TYPES[$target['type']];
        $model = $config['model']::query()->find($target['id']);
        if ($model === null) {
            return null;
        }

        $name = trim((string) $model->getAttribute($config['name']));
        $excerpt = (string) $model->getAttribute($config['excerpt']);
        $body = $config['body'] !== null ? (string) $model->getAttribute($config['body']) : '';
        $override = trim((string) $meta->title);

        return new AnalysisInput(
            type: $config['type'],
            title: $seo->title($override !== '' ? $override : $name),
            heading: $name,
            description: self::description($meta->description, $excerpt, $body, $seo),
            slug: (string) $model->getAttribute('slug'),
            focusKeyword: trim((string) $meta->focus_keyword),
            contentHtml: $config['body'] !== null ? $body : (trim($excerpt) !== '' ? $excerpt : null),
            ownHosts: $hosts,
            cornerstone: $meta->cornerstone,
            duplicateTitle: $duplicates['title'],
            duplicateDescription: $duplicates['description'],
        );
    }
}
