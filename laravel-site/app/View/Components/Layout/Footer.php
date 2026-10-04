<?php

declare(strict_types=1);

namespace App\View\Components\Layout;

use App\Domain\Content\Data\LinkData;
use App\Domain\Content\Enums\StaticPage;
use App\Domain\Content\SiteNavigation;
use App\Domain\Content\Support\AllowedHtml;
use App\Domain\Settings\Contracts\SettingsRepository;
use App\Domain\Settings\Data\SiteSettings;
use App\Support\Cache\CacheAside;
use App\Support\Cache\CacheKey;
use App\Support\Cache\NamespaceVersions;
use App\Support\Jalali\JalaliDate;
use App\Support\Text\PersianDigits;
use DateTimeImmutable;
use Illuminate\Contracts\View\View;
use Illuminate\View\Component;

/**
 * <x-layout.footer /> — brand + store badges, five `<nav>` link columns from the StaticPage registry, the enamad
 * trust seal (admin HTML reduced to the allowed tags), the emergency note and socials (empty settings are hidden).
 *
 * Cached as a `menu` fragment keyed by the `settings` version, the Jalali year and the build hash.
 */
final class Footer extends Component
{
    private const STORES = [
        'bazaar' => 'کافه‌بازار',
        'myket' => 'مایکت',
        'google_play' => 'گوگل‌پلی',
        'app_store' => 'نسخه iOS',
    ];

    private const SOCIALS = [
        'instagram' => 'اینستاگرام',
        'telegram' => 'تلگرام',
        'linkedin' => 'لینکدین',
    ];

    public function render(): View
    {
        $year = self::jalaliYear();
        $key = CacheKey::make('menu', 'footer', $year, 's'.app(NamespaceVersions::class)->version('settings'), Header::buildHash());

        $html = app(CacheAside::class)->remember($key, null, function () use ($year): string {
            $settings = app(SettingsRepository::class)->all();
            $navigation = app(SiteNavigation::class);

            return view('components.layout.footer', [
                'homeUrl' => $navigation->url(StaticPage::Home),
                'siteName' => $settings->general->siteName,
                'tagline' => self::tagline($settings),
                'columns' => $navigation->footer(),
                'stores' => self::links($settings->appLinks->toArray(), self::STORES),
                'socials' => self::links($settings->social->toArray(), self::SOCIALS),
                'enamadHtml' => AllowedHtml::clean($settings->legal->enamadHtml, $settings->legal->enamadAllowedTags, 'نماد اعتماد الکترونیکی'),
                'enamadCode' => $settings->legal->enamadCode,
                'year' => PersianDigits::toPersian($year),
                'emergency' => $settings->general->emergencyNumber,
                'emergencyLabel' => PersianDigits::toPersian($settings->general->emergencyNumber),
            ])->render();
        });

        return view('components.layout.fragment', ['html' => $html]);
    }

    /** Current Solar Hijri year in Tehran. */
    public static function jalaliYear(?DateTimeImmutable $now = null): int
    {
        return JalaliDate::fromDateTime($now ?? new DateTimeImmutable('now'))->year;
    }

    private static function tagline(SiteSettings $settings): string
    {
        $parts = array_filter([rtrim($settings->general->tagline, '.'), $settings->general->footerNote], static fn (string $part): bool => $part !== '');

        return implode('. ', $parts);
    }

    /**
     * @param  array<string, mixed>  $urls
     * @param  array<string, string>  $labels
     * @return list<LinkData>
     */
    private static function links(array $urls, array $labels): array
    {
        $links = [];
        foreach ($labels as $key => $label) {
            $url = $urls[$key] ?? null;
            if (is_string($url) && preg_match('#^https?://#i', $url) === 1) {
                $links[] = new LinkData($label, $url, external: true);
            }
        }

        return $links;
    }
}
