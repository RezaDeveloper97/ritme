<?php

declare(strict_types=1);

namespace App\Domain\Directory\Data;

final readonly class ReviewPage
{
    /**
     * @param  list<ReviewData>  $items
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

    /**
     * @return array<string, mixed>
     */
    public function toArray(): array
    {
        return [
            'items' => array_map(static fn (ReviewData $r): array => $r->toArray(), $this->items),
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

        return new self(array_map(ReviewData::fromArray(...), $items), (int) $data['total'], (int) $data['page'], (int) $data['perPage']);
    }
}
