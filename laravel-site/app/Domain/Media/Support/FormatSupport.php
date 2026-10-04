<?php

declare(strict_types=1);

namespace App\Domain\Media\Support;

use App\Domain\Media\Enums\MediaFormat;
use Intervention\Image\Drivers\Imagick\Driver as ImagickDriver;
use Intervention\Image\ImageManager;

/**
 * Which formats the active driver can encode, and the resolution of the configured format list (`fallback` = jpg
 * for photos, png when the image has transparency). AVIF is dropped silently when unsupported (e.g. old GD).
 */
final class FormatSupport
{
    /** @var array<string, bool> */
    private array $cache = [];

    public function __construct(private readonly ImageManager $images) {}

    public function canEncode(MediaFormat $format): bool
    {
        return $this->cache[$format->value] ??= $this->detect($format);
    }

    /**
     * @param  list<string>  $configured  e.g. ['avif', 'webp', 'fallback']
     * @return list<MediaFormat>
     */
    public function resolve(array $configured, bool $hasAlpha): array
    {
        $formats = [];
        foreach ($configured as $name) {
            $format = $name === 'fallback'
                ? ($hasAlpha ? MediaFormat::Png : MediaFormat::Jpg)
                : MediaFormat::tryFrom($name);

            if ($format !== null && $format->isRaster() && $format !== MediaFormat::Gif && $this->canEncode($format)) {
                $formats[$format->value] = $format;
            }
        }

        return array_values($formats);
    }

    private function detect(MediaFormat $format): bool
    {
        if ($this->images->driver() instanceof ImagickDriver) {
            return \Imagick::queryFormats(strtoupper($format->value === 'jpg' ? 'jpeg' : $format->value)) !== [];
        }

        $info = gd_info();

        return match ($format) {
            MediaFormat::Avif => function_exists('imageavif') && (bool) ($info['AVIF Support'] ?? false),
            MediaFormat::Webp => function_exists('imagewebp') && (bool) ($info['WebP Support'] ?? false),
            MediaFormat::Jpg => (bool) ($info['JPEG Support'] ?? false),
            MediaFormat::Png => (bool) ($info['PNG Support'] ?? false),
            MediaFormat::Gif => (bool) ($info['GIF Create Support'] ?? false),
            MediaFormat::Svg => false,
        };
    }
}
