<?php

declare(strict_types=1);

namespace App\Domain\Seo\Data;

use App\Domain\Content\Enums\StaticPage;

/**
 * One row of the admin static-pages SEO list: the effective head of the page (admin override or lang default) and
 * its health checks. Built by App\Domain\Seo\Actions\ListStaticPageSeo.
 */
final readonly class StaticPageSeoData
{
    public function __construct(
        public StaticPage $page,
        public string $url,
        public string $title,
        public string $description,
        public bool $titleOverridden,
        public bool $descriptionOverridden,
        public StaticPageDefaults $defaults,
        public bool $titleLengthOk,
        public bool $descriptionLengthOk,
        public bool $unique,
        public bool $ogImageSet,
        public bool $indexable,
        public bool $hasOverrides,
        public ?string $focusKeyword,
    ) {}

    /**
     * Flat array for Filament's custom-data table (key = route name).
     *
     * @return array<string, mixed>
     */
    public function toArray(): array
    {
        return [
            'key' => $this->page->routeName(),
            'route' => $this->page->routeName(),
            'label' => $this->page->label(),
            'path' => $this->page->path(),
            'url' => $this->url,
            'title' => $this->title,
            'description' => $this->description,
            'title_overridden' => $this->titleOverridden,
            'description_overridden' => $this->descriptionOverridden,
            'title_length' => mb_strlen($this->title),
            'description_length' => mb_strlen($this->description),
            'title_length_ok' => $this->titleLengthOk,
            'description_length_ok' => $this->descriptionLengthOk,
            'unique' => $this->unique,
            'og_image_set' => $this->ogImageSet,
            'indexable' => $this->indexable,
            'has_overrides' => $this->hasOverrides,
            'focus_keyword' => $this->focusKeyword,
        ];
    }
}
