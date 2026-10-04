<?php

declare(strict_types=1);

namespace App\Domain\Media\Support;

use App\Domain\Media\Data\MediaData;
use App\Domain\Media\Enums\MediaFormat;
use App\Domain\Media\Models\Media;
use Illuminate\Contracts\Filesystem\Factory as Filesystems;

/**
 * Model → MediaData, resolving every path to a URL on the media's disk.
 */
final class MediaDataFactory
{
    public function __construct(private readonly Filesystems $filesystems) {}

    public function make(Media $media): MediaData
    {
        $disk = $this->filesystems->disk($media->disk);

        $variants = [];
        foreach ($media->variants ?? [] as $name => $formats) {
            foreach ($formats as $format => $file) {
                $variants[$name][$format] = ['w' => (int) $file['w'], 'h' => (int) $file['h'], 'url' => $disk->url($file['path']), 'size' => (int) $file['size']];
            }
        }

        return new MediaData(
            id: $media->id,
            url: $disk->url($media->path()),
            mime: $media->mime,
            extension: MediaFormat::fromMime($media->mime)->value ?? pathinfo($media->filename, PATHINFO_EXTENSION),
            width: $media->width,
            height: $media->height,
            alt: $media->alt,
            title: $media->title,
            caption: $media->caption,
            focalX: $media->focal_x,
            focalY: $media->focal_y,
            dominantColor: $media->dominant_color,
            lqip: $media->lqip,
            variants: $variants,
            optimized: $media->optimized_at !== null,
        );
    }
}
