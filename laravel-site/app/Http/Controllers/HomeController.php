<?php

declare(strict_types=1);

namespace App\Http\Controllers;

use App\Domain\Blog\Contracts\PostRepository;
use App\Domain\Blog\Data\PostCardData;
use App\Domain\Content\Enums\StaticPage;
use App\Domain\Content\SiteNavigation;
use App\Domain\Seo\Contracts\SeoMetaRepository;
use App\Domain\Seo\Schema\Nodes\MobileApplicationNode;
use App\Domain\Seo\Schema\SchemaGraph;
use App\Domain\Seo\Schema\SchemaIds;
use App\Domain\Seo\SeoManager;
use App\Domain\Settings\Contracts\SettingsRepository;
use App\Domain\Settings\Data\SiteSettings;
use App\Support\Cache\CacheAside;
use App\Support\Cache\CacheKey;
use Illuminate\Contracts\Config\Repository as Config;
use Illuminate\Contracts\View\Factory as ViewFactory;
use Illuminate\Contracts\View\View;
use Illuminate\Foundation\Vite;
use Illuminate\Support\HtmlString;

/**
 * The home page (`/`, design/html/index.html) — pages/home.blade.php.
 *
 * Data handed to the view: the latest magazine posts (cached `blog` repository; the block hides itself while there
 * are none), the app links + QR target from settings, and the static middle of the page (stage cards → tools) as one
 * pre-rendered fragment cached in the `pages` namespace (on top of the guest full-page cache, so logged-in and
 * cache-bypassing requests stay cheap too). Copy lives in lang/fa/home.php.
 */
final class HomeController
{
    /** «خواندنی‌های این هفته»: newest posts shown. */
    public const READINGS = 3;

    public function __construct(
        private readonly SeoManager $seo,
        private readonly SeoMetaRepository $meta,
        private readonly SchemaGraph $graph,
        private readonly SettingsRepository $settings,
        private readonly PostRepository $posts,
        private readonly SiteNavigation $navigation,
        private readonly CacheAside $cache,
        private readonly ViewFactory $views,
        private readonly Vite $vite,
        private readonly Config $config,
    ) {}

    public function __invoke(): View
    {
        $settings = $this->settings->all();
        $this->describe($settings);

        return $this->views->make('pages.home', [
            'staticSections' => $this->staticSections(),
            'readings' => $this->readings(),
            'appLinks' => $settings->appLinks,
            'qrUrl' => $settings->appLinks->webApp ?? $this->navigation->url(StaticPage::Home, 'download', absolute: true),
        ]);
    }

    /**
     * Title/description: the admin's (seo_meta of `home`) when set, else the design's. JSON-LD: Organization,
     * WebSite and WebPage are automatic (no breadcrumbs on the home page); the app the page offers is added here.
     */
    private function describe(SiteSettings $settings): void
    {
        $meta = $this->meta->forRoute(StaticPage::Home->routeName());

        if (($meta->title ?? '') === '') {
            // The design title already carries the brand («ریتمی — …»): no «%s — ریتمی» template.
            $this->seo->rawTitle(self::text('home.seo.title'));
        }
        if (($meta->description ?? '') === '') {
            $this->seo->description(self::text('home.seo.description'));
        }

        $siteUrl = SchemaIds::root((string) $this->config->get('app.url'));
        $this->graph->add(MobileApplicationNode::make($settings, $siteUrl, self::text('home.seo.description')));
    }

    /**
     * @return list<PostCardData>
     */
    private function readings(): array
    {
        return $this->posts->latest(1, self::READINGS)->items;
    }

    /**
     * Stage cards, «یک اپ، پنج بخش», the two feature splits, help, privacy and tools: copy + routes only, so the
     * rendered HTML is cached until the `pages` namespace is bumped, the asset build changes (sprite URL) or a
     * template is edited.
     */
    private function staticSections(): HtmlString
    {
        $key = CacheKey::make('pages', 'home', 'static', $this->templateVersion());

        /** @var string $html */
        $html = $this->cache->remember($key, null, fn (): string => $this->views->make('pages.home.static')->render());

        return new HtmlString($html);
    }

    private function templateVersion(): string
    {
        $hash = $this->vite->manifestHash();
        $files = array_merge(
            glob(resource_path('views/pages/home/*.blade.php')) ?: [],
            glob(resource_path('views/pages/home/*/*.blade.php')) ?: [],
            glob(resource_path('views/components/*/*.blade.php')) ?: [],
            [lang_path('fa/home.php')],
        );
        $mtime = max(array_map(static fn (string $file): int => is_file($file) ? (int) filemtime($file) : 0, $files));

        return (is_string($hash) && $hash !== '' ? $hash : 'nobuild').'-'.$mtime;
    }

    private static function text(string $key): string
    {
        $text = __($key);

        return is_string($text) ? $text : '';
    }
}
