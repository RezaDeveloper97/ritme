<?php

declare(strict_types=1);

namespace App\Http\Controllers;

use App\Domain\Content\Enums\StaticPage;
use App\Domain\Content\SiteNavigation;
use App\Domain\Seo\Schema\SchemaGraph;
use App\Domain\Seo\SeoManager;
use App\Domain\Seo\StaticPages\StaticPageSeo;
use App\Domain\Settings\Contracts\SettingsRepository;
use Illuminate\Contracts\View\Factory as ViewFactory;
use Illuminate\Contracts\View\View;

/**
 * /social-responsibility (design/html/social-responsibility.html) — pages/social-responsibility.blade.php: the
 * always-free commitment, programs, support + partnership cards, founder quote, transparency report, #download CTA.
 *
 * AUDIT §8: no payment gateway exists, so «از ریتمی حمایت کن» is a contact CTA and no donation amounts are shown.
 * The transparency report link is shown only when `social.transparency.url` is set. Copy: lang/fa/social.php.
 */
final class SocialResponsibilityController
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
        $this->staticSeo->apply($seo, StaticPage::SocialResponsibility);
        $graph->pageName(StaticPage::SocialResponsibility->label());

        $appLinks = $this->settings->all()->appLinks;
        $report = __('social.transparency.url');

        return $this->views->make('pages.social-responsibility', [
            'freeItems' => self::strings('social.free.items'),
            'programs' => self::programs(),
            'transparencyUrl' => is_string($report) && $report !== '' && $report !== 'social.transparency.url' ? $report : null,
            'appLinks' => $appLinks,
            'qrUrl' => $appLinks->webApp ?? $this->navigation->url(StaticPage::SocialResponsibility, 'download', absolute: true),
        ]);
    }

    /**
     * @return list<array{title: string, text: string}>
     */
    private static function programs(): array
    {
        $value = __('social.programs.items');
        $programs = [];
        foreach (is_array($value) ? $value : [] as $item) {
            if (is_array($item)) {
                $programs[] = [
                    'title' => is_string($item['title'] ?? null) ? $item['title'] : '',
                    'text' => is_string($item['text'] ?? null) ? $item['text'] : '',
                ];
            }
        }

        return $programs;
    }

    /**
     * @return list<string>
     */
    private static function strings(string $key): array
    {
        $value = __($key);

        return array_values(array_filter(is_array($value) ? $value : [], is_string(...)));
    }
}
