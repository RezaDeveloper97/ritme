<?php

declare(strict_types=1);

namespace App\View\Components\Layout;

use App\Domain\Settings\Contracts\SettingsRepository;
use Illuminate\Contracts\View\View;
use Illuminate\Foundation\Vite;
use Illuminate\View\Component;
use Throwable;

/**
 * <x-layout.assets /> — the layout's non-SEO head tags: theme-color (PWA settings), preloads for the two
 * above-the-fold font files (Lalezar for the logo/h1, Vazirmatn 600 for nav and body copy) and the Vite entries.
 * Everything else (other weights, Latin subsets) loads on demand through unicode-range.
 */
final class Assets extends Component
{
    public const PRELOAD_FONTS = [
        'resources/fonts/lalezar-arabic-400-normal.woff2',
        'resources/fonts/vazirmatn-arabic-600-normal.woff2',
    ];

    /** @var list<string> */
    public readonly array $fonts;

    public readonly ?string $themeColor;

    public function __construct(?Vite $vite = null)
    {
        $vite ??= app(Vite::class);
        $fonts = [];
        if (! $vite->isRunningHot()) {
            foreach (self::PRELOAD_FONTS as $font) {
                try {
                    $url = (string) $vite->asset($font);
                } catch (Throwable) {
                    $url = ''; // no build yet
                }
                if ($url !== '') {
                    $fonts[] = $url;
                }
            }
        }
        $this->fonts = $fonts;

        $color = app(SettingsRepository::class)->all()->pwa->themeColor;
        $this->themeColor = preg_match('/^#[0-9a-f]{3,8}$/i', $color) === 1 ? $color : null;
    }

    public function render(): View
    {
        return view('components.layout.assets');
    }
}
