<?php

declare(strict_types=1);

namespace App\Http\Controllers;

use App\Domain\Content\Enums\StaticPage;
use App\Domain\Content\SiteNavigation;
use App\Domain\Seo\Schema\Enums\WebPageType;
use App\Domain\Seo\Schema\SchemaGraph;
use App\Domain\Seo\SeoManager;
use App\Domain\Seo\StaticPages\StaticPageSeo;
use App\Domain\Settings\Contracts\SettingsRepository;
use Illuminate\Contracts\View\Factory as ViewFactory;
use Illuminate\Contracts\View\View;

/**
 * /services (design/html/services.html) — pages/services.blade.php: intro, «از سؤال ساده تا ویزیت» care cards (in-app
 * services, not links), directory + shop promo cards, emergency note, #download CTA. Copy: lang/fa/services.php;
 * emergency number + app links from settings. JSON-LD: CollectionPage (it collects the site's service sections).
 */
final class ServicesController
{
    public function __construct(
        private readonly StaticPageSeo $staticSeo,
        private readonly SettingsRepository $settings,
        private readonly SiteNavigation $navigation,
        private readonly ViewFactory $views,
    ) {}

    /** SeoManager and SchemaGraph are request-scoped: injected per call, not into the (route-cached) controller. */
    public function __invoke(SeoManager $seo, SchemaGraph $graph): View
    {
        $this->staticSeo->apply($seo, StaticPage::Services);
        $graph->pageType(WebPageType::CollectionPage)->pageName(StaticPage::Services->label());

        $settings = $this->settings->all();
        $appLinks = $settings->appLinks;

        return $this->views->make('pages.services', [
            'careItems' => self::careItems(),
            'emergencyNumber' => $settings->general->emergencyNumber,
            'appLinks' => $appLinks,
            'qrUrl' => $appLinks->webApp ?? $this->navigation->url(StaticPage::Services, 'download', absolute: true),
        ]);
    }

    /**
     * @return list<array{icon: string, color: string, title: string, text: string, points: list<string>}>
     */
    private static function careItems(): array
    {
        $value = __('services.care.items');
        $items = [];
        foreach (is_array($value) ? $value : [] as $item) {
            if (! is_array($item)) {
                continue;
            }
            $points = is_array($item['points'] ?? null) ? $item['points'] : [];
            $items[] = [
                'icon' => is_string($item['icon'] ?? null) ? $item['icon'] : 'check',
                'color' => is_string($item['color'] ?? null) ? $item['color'] : 'primary',
                'title' => is_string($item['title'] ?? null) ? $item['title'] : '',
                'text' => is_string($item['text'] ?? null) ? $item['text'] : '',
                'points' => array_values(array_filter($points, is_string(...))),
            ];
        }

        return $items;
    }
}
