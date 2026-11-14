<?php

declare(strict_types=1);

namespace App\Http\Controllers;

use App\Domain\Content\Enums\StaticPage;
use App\Domain\Seo\Schema\SchemaGraph;
use App\Domain\Seo\SeoManager;
use App\Domain\Seo\StaticPages\StaticPageSeo;
use App\Domain\Settings\Contracts\SettingsRepository;
use App\Support\Text\Toman;
use Illuminate\Contracts\View\Factory as ViewFactory;
use Illuminate\Contracts\View\View;

/**
 * /plus (design/Web_Pricing.html) — pages/plus.blade.php: intro, four plan cards (monthly · 3 · 6 · 12 months),
 * what Plus adds, comparison table, FAQ group `plus` (`$faq` + its FAQPage JSON-LD come from the FaqServiceProvider
 * view composer), «why paid» text, #download band. Copy: lang/fa/plus.php.
 *
 * Prices: `plus.pricing.show` (lang) is the single switch; off, each card reads «قیمت در اپ». JSON-LD stays a plain
 * WebPage (+ FAQPage) — no Product/Offer, the prices are drafts per the design's own note.
 */
final class PlusController
{
    public function __construct(
        private readonly StaticPageSeo $staticSeo,
        private readonly SettingsRepository $settings,
        private readonly ViewFactory $views,
    ) {}

    /** SeoManager and SchemaGraph are request-scoped: injected per call, not into the (route-cached) controller. */
    public function __invoke(SeoManager $seo, SchemaGraph $graph): View
    {
        $this->staticSeo->apply($seo, StaticPage::Plus);
        $graph->pageName(StaticPage::Plus->label());

        $appLinks = $this->settings->all()->appLinks;

        return $this->views->make('pages.plus', [
            'plans' => self::plans(),
            'showPrices' => self::showPrices(),
            'appLinks' => $appLinks,
            // No checkout on the site: «انتخاب» goes to the web app when configured, else to the download band.
            'chooseUrl' => $appLinks->webApp ?? '#download',
        ]);
    }

    /**
     * The four plan cards in design order.
     *
     * @return list<array{key: string, title: string, price: string, unit: string, note: string, featured: bool}>
     */
    public static function plans(): array
    {
        // Read as one array: the translator turns non-string leaves (bool flag, int amounts) into the key string.
        $pricing = __('plus.pricing');
        $pricing = is_array($pricing) ? $pricing : [];
        $show = self::showPrices();
        $cancel = self::str($pricing['cancel_anytime'] ?? null);

        $plans = [];
        foreach (is_array($pricing['plans'] ?? null) ? $pricing['plans'] : [] as $plan) {
            if (! is_array($plan)) {
                continue;
            }

            $perMonth = self::amount($plan['per_month'] ?? null);
            $total = self::amount($plan['total'] ?? null);
            $saving = self::amount($plan['saving'] ?? null);
            $priced = $show && $perMonth !== null;

            $tail = $saving !== null
                ? str_replace(':percent', fa_digits((string) $saving), self::str($pricing['saving'] ?? null))
                : $cancel;
            $note = match (true) {
                $priced && $total !== null => str_replace([':total', ':tail'], [Toman::format($total, false), $tail], self::str($pricing['total_line'] ?? null)),
                $saving === null => $cancel,
                default => '',
            };

            $plans[] = [
                'key' => self::str($plan['key'] ?? null),
                'title' => self::str($plan['title'] ?? null),
                'price' => $priced ? Toman::format($perMonth, false) : self::str($pricing['in_app'] ?? null),
                'unit' => $priced ? self::str($pricing['per_month'] ?? null) : '',
                'note' => $note,
                'featured' => ($plan['featured'] ?? false) === true,
            ];
        }

        return $plans;
    }

    private static function showPrices(): bool
    {
        $pricing = __('plus.pricing');

        return is_array($pricing) && ($pricing['show'] ?? false) === true;
    }

    private static function amount(mixed $value): ?int
    {
        return is_int($value) && $value > 0 ? $value : null;
    }

    private static function str(mixed $value): string
    {
        return is_string($value) ? $value : '';
    }
}
