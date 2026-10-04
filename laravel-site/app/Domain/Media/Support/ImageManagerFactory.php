<?php

declare(strict_types=1);

namespace App\Domain\Media\Support;

use Intervention\Image\ImageManager;
use InvalidArgumentException;

/**
 * Builds the Intervention manager: Imagick when the extension is loaded (driver `auto`), otherwise GD — the usual
 * cPanel setup. Decoding auto-orients by EXIF; encoders strip metadata (GD never writes it, Imagick is told to).
 */
final class ImageManagerFactory
{
    public static function make(string $driver = 'auto'): ImageManager
    {
        $driver = $driver === 'auto'
            ? (self::imagickAvailable() ? 'imagick' : 'gd')
            : $driver;

        return match ($driver) {
            'imagick' => ImageManager::imagick(autoOrientation: true, decodeAnimation: true, strip: true),
            'gd' => ImageManager::gd(autoOrientation: true, decodeAnimation: true, strip: true),
            default => throw new InvalidArgumentException("Unknown media driver [{$driver}]."),
        };
    }

    public static function imagickAvailable(): bool
    {
        return extension_loaded('imagick') && class_exists(\Imagick::class);
    }
}
