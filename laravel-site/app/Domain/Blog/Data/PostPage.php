<?php

declare(strict_types=1);

namespace App\Domain\Blog\Data;

/**
 * One page of post cards plus the totals a paginated list needs (out-of-range page → 404 in L4-02).
 */
final readonly class PostPage
{
    /**
     * @param  list<PostCardData>  $items
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
            'items' => array_map(static fn (PostCardData $card): array => $card->toArray(), $this->items),
            'total' => $this->total,
            'page' => $this->page,
            'perPage' => $this->perPage,
        ];
    }

    /**
     * @param  array<string, mixed>  $data
     */
    public static function fromArray(array $data): self
    {
        /** @var list<array<string, mixed>> $items */
        $items = (array) ($data['items'] ?? []);

        return new self(
            array_map(PostCardData::fromArray(...), $items),
            (int) $data['total'],
            (int) $data['page'],
            (int) $data['perPage'],
        );
    }
}
