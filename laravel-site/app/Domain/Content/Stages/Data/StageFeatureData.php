<?php

declare(strict_types=1);

namespace App\Domain\Content\Stages\Data;

/**
 * One feature split: text column (eyebrow, h2, text, check list) + phone mock-up. `mediaFirst` alternates the
 * column order, `surface` the band colour (AUDIT §2.4).
 */
final readonly class StageFeatureData
{
    /**
     * @param  list<string>  $points
     * @param  array<string, mixed>  $screenData  copy for the mock screen partial
     */
    public function __construct(
        public string $key,
        public ?string $anchor,
        public string $eyebrow,
        public string $title,
        public string $text,
        public array $points,
        public string $partial,
        public string $screen,
        public array $screenData,
        public bool $mediaFirst,
        public bool $surface,
    ) {}

    public function headingId(): string
    {
        return 'feature-'.$this->key.'-title';
    }
}
