<?php

declare(strict_types=1);

namespace App\Domain\Directory\Data;

/**
 * One page of place cards plus the totals a paginated list needs.
 */
final readonly class PlacePage
{
    /**
     * @param  list<PlaceCardData>  $items
     */
    public function __construct(
        public array $items,
        public int $total,
        public int $page,
        public int $perPage,
    ) {}

    public function lastPage(): int
    {
        return max(1, (int) ceil($this->total / max(1, $this->perPage)));
    }

    public function isOutOfRange(): bool
    {
        return $this->page < 1 || $this->page > $this->lastPage();
    }

    public function isEmpty(): bool
    {
        return $this->items === [];
    }

    /**
     * @return array<string, mixed>
     */
    public function toArray(): array
    {
        return [
            'items' => array_map(static fn (PlaceCardData $card): array => $card->toArray(), $this->items),
            'total' => $this->total, 'page' => $this->page, 'perPage' => $this->perPage,
        ];
    }

    /**
     * @param  array<string, mixed>  $data
     */
    public static function fromArray(array $data): self
    {
        /** @var list<array<string, mixed>> $items */
        $items = (array) ($data['items'] ?? []);

        return new self(array_map(PlaceCardData::fromArray(...), $items), (int) $data['total'], (int) $data['page'], (int) $data['perPage']);
    }
}
