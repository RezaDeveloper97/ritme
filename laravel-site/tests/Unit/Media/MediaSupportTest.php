<?php

declare(strict_types=1);

use App\Domain\Media\Enums\MediaFormat;
use App\Domain\Media\Support\ImageAnalyzer;
use App\Domain\Media\Support\ImageManagerFactory;
use App\Domain\Media\Support\MemoryLimit;
use App\Domain\Media\Support\MimeSniffer;
use Intervention\Image\Drivers\Gd\Driver as GdDriver;
use Intervention\Image\ImageManager;
use Tests\Feature\Media\MediaFixtures;

it('sniffs real mime types and recognises SVG served as text', function (): void {
    $sniffer = new MimeSniffer;

    expect($sniffer->sniff(MediaFixtures::png(20, 20, false)))->toBe('image/png')
        ->and($sniffer->sniff(MediaFixtures::jpeg(20, 20, name: 'sniff')))->toBe('image/jpeg')
        ->and($sniffer->sniff(MediaFixtures::svg('<svg xmlns="http://www.w3.org/2000/svg"></svg>', 'bare.svg')))->toBe('image/svg+xml')
        ->and($sniffer->sniff(MediaFixtures::text('plain.txt')))->toBe('text/plain');
});

it('maps mimes to formats', function (): void {
    expect(MediaFormat::fromMime('image/jpeg'))->toBe(MediaFormat::Jpg)
        ->and(MediaFormat::fromMime('IMAGE/PNG'))->toBe(MediaFormat::Png)
        ->and(MediaFormat::fromMime('application/pdf'))->toBeNull()
        ->and(MediaFormat::Avif->mime())->toBe('image/avif');
});

it('picks GD when Imagick is not loaded, or the explicitly configured driver', function (): void {
    expect(ImageManagerFactory::make('gd')->driver())->toBeInstanceOf(GdDriver::class);

    if (! ImageManagerFactory::imagickAvailable()) {
        expect(ImageManagerFactory::make('auto')->driver())->toBeInstanceOf(GdDriver::class);
    }

    expect(fn () => ImageManagerFactory::make('vips'))->toThrow(InvalidArgumentException::class);
});

it('computes dominant colour, transparency and a tiny LQIP', function (): void {
    $analyzer = new ImageAnalyzer;
    $red = ImageManager::gd()->create(200, 100)->fill('dc1e1e');

    expect($analyzer->dominantColor($red))->toBe('#dc1e1e')
        ->and($analyzer->hasAlpha($red))->toBeFalse()
        ->and($analyzer->hasAlpha(ImageManager::gd()->read(MediaFixtures::png(100, 100, true))))->toBeTrue();

    $lqip = (string) $analyzer->lqip($red);
    expect($lqip)->toStartWith('data:image/webp;base64,')
        ->and(strlen((string) base64_decode(substr($lqip, 23))))->toBeLessThanOrEqual(600);
});

it('parses memory limits', function (): void {
    expect(MemoryLimit::bytes('512M'))->toBe(512 * 1024 * 1024)
        ->and(MemoryLimit::bytes('1G'))->toBe(1024 ** 3)
        ->and(MemoryLimit::bytes('-1'))->toBe(-1)
        ->and(MemoryLimit::bytes('2048'))->toBe(2048);
});
