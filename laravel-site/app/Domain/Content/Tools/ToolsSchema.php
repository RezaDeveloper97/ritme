<?php

declare(strict_types=1);

namespace App\Domain\Content\Tools;

use App\Domain\Content\Tools\Enums\CalculatorKind;
use App\Domain\Seo\Schema\Data\OfferData;
use App\Domain\Seo\Schema\Node;
use App\Domain\Seo\Schema\SchemaIds;

/**
 * JSON-LD for the two calculators: one `WebApplication` each (`{page}#due-date-app`, `{page}#fertility-app`), free
 * (Offer price 0 IRR), browser-based, part of the page's WebPage node. No aggregateRating — there are no real ratings
 * and we never invent them, so the nodes are valid but not rich-result eligible (docs/SEO.md).
 */
final class ToolsSchema
{
    /**
     * @param  array<string, array{name: string, description: string}>  $copy  per kind value
     * @return list<array<string, mixed>>
     */
    public static function nodes(string $pageUrl, string $siteUrl, array $copy): array
    {
        $nodes = [];
        foreach (CalculatorKind::cases() as $kind) {
            $anchor = $pageUrl.'#'.$kind->anchor();
            $nodes[] = Node::clean([
                '@type' => 'WebApplication',
                '@id' => $anchor.'-app',
                'name' => $copy[$kind->value]['name'] ?? null,
                'description' => $copy[$kind->value]['description'] ?? null,
                'url' => $anchor,
                'applicationCategory' => 'HealthApplication',
                'operatingSystem' => 'Any',
                'browserRequirements' => 'Requires a web browser; works without JavaScript.',
                'inLanguage' => 'fa-IR',
                'isAccessibleForFree' => true,
                'offers' => (new OfferData(price: 0))->toNode(),
                'publisher' => Node::ref(SchemaIds::organization($siteUrl)),
                'isPartOf' => Node::ref(SchemaIds::webPage($pageUrl)),
            ]);
        }

        return $nodes;
    }
}
