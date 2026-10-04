<?php

declare(strict_types=1);

use App\Domain\Media\Actions\StoreMedia;
use App\Domain\Media\Data\MediaUpload;
use App\Domain\Media\Exceptions\InvalidMediaException;
use App\Domain\Media\Jobs\OptimizeMedia;
use App\Domain\Media\Models\Media;
use Illuminate\Contracts\Filesystem\Filesystem;
use Illuminate\Support\Facades\Queue;
use Illuminate\Support\Facades\Storage;
use Intervention\Image\ImageManager;
use Tests\Feature\Media\MediaFixtures;

function storeMedia(string $path, ?string $name = null, float $focalX = 0.5, float $focalY = 0.5): Media
{
    return app(StoreMedia::class)->handle(new MediaUpload($path, $name ?? basename($path), alt: 'تصویر', focalX: $focalX, focalY: $focalY));
}

function mediaDisk(): Filesystem
{
    return Storage::disk('public');
}

/**
 * @return array{0: int, 1: int, mime: string}
 */
function storedImageSize(string $path): array
{
    $info = getimagesizefromstring((string) mediaDisk()->get($path));
    expect($info)->not->toBeFalse();

    return $info;
}

beforeEach(function (): void {
    Storage::fake('public');
});

it('turns a 4000x3000 JPEG into avif/webp/jpg at 480/768/1280/1920 plus thumb and og, all smaller than the source', function (): void {
    $source = MediaFixtures::jpeg(4000, 3000);
    $sourceSize = filesize($source);

    $media = storeMedia($source);

    expect($media->mime)->toBe('image/jpeg')
        ->and($media->width)->toBe(2560)
        ->and($media->height)->toBe(1920)
        ->and($media->optimized_at)->not->toBeNull()
        ->and($media->hash)->toBe(hash_file('sha256', $source));

    $expected = [
        'mobile_480' => [480, 360], 'mobile_768' => [768, 576],
        'desktop_1280' => [1280, 960], 'desktop_1920' => [1920, 1440],
        'thumb' => [320, 320],
    ];
    foreach ($expected as $name => [$w, $h]) {
        expect(array_keys($media->variants[$name]))->toBe(['avif', 'webp', 'jpg']);
        foreach ($media->variants[$name] as $format => $file) {
            $info = storedImageSize($file['path']);
            expect([$info[0], $info[1]])->toBe([$w, $h])
                ->and([$file['w'], $file['h']])->toBe([$w, $h])
                ->and($info['mime'])->toBe(['avif' => 'image/avif', 'webp' => 'image/webp', 'jpg' => 'image/jpeg'][$format])
                ->and($file['size'])->toBe(mediaDisk()->size($file['path']))
                ->and($file['size'])->toBeLessThan($sourceSize);
        }
    }

    expect(array_keys($media->variants['og']))->toBe(['jpg']);
    $og = storedImageSize($media->variants['og']['jpg']['path']);
    expect([$og[0], $og[1]])->toBe([1200, 630])
        ->and($media->variants)->not->toHaveKey('square'); // on demand only

    expect(mediaDisk()->size($media->path()))->toBeLessThan($sourceSize);
});

it('stores a dominant colour and an LQIP WebP of at most 600 bytes', function (): void {
    $media = storeMedia(MediaFixtures::jpeg(1600, 1200));

    expect($media->dominant_color)->toMatch('/^#[0-9a-f]{6}$/')
        ->and($media->lqip)->toStartWith('data:image/webp;base64,');

    $bytes = base64_decode(substr((string) $media->lqip, strlen('data:image/webp;base64,')), true);
    expect($bytes)->not->toBeFalse()
        ->and(strlen((string) $bytes))->toBeLessThanOrEqual(600);
});

it('auto-orients by EXIF and strips metadata from the original and every variant', function (): void {
    $source = MediaFixtures::jpeg(1600, 1200, orientation: 6);
    expect((string) file_get_contents($source))->toContain(MediaFixtures::EXIF_MARKER);

    $media = storeMedia($source);

    expect([$media->width, $media->height])->toBe([1200, 1600]);

    // Orientation 6 = rotate 90° clockwise: the red left half ends up on top.
    $image = ImageManager::gd()->read((string) mediaDisk()->get($media->path()));
    $top = (clone $image)->crop(1200, 400, 0, 0)->resize(1, 1)->pickColor(0, 0)->toArray();
    $bottom = (clone $image)->crop(1200, 400, 0, 1200)->resize(1, 1)->pickColor(0, 0)->toArray();
    expect($top[0])->toBeGreaterThan($top[2])
        ->and($bottom[2])->toBeGreaterThan($bottom[0]);

    foreach ($media->allPaths() as $path) {
        $bytes = (string) mediaDisk()->get($path);
        expect($bytes)->not->toContain(MediaFixtures::EXIF_MARKER)
            ->and($bytes)->not->toContain("Exif\0\0");
    }

    expect(storedImageSize($media->variants['mobile_768']['jpg']['path'])[1])->toBe(1024)
        ->and($media->variants)->not->toHaveKey('desktop_1280'); // 1200 px wide after rotation
});

it('never upscales: small sources skip wider widths and crops shrink to fit', function (): void {
    $media = storeMedia(MediaFixtures::jpeg(600, 400));

    expect([$media->width, $media->height])->toBe([600, 400])
        ->and(array_keys($media->variants))->toBe(['mobile_480', 'thumb', 'og']);

    $og = storedImageSize($media->variants['og']['jpg']['path']);
    expect([$og[0], $og[1]])->toBe([600, 315]);

    $thumb = storedImageSize($media->variants['thumb']['webp']['path']);
    expect([$thumb[0], $thumb[1]])->toBe([320, 320]);
});

it('caps the stored original at 2560 px', function (): void {
    config(['media.presets' => []]); // only the original matters here

    $media = storeMedia(MediaFixtures::jpeg(3000, 5000));

    expect([$media->width, $media->height])->toBe([1536, 2560]);
    $info = storedImageSize($media->path());
    expect([$info[0], $info[1]])->toBe([1536, 2560]);
});

it('crops thumb and og around the focal point', function (): void {
    // Focal point far right: the blue right half fills the square thumb.
    $media = storeMedia(MediaFixtures::jpeg(1200, 600, name: 'focal'), focalX: 1.0);

    $thumb = ImageManager::gd()->read((string) mediaDisk()->get($media->variants['thumb']['jpg']['path']));
    $avg = $thumb->resize(1, 1)->pickColor(0, 0)->toArray();
    expect($avg[2])->toBeGreaterThan($avg[0]);
});

it('keeps transparency: png fallback instead of jpg, alpha preserved', function (): void {
    $media = storeMedia(MediaFixtures::png(1000, 600, transparent: true));

    expect($media->mime)->toBe('image/png')
        ->and(array_keys($media->variants['mobile_480']))->toBe(['avif', 'webp', 'png'])
        ->and(array_keys($media->variants['og']))->toBe(['png']);

    $variant = ImageManager::gd()->read((string) mediaDisk()->get($media->variants['mobile_768']['png']['path']));
    expect($variant->pickColor(10, 10)->isTransparent())->toBeTrue()
        ->and($variant->pickColor(700, 10)->isTransparent())->toBeFalse();

    $webp = ImageManager::gd()->read((string) mediaDisk()->get($media->variants['mobile_768']['webp']['path']));
    expect($webp->pickColor(10, 10)->isTransparent())->toBeTrue();
});

it('uses jpg as fallback for an opaque PNG', function (): void {
    $media = storeMedia(MediaFixtures::png(800, 600, transparent: false));

    expect(array_keys($media->variants['mobile_480']))->toBe(['avif', 'webp', 'jpg']);
});

it('keeps animated GIFs as-is and adds a still poster and thumb', function (): void {
    $source = MediaFixtures::animatedGif(400, 300);

    $media = storeMedia($source);

    expect($media->mime)->toBe('image/gif')
        ->and(mediaDisk()->get($media->path()))->toBe(file_get_contents($source))
        ->and(array_keys($media->variants))->toBe(['poster', 'thumb'])
        ->and(array_keys($media->variants['poster']))->toBe(['avif', 'webp', 'jpg']);

    $poster = storedImageSize($media->variants['poster']['webp']['path']);
    expect([$poster[0], $poster[1]])->toBe([400, 300]);
});

it('sanitises SVG uploads and stores no variants', function (): void {
    $svg = '<?xml version="1.0"?><svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 48 24" onload="alert(1)">'
        .'<script>alert(1)</script><a href="javascript:alert(2)"><rect width="48" height="24" fill="#f00"/></a>'
        .'<image href="https://evil.example/x.png"/></svg>';

    $media = storeMedia(MediaFixtures::svg($svg));
    $stored = (string) mediaDisk()->get($media->path());

    expect($media->mime)->toBe('image/svg+xml')
        ->and($media->filename)->toEndWith('.svg')
        ->and([$media->width, $media->height])->toBe([48, 24])
        ->and($media->variants)->toBe([])
        ->and($media->optimized_at)->not->toBeNull()
        ->and($stored)->toContain('<rect')
        ->and($stored)->not->toContain('<script')
        ->and($stored)->not->toContain('onload')
        ->and($stored)->not->toContain('javascript:')
        ->and($stored)->not->toContain('evil.example');
});

it('rejects SVG when disabled', function (): void {
    config(['media.svg' => false]);

    storeMedia(MediaFixtures::svg('<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 1 1"/>', 'off.svg'));
})->throws(InvalidMediaException::class);

it('rejects invalid uploads', function (Closure $prepare, string $reason): void {
    $path = $prepare();

    try {
        storeMedia($path);
        $this->fail('expected an InvalidMediaException');
    } catch (InvalidMediaException $e) {
        expect($e->reason)->toBe($reason)
            ->and(Media::query()->count())->toBe(0)
            ->and(mediaDisk()->allFiles())->toBe([]);
    }
})->with([
    'non-image with an image extension' => [fn (): string => MediaFixtures::text('fake.jpg'), 'unsupported_type'],
    'over the byte limit' => [function (): string {
        config(['media.max_bytes' => 1024]);

        return MediaFixtures::jpeg(800, 600);
    }, 'too_large'],
    'over the pixel limit' => [function (): string {
        config(['media.max_dimension' => 500]);

        return MediaFixtures::jpeg(800, 600);
    }, 'too_many_pixels'],
    'missing file' => [fn (): string => MediaFixtures::dir().'/does-not-exist.jpg', 'unreadable'],
    'truncated JPEG' => [fn (): string => MediaFixtures::write('broken.jpg', substr((string) file_get_contents(MediaFixtures::jpeg(300, 200)), 0, 200)), 'corrupt'],
]);

it('trusts sniffed content over the file name', function (): void {
    $png = MediaFixtures::png(300, 200, transparent: false);

    $media = storeMedia($png, 'disguised.jpg');

    expect($media->mime)->toBe('image/png')
        ->and($media->filename)->toBe('disguised.png');
});

it('dedupes identical uploads by hash', function (): void {
    $source = MediaFixtures::jpeg(500, 400);

    $first = storeMedia($source);
    $second = storeMedia($source, 'again.jpg');

    expect($second->id)->toBe($first->id)
        ->and(Media::query()->count())->toBe(1);
});

it('queues OptimizeMedia and serves the original until variants exist', function (): void {
    Queue::fake();

    $media = storeMedia(MediaFixtures::jpeg(900, 600));

    expect($media->variants)->toBe([])
        ->and($media->optimized_at)->toBeNull()
        ->and($media->lqip)->not->toBeNull();

    Queue::assertPushed(OptimizeMedia::class, fn (OptimizeMedia $job): bool => $job->mediaId === $media->id && $job->presets === null);

    app()->call([new OptimizeMedia($media->id), 'handle']);

    expect($media->refresh()->variants)->toHaveKeys(['mobile_480', 'mobile_768', 'thumb', 'og'])
        ->and($media->optimized_at)->not->toBeNull();
});

it('puts OptimizeMedia on the configured connection and queue', function (): void {
    config(['media.queue.connection' => 'database', 'media.queue.name' => 'media']);

    $job = new OptimizeMedia(5);

    expect($job->connection)->toBe('database')
        ->and($job->queue)->toBe('media');
});

it('ignores an OptimizeMedia job whose media was deleted', function (): void {
    app()->call([new OptimizeMedia(999), 'handle']);

    expect(Media::query()->count())->toBe(0);
});

it('names files after a slug of the upload and falls back to image', function (): void {
    expect(StoreMedia::stem('My Holiday Photo.JPG'))->toBe('my-holiday-photo')
        ->and(StoreMedia::stem('---.png'))->toBe('image');

    $media = storeMedia(MediaFixtures::jpeg(320, 240), 'Summer Day.jpeg');

    expect($media->filename)->toBe('summer-day.jpg')
        ->and($media->directory)->toMatch('#^\d{4}/\d{2}/[a-z0-9]{10}$#')
        ->and($media->variants['mobile_480'] ?? null)->toBeNull(); // 320 px wide: below every responsive width
});
