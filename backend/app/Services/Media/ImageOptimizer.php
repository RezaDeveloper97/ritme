<?php

namespace App\Services\Media;

use Illuminate\Http\UploadedFile;
use Illuminate\Support\Facades\Log;
use Illuminate\Support\Facades\Storage;
use Illuminate\Support\Str;

/**
 * Shrinks and re-encodes admin-uploaded images before they hit the disk.
 *
 * The app is mobile-only, so a 4000px camera JPEG is pure waste on a phone
 * screen (and on an Iranian mobile connection). Every upload is therefore
 * scaled to fit a box, re-encoded to WebP when the runtime supports it, and
 * stripped of EXIF (GD keeps no metadata) after honouring the orientation tag.
 *
 * Uses plain GD — no extra composer dependency. When GD is unavailable the
 * original file is stored untouched rather than failing the upload, so a
 * runtime without the extension degrades instead of breaking the panel.
 */
class ImageOptimizer
{
    /** Bounding box the stored image is scaled into (never upscaled). */
    public const MAX_WIDTH = 1080;

    public const MAX_HEIGHT = 1080;

    /** Encoder quality for WebP/JPEG output. */
    public const QUALITY = 82;

    /**
     * Above this pixel count decoding costs more memory than the saving is
     * worth, so the original is kept as-is (a decompression-bomb guard).
     */
    private const MAX_PIXELS = 30_000_000;

    public function __construct(
        private readonly int $maxWidth = self::MAX_WIDTH,
        private readonly int $maxHeight = self::MAX_HEIGHT,
        private readonly int $quality = self::QUALITY,
    ) {}

    /**
     * Optimize and store the upload, returning its path on the disk.
     */
    public function store(UploadedFile $file, string $directory, string $disk = 'public'): string
    {
        $optimized = $this->optimize($file);

        if ($optimized === null) {
            return $file->store($directory, $disk);
        }

        [$binary, $extension] = $optimized;
        $path = trim($directory, '/').'/'.Str::random(40).'.'.$extension;

        Storage::disk($disk)->put($path, $binary);

        return $path;
    }

    /**
     * @return array{0: string, 1: string}|null [binary, extension], or null
     *                                          when the file can't be processed
     */
    private function optimize(UploadedFile $file): ?array
    {
        if (! function_exists('imagecreatefromstring')) {
            return null;
        }

        $path = $file->getRealPath();

        if ($path === false || ! $this->isSaneSize($path)) {
            return null;
        }

        $raw = @file_get_contents($path);

        if ($raw === false) {
            return null;
        }

        $image = @imagecreatefromstring($raw);

        if ($image === false) {
            return null;
        }

        try {
            $image = $this->autoRotate($image, $path);
            $image = $this->scaleToFit($image);

            return $this->encode($image);
        } catch (\Throwable $e) {
            Log::warning('Image optimization failed; storing the original.', ['error' => $e->getMessage()]);

            return null;
        } finally {
            if ($image instanceof \GdImage) {
                imagedestroy($image);
            }
        }
    }

    /**
     * Guard against images too large to decode comfortably.
     */
    private function isSaneSize(string $path): bool
    {
        $info = @getimagesize($path);

        return $info !== false && ($info[0] * $info[1]) <= self::MAX_PIXELS;
    }

    /**
     * Apply the JPEG EXIF orientation tag, since re-encoding drops the tag and
     * a phone photo would otherwise come out sideways.
     */
    private function autoRotate(\GdImage $image, string $path): \GdImage
    {
        if (! function_exists('exif_read_data')) {
            return $image;
        }

        $exif = @exif_read_data($path);
        $orientation = $exif['Orientation'] ?? null;

        $degrees = match ($orientation) {
            3 => 180,
            6 => -90,
            8 => 90,
            default => 0,
        };

        if ($degrees === 0) {
            return $image;
        }

        $rotated = @imagerotate($image, $degrees, 0);

        if ($rotated === false) {
            return $image;
        }

        imagedestroy($image);

        return $rotated;
    }

    /**
     * Scale down into the configured box, preserving the aspect ratio and
     * transparency. Images already inside the box are returned untouched.
     */
    private function scaleToFit(\GdImage $image): \GdImage
    {
        $width = imagesx($image);
        $height = imagesy($image);
        $ratio = min($this->maxWidth / $width, $this->maxHeight / $height, 1);

        if ($ratio >= 1) {
            return $image;
        }

        $target = imagecreatetruecolor((int) max(1, round($width * $ratio)), (int) max(1, round($height * $ratio)));

        imagealphablending($target, false);
        imagesavealpha($target, true);
        imagecopyresampled(
            $target, $image,
            0, 0, 0, 0,
            imagesx($target), imagesy($target),
            $width, $height,
        );

        imagedestroy($image);

        return $target;
    }

    /**
     * WebP where available (roughly 30% smaller than JPEG at equal quality),
     * otherwise JPEG — flattened onto white so transparency doesn't turn black.
     *
     * @return array{0: string, 1: string}
     */
    private function encode(\GdImage $image): array
    {
        if (function_exists('imagewebp')) {
            imagealphablending($image, false);
            imagesavealpha($image, true);

            return [$this->capture(fn () => imagewebp($image, null, $this->quality)), 'webp'];
        }

        $flattened = imagecreatetruecolor(imagesx($image), imagesy($image));
        imagefill($flattened, 0, 0, imagecolorallocate($flattened, 255, 255, 255));
        imagecopy($flattened, $image, 0, 0, 0, 0, imagesx($image), imagesy($image));

        try {
            return [$this->capture(fn () => imagejpeg($flattened, null, $this->quality)), 'jpg'];
        } finally {
            imagedestroy($flattened);
        }
    }

    /**
     * GD encoders write to stdout when no filename is given; capture that.
     */
    private function capture(callable $writer): string
    {
        ob_start();

        try {
            $writer();

            return (string) ob_get_contents();
        } finally {
            ob_end_clean();
        }
    }
}
