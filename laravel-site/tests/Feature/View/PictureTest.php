<?php

declare(strict_types=1);

use App\Domain\Media\Contracts\MediaRepository;
use App\Domain\Media\Data\MediaData;
use App\View\Components\Picture;
use Illuminate\Support\Facades\Blade;
use Illuminate\View\ViewException;

/**
 * @param  list<string>  $formats
 * @param  array<string, mixed>  $overrides
 */
function pictureMedia(int $id = 7, array $formats = ['avif', 'webp', 'jpg'], array $overrides = [], int $w = 2400, int $h = 1600): MediaData
{
    $variants = [];
    foreach (['mobile_480' => 480, 'mobile_768' => 768, 'desktop_1280' => 1280, 'desktop_1920' => 1920] as $name => $vw) {
        foreach ($formats as $format) {
            $variants[$name][$format] = ['w' => $vw, 'h' => intdiv($vw * $h, $w), 'url' => "/media/{$id}/{$name}.{$format}", 'size' => 1000];
        }
    }
    $variants['og']['jpg'] = ['w' => 1200, 'h' => 630, 'url' => "/media/{$id}/og.jpg", 'size' => 1000];

    return MediaData::fromArray(array_merge([
        'id' => $id,
        'url' => "/media/{$id}/original.jpg",
        'mime' => 'image/jpeg',
        'extension' => 'jpg',
        'width' => $w,
        'height' => $h,
        'alt' => 'زنی در حال نوشیدن چای',
        'dominantColor' => '#aabbcc',
        'lqip' => 'data:image/webp;base64,AAAA',
        'variants' => $variants,
        'optimized' => true,
    ], $overrides));
}

function renderPicture(string $template, array $data = []): string
{
    app('view')->flushState();

    return normalisePicture(Blade::render($template, $data, deleteCachedView: true));
}

/**
 * Renders the template followed by the `head` stack (pushes are flushed when the outermost render finishes) and
 * returns [body, head].
 *
 * @return array{0: string, 1: string}
 */
function renderPictureWithHead(string $template, array $data = []): array
{
    $html = renderPicture($template.'<!--HEAD-->@stack(\'head\')', $data);

    return explode('<!--HEAD-->', $html, 2) + [1 => ''];
}

function normalisePicture(string $html): string
{
    return (string) preg_replace(['/\s+/', '/\s+>/', '/<(\w+)\s+/'], [' ', '>', '<$1 '], $html);
}

it('renders avif + webp sources, a jpg fallback img and lazy defaults', function (): void {
    $html = renderPicture('<x-picture :media="$m" sizes="(max-width: 768px) 100vw, 50vw" class="w-full" />', ['m' => pictureMedia()]);

    expect($html)
        ->toContain('<source type="image/avif" srcset="/media/7/mobile_480.avif 480w, /media/7/mobile_768.avif 768w, /media/7/desktop_1280.avif 1280w, /media/7/desktop_1920.avif 1920w" sizes="(max-width: 768px) 100vw, 50vw">')
        ->toContain('<source type="image/webp" srcset="/media/7/mobile_480.webp 480w')
        ->toContain('src="/media/7/desktop_1280.jpg"')
        ->toContain('srcset="/media/7/mobile_480.jpg 480w, /media/7/mobile_768.jpg 768w, /media/7/desktop_1280.jpg 1280w, /media/7/desktop_1920.jpg 1920w, /media/7/original.jpg 2400w"')
        ->toContain('width="2400" height="1600"', 'alt="زنی در حال نوشیدن چای"', 'loading="lazy"', 'decoding="async"', 'class="bg-lavender w-full"')
        ->not->toContain('fetchpriority', 'style=', 'image/jpeg', 'base64', 'http')
        ->and(substr_count($html, '<source'))->toBe(2)
        ->and(renderPictureWithHead('<x-picture :media="$m" />', ['m' => pictureMedia()])[1])->not->toContain('preload');
});

it('falls back to a plain img when no variants exist yet', function (): void {
    $html = renderPicture('<x-picture :media="$m" />', ['m' => pictureMedia(formats: [], overrides: ['variants' => []])]);

    expect($html)
        ->toContain('src="/media/7/original.jpg"', 'width="2400" height="1600"', 'alt="زنی در حال نوشیدن چای"')
        ->not->toContain('<source', 'srcset', 'sizes=');
});

it('omits the avif source when avif was not encoded', function (): void {
    $html = renderPicture('<x-picture :media="$m" />', ['m' => pictureMedia(formats: ['webp', 'jpg'])]);

    expect($html)->toContain('type="image/webp"', 'sizes="100vw"')->not->toContain('avif')
        ->and(substr_count($html, '<source'))->toBe(1);
});

it('keeps transparent png images free of the placeholder background', function (): void {
    $html = renderPicture('<x-picture :media="$m" />', ['m' => pictureMedia(formats: ['webp', 'png'], overrides: ['url' => '/media/7/original.png', 'mime' => 'image/png', 'extension' => 'png'])]);

    expect($html)->toContain('src="/media/7/desktop_1280.png"', '/media/7/original.png 2400w')->not->toContain('bg-lavender');
});

it('adds art-directed mobile sources with their own dimensions first', function (): void {
    $html = renderPicture('<x-picture :media="$m" :mobile="$mm" picture-class="block" />', [
        'm' => pictureMedia(),
        'mm' => pictureMedia(9, ['webp', 'jpg'], ['alt' => null], 900, 1200),
    ]);

    $mobileWebp = strpos($html, 'media="(max-width: 767px)" type="image/webp" srcset="/media/9/mobile_480.webp 480w');
    $desktopAvif = strpos($html, '<source type="image/avif"');

    expect($html)
        ->toContain('<picture class="block">')
        ->toContain('media="(max-width: 767px)" type="image/jpeg" srcset="/media/9/mobile_480.jpg 480w')
        ->toContain('width="900" height="1200"', 'width="2400" height="1600"')
        ->and($mobileWebp)->toBeInt()
        ->and($desktopAvif)->toBeInt()->toBeGreaterThan($mobileWebp)
        ->and(substr_count($html, '<source'))->toBe(4);
});

it('preloads the LCP image once and loads it eagerly with high priority', function (): void {
    [$html, $head] = renderPictureWithHead('<x-picture :media="$m" sizes="50vw" priority /><x-picture :media="$m" priority />', ['m' => pictureMedia()]);

    expect($html)->toContain('loading="eager"', 'fetchpriority="high"', 'decoding="async"')->not->toContain('loading="lazy"')
        ->and($head)->toContain('<link rel="preload" as="image" imagesrcset="/media/7/mobile_480.avif 480w', 'imagesizes="50vw" type="image/avif" fetchpriority="high">')
        ->and(substr_count($head, 'rel="preload"'))->toBe(1);
});

it('preloads mobile and desktop candidates separately with art direction', function (): void {
    [, $head] = renderPictureWithHead('<x-picture :media="$m" :mobile="$mm" priority />', ['m' => pictureMedia(), 'mm' => pictureMedia(9, ['webp', 'jpg'])]);

    expect($head)
        ->toContain('imagesrcset="/media/9/mobile_480.webp 480w', 'type="image/webp" media="(max-width: 767px)"')
        ->toContain('imagesrcset="/media/7/mobile_480.avif 480w', 'type="image/avif" media="(min-width: 768px)"');
});

it('lets loading, fetchpriority, decoding and dimensions be overridden', function (): void {
    $html = renderPicture('<x-picture :media="$m" loading="eager" fetchpriority="low" decoding="sync" :width="600" :height="400" />', ['m' => pictureMedia()]);

    expect($html)->toContain('loading="eager"', 'fetchpriority="low"', 'decoding="sync"', 'width="600" height="400"');
});

it('requires an alt unless the image is decorative', function (): void {
    $m = pictureMedia(overrides: ['alt' => null]);

    expect(renderPicture('<x-picture :media="$m" decorative />', ['m' => $m]))->toContain('alt=""')
        ->and(renderPicture('<x-picture :media="$m" alt="متن جایگزین" />', ['m' => $m]))->toContain('alt="متن جایگزین"');

    renderPicture('<x-picture :media="$m" />', ['m' => $m]);
})->throws(ViewException::class, 'needs an alt');

it('renders svg media as a sized img without sources', function (): void {
    $svg = pictureMedia(overrides: ['url' => '/media/7/logo.svg', 'mime' => 'image/svg+xml', 'extension' => 'svg', 'variants' => []], w: 120, h: 40);

    expect(renderPicture('<x-picture :media="$m" />', ['m' => $svg]))
        ->toContain('src="/media/7/logo.svg"', 'width="120" height="40"')->not->toContain('<source', 'srcset');
});

it('resolves media ids through the repository and renders nothing for missing media', function (): void {
    $repo = Mockery::mock(MediaRepository::class);
    $repo->shouldReceive('find')->with(7)->andReturn(pictureMedia());
    $repo->shouldReceive('find')->with(404)->andReturn(null);
    app()->instance(MediaRepository::class, $repo);

    expect(renderPicture('<x-picture :media="7" />'))->toContain('src="/media/7/desktop_1280.jpg"')
        ->and(trim(renderPicture('<x-picture :media="404" />')))->toBe('')
        ->and(trim(renderPicture('<x-picture :media="null" />')))->toBe('');
});

it('builds absolute variant urls for meta tags (media_url)', function (): void {
    $repo = Mockery::mock(MediaRepository::class);
    $repo->shouldReceive('find')->with(7)->andReturn(pictureMedia());
    $repo->shouldReceive('find')->with(8)->andReturn(null);
    app()->instance(MediaRepository::class, $repo);

    expect(Picture::mediaUrl(pictureMedia(), 'og'))->toBe(url('/media/7/og.jpg'))
        ->and(Picture::mediaUrl(7, 'missing'))->toBe(url('/media/7/original.jpg'))
        ->and(Picture::mediaUrl(7, 'desktop_1280', 'webp'))->toBe(url('/media/7/desktop_1280.webp'))
        ->and(Picture::mediaUrl(8, 'og'))->toBeNull()
        ->and(Picture::mediaUrl(null))->toBeNull();
});
