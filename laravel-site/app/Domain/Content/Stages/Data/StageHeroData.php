<?php

declare(strict_types=1);

namespace App\Domain\Content\Stages\Data;

/**
 * Hero of a stage page: eyebrow pill, h1 (`highlight` in lilac + `title`), lead, the two CTAs and the phone.
 */
final readonly class StageHeroData
{
    /**
     * @param  array<string, string>  $screenData  copy for the mock screen partial
     * @param  list<StageFloatCardData>  $floatCards
     */
    public function __construct(
        public string $eyebrow,
        public string $eyebrowIcon,
        public string $highlight,
        public string $title,
        public string $lead,
        public string $downloadLabel,
        public string $howLabel,
        public string $screen,
        public array $screenData,
        public array $floatCards,
    ) {}
}
