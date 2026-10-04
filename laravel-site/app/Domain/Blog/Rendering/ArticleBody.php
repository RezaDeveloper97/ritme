<?php

declare(strict_types=1);

namespace App\Domain\Blog\Rendering;

use App\Support\Html\HeadingAnchors;

/**
 * An article body ready to print unescaped: re-sanitised, body images rendered through <x-picture>, tables
 * scroll-wrapped, h2/h3 with ids. `outline` is the TOC read from the same HTML (so links always match).
 *
 * @phpstan-import-type Heading from HeadingAnchors
 */
final readonly class ArticleBody
{
    /**
     * @param  list<Heading>  $outline
     */
    public function __construct(
        public string $html,
        public array $outline,
        public int $images = 0,
    ) {}

    /**
     * @return array{html: string, outline: list<Heading>, images: int}
     */
    public function toArray(): array
    {
        return ['html' => $this->html, 'outline' => $this->outline, 'images' => $this->images];
    }

    /**
     * @param  array<string, mixed>  $data
     */
    public static function fromArray(array $data): self
    {
        $outline = [];
        foreach ((array) ($data['outline'] ?? []) as $heading) {
            if (is_array($heading)) {
                $outline[] = [
                    'level' => (int) ($heading['level'] ?? 2),
                    'id' => (string) ($heading['id'] ?? ''),
                    'text' => (string) ($heading['text'] ?? ''),
                ];
            }
        }

        return new self((string) ($data['html'] ?? ''), $outline, (int) ($data['images'] ?? 0));
    }
}
