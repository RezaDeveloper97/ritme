<?php

declare(strict_types=1);

namespace App\Domain\Pwa\Manifest;

use App\Domain\Pwa\Data\IconSet;
use App\Domain\Pwa\Data\InstallMetadata;
use App\Domain\Pwa\Data\ManifestIcon;
use App\Domain\Pwa\Support\PwaIconFiles;
use App\Domain\Settings\Contracts\SettingsRepository;
use App\Domain\Settings\Data\PwaSettings;
use App\Support\Cache\CacheAside;
use App\Support\Cache\CacheKey;
use App\Support\Cache\NamespaceVersions;
use Illuminate\Contracts\Routing\UrlGenerator;

/**
 * `/manifest.webmanifest` and the install head tags, both from PwaSettings + the resolved IconSet. Cache-aside in
 * the `settings` namespace (SettingObserver bumps it on any settings save); the key also carries the `media`
 * namespace version so a replaced or regenerated logo shows up, and REVISION for code changes.
 */
final class WebManifest
{
    public const URL = '/manifest.webmanifest';

    public const CONTENT_TYPE = 'application/manifest+json';

    /** Stable app identity; never change it or installed apps become a different app. */
    public const ID = '/';

    /** Launch URL; `source` is dropped from canonicals (SeoManager canonical = path only). */
    public const START_URL = '/?source=pwa';

    /** Dark-scheme browser chrome = the `night` token (the dark hero/header colour). */
    public const DARK_THEME_COLOR = '#17112B';

    public const DEFAULT_DESCRIPTION = 'همراه سلامت زنان، از اولین پریود تا یائسگی';

    /** Shortcuts: route name => label. */
    public const SHORTCUTS = [
        'tools' => 'ابزارها',
        'blog.index' => 'مجله',
        'shop.index' => 'فروشگاه',
    ];

    public const REVISION = 1;

    public function __construct(
        private readonly SettingsRepository $settings,
        private readonly IconSetResolver $icons,
        private readonly CacheAside $cache,
        private readonly NamespaceVersions $versions,
        private readonly UrlGenerator $urls,
    ) {}

    public function json(): string
    {
        return $this->payload()['json'];
    }

    /**
     * @return array<string, mixed>
     */
    public function document(): array
    {
        return (array) json_decode($this->json(), true, 512, JSON_THROW_ON_ERROR);
    }

    public function installMetadata(): InstallMetadata
    {
        return InstallMetadata::fromArray($this->payload()['head']);
    }

    /**
     * @return array{json: string, head: array<string, string|null>}
     */
    private function payload(): array
    {
        $key = CacheKey::make('settings', 'pwa', 'manifest', 'm'.$this->versions->version('media'), 'r'.self::REVISION.'-'.PwaIconFiles::REVISION);

        /** @var array{json: string, head: array<string, string|null>} */
        return $this->cache->remember($key, null, function (): array {
            $pwa = $this->settings->all()->pwa;
            $icons = $this->icons->resolve($pwa);

            return [
                'json' => json_encode($this->build($pwa, $icons), JSON_UNESCAPED_UNICODE | JSON_UNESCAPED_SLASHES | JSON_PRETTY_PRINT | JSON_THROW_ON_ERROR),
                'head' => $this->head($pwa, $icons)->toArray(),
            ];
        });
    }

    /**
     * @return array<string, mixed>
     */
    private function build(PwaSettings $pwa, IconSet $icons): array
    {
        $shortcuts = [];
        foreach (self::SHORTCUTS as $route => $label) {
            $shortcuts[] = [
                'name' => $label,
                'short_name' => $label,
                'url' => $this->urls->route($route, [], false),
                'icons' => [$icons->shortcutIcon->toArray()],
            ];
        }

        $screenshots = [];
        foreach (PwaIconFiles::SCREENSHOTS as $path => [$width, $height, $formFactor, $label]) {
            if (is_file(public_path($path))) {
                $screenshots[] = [
                    'src' => PwaIconFiles::publicUrl($path),
                    'sizes' => "{$width}x{$height}",
                    'type' => 'image/webp',
                    'form_factor' => $formFactor,
                    'label' => $label,
                ];
            }
        }

        return array_filter([
            'id' => self::ID,
            'name' => $pwa->name,
            'short_name' => $pwa->shortName,
            'description' => $pwa->description !== '' ? $pwa->description : self::DEFAULT_DESCRIPTION,
            'lang' => 'fa',
            'dir' => 'rtl',
            'start_url' => self::START_URL,
            'scope' => '/',
            'display' => 'standalone',
            'theme_color' => $pwa->themeColor,
            'background_color' => $pwa->backgroundColor,
            'categories' => ['health', 'medical', 'lifestyle'],
            'icons' => array_map(static fn (ManifestIcon $icon): array => $icon->toArray(), $icons->icons),
            'screenshots' => $screenshots,
            'shortcuts' => $shortcuts,
            'prefer_related_applications' => false,
        ], static fn (mixed $value): bool => $value !== []);
    }

    private function head(PwaSettings $pwa, IconSet $icons): InstallMetadata
    {
        return new InstallMetadata(
            manifestUrl: self::URL,
            applicationName: $pwa->shortName,
            themeColorLight: $pwa->themeColor,
            themeColorDark: self::DARK_THEME_COLOR,
            faviconIco: $icons->faviconIco,
            faviconSvg: $icons->faviconSvg,
            faviconPng: $icons->faviconPng,
            appleTouchIcon: $icons->appleTouchIcon,
            maskIcon: $icons->maskIcon,
            maskIconColor: $pwa->themeColor,
        );
    }
}
