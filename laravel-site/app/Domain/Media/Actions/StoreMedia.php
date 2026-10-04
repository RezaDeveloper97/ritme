<?php

declare(strict_types=1);

namespace App\Domain\Media\Actions;

use App\Domain\Media\Data\MediaUpload;
use App\Domain\Media\Enums\MediaFormat;
use App\Domain\Media\Exceptions\InvalidMediaException;
use App\Domain\Media\Jobs\OptimizeMedia;
use App\Domain\Media\Models\Media;
use App\Domain\Media\Support\FormatSupport;
use App\Domain\Media\Support\ImageAnalyzer;
use App\Domain\Media\Support\ImageEncoder;
use App\Domain\Media\Support\MemoryLimit;
use App\Domain\Media\Support\MimeSniffer;
use App\Domain\Media\Support\SvgSanitizer;
use Illuminate\Contracts\Bus\Dispatcher;
use Illuminate\Contracts\Config\Repository as Config;
use Illuminate\Contracts\Filesystem\Factory as Filesystems;
use Illuminate\Support\Str;
use Intervention\Image\ImageManager;
use Intervention\Image\Interfaces\ImageInterface;
use Throwable;

/**
 * Upload use case: validate (sniffed mime, 15 MB, 8000 px, SVG sanitised), dedupe by sha256, auto-orient + strip
 * metadata + cap the original at `media.original_max`, compute dominant colour + LQIP, store, then queue
 * OptimizeMedia for the variants. Animated GIFs are stored byte-for-byte.
 */
final class StoreMedia
{
    public function __construct(
        private readonly ImageManager $images,
        private readonly ImageAnalyzer $analyzer,
        private readonly ImageEncoder $encoder,
        private readonly FormatSupport $formats,
        private readonly MimeSniffer $sniffer,
        private readonly SvgSanitizer $svg,
        private readonly Filesystems $filesystems,
        private readonly Dispatcher $bus,
        private readonly Config $config,
    ) {}

    public function handle(MediaUpload $upload): Media
    {
        $path = $upload->path;
        if (! is_file($path) || ! is_readable($path)) {
            throw InvalidMediaException::unreadable();
        }

        $maxBytes = (int) $this->config->get('media.max_bytes');
        if ((int) filesize($path) > $maxBytes) {
            throw InvalidMediaException::tooLarge($maxBytes);
        }

        $mime = $this->sniffer->sniff($path);
        $format = MediaFormat::fromMime($mime);
        $this->assertAllowed($mime, $format);

        $hash = (string) hash_file('sha256', $path);
        $existing = Media::query()->where('hash', $hash)->first();
        if ($existing !== null) {
            return $existing;
        }

        /** @var MediaFormat $format */
        $stored = $format === MediaFormat::Svg ? $this->processSvg($path) : $this->processRaster($path, $format);

        $media = $this->persist($upload, $stored, $hash);

        $this->queueOptimization($media);

        return $media->refresh();
    }

    private function assertAllowed(string $mime, ?MediaFormat $format): void
    {
        if ($format === MediaFormat::Svg) {
            if (! (bool) $this->config->get('media.svg', true)) {
                throw InvalidMediaException::unsupportedType($mime);
            }

            return;
        }

        /** @var list<string> $allowed */
        $allowed = (array) $this->config->get('media.mimes', []);
        if ($format === null || ! in_array($format->mime(), $allowed, true)) {
            throw InvalidMediaException::unsupportedType($mime);
        }

        if ($format === MediaFormat::Avif && ! $this->formats->canEncode(MediaFormat::Avif)) {
            throw InvalidMediaException::unsupportedType($mime); // the driver could not decode it either
        }
    }

    /**
     * @return array{contents: string, format: MediaFormat, width: int|null, height: int|null, dominant: string|null, lqip: string|null}
     */
    private function processSvg(string $path): array
    {
        $clean = $this->svg->sanitize((string) file_get_contents($path));

        return ['contents' => $clean['svg'], 'format' => MediaFormat::Svg, 'width' => $clean['width'], 'height' => $clean['height'], 'dominant' => null, 'lqip' => null];
    }

    /**
     * @return array{contents: string, format: MediaFormat, width: int|null, height: int|null, dominant: string|null, lqip: string|null}
     */
    private function processRaster(string $path, MediaFormat $format): array
    {
        $size = @getimagesize($path);
        if ($size === false || $size[0] < 1 || $size[1] < 1) {
            throw InvalidMediaException::corrupt();
        }

        $maxDimension = (int) $this->config->get('media.max_dimension');
        if ($size[0] > $maxDimension || $size[1] > $maxDimension) {
            throw InvalidMediaException::tooManyPixels($maxDimension);
        }

        MemoryLimit::raise((string) $this->config->get('media.memory_limit', '512M'));

        try {
            $image = $this->images->read($path); // auto-oriented by EXIF
        } catch (Throwable) {
            throw InvalidMediaException::corrupt();
        }

        if ($format === MediaFormat::Gif && $image->isAnimated()) {
            $contents = (string) file_get_contents($path); // kept as-is
            $still = (clone $image)->removeAnimation(0);
        } else {
            $max = (int) $this->config->get('media.original_max', 2560);
            $image->scaleDown($max, $max);
            $contents = $this->encoder->encode($image, $format, original: true)->toString();
            $still = $image;
        }

        return [
            'contents' => $contents,
            'format' => $format,
            'width' => $image->width(),
            'height' => $image->height(),
            'dominant' => $this->analyzer->dominantColor($still),
            'lqip' => $this->lqip($still),
        ];
    }

    private function lqip(ImageInterface $image): ?string
    {
        return $this->analyzer->lqip(
            $image,
            (int) $this->config->get('media.lqip.width', 16),
            (int) $this->config->get('media.lqip.quality', 40),
            (int) $this->config->get('media.lqip.max_bytes', 600),
        );
    }

    /**
     * @param  array{contents: string, format: MediaFormat, width: int|null, height: int|null, dominant: string|null, lqip: string|null}  $stored
     */
    private function persist(MediaUpload $upload, array $stored, string $hash): Media
    {
        $diskName = (string) $this->config->get('media.disk', 'public');
        $disk = $this->filesystems->disk($diskName);

        $directory = now()->format('Y/m').'/'.Str::lower(Str::random(10));
        $filename = self::stem($upload->originalName).'.'.$stored['format']->value;
        $path = "{$directory}/{$filename}";

        $disk->put($path, $stored['contents']);

        try {
            return Media::query()->create([
                'disk' => $diskName,
                'directory' => $directory,
                'filename' => $filename,
                'original_name' => mb_substr($upload->originalName, 0, 255),
                'mime' => $stored['format']->mime(),
                'size' => strlen($stored['contents']),
                'width' => $stored['width'],
                'height' => $stored['height'],
                'alt' => $upload->alt,
                'title' => $upload->title,
                'caption' => $upload->caption,
                'focal_x' => max(0.0, min(1.0, $upload->focalX)),
                'focal_y' => max(0.0, min(1.0, $upload->focalY)),
                'dominant_color' => $stored['dominant'],
                'lqip' => $stored['lqip'],
                'variants' => [],
                'uploaded_by' => $upload->uploadedBy,
                'hash' => $hash,
            ]);
        } catch (Throwable $e) {
            $disk->deleteDirectory($directory);

            throw $e;
        }
    }

    private function queueOptimization(Media $media): void
    {
        if ($media->mime === MediaFormat::Svg->mime()) {
            $media->forceFill(['optimized_at' => now()])->save(); // vector: nothing to generate

            return;
        }

        try {
            $this->bus->dispatch((new OptimizeMedia($media->id))->afterCommit());
        } catch (Throwable $e) {
            report($e); // sync queue: a failed optimisation must not fail the upload — the original is served
        }
    }

    /**
     * File name stem: ASCII slug of the uploaded name (Persian names transliterate or fall back to `image`).
     */
    public static function stem(string $originalName): string
    {
        $slug = Str::limit(Str::slug(pathinfo($originalName, PATHINFO_FILENAME)), 80, '');

        return trim($slug, '-') !== '' ? trim($slug, '-') : 'image';
    }
}
