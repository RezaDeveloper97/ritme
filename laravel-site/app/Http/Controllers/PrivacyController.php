<?php

declare(strict_types=1);

namespace App\Http\Controllers;

use App\Domain\Content\Enums\StaticPage;
use App\Domain\Seo\Contracts\SeoMetaRepository;
use App\Domain\Seo\Schema\SchemaGraph;
use App\Domain\Seo\SeoManager;
use App\Domain\Settings\Contracts\SettingsRepository;
use Carbon\CarbonImmutable;
use Illuminate\Contracts\View\Factory as ViewFactory;
use Illuminate\Contracts\View\View;
use Throwable;

/**
 * /privacy (design/html/privacy.html) — pages/privacy.blade.php: principles, in-app privacy tools, consents and the
 * full privacy policy (AUDIT §8 decision: a section on this page, `#policy`, opened by «خواندن متن کامل» — no
 * separate URL, so nothing new in the StaticPage registry or the URL map).
 *
 * The data-protection email comes from LegalSettings (a visible «[…]» marker while it is empty); the last-updated
 * date (`privacy.updated_at`, lang/fa/privacy.php) is shown and sent as the page's `dateModified`.
 */
final class PrivacyController
{
    public function __construct(
        private readonly SeoManager $seo,
        private readonly SeoMetaRepository $meta,
        private readonly SchemaGraph $graph,
        private readonly SettingsRepository $settings,
        private readonly ViewFactory $views,
    ) {}

    public function __invoke(): View
    {
        $meta = $this->meta->forRoute(StaticPage::Privacy->routeName());
        if (($meta->title ?? '') === '') {
            $this->seo->rawTitle(self::text('privacy.seo.title'));
        }
        if (($meta->description ?? '') === '') {
            $this->seo->description(self::text('privacy.seo.description'));
        }

        $updated = self::date(self::text('privacy.updated_at'));
        $this->graph->pageName(StaticPage::Privacy->label())->dates(null, $updated?->toAtomString());

        $email = $this->settings->all()->legal->dataProtectionEmail;
        $emailText = $email ?? self::text('privacy.policy.dpo_missing');

        return $this->views->make('pages.privacy', [
            'updatedAt' => $updated,
            'dpoEmail' => $email,
            'dpoText' => $emailText,
            'sections' => self::legalSections(__('privacy.policy.sections'), [':dpo_email' => $emailText]),
        ]);
    }

    /**
     * Normalises a legal text from a lang file (privacy policy, terms) for pages/privacy and pages/terms:
     * `[title, paragraphs?, items?, link? => [text, route]]` → strings with `$tokens` replaced and link URLs resolved.
     *
     * @param  array<string, string>  $tokens
     * @return list<array{title: string, paragraphs: list<string>, items: list<string>, link: array{text: string, url: string}|null}>
     */
    public static function legalSections(mixed $raw, array $tokens): array
    {
        $strings = static fn (mixed $list): array => array_values(array_map(
            static fn (string $line): string => strtr($line, $tokens),
            array_filter(is_array($list) ? $list : [], is_string(...)),
        ));

        $sections = [];
        foreach (is_array($raw) ? $raw : [] as $section) {
            if (! is_array($section) || ! is_string($section['title'] ?? null)) {
                continue;
            }
            $link = $section['link'] ?? null;
            $sections[] = [
                'title' => $section['title'],
                'paragraphs' => $strings($section['paragraphs'] ?? []),
                'items' => $strings($section['items'] ?? []),
                'link' => is_array($link) && is_string($link['text'] ?? null) && is_string($link['route'] ?? null)
                    ? ['text' => $link['text'], 'url' => route($link['route'])]
                    : null,
            ];
        }

        return $sections;
    }

    /** ISO date from a content file → Tehran-midnight date, or null when missing/invalid. */
    public static function date(string $iso): ?CarbonImmutable
    {
        if (preg_match('/^\d{4}-\d{2}-\d{2}$/', $iso) !== 1) {
            return null;
        }

        try {
            return CarbonImmutable::createFromFormat('!Y-m-d', $iso, 'Asia/Tehran') ?: null;
        } catch (Throwable) {
            return null;
        }
    }

    private static function text(string $key): string
    {
        $text = __($key);

        return is_string($text) ? $text : '';
    }
}
