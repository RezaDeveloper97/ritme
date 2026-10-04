<?php

declare(strict_types=1);

namespace App\Domain\Blog\Rendering;

use App\Domain\Blog\Data\PostCardData;
use App\Domain\Blog\Data\PostData;
use App\Domain\Media\Data\MediaData;
use App\Domain\Seo\Schema\Data\BreadcrumbItem;
use App\Domain\Settings\Data\AppLinksSettings;

/**
 * Everything `pages/blog/show.blade.php` renders for one article (DTOs only; no queries in the view).
 */
final readonly class ArticlePage
{
    /**
     * @param  list<BreadcrumbItem>  $breadcrumbs  خانه › مجله › دسته › عنوان (also the BreadcrumbList JSON-LD)
     * @param  list<PostCardData>  $related
     * @param  list<array{key: string, label: string, href: string}>  $share
     */
    public function __construct(
        public PostData $post,
        public ArticleBody $body,
        public ?MediaData $cover,
        public ?MediaData $mobileCover,
        public string $coverAlt,
        public string $url,
        public array $breadcrumbs,
        public array $related,
        public ?PostCardData $previous,
        public ?PostCardData $next,
        public array $share,
        public ?string $authorUrl,
        public ?string $reviewerUrl,
        public ?string $categoryUrl,
        public AppLinksSettings $appLinks,
        public string $downloadUrl,
    ) {}
}
