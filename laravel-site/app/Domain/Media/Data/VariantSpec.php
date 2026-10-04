<?php

declare(strict_types=1);

namespace App\Domain\Media\Data;

use App\Domain\Media\Enums\MediaFormat;

/**
 * One variant to generate: its name (e.g. `mobile_480`, `thumb`), the preset it belongs to, the output size, whether
 * it is a focal crop (otherwise an aspect-keeping scale) and the formats to encode.
 */
final readonly class VariantSpec
{
    /**
     * @param  list<MediaFormat>  $formats
     */
    public function __construct(
        public string $name,
        public string $preset,
        public int $width,
        public int $height,
        public bool $crop,
        public array $formats,
    ) {}
}
