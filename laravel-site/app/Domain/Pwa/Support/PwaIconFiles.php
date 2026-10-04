<?php

declare(strict_types=1);

namespace App\Domain\Pwa\Support;

use App\Domain\Media\Enums\MediaFormat;
use App\Domain\Media\Models\Media;

/**
 * File names of an icon set — the same names in public/icons (committed defaults, tools/pwa-icons.mjs) and in a
 * generated set on the media disk (`pwa/{media id}-{version}/`, GeneratePwaIcons).
 */
final class PwaIconFiles
{
    /** Bump when the generated artwork changes (new sizes, padding …): every generated set gets a new URL. */
    public const REVISION = 1;

    /** Root of generated sets on the media disk (served under /media, immutable-cached, hence versioned names). */
    public const GENERATED_ROOT = 'pwa';

    /** Smallest logo edge (px) the admin may upload: the 512 px icons are never upscaled from less. */
    public const MIN_SOURCE_SIZE = 512;

    /** Manifest icons: file => [size, purpose]. `monochrome-512` is optional for generated sets (needs transparency). */
    public const ICONS = [
        'icon-192' => [192, 'any'],
        'icon-512' => [512, 'any'],
        'maskable-192' => [192, 'maskable'],
        'maskable-512' => [512, 'maskable'],
        'monochrome-512' => [512, 'monochrome'],
    ];

    public const SHORTCUT_ICON = 'icon-96';

    public const SHORTCUT_SIZE = 96;

    public const APPLE_TOUCH_ICON = 'apple-touch-icon';

    public const APPLE_TOUCH_SIZE = 180;

    public const FAVICON_PNG = 'favicon-32';

    /** favicon.ico entries (PNG-encoded). */
    public const ICO_SIZES = [16, 32, 48];

    /** Defaults, relative to public/. */
    public const DEFAULT_DIRECTORY = 'icons';

    public const DEFAULT_FAVICON_SVG = 'icons/favicon.svg';

    public const DEFAULT_MASK_ICON = 'icons/mask-icon.svg';

    public const DEFAULT_FAVICON_ICO = 'favicon.ico';

    /** Manifest screenshots, relative to public/: file => [width, height, form_factor, label]. */
    public const SCREENSHOTS = [
        'icons/screenshot-mobile.webp' => [1080, 2340, 'narrow', 'صفحه اصلی ریتمی در موبایل'],
        'icons/screenshot-desktop.webp' => [1280, 800, 'wide', 'صفحه اصلی ریتمی در دسکتاپ'],
    ];

    public static function generatedDirectory(Media $media, string $backgroundColor): string
    {
        $version = substr(sha1($media->hash.'|'.strtoupper($backgroundColor).'|'.self::REVISION), 0, 10);

        return self::GENERATED_ROOT.'/'.$media->id.'-'.$version;
    }

    public static function usableSource(Media $media): bool
    {
        $format = MediaFormat::fromMime($media->mime);

        return $format !== null && $format->isRaster() && $format !== MediaFormat::Gif
            && min((int) $media->width, (int) $media->height) >= self::MIN_SOURCE_SIZE;
    }

    /** Root-relative URL of a file in public/, with `?v=` + content hash so long-cached URLs change with the file. */
    public static function publicUrl(string $path): string
    {
        $file = public_path($path);
        $hash = is_file($file) ? hash_file('xxh3', $file) : false;

        return '/'.$path.($hash !== false ? '?v='.substr($hash, 0, 8) : '');
    }
}
