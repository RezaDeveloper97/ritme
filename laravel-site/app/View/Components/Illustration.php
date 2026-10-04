<?php

declare(strict_types=1);

namespace App\View\Components;

use App\Support\Cache\CacheAside;
use App\Support\Cache\CacheKey;
use Illuminate\View\Component;
use InvalidArgumentException;

/**
 * <x-illustration name="hero-orbit" class="…" /> — inlines an SVGO-optimised file from resources/svg/illustrations.
 *
 *  - width/height attributes default to the viewBox size (prevents CLS); override with :width / :height.
 *  - Decorative by default (aria-hidden); label="…" → role="img" + <title>.
 *  - Parsed markup is cached through CacheAside in the `media` namespace, keyed by name + file mtime.
 *  - Unknown names throw outside production and render nothing in production.
 */
final class Illustration extends Component
{
    public readonly string $inner;

    public readonly string $viewBox;

    public readonly ?string $preserveAspectRatio;

    public readonly int|string $width;

    public readonly int|string $height;

    public readonly bool $missing;

    public function __construct(
        public readonly string $name,
        int|string|null $width = null,
        int|string|null $height = null,
        public readonly ?string $label = null,
        ?CacheAside $cache = null,
    ) {
        $path = self::path($name);
        $this->missing = $path === null;

        if ($path === null) {
            if (! app()->isProduction()) {
                throw new InvalidArgumentException("Unknown illustration [{$name}]. Files live in resources/svg/illustrations.");
            }
            $this->inner = $this->viewBox = '';
            $this->preserveAspectRatio = null;
            $this->width = $this->height = 0;

            return;
        }

        $cache ??= app(CacheAside::class);
        /** @var array{inner: string, viewBox: string, par: string|null, w: int, h: int} $data */
        $data = $cache->remember(
            CacheKey::make('media', 'illustration', $name, (string) filemtime($path)),
            null,
            static fn (): array => self::parse((string) file_get_contents($path)),
        );

        $this->inner = $data['inner'];
        $this->viewBox = $data['viewBox'];
        $this->preserveAspectRatio = $data['par'];
        $this->width = $width ?? $data['w'];
        $this->height = $height ?? $data['h'];
    }

    public static function path(string $name): ?string
    {
        if (preg_match('/^[a-z0-9]+(?:-[a-z0-9]+)*$/', $name) !== 1) {
            return null;
        }

        $path = resource_path('svg/illustrations/'.$name.'.svg');

        return is_file($path) ? $path : null;
    }

    /**
     * @return array{inner: string, viewBox: string, par: string|null, w: int, h: int}
     */
    public static function parse(string $svg): array
    {
        preg_match('/<svg\b([^>]*)>/', $svg, $root);
        $attrs = $root[1] ?? '';
        preg_match('/viewBox="([^"]+)"/', $attrs, $vb);
        preg_match('/preserveAspectRatio="([^"]+)"/', $attrs, $par);

        $viewBox = $vb[1] ?? '0 0 100 100';
        $parts = preg_split('/[\s,]+/', trim($viewBox)) ?: [];

        return [
            'inner' => trim(preg_replace('/^.*?<svg\b[^>]*>|<\/svg>\s*$/s', '', $svg) ?? ''),
            'viewBox' => $viewBox,
            'par' => $par[1] ?? null,
            'w' => (int) round((float) ($parts[2] ?? 100)),
            'h' => (int) round((float) ($parts[3] ?? 100)),
        ];
    }

    public function shouldRender(): bool
    {
        return ! $this->missing;
    }

    public function render(): string
    {
        return <<<'BLADE'
            @php($labelled = $label !== null && $label !== '')
            <svg {{ $attributes->except(['aria-hidden', 'role', 'viewBox']) }} xmlns="http://www.w3.org/2000/svg" viewBox="{{ $viewBox }}" width="{{ $width }}" height="{{ $height }}" @if ($preserveAspectRatio) preserveAspectRatio="{{ $preserveAspectRatio }}" @endif @if ($labelled) role="img" aria-label="{{ $label }}" @else aria-hidden="true" @endif focusable="false">@if ($labelled)<title>{{ $label }}</title>@endif{!! $inner !!}</svg>
            BLADE;
    }
}
