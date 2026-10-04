<?php

declare(strict_types=1);

namespace Tests\Feature\Media;

use GdImage;
use Intervention\Image\ImageManager;

/**
 * Generates image fixtures at runtime (no binaries in git): photo-like JPEGs (optionally with an EXIF orientation +
 * a camera "Make" tag to prove metadata stripping), transparent PNGs, animated GIFs and SVGs.
 */
final class MediaFixtures
{
    public const EXIF_MARKER = 'SECRETCAM';

    public static function dir(): string
    {
        $dir = sys_get_temp_dir().'/ritme-media-fixtures';
        if (! is_dir($dir)) {
            mkdir($dir, 0777, true);
        }

        return $dir;
    }

    /**
     * Left half red, right half blue, plus random blobs (so encoders have real work and sizes are realistic).
     */
    public static function jpeg(int $width, int $height, ?int $orientation = null, int $quality = 92, string $name = 'photo'): string
    {
        $gd = self::canvas($width, $height);
        ob_start();
        imagejpeg($gd, null, $quality);
        $bytes = (string) ob_get_clean();

        if ($orientation !== null) {
            $bytes = substr($bytes, 0, 2).self::exifSegment($orientation).substr($bytes, 2);
        }

        return self::write("{$name}-{$width}x{$height}-".($orientation ?? 0).'.jpg', $bytes);
    }

    public static function png(int $width, int $height, bool $transparent): string
    {
        $gd = self::canvas($width, $height);
        if ($transparent) {
            imagealphablending($gd, false);
            imagesavealpha($gd, true);
            $clear = imagecolorallocatealpha($gd, 0, 0, 0, 127);
            imagefilledrectangle($gd, 0, 0, intdiv($width, 2), $height - 1, $clear);
        }
        ob_start();
        imagepng($gd);

        return self::write("image-{$width}x{$height}-".($transparent ? 'alpha' : 'opaque').'.png', (string) ob_get_clean());
    }

    public static function animatedGif(int $width, int $height, int $frames = 3): string
    {
        $animation = ImageManager::gd()->animate(static function ($animation) use ($width, $height, $frames): void {
            for ($i = 0; $i < $frames; $i++) {
                $frame = ImageManager::gd()->create($width, $height)->fill(['ff0000', '00ff00', '0000ff'][$i % 3]);
                $animation->add($frame, 0.2);
            }
        });

        return self::write("anim-{$width}x{$height}.gif", $animation->toGif()->toString());
    }

    public static function svg(string $markup, string $name = 'icon.svg'): string
    {
        return self::write($name, $markup);
    }

    public static function text(string $name = 'notes.jpg'): string
    {
        return self::write($name, "this is not an image\n");
    }

    public static function write(string $name, string $bytes): string
    {
        $path = self::dir().'/'.$name;
        file_put_contents($path, $bytes);

        return $path;
    }

    private static function canvas(int $width, int $height): GdImage
    {
        $gd = imagecreatetruecolor($width, $height);
        mt_srand($width * 31 + $height);
        imagefilledrectangle($gd, 0, 0, intdiv($width, 2) - 1, $height - 1, (int) imagecolorallocate($gd, 220, 30, 30));
        imagefilledrectangle($gd, intdiv($width, 2), 0, $width - 1, $height - 1, (int) imagecolorallocate($gd, 30, 30, 220));
        $blobs = max(10, intdiv($width * $height, 60000));
        for ($i = 0; $i < $blobs; $i++) {
            $r = mt_rand(4, max(5, intdiv(min($width, $height), 12)));
            imagefilledellipse($gd, mt_rand(0, $width), mt_rand(0, $height), $r, $r, (int) imagecolorallocate($gd, mt_rand(0, 255), mt_rand(0, 255), mt_rand(0, 255)));
        }

        return $gd;
    }

    /**
     * APP1 Exif segment (big-endian TIFF) with Make=SECRETCAM and the given Orientation.
     */
    private static function exifSegment(int $orientation): string
    {
        $make = self::EXIF_MARKER."\0"; // 10 bytes, stored after the IFD
        $ifd = pack('n', 2)
            .pack('nnN', 0x010F, 2, strlen($make)).pack('N', 8 + 2 + 2 * 12 + 4)
            .pack('nnN', 0x0112, 3, 1).pack('nn', $orientation, 0)
            .pack('N', 0);
        $tiff = "MM\0\x2A".pack('N', 8).$ifd.$make;
        $payload = "Exif\0\0".$tiff;

        return "\xFF\xE1".pack('n', strlen($payload) + 2).$payload;
    }
}
