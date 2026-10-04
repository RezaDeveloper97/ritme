<?php

declare(strict_types=1);

namespace App\Domain\Seo\Schema\Nodes;

use App\Domain\Seo\Schema\Data\BlogPostingData;
use App\Domain\Seo\Schema\Data\PersonData;
use App\Domain\Seo\Schema\Node;
use App\Domain\Seo\Schema\SchemaIds;

/**
 * BlogPosting (`#article`) with author Person(s), the organization as publisher and the page as mainEntityOfPage.
 * For YMYL articles pass the medical reviewer to the page (SchemaGraph::reviewedBy()) — `reviewedBy` is a WebPage
 * property.
 */
final class BlogPostingNode
{
    /**
     * @return array<string, mixed>
     */
    public static function make(BlogPostingData $post, string $siteUrl): array
    {
        return Node::clean([
            '@type' => 'BlogPosting',
            '@id' => SchemaIds::article($post->url),
            'headline' => mb_substr(trim($post->headline), 0, 110),
            'description' => $post->description,
            'url' => $post->url,
            'mainEntityOfPage' => Node::ref(SchemaIds::webPage($post->url)),
            'isPartOf' => Node::ref(SchemaIds::webPage($post->url)),
            'image' => $post->image !== null ? Node::image($post->image) : null,
            'datePublished' => $post->datePublished,
            'dateModified' => $post->dateModified ?? $post->datePublished,
            'author' => array_map(
                static fn (PersonData $author): array => PersonNode::make($author, $siteUrl),
                $post->authors,
            ),
            'publisher' => Node::ref(SchemaIds::organization($siteUrl)),
            'articleSection' => $post->section,
            'keywords' => Node::strings($post->keywords),
            'wordCount' => $post->wordCount,
            'inLanguage' => 'fa-IR',
        ]);
    }
}
