<?php

declare(strict_types=1);

use App\Domain\Media\Actions\StoreMedia;
use App\Domain\Media\Contracts\MediaRepository;
use App\Domain\Media\Data\MediaUpload;
use App\Domain\Media\Models\Media;
use App\Domain\Media\Repositories\CachedMediaRepository;
use App\Domain\Seo\Contracts\OgImageResolver;
use App\Support\Cache\NamespaceVersions;
use Illuminate\Support\Facades\DB;
use Illuminate\Support\Facades\Storage;
use Tests\Feature\Media\MediaFixtures;

function libraryMedia(string $path, ?string $alt = 'تصویر'): Media
{
    return app(StoreMedia::class)->handle(new MediaUpload($path, basename($path), alt: $alt));
}

function rawMedia(array $attributes = []): Media
{
    return Media::query()->create([
        'disk' => 'public', 'directory' => '2026/10/abcdefghij', 'filename' => 'pic.jpg', 'mime' => 'image/jpeg',
        'size' => 1000, 'width' => 1600, 'height' => 900, 'variants' => [], 'hash' => str_repeat('a', 64),
        ...$attributes,
    ]);
}

function mediaQueries(Closure $callback): int
{
    DB::flushQueryLog();
    DB::enableQueryLog();
    $callback();
    $count = count(array_filter(DB::getQueryLog(), static fn (array $q): bool => str_contains((string) $q['query'], '"media"')));
    DB::disableQueryLog();

    return $count;
}

beforeEach(function (): void {
    Storage::fake('public');
});

it('writes the public disk into public/media with root-relative /media URLs by default', function (): void {
    // Re-read the shipped config (Storage::fake replaced the runtime disk).
    $disks = require config_path('filesystems.php');

    expect($disks['disks']['public']['root'])->toBe(public_path('media'))
        ->and($disks['disks']['public']['url'])->toBe('/media')
        ->and(config('media.disk'))->toBe('public');
});

it('resolves the repository contract to the cached decorator and caches reads in the media namespace', function (): void {
    $media = libraryMedia(MediaFixtures::jpeg(1000, 700));
    $repository = app(MediaRepository::class);

    expect($repository)->toBeInstanceOf(CachedMediaRepository::class);

    expect(mediaQueries(fn () => $repository->find($media->id)))->toBe(1)
        ->and(mediaQueries(fn () => $repository->find($media->id)))->toBe(0)
        ->and(mediaQueries(fn () => $repository->find(424242)))->toBe(1)
        ->and(mediaQueries(fn () => $repository->find(424242)))->toBe(0);

    $media->update(['alt' => 'جدید']);

    expect($repository->find($media->id)?->alt)->toBe('جدید')
        ->and(array_keys($repository->findMany([$media->id, 424242])))->toBe([$media->id]);
});

it('returns DTOs with resolved URLs, srcsets and LQIP', function (): void {
    $media = libraryMedia(MediaFixtures::jpeg(2000, 1000));
    $data = app(MediaRepository::class)->find($media->id);

    expect($data)->not->toBeNull()
        ->and($data->optimized)->toBeTrue()
        ->and($data->formats())->toBe(['avif', 'webp', 'jpg'])
        ->and($data->fallbackFormat())->toBe('jpg')
        ->and($data->url('desktop_1280', 'avif'))->toEndWith('-desktop_1280.avif')
        ->and($data->url('thumb'))->toEndWith('-thumb.jpg')
        ->and($data->srcset('webp'))->toMatch('/-mobile_480\.webp 480w, .*-mobile_768\.webp 768w, .*-desktop_1280\.webp 1280w, .*-desktop_1920\.webp 1920w$/')
        ->and($data->srcset('jpg'))->toEndWith('.jpg 2000w')
        ->and($data->lqip)->toStartWith('data:image/webp;base64,');
});

it('falls back to the original when variants are missing', function (): void {
    $data = app(MediaRepository::class)->find(rawMedia()->id);

    expect($data->optimized)->toBeFalse()
        ->and($data->url('desktop_1280', 'avif'))->toBe($data->url)
        ->and($data->url('og'))->toBe($data->url)
        ->and($data->url)->toEndWith('2026/10/abcdefghij/pic.jpg')
        ->and($data->formats())->toBe([])
        ->and($data->srcset('avif'))->toBe('')
        ->and($data->srcset('jpg'))->toEndWith('pic.jpg 1600w');
});

it('bumps the media, seo and pages namespaces on change', function (): void {
    $versions = app(NamespaceVersions::class);
    $before = [$versions->version('media'), $versions->version('seo'), $versions->version('pages')];

    rawMedia();

    expect([$versions->version('media'), $versions->version('seo'), $versions->version('pages')])
        ->toBe([$before[0] + 1, $before[1] + 1, $before[2] + 1]);
});

it('deletes every file and the directory when media is deleted', function (): void {
    $media = libraryMedia(MediaFixtures::jpeg(1400, 900));
    $paths = $media->allPaths();

    expect(count($paths))->toBeGreaterThan(10);
    foreach ($paths as $path) {
        Storage::disk('public')->assertExists($path);
    }

    $media->delete();

    foreach ($paths as $path) {
        Storage::disk('public')->assertMissing($path);
    }
    expect(Storage::disk('public')->allFiles())->toBe([])
        ->and(Storage::disk('public')->directories(dirname($media->directory)))->toBe([]);
});

it('resolves OG images from the og variant with an absolute URL', function (): void {
    $media = libraryMedia(MediaFixtures::jpeg(1300, 700), alt: null);

    $image = app(OgImageResolver::class)->resolve($media->id, 'عنوان صفحه');

    expect($image)->not->toBeNull()
        ->and($image->url)->toStartWith('http')
        ->and($image->url)->toEndWith('-og.jpg')
        ->and([$image->width, $image->height])->toBe([1200, 630])
        ->and($image->type)->toBe('image/jpeg')
        ->and($image->alt)->toBe('عنوان صفحه');
});

it('falls back to the original for OG, and returns null for SVG or missing media', function (): void {
    $plain = rawMedia(['alt' => 'متن جایگزین']);
    $svg = rawMedia(['mime' => 'image/svg+xml', 'filename' => 'logo.svg', 'hash' => str_repeat('b', 64)]);
    $resolver = app(OgImageResolver::class);

    $image = $resolver->resolve($plain->id, 'x');

    expect($image?->url)->toEndWith('pic.jpg')
        ->and([$image?->width, $image?->height])->toBe([1600, 900])
        ->and($image?->alt)->toBe('متن جایگزین')
        ->and($resolver->resolve($svg->id, 'x'))->toBeNull()
        ->and($resolver->resolve(999, 'x'))->toBeNull();
});

it('regenerates an on-demand preset and merges it into existing variants', function (): void {
    $media = libraryMedia(MediaFixtures::jpeg(1300, 900));
    $before = $media->variants;

    $this->artisan('media:regenerate', ['--id' => [$media->id], '--preset' => ['square']])
        ->expectsOutputToContain('Regenerated 1 of 1 media.')
        ->assertSuccessful();

    $after = $media->refresh()->variants;
    expect($after)->toHaveKey('square')
        ->and(array_diff_key($after, ['square' => true]))->toBe($before)
        ->and([$after['square']['jpg']['w'], $after['square']['jpg']['h']])->toBe([600, 600]);
    Storage::disk('public')->assertExists($after['square']['webp']['path']);
});

it('regenerates everything and removes files that are no longer produced', function (): void {
    $media = libraryMedia(MediaFixtures::jpeg(1300, 900));
    $this->artisan('media:regenerate', ['--preset' => ['square']])->assertSuccessful();
    $squarePath = $media->refresh()->variants['square']['jpg']['path'];

    config(['media.formats' => ['webp', 'fallback']]);
    $this->artisan('media:regenerate')->assertSuccessful();

    $variants = $media->refresh()->variants;
    expect($variants)->not->toHaveKey('square')
        ->and(array_keys($variants['mobile_480']))->toBe(['webp', 'jpg']);
    Storage::disk('public')->assertMissing($squarePath);
    expect(collect(Storage::disk('public')->allFiles())->filter(fn (string $p): bool => str_ends_with($p, '.avif'))->all())->toBe([]);
});

it('rejects unknown presets in media:regenerate', function (): void {
    $this->artisan('media:regenerate', ['--preset' => ['huge']])
        ->expectsOutputToContain('Unknown preset(s): huge')
        ->assertFailed();
});
