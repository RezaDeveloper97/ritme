<?php

declare(strict_types=1);

namespace App\Domain\Seo\Schema\Nodes;

use App\Domain\Seo\Schema\Node;
use App\Domain\Seo\Schema\SchemaIds;
use App\Domain\Settings\Data\SiteSettings;

/**
 * The site itself. `$searchUrlTemplate` (e.g. `https://ritme.ir/search?q={search_term_string}`) adds the
 * SearchAction once site search ships (L4-04); it must contain `{search_term_string}`.
 */
final class WebSiteNode
{
    /**
     * @return array<string, mixed>
     */
    public static function make(SiteSettings $settings, string $siteUrl, ?string $searchUrlTemplate = null): array
    {
        $search = $searchUrlTemplate !== null && str_contains($searchUrlTemplate, '{search_term_string}');

        return Node::clean([
            '@type' => 'WebSite',
            '@id' => SchemaIds::website($siteUrl),
            'url' => SchemaIds::root($siteUrl),
            'name' => $settings->general->siteName,
            'alternateName' => $settings->general->alternateName,
            'description' => $settings->seo->defaultDescription,
            'inLanguage' => 'fa-IR',
            'publisher' => Node::ref(SchemaIds::organization($siteUrl)),
            'potentialAction' => $search ? [
                '@type' => 'SearchAction',
                'target' => ['@type' => 'EntryPoint', 'urlTemplate' => $searchUrlTemplate],
                'query-input' => 'required name=search_term_string',
            ] : null,
        ]);
    }
}
