<?php

declare(strict_types=1);

namespace App\Domain\Content\Stages;

/**
 * A glass card floating next to the hero phone. Copy is `hero.float.<key>` (title, text) of the stage's lang file.
 */
final readonly class FloatCardSpec
{
    public const TOP_START = 'top-start';

    public const BOTTOM_END = 'bottom-end';

    public function __construct(
        public string $key,
        public string $icon,
        public string $tint,
        public string $position,
    ) {}
}
