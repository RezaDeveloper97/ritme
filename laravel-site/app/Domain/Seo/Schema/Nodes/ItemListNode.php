<?php

declare(strict_types=1);

namespace App\Domain\Seo\Schema\Nodes;

use App\Domain\Seo\Schema\Data\ListEntry;
use App\Domain\Seo\Schema\Node;
use App\Domain\Seo\Schema\SchemaIds;

/**
 * ItemList (`#itemlist`) for listing pages, in the "summary page" form Google uses for carousels: ListItem with
 * position + url (detail pages carry their own full markup).
 */
final class ItemListNode
{
    /**
     * @param  list<ListEntry>  $entries
     * @return array<string, mixed>
     */
    public static function make(string $pageUrl, array $entries, ?string $name = null): array
    {
        $elements = [];
        foreach ($entries as $i => $entry) {
            $elements[] = [
                '@type' => 'ListItem',
                'position' => $i + 1,
                'url' => $entry->url,
                'name' => $entry->name,
                'image' => $entry->imageUrl,
            ];
        }

        return Node::clean([
            '@type' => 'ItemList',
            '@id' => SchemaIds::itemList($pageUrl),
            'name' => $name,
            'numberOfItems' => count($elements),
            'itemListElement' => $elements,
            'mainEntityOfPage' => Node::ref(SchemaIds::webPage($pageUrl)),
        ]);
    }
}
