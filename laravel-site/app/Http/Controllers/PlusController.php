<?php

declare(strict_types=1);

namespace App\Http\Controllers;

use App\Domain\Content\Enums\StaticPage;
use App\Domain\Content\SiteNavigation;
use App\Domain\Seo\Contracts\SeoMetaRepository;
use App\Domain\Seo\Schema\SchemaGraph;
use App\Domain\Seo\SeoManager;
use App\Domain\Settings\Contracts\SettingsRepository;
use App\Support\Text\Toman;
use Illuminate\Contracts\View\Factory as ViewFactory;
use Illuminate\Contracts\View\View;

/**
 * /plus (design/html/plus.html) — pages/plus.blade.php: intro, three plan cards (always free · Ritme Plus · course
 * packs), FAQ group `plus` as a 2-column grid (`$faq` + its FAQPage JSON-LD come from the FaqServiceProvider view
 * composer), #download CTA. Copy: lang/fa/plus.php.
 *
 * Prices: `plus.pricing.show` (lang) is the single switch. The design has only placeholders, so it is off and paid
 * plans read «قیمت در اپ»; when on, each filled amount (toman) is shown instead. JSON-LD stays a plain WebPage (+
 * FAQPage) — no Product/Offer until prices are real and published.
 */
final class PlusController
{
    public function __construct(
        private readonly SeoManager $seo,
        private readonly SeoMetaRepository $meta,
        private readonly SchemaGraph $graph,
        private readonly SettingsRepository $settings,
        private readonly SiteNavigation $navigation,
        private readonly ViewFactory $views,
    ) {}

    public function __invoke(): View
    {
        $meta = $this->meta->forRoute(StaticPage::Plus->routeName());
        if (($meta->title ?? '') === '') {
            $this->seo->rawTitle(self::text('plus.seo.title'));
        }
        if (($meta->description ?? '') === '') {
            $this->seo->description(self::text('plus.seo.description'));
        }
        $this->graph->pageName(StaticPage::Plus->label());

        $appLinks = $this->settings->all()->appLinks;

        return $this->views->make('pages.plus', [
            'plans' => self::plans(),
            'appLinks' => $appLinks,
            'qrUrl' => $appLinks->webApp ?? $this->navigation->url(StaticPage::Plus, 'download', absolute: true),
        ]);
    }

    /**
     * The three plan cards in design order.
     *
     * @return list<array{key: string, title: string, price: string, note: string, items: list<string>, featured: bool}>
     */
    public static function plans(): array
    {
        // Read as one array: the translator turns non-string leaves (bool flag, int amounts) into the key string.
        $pricing = __('plus.pricing');
        $pricing = is_array($pricing) ? $pricing : [];
        $show = ($pricing['show'] ?? false) === true;
        $inApp = self::text('plus.pricing.in_app');

        $yearly = $show ? self::amount($pricing['plus_yearly'] ?? null) : null;
        $monthly = $show ? self::amount($pricing['plus_monthly'] ?? null) : null;
        $pack = $show ? self::amount($pricing['pack'] ?? null) : null;

        $plusNote = self::text('plus.plans.plus.note');
        if ($monthly !== null) {
            $plusNote .= ' · '.str_replace(':price', Toman::withUnit($monthly), self::text('plus.pricing.monthly'));
        }

        return [
            self::plan('free', self::text('plus.plans.free.price'), self::text('plus.plans.free.note')),
            self::plan('plus', $yearly !== null ? Toman::withUnit($yearly).' '.self::text('plus.pricing.yearly_suffix') : $inApp, $plusNote, featured: true),
            self::plan('packs', $pack !== null ? Toman::withUnit($pack).' '.self::text('plus.pricing.pack_suffix') : $inApp, self::text('plus.plans.packs.note')),
        ];
    }

    /**
     * @return array{key: string, title: string, price: string, note: string, items: list<string>, featured: bool}
     */
    private static function plan(string $key, string $price, string $note, bool $featured = false): array
    {
        $items = __("plus.plans.{$key}.items");

        return [
            'key' => $key,
            'title' => self::text("plus.plans.{$key}.title"),
            'price' => $price,
            'note' => $note,
            'items' => array_values(array_filter(is_array($items) ? $items : [], is_string(...))),
            'featured' => $featured,
        ];
    }

    private static function amount(mixed $value): ?int
    {
        return is_int($value) && $value > 0 ? $value : null;
    }

    private static function text(string $key): string
    {
        $text = __($key);

        return is_string($text) ? $text : '';
    }
}
