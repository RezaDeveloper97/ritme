<?php

declare(strict_types=1);

namespace App\Domain\Seo\Data;

/**
 * Everything `<x-seo.head/>` renders, fully resolved. Maps are ordered `name => content`.
 */
final readonly class SeoHead
{
    /**
     * @param  array<string, string>  $openGraph  property => content
     * @param  array<string, string>  $twitter  name => content
     * @param  array<string, string>  $verification  meta name => token
     * @param  list<array{href: string, title: string}>  $feeds  RSS alternates
     * @param  array<string, mixed>  $schemaOverrides  for the JSON-LD graph (L1-04)
     */
    public function __construct(
        public string $title,
        public string $description,
        public string $canonical,
        public string $robots,
        public bool $indexable,
        public array $openGraph,
        public array $twitter,
        public array $verification,
        public array $feeds,
        public ?SeoImage $image,
        public array $schemaOverrides,
    ) {}
}
