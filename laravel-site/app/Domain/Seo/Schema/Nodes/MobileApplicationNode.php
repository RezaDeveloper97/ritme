<?php

declare(strict_types=1);

namespace App\Domain\Seo\Schema\Nodes;

use App\Domain\Seo\Schema\Data\AggregateRatingData;
use App\Domain\Seo\Schema\Data\OfferData;
use App\Domain\Seo\Schema\Node;
use App\Domain\Seo\Schema\SchemaIds;
use App\Domain\Settings\Data\SiteSettings;

/**
 * The Ritme app (`#app`): store links from settings, free offer (price 0), HealthApplication. A rating only when real
 * store / on-site ratings are passed — never invented. Google shows the app rich result only with a rating or review,
 * so without one the node is still valid, just not rich-result eligible.
 */
final class MobileApplicationNode
{
    /**
     * @return array<string, mixed>
     */
    public static function make(
        SiteSettings $settings,
        string $siteUrl,
        ?string $description = null,
        ?string $imageUrl = null,
        ?AggregateRatingData $rating = null,
    ): array {
        $links = $settings->appLinks;
        $stores = Node::strings([$links->bazaar, $links->myket, $links->googlePlay, $links->appStore]);
        $systems = array_filter([
            'Android' => $links->bazaar !== null || $links->myket !== null || $links->googlePlay !== null,
            'iOS' => $links->appStore !== null,
            'Web' => $links->webApp !== null,
        ]);

        return Node::clean([
            '@type' => 'MobileApplication',
            '@id' => SchemaIds::app($siteUrl),
            'name' => $settings->general->siteName,
            'alternateName' => $settings->general->alternateName,
            'description' => $description ?? $settings->pwa->description,
            'url' => SchemaIds::root($siteUrl),
            'image' => $imageUrl,
            'applicationCategory' => 'HealthApplication',
            'operatingSystem' => $systems !== [] ? implode(', ', array_keys($systems)) : 'Android, iOS',
            'inLanguage' => 'fa-IR',
            'installUrl' => $stores[0] ?? $links->webApp,
            'downloadUrl' => $stores,
            'sameAs' => $stores,
            'offers' => (new OfferData(price: 0))->toNode(),
            'publisher' => Node::ref(SchemaIds::organization($siteUrl)),
            'aggregateRating' => $rating?->toNode(),
        ]);
    }
}
