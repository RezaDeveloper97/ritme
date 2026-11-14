<?php

declare(strict_types=1);

namespace App\Domain\Content\Stages\Data;

use App\Domain\Media\Data\MediaData;

/**
 * What `pages/stages/redesign.blade.php` renders on top of StagePageData: hero copy and chips, photos, six feature
 * cards and the shared block headings.
 */
final readonly class StageRedesignData
{
    /**
     * @param  list<array{title: string, text: string}>  $chips  top (overhanging end) chip first, bottom chip second
     * @param  list<array{title: string, text: string}>  $features
     * @param  array<string, string>  $copy  shared headings and labels
     */
    public function __construct(
        public string $accent,
        public string $eyebrow,
        public string $title,
        public string $lead,
        public array $chips,
        public ?MediaData $heroPhoto,
        public string $heroAlt,
        public ?MediaData $featurePhoto,
        public string $featureAlt,
        public string $featuresTitle,
        public array $features,
        public array $copy,
    ) {}
}
