<?php

declare(strict_types=1);

namespace App\Domain\Content\Stages\Data;

final readonly class StageFloatCardData
{
    /**
     * @param  string  $position  FloatCardSpec::TOP_START | FloatCardSpec::BOTTOM_END
     */
    public function __construct(
        public string $icon,
        public string $tint,
        public string $title,
        public string $text,
        public string $position,
    ) {}
}
