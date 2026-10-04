<?php

declare(strict_types=1);

namespace App\View\Components\Layout;

use App\Domain\Content\Enums\HeaderVariant;
use App\Domain\Content\Enums\StaticPage;
use App\Domain\Content\SiteNavigation;
use App\Domain\Settings\Contracts\SettingsRepository;
use App\Support\Cache\CacheAside;
use App\Support\Cache\CacheKey;
use App\Support\Cache\NamespaceVersions;
use Illuminate\Contracts\View\View;
use Illuminate\Foundation\Vite;
use Illuminate\Routing\Router;
use Illuminate\View\Component;

/**
 * <x-layout.header variant="dark" /> — logo, main nav, login/download actions, burger + mobile menu.
 *
 * The HTML is a cache-aside fragment in the `menu` namespace, keyed by variant, active route, the download
 * target, the `settings` namespace version (admin edits of app links / site name) and the Vite manifest hash
 * (sprite URLs change per build).
 */
final class Header extends Component
{
    public readonly HeaderVariant $headerVariant;

    public readonly ?string $routeName;

    public function __construct(
        HeaderVariant|string|null $variant = null,
        ?string $route = null,
        public readonly ?bool $appCta = null,
        ?Router $router = null,
    ) {
        $this->headerVariant = HeaderVariant::resolve($variant);
        $this->routeName = $route ?? ($router ?? app(Router::class))->currentRouteName();
    }

    public function render(): View
    {
        $navigation = app(SiteNavigation::class);
        $downloadHref = $navigation->downloadHref($this->routeName, $this->appCta);

        $key = CacheKey::make(
            'menu',
            'header',
            $this->headerVariant->value,
            $this->routeName ?? '-',
            $downloadHref === '#download' ? 'cta' : 'home-cta',
            's'.app(NamespaceVersions::class)->version('settings'),
            self::buildHash(),
        );

        $html = app(CacheAside::class)->remember($key, null, function () use ($navigation, $downloadHref): string {
            $settings = app(SettingsRepository::class)->all();

            return view('components.layout.header', [
                'variant' => $this->headerVariant,
                'links' => $navigation->main($this->routeName),
                'homeUrl' => $navigation->url(StaticPage::Home),
                'siteName' => $settings->general->siteName,
                'loginUrl' => $settings->appLinks->webApp,
                'downloadHref' => $downloadHref,
            ])->render();
        });

        return view('components.layout.fragment', ['html' => $html]);
    }

    /**
     * Fragment version: changes with every asset build (hashed sprite URL inside the cached HTML) and with every
     * edit of the shell templates, so neither a deploy nor a template change can serve stale markup.
     */
    public static function buildHash(): string
    {
        $hash = app(Vite::class)->manifestHash();
        $templates = array_merge(glob(resource_path('views/components/layout/*.blade.php')) ?: [], glob(resource_path('views/components/ui/*.blade.php')) ?: []);
        $mtime = $templates === [] ? 0 : max(array_map(static fn (string $file): int => (int) filemtime($file), $templates));

        return (is_string($hash) && $hash !== '' ? $hash : 'nobuild').'-'.$mtime;
    }
}
