<?php

declare(strict_types=1);

namespace App\Domain\Seo\Analysis;

/**
 * What kind of page is analysed: sets the recommended body length and which content checks apply.
 */
enum ContentType: string
{
    case Post = 'post';
    case Product = 'product';
    case Place = 'place';
    case Page = 'page';
    case Archive = 'archive';

    public function label(): string
    {
        return match ($this) {
            self::Post => 'مقاله',
            self::Product => 'محصول',
            self::Place => 'مرکز',
            self::Page => 'صفحه',
            self::Archive => 'صفحه فهرست',
        };
    }

    /**
     * Recommended minimum words of body text (cornerstone content: the stricter value).
     */
    public function minWords(bool $cornerstone = false): int
    {
        $words = match ($this) {
            self::Post => 600,
            self::Product => 200,
            self::Place => 150,
            self::Page => 300,
            self::Archive => 50,
        };

        return $cornerstone ? max($words * 2, 900) : $words;
    }

    /**
     * Recommended minimum internal links in the body.
     */
    public function minInternalLinks(bool $cornerstone = false): int
    {
        $links = match ($this) {
            self::Post, self::Page => 2,
            self::Product, self::Place => 1,
            self::Archive => 0,
        };

        return $cornerstone ? $links + 2 : $links;
    }
}
