<?php

declare(strict_types=1);

namespace App\Domain\Content\Stages\Data;

/**
 * Hero of a stage page: eyebrow pill, h1 (`titleBefore` + `highlight` in lilac + `titleAfter`, spacing included),
 * lead, the two CTAs and the phone.
 */
final readonly class StageHeroData
{
    /**
     * @param  array<string, mixed>  $screenData  copy for the mock screen partial
     * @param  list<StageFloatCardData>  $floatCards
     */
    public function __construct(
        public string $eyebrow,
        public string $eyebrowIcon,
        public string $titleBefore,
        public string $highlight,
        public string $titleAfter,
        public string $lead,
        public string $downloadLabel,
        public string $howLabel,
        public string $screen,
        public array $screenData,
        public array $floatCards,
    ) {}
}
