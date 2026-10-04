<?php

declare(strict_types=1);

namespace App\Domain\Seo\Schema\Nodes;

use App\Domain\Seo\Schema\Data\BreadcrumbItem;
use App\Domain\Seo\Schema\Node;
use App\Domain\Seo\Schema\SchemaIds;

/**
 * BreadcrumbList (`#breadcrumb`) from the same items as `x-ui.breadcrumbs`. Every item except the last needs a URL;
 * the last one (the current page) gets the page URL when it has none.
 */
final class BreadcrumbListNode
{
    /**
     * @param  list<BreadcrumbItem>  $items
     * @return array<string, mixed>
     */
    public static function make(string $pageUrl, array $items): array
    {
        $elements = [];
        $last = count($items) - 1;
        foreach ($items as $i => $item) {
            $url = $item->url ?? ($i === $last ? $pageUrl : null);
            if ($url === null) {
                continue; // an intermediate step without a page cannot be a ListItem
            }
            $elements[] = [
                '@type' => 'ListItem',
                'position' => count($elements) + 1,
                'name' => $item->name,
                'item' => $url,
            ];
        }

        return Node::clean([
            '@type' => 'BreadcrumbList',
            '@id' => SchemaIds::breadcrumb($pageUrl),
            'itemListElement' => $elements,
        ]);
    }
}
