<?php

declare(strict_types=1);

namespace App\Domain\Media\Support;

use Intervention\Image\Interfaces\ImageInterface;

/**
 * Cheap image facts computed on downscaled clones: dominant (average) colour, transparency, and the LQIP —
 * a tiny WebP data URI of at most `lqip.max_bytes` bytes used as a blurred placeholder.
 */
final class ImageAnalyzer
{
    public function dominantColor(ImageInterface $image): string
    {
        $sample = (clone $image)->resize(1, 1);

        return '#'.strtolower($sample->pickColor(0, 0)->toHex());
    }

    public function hasAlpha(ImageInterface $image): bool
    {
        $sample = (clone $image)->scaleDown(64, 64);
        $width = $sample->width();
        $height = $sample->height();

        for ($y = 0; $y < $height; $y++) {
            for ($x = 0; $x < $width; $x++) {
                if ($sample->pickColor($x, $y)->isTransparent()) {
                    return true;
                }
            }
        }

        return false;
    }

    public function lqip(ImageInterface $image, int $width = 16, int $quality = 40, int $maxBytes = 600): ?string
    {
        for ($w = $width, $q = $quality; $w >= 4; $w = intdiv($w * 3, 4), $q = max(10, $q - 10)) {
            $encoded = (clone $image)->scaleDown($w, $w)->toWebp($q);

            if ($encoded->size() <= $maxBytes) {
                return $encoded->toDataUri();
            }
        }

        return null;
    }
}
