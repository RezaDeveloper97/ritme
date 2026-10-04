<?php

declare(strict_types=1);

namespace App\Domain\Search\Data;

/**
 * One page of search results. `query` is what the visitor typed (trimmed), `normalized` what was searched for.
 */
final readonly class SearchResults
{
    /**
     * @param  list<SearchHit>  $hits
     */
    public function __construct(
        public string $query,
        public string $normalized,
        public array $hits,
        public int $total,
        public int $page,
        public int $perPage,
    ) {}

    public static function empty(string $query = ''): self
    {
        return new self($query, '', [], 0, 1, 1);
    }

    public function searched(): bool
    {
        return $this->normalized !== '';
    }

    public function lastPage(): int
    {
        return max(1, (int) ceil($this->total / max(1, $this->perPage)));
    }
}
