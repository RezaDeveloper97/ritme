<?php

declare(strict_types=1);

namespace App\Domain\Seo\Schema\Nodes;

use App\Domain\Seo\Data\SeoImage;
use App\Domain\Seo\Schema\Data\PersonData;
use App\Domain\Seo\Schema\Enums\WebPageType;
use App\Domain\Seo\Schema\Node;
use App\Domain\Seo\Schema\SchemaIds;

/**
 * The page (`#webpage`): WebPage / AboutPage / ContactPage / CollectionPage …, linked to the website, its breadcrumb
 * and primary image. `reviewedBy` + `lastReviewed` are the YMYL hooks (medically reviewed health content).
 */
final class WebPageNode
{
    /**
     * @return array<string, mixed>
     */
    public static function make(
        string $url,
        string $siteUrl,
        string $name,
        ?string $description = null,
        WebPageType $type = WebPageType::WebPage,
        ?SeoImage $image = null,
        bool $hasBreadcrumb = false,
        ?string $datePublished = null,
        ?string $dateModified = null,
        ?PersonData $reviewedBy = null,
        ?string $lastReviewed = null,
    ): array {
        return Node::clean([
            '@type' => $type->value,
            '@id' => SchemaIds::webPage($url),
            'url' => $url,
            'name' => $name,
            'description' => $description,
            'inLanguage' => 'fa-IR',
            'isPartOf' => Node::ref(SchemaIds::website($siteUrl)),
            'about' => rtrim($url, '/') === rtrim($siteUrl, '/') ? Node::ref(SchemaIds::organization($siteUrl)) : null,
            'primaryImageOfPage' => $image !== null ? Node::image($image, SchemaIds::primaryImage($url)) : null,
            'breadcrumb' => $hasBreadcrumb ? Node::ref(SchemaIds::breadcrumb($url)) : null,
            'datePublished' => $datePublished,
            'dateModified' => $dateModified,
            'reviewedBy' => $reviewedBy !== null ? PersonNode::make($reviewedBy, $siteUrl) : null,
            'lastReviewed' => $lastReviewed,
        ]);
    }
}
