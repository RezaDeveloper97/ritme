<?php

declare(strict_types=1);

namespace App\Domain\Media\Support;

use App\Domain\Media\Enums\MediaFormat;
use Intervention\Image\Interfaces\EncodedImageInterface;
use Intervention\Image\Interfaces\ImageInterface;
use InvalidArgumentException;

/**
 * Encodes an image in one format with the configured quality, always stripping metadata. JPEGs are progressive.
 */
final class ImageEncoder
{
    /**
     * @param  array<string, int>  $quality  config('media.quality')
     */
    public function __construct(private readonly array $quality) {}

    public function encode(ImageInterface $image, MediaFormat $format, bool $original = false): EncodedImageInterface
    {
        $quality = $original ? ($this->quality['original'] ?? 86) : ($this->quality[$format->value] ?? 80);

        return match ($format) {
            MediaFormat::Avif => $image->toAvif(quality: $quality, strip: true),
            MediaFormat::Webp => $image->toWebp(quality: $quality, strip: true),
            MediaFormat::Jpg => $image->toJpeg(quality: $quality, progressive: true, strip: true),
            MediaFormat::Png => $image->toPng(),
            MediaFormat::Gif => $image->toGif(),
            MediaFormat::Svg => throw new InvalidArgumentException('SVG is not a raster encoding.'),
        };
    }
}
