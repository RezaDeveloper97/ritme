<?php

declare(strict_types=1);

namespace App\View\Components;

use Illuminate\Contracts\View\View;
use Illuminate\Foundation\Vite;
use Illuminate\View\Component;
use InvalidArgumentException;
use Throwable;

/**
 * <x-icon name="drop" class="size-5" /> — one <symbol> of the hashed SVG sprite built by tools/build-sprite.mjs.
 *
 *  - Decorative by default (aria-hidden); pass label="…" to expose it as role="img" with a <title>.
 *  - Colour comes from CSS (`currentColor`), stroke width from the `stroke` prop (the design uses 1.8 / 2 / 2.2 / 2.6).
 *  - `size` sets the width/height attributes so the icon has its box before CSS loads; a class (size-5) overrides it.
 *  - Unknown names throw outside production and render nothing in production.
 *  - When the sprite is not available as a same-origin build asset (Vite dev server, tests without a build) the
 *    symbol's geometry is inlined instead, because <use> cannot reference a cross-origin file.
 */
final class Icon extends Component
{
    public const SPRITE_ENTRY = 'resources/svg/sprite.svg';

    /** @var array<string, string> name => inner markup, per request */
    private static array $inline = [];

    public readonly string $spriteHref;

    public readonly string $paths;

    public readonly bool $missing;

    public function __construct(
        public readonly string $name,
        public readonly ?string $label = null,
        public readonly int|float|string $stroke = 2,
        public readonly int|string $size = 24,
        ?Vite $vite = null,
    ) {
        $this->missing = ! self::exists($name);

        if ($this->missing) {
            if (! app()->isProduction()) {
                throw new InvalidArgumentException("Unknown icon [{$name}]. Available icons live in resources/svg/icons.");
            }

            $this->spriteHref = '';
            $this->paths = '';

            return;
        }

        $url = self::spriteUrl($vite ?? app(Vite::class));
        $this->spriteHref = $url === '' ? '' : $url.'#'.$name;
        $this->paths = $url === '' ? self::inlineMarkup($name) : '';
    }

    public static function directory(): string
    {
        return resource_path('svg/icons');
    }

    public static function exists(string $name): bool
    {
        return preg_match('/^[a-z0-9]+(?:-[a-z0-9]+)*$/', $name) === 1
            && is_file(self::directory().'/'.$name.'.svg');
    }

    /**
     * @return list<string>
     */
    public static function names(): array
    {
        $names = array_map(
            static fn (string $file): string => basename($file, '.svg'),
            glob(self::directory().'/*.svg') ?: [],
        );
        sort($names);

        return $names;
    }

    /** Clears the per-request memo (tests, Octane-style workers). */
    public static function flush(): void
    {
        self::$inline = [];
    }

    public function shouldRender(): bool
    {
        return ! $this->missing;
    }

    public function render(): View
    {
        return view('components.icon');
    }

    private static function spriteUrl(Vite $vite): string
    {
        if ($vite->isRunningHot()) {
            return '';
        }

        try {
            $url = (string) $vite->asset(self::SPRITE_ENTRY);
        } catch (Throwable) {
            return ''; // no build yet (or sprite missing from the manifest): fall back to inline geometry
        }

        // <use> only follows same-origin URLs: drop scheme + host so www/non-www or a mismatched APP_URL still work.
        $path = parse_url($url, PHP_URL_PATH);

        return is_string($path) && $path !== '' ? $path : '';
    }

    private static function inlineMarkup(string $name): string
    {
        if (isset(self::$inline[$name])) {
            return self::$inline[$name];
        }

        $svg = (string) file_get_contents(self::directory().'/'.$name.'.svg');
        $inner = preg_replace('/^.*?<svg\b[^>]*>|<\/svg>\s*$/s', '', $svg) ?? '';

        return self::$inline[$name] = trim($inner);
    }
}
