<?php

declare(strict_types=1);

use App\Domain\Media\Data\MediaData;

function mediaData(array $variants = [], string $extension = 'jpg', int $width = 2560): MediaData
{
    return new MediaData(
        id: 1, url: '/media/a/pic.'.$extension, mime: $extension === 'png' ? 'image/png' : 'image/jpeg', extension: $extension,
        width: $width, height: 1440, alt: 'alt', title: null, caption: null, focalX: 0.5, focalY: 0.5,
        dominantColor: '#aabbcc', lqip: null, variants: $variants, optimized: $variants !== [],
    );
}

function variantFile(int $w, string $url): array
{
    return ['w' => $w, 'h' => intdiv($w * 9, 16), 'url' => $url, 'size' => 100];
}

it('builds srcsets per format in ascending width, with the original as the widest fallback candidate', function (): void {
    $data = mediaData([
        'desktop_1280' => ['webp' => variantFile(1280, '/d.webp'), 'jpg' => variantFile(1280, '/d.jpg')],
        'mobile_480' => ['webp' => variantFile(480, '/m.webp'), 'jpg' => variantFile(480, '/m.jpg')],
        'thumb' => ['webp' => variantFile(320, '/t.webp')],
    ]);

    expect($data->srcset('webp'))->toBe('/m.webp 480w, /d.webp 1280w')
        ->and($data->srcset('jpg'))->toBe('/m.jpg 480w, /d.jpg 1280w, /media/a/pic.jpg 2560w')
        ->and($data->srcset('avif'))->toBe('')
        ->and($data->formats())->toBe(['webp', 'jpg'])
        ->and($data->formats(['thumb']))->toBe(['webp']);
});

it('falls back from a missing format to the fallback format, then to the original', function (): void {
    $data = mediaData(['og' => ['jpg' => variantFile(1200, '/og.jpg')]]);

    expect($data->url('og', 'avif'))->toBe('/og.jpg')
        ->and($data->url('og'))->toBe('/og.jpg')
        ->and($data->url('thumb', 'webp'))->toBe('/media/a/pic.jpg')
        ->and($data->url())->toBe('/media/a/pic.jpg')
        ->and($data->file('og', 'jpg'))->toBe(variantFile(1200, '/og.jpg'))
        ->and($data->file('og', 'webp'))->toBeNull();
});

it('reports png as fallback for transparent images and the original extension without variants', function (): void {
    expect(mediaData(['mobile_480' => ['webp' => variantFile(480, '/m.webp'), 'png' => variantFile(480, '/m.png')]], 'png')->fallbackFormat())->toBe('png')
        ->and(mediaData([], 'png')->fallbackFormat())->toBe('png');
});

it('round-trips through arrays (the cached form)', function (): void {
    $data = mediaData(['thumb' => ['jpg' => variantFile(320, '/t.jpg')]]);

    expect(MediaData::fromArray($data->toArray()))->toEqual($data);
});
