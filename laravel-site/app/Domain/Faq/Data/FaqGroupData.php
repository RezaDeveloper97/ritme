<?php

declare(strict_types=1);

namespace App\Domain\Faq\Data;

use App\Domain\Seo\Schema\Data\FaqItem;

/**
 * A FAQ group with its published items in order. `slug` doubles as the in-page anchor on /faq.
 */
final readonly class FaqGroupData
{
    /**
     * @param  list<FaqItemData>  $items
     */
    public function __construct(
        public int $id,
        public string $slug,
        public string $title,
        public bool $listed,
        public array $items,
    ) {}

    public function isEmpty(): bool
    {
        return $this->items === [];
    }

    /**
     * @return list<FaqItem>
     */
    public function schemaItems(): array
    {
        return array_map(static fn (FaqItemData $item): FaqItem => $item->toSchema(), $this->items);
    }

    /**
     * @return array<string, mixed>
     */
    public function toArray(): array
    {
        return [
            'id' => $this->id, 'slug' => $this->slug, 'title' => $this->title, 'listed' => $this->listed,
            'items' => array_map(static fn (FaqItemData $item): array => $item->toArray(), $this->items),
        ];
    }

    /**
     * @param  array<string, mixed>  $data
     */
    public static function fromArray(array $data): self
    {
        $items = is_array($data['items'] ?? null) ? $data['items'] : [];

        return new self(
            id: (int) $data['id'],
            slug: (string) $data['slug'],
            title: (string) $data['title'],
            listed: (bool) $data['listed'],
            items: array_values(array_map(static fn (array $item): FaqItemData => FaqItemData::fromArray($item), array_filter($items, 'is_array'))),
        );
    }
}
