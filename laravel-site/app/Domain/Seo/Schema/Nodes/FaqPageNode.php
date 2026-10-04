<?php

declare(strict_types=1);

namespace App\Domain\Seo\Schema\Nodes;

use App\Domain\Seo\Schema\Data\FaqItem;
use App\Domain\Seo\Schema\Node;
use App\Domain\Seo\Schema\SchemaIds;

/**
 * FAQPage (`#faq`) with Question/Answer pairs that are visible on the page. Only for site-authored FAQs (not user
 * Q&A). Since 2023 Google shows FAQ rich results only for well-known authoritative health/government sites; the
 * markup stays useful for other engines and AI answers.
 */
final class FaqPageNode
{
    /**
     * @param  list<FaqItem>  $items
     * @return array<string, mixed>
     */
    public static function make(string $pageUrl, array $items): array
    {
        return Node::clean([
            '@type' => 'FAQPage',
            '@id' => SchemaIds::faq($pageUrl),
            'url' => $pageUrl,
            'inLanguage' => 'fa-IR',
            'isPartOf' => Node::ref(SchemaIds::webPage($pageUrl)),
            'mainEntity' => array_map(static fn (FaqItem $item): array => [
                '@type' => 'Question',
                'name' => trim($item->question),
                'acceptedAnswer' => ['@type' => 'Answer', 'text' => trim($item->answer)],
            ], $items),
        ]);
    }
}
