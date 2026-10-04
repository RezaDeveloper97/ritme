<?php

declare(strict_types=1);

namespace App\Domain\Search\Data;

/**
 * One search result for the view. `score`: 2 = a token matched the title, 1 = text only. `url` is absolute.
 */
final readonly class SearchHit
{
    public function __construct(
        public string $type,
        public string $typeLabel,
        public string $title,
        public string $url,
        public ?string $snippet = null,
        public int $score = 1,
    ) {}

    /**
     * @return array{type: string, typeLabel: string, title: string, url: string, snippet: string|null, score: int}
     */
    public function toArray(): array
    {
        return [
            'type' => $this->type, 'typeLabel' => $this->typeLabel, 'title' => $this->title, 'url' => $this->url,
            'snippet' => $this->snippet, 'score' => $this->score,
        ];
    }

    /**
     * @param  array<string, mixed>  $data
     */
    public static function fromArray(array $data): self
    {
        return new self(
            type: (string) $data['type'],
            typeLabel: (string) $data['typeLabel'],
            title: (string) $data['title'],
            url: (string) $data['url'],
            snippet: isset($data['snippet']) ? (string) $data['snippet'] : null,
            score: (int) ($data['score'] ?? 1),
        );
    }
}
