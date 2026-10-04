<?php

declare(strict_types=1);

use App\Support\Cache\CacheKey;
use App\Support\Cache\NamespaceVersions;
use App\View\Components\Icon;
use Illuminate\Foundation\Vite;
use Illuminate\Support\Facades\Blade;
use Illuminate\View\ViewException;

beforeEach(fn () => Icon::flush());

it('extracted the 76 audited icons and 28 illustrations', function (): void {
    expect(Icon::names())->toHaveCount(76)
        ->toContain('drop', 'download', 'check', 'arrow-left', 'chevron-left', 'cart', 'flask')
        ->not->toContain('cart-alt');
    expect(glob(resource_path('svg/illustrations/*.svg')))->toHaveCount(28);
});

it('keeps icon sources colour-free and free of inline styles', function (): void {
    foreach (glob(resource_path('svg/icons/*.svg')) ?: [] as $file) {
        $svg = (string) file_get_contents($file);
        expect($svg)->not->toMatch('/#[0-9a-f]{3,8}/i')->not->toContain('style=');
    }
});

it('renders a decorative sprite reference with a same-origin hashed URL', function (): void {
    if (! is_file(public_path('build/manifest.json'))) {
        $this->markTestSkipped('Run npm run build first.');
    }
    app()->forgetInstance(Vite::class); // the base TestCase fakes Vite

    $html = Blade::render('<x-icon name="drop" class="size-5" />');

    expect($html)
        ->toMatch('#<use href="/build/assets/sprite-[A-Za-z0-9_-]+\.svg\#drop"/>#')
        ->toContain('aria-hidden="true"', 'focusable="false"', 'class="size-5"', 'stroke="currentColor"', 'width="24"')
        ->not->toContain('role="img"', 'http');
});

it('inlines the geometry when no build manifest is available', function (): void {
    $html = Blade::render('<x-icon name="info" />'); // TestCase::withoutVite() → asset() returns ''

    expect($html)->not->toContain('<use')->toContain('<circle cx="12" cy="12" r="9"/>');
});

it('exposes a labelled icon as role=img with a title', function (): void {
    $html = Blade::render('<x-icon name="heart" label="علاقه‌مندی" :stroke="1.8" size="18" />');

    expect($html)
        ->toContain('role="img"', 'aria-label="علاقه‌مندی"', '<title>علاقه‌مندی</title>', 'stroke-width="1.8"', 'width="18"')
        ->not->toContain('aria-hidden');
});

it('throws on an unknown icon outside production', function (): void {
    Blade::render('<x-icon name="no-such-icon" />');
})->throws(ViewException::class, 'Unknown icon [no-such-icon]');

it('renders nothing for an unknown icon in production', function (): void {
    app()->detectEnvironment(fn (): string => 'production');

    expect(trim(Blade::render('<x-icon name="no-such-icon" />')))->toBe('');
});

it('rejects path traversal in icon names', function (): void {
    expect(Icon::exists('../icons/drop'))->toBeFalse()->and(Icon::exists('drop'))->toBeTrue();
});

it('inlines an illustration with viewBox-derived width and height', function (): void {
    $html = Blade::render('<x-illustration name="hero-orbit" class="absolute" />');

    expect($html)
        ->toContain('viewBox="0 0 100 100"', 'width="100"', 'height="100"', 'aria-hidden="true"', 'class="absolute"', '<circle')
        ->not->toContain('style=');
});

it('keeps preserveAspectRatio and accepts explicit dimensions', function (): void {
    $html = Blade::render('<x-illustration name="chart-sparkline" width="200" height="50" label="نمودار" />');

    expect($html)->toContain('preserveAspectRatio="none"', 'height="50"', 'role="img"', '<title>نمودار</title>');
});

it('caches parsed illustrations in the media namespace', function (): void {
    Blade::render('<x-illustration name="map-place" />');
    $path = resource_path('svg/illustrations/map-place.svg');
    $key = CacheKey::make('media', 'illustration', 'map-place', (string) filemtime($path));

    $versions = app(NamespaceVersions::class);
    expect($versions->store()->get($versions->qualify($key)))->toBeArray();
});

it('throws on an unknown illustration outside production', function (): void {
    Blade::render('<x-illustration name="nope" />');
})->throws(ViewException::class, 'Unknown illustration [nope]');
