<?php

declare(strict_types=1);

namespace App\Domain\Content\Stages;

use App\Domain\Content\Enums\SitePhoto;

/**
 * Structure of the Nov 2026 stage-page design (`pages/stages/redesign.blade.php`): the two photos, the stage's accent
 * and band colour, and the keys of the six feature cards. Copy is `redesign.*` of the stage's lang file.
 */
final readonly class RedesignSpec
{
    /**
     * @param  string  $accent  `cycle`, `ttc` or `pregnancy`: the view maps it to literal colour utilities (the class
     *                          scanner does not read app/Domain)
     * @param  list<string>  $featureKeys  keys of `redesign.features.items`
     */
    public function __construct(
        public SitePhoto $heroPhoto,
        public SitePhoto $featurePhoto,
        public string $accent,
        public array $featureKeys,
    ) {}
}
