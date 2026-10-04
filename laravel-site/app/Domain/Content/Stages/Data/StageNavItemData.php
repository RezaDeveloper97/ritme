<?php

declare(strict_types=1);

namespace App\Domain\Content\Stages\Data;

final readonly class StageNavItemData
{
    /**
     * @param  string  $color  stage colour key (cycle, ttc, …)
     */
    public function __construct(
        public string $label,
        public string $url,
        public string $icon,
        public string $color,
        public bool $active,
    ) {}
}
