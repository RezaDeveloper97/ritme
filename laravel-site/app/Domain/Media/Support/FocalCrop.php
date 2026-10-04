<?php

declare(strict_types=1);

namespace App\Domain\Media\Support;

/**
 * Crop box for a target aspect ratio centred on a focal point (0..1, clamped to the image), and the output size —
 * never larger than the crop itself (no upscaling: a small source yields a smaller image of the same ratio).
 */
final class FocalCrop
{
    /**
     * @return array{x: int, y: int, width: int, height: int, outWidth: int, outHeight: int}
     */
    public static function box(int $width, int $height, int $targetWidth, int $targetHeight, float $focalX = 0.5, float $focalY = 0.5): array
    {
        $ratio = $targetWidth / $targetHeight;

        if ($width / $height > $ratio) {
            $cropHeight = $height;
            $cropWidth = max(1, (int) round($height * $ratio));
        } else {
            $cropWidth = $width;
            $cropHeight = max(1, (int) round($width / $ratio));
        }

        $x = self::clamp((int) round(self::unit($focalX) * $width - $cropWidth / 2), 0, $width - $cropWidth);
        $y = self::clamp((int) round(self::unit($focalY) * $height - $cropHeight / 2), 0, $height - $cropHeight);

        [$outWidth, $outHeight] = $cropWidth > $targetWidth
            ? [$targetWidth, $targetHeight]
            : [$cropWidth, $cropHeight];

        return ['x' => $x, 'y' => $y, 'width' => $cropWidth, 'height' => $cropHeight, 'outWidth' => $outWidth, 'outHeight' => $outHeight];
    }

    private static function unit(float $value): float
    {
        return max(0.0, min(1.0, $value));
    }

    private static function clamp(int $value, int $min, int $max): int
    {
        return max($min, min($max, $value));
    }
}
