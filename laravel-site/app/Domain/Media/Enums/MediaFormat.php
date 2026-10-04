<?php

declare(strict_types=1);

namespace App\Domain\Media\Enums;

/**
 * File formats the pipeline reads or writes. The value is the file extension.
 */
enum MediaFormat: string
{
    case Avif = 'avif';
    case Webp = 'webp';
    case Jpg = 'jpg';
    case Png = 'png';
    case Gif = 'gif';
    case Svg = 'svg';

    public static function fromMime(string $mime): ?self
    {
        return match (strtolower($mime)) {
            'image/avif' => self::Avif,
            'image/webp' => self::Webp,
            'image/jpeg', 'image/jpg', 'image/pjpeg' => self::Jpg,
            'image/png' => self::Png,
            'image/gif' => self::Gif,
            'image/svg+xml' => self::Svg,
            default => null,
        };
    }

    public function mime(): string
    {
        return match ($this) {
            self::Avif => 'image/avif',
            self::Webp => 'image/webp',
            self::Jpg => 'image/jpeg',
            self::Png => 'image/png',
            self::Gif => 'image/gif',
            self::Svg => 'image/svg+xml',
        };
    }

    public function isRaster(): bool
    {
        return $this !== self::Svg;
    }
}
