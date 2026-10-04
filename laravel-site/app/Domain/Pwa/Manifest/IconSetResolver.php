<?php

declare(strict_types=1);

namespace App\Domain\Pwa\Manifest;

use App\Domain\Media\Models\Media;
use App\Domain\Pwa\Actions\GeneratePwaIcons;
use App\Domain\Pwa\Data\IconSet;
use App\Domain\Pwa\Data\ManifestIcon;
use App\Domain\Pwa\Support\PwaIconFiles;
use App\Domain\Settings\Data\PwaSettings;
use Illuminate\Contracts\Filesystem\Factory as Filesystems;
use Illuminate\Contracts\Filesystem\Filesystem;
use Throwable;

/**
 * Picks the icon set for the manifest and head tags: the set generated from the admin logo when one is configured
 * (generated on first use if missing, e.g. after a database restore), else the committed defaults in public/icons
 * with a content hash in the query (those URLs are long-cached by .htaccess). Only called inside WebManifest's
 * cache loader.
 */
final class IconSetResolver
{
    public function __construct(
        private readonly GeneratePwaIcons $generate,
        private readonly Filesystems $filesystems,
    ) {}

    public function resolve(PwaSettings $settings): IconSet
    {
        return ($settings->iconMediaId !== null ? $this->generated($settings->iconMediaId, $settings->backgroundColor) : null)
            ?? $this->defaults();
    }

    public function defaults(): IconSet
    {
        $png = fn (string $name): string => PwaIconFiles::publicUrl(PwaIconFiles::DEFAULT_DIRECTORY."/{$name}.png");

        $icons = [];
        foreach (PwaIconFiles::ICONS as $name => [$size, $purpose]) {
            $icons[] = new ManifestIcon($png($name), $size, $purpose);
        }

        return new IconSet(
            icons: $icons,
            shortcutIcon: new ManifestIcon($png(PwaIconFiles::SHORTCUT_ICON), PwaIconFiles::SHORTCUT_SIZE),
            appleTouchIcon: $png(PwaIconFiles::APPLE_TOUCH_ICON),
            faviconIco: '/'.PwaIconFiles::DEFAULT_FAVICON_ICO,
            faviconSvg: PwaIconFiles::publicUrl(PwaIconFiles::DEFAULT_FAVICON_SVG),
            faviconPng: null,
            maskIcon: PwaIconFiles::publicUrl(PwaIconFiles::DEFAULT_MASK_ICON),
        );
    }

    private function generated(int $mediaId, string $backgroundColor): ?IconSet
    {
        $media = Media::query()->find($mediaId);
        if (! $media instanceof Media || ! PwaIconFiles::usableSource($media)) {
            return null;
        }

        $disk = $this->filesystems->disk($media->disk);
        $directory = PwaIconFiles::generatedDirectory($media, $backgroundColor);

        if (! $disk->exists("{$directory}/icon-512.png")) {
            try {
                $this->generate->handle($media, $backgroundColor);
            } catch (Throwable $e) {
                report($e);

                return null;
            }
        }

        $url = static fn (Filesystem $disk, string $file): string => $disk->url("{$directory}/{$file}");

        $icons = [];
        foreach (PwaIconFiles::ICONS as $name => [$size, $purpose]) {
            if ($disk->exists("{$directory}/{$name}.png")) {
                $icons[] = new ManifestIcon($url($disk, "{$name}.png"), $size, $purpose);
            }
        }

        return new IconSet(
            icons: $icons,
            shortcutIcon: new ManifestIcon($url($disk, PwaIconFiles::SHORTCUT_ICON.'.png'), PwaIconFiles::SHORTCUT_SIZE),
            appleTouchIcon: $url($disk, PwaIconFiles::APPLE_TOUCH_ICON.'.png'),
            faviconIco: $url($disk, 'favicon.ico'),
            faviconSvg: null,
            faviconPng: $url($disk, PwaIconFiles::FAVICON_PNG.'.png'),
            maskIcon: PwaIconFiles::publicUrl(PwaIconFiles::DEFAULT_MASK_ICON),
        );
    }
}
