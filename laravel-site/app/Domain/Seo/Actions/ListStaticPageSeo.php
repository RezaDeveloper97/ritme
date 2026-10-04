<?php

declare(strict_types=1);

namespace App\Domain\Seo\Actions;

use App\Domain\Content\Enums\StaticPage;
use App\Domain\Seo\Audit\SeoAuditor;
use App\Domain\Seo\Contracts\SeoMetaRepository;
use App\Domain\Seo\Data\SeoMetaData;
use App\Domain\Seo\Data\StaticPageSeoData;
use App\Domain\Seo\StaticPages\StaticPageSeoDefaults;
use App\Domain\Seo\Support\CanonicalUrl;
use App\Domain\Seo\Support\Robots;
use App\Domain\Settings\Contracts\SettingsRepository;
use App\Domain\Settings\Data\SeoDefaults;
use Illuminate\Contracts\Config\Repository as Config;
use Illuminate\Contracts\Routing\UrlGenerator;
use Illuminate\Routing\Router;

/**
 * Every indexable page of the StaticPage registry (with or without a seo_meta row) with its effective title /
 * description — the admin override through the title template, else the controller's lang default — and the checks
 * the admin list shows: length (seo:audit limits), unique among these pages, OG image chosen, indexable.
 * Reads go through the cached SeoMetaRepository / SettingsRepository.
 */
final class ListStaticPageSeo
{
    public function __construct(
        private readonly SeoMetaRepository $meta,
        private readonly StaticPageSeoDefaults $defaults,
        private readonly SettingsRepository $settings,
        private readonly Router $router,
        private readonly UrlGenerator $url,
        private readonly Config $config,
    ) {}

    /**
     * @return list<StaticPage>
     */
    public static function pages(): array
    {
        return array_values(array_filter(StaticPage::cases(), static fn (StaticPage $page): bool => $page->indexable()));
    }

    /**
     * @return list<StaticPageSeoData>
     */
    public function handle(): array
    {
        $seo = $this->settings->all()->seo;
        $rows = [];
        foreach (self::pages() as $page) {
            $rows[] = $this->build($page, $seo);
        }

        // Duplicate titles / descriptions among the indexable pages (the seo:audit cross-check).
        $titles = array_count_values(array_map(static fn (array $r): string => $r['title'], $rows));
        $descriptions = array_count_values(array_map(static fn (array $r): string => $r['description'], $rows));

        return array_map(fn (array $r): StaticPageSeoData => $this->data(
            $r,
            ! $r['indexable'] || (($titles[$r['title']] ?? 0) < 2 && ($descriptions[$r['description']] ?? 0) < 2),
        ), $rows);
    }

    public function find(StaticPage $page): StaticPageSeoData
    {
        foreach ($this->handle() as $row) {
            if ($row->page === $page) {
                return $row;
            }
        }

        return $this->data($this->build($page, $this->settings->all()->seo), true);
    }

    public function url(StaticPage $page): string
    {
        $url = $this->router->has($page->routeName()) ? $this->url->route($page->routeName()) : $this->url->to($page->path());

        return CanonicalUrl::normalize($url, (string) $this->config->get('app.url'));
    }

    /**
     * @return array{page: StaticPage, meta: SeoMetaData|null, title: string, description: string, indexable: bool}
     */
    private function build(StaticPage $page, SeoDefaults $seo): array
    {
        $meta = $this->meta->forRoute($page->routeName());
        $defaults = $this->defaults->for($page);

        $title = match (true) {
            ($meta->title ?? '') !== '' => $seo->title($meta?->title),
            $defaults->titleIsComplete => $defaults->title,
            default => $seo->title($defaults->title),
        };
        $description = ($meta->description ?? '') !== '' ? (string) $meta?->description : $defaults->description;
        if ($description === '') {
            $description = $seo->defaultDescription;
        }

        return [
            'page' => $page,
            'meta' => $meta,
            'title' => trim($title),
            'description' => trim($description),
            'indexable' => $page->indexable() && Robots::parse($meta?->robots)->index,
        ];
    }

    /**
     * @param  array{page: StaticPage, meta: SeoMetaData|null, title: string, description: string, indexable: bool}  $row
     */
    private function data(array $row, bool $unique): StaticPageSeoData
    {
        $meta = $row['meta'];
        $titleLength = mb_strlen($row['title']);
        $descriptionLength = mb_strlen($row['description']);

        return new StaticPageSeoData(
            page: $row['page'],
            url: $this->url($row['page']),
            title: $row['title'],
            description: $row['description'],
            titleOverridden: ($meta->title ?? '') !== '',
            descriptionOverridden: ($meta->description ?? '') !== '',
            defaults: $this->defaults->for($row['page']),
            titleLengthOk: $titleLength >= SeoAuditor::TITLE_MIN && $titleLength <= SeoAuditor::TITLE_MAX,
            descriptionLengthOk: $descriptionLength >= SeoAuditor::DESCRIPTION_MIN && $descriptionLength <= SeoAuditor::DESCRIPTION_MAX,
            unique: $unique,
            ogImageSet: $meta?->ogMediaId !== null,
            indexable: $row['indexable'],
            hasOverrides: $meta !== null,
            focusKeyword: $meta?->focusKeyword,
        );
    }
}
