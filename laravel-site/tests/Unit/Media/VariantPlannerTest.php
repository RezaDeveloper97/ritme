<?php

declare(strict_types=1);

use App\Domain\Media\Data\VariantSpec;
use App\Domain\Media\Enums\MediaFormat;
use App\Domain\Media\Support\FormatSupport;
use App\Domain\Media\Support\VariantPlanner;
use Intervention\Image\ImageManager;

function mediaPlanner(?FormatSupport $support = null): VariantPlanner
{
    $config = require dirname(__DIR__, 3).'/config/media.php';

    return new VariantPlanner($config['presets'], $config['formats'], $support ?? new FormatSupport(ImageManager::gd()), $config['poster_width']);
}

/**
 * @param  list<VariantSpec>  $specs
 * @return array<string, array{0: int, 1: int, 2: list<string>}>
 */
function planSummary(array $specs): array
{
    $out = [];
    foreach ($specs as $spec) {
        $out[$spec->name] = [$spec->width, $spec->height, array_map(static fn (MediaFormat $f): string => $f->value, $spec->formats)];
    }

    return $out;
}

it('plans every automatic preset for a large photo', function (): void {
    expect(planSummary(mediaPlanner()->plan(2560, 1920, hasAlpha: false)))->toBe([
        'mobile_480' => [480, 360, ['avif', 'webp', 'jpg']],
        'mobile_768' => [768, 576, ['avif', 'webp', 'jpg']],
        'desktop_1280' => [1280, 960, ['avif', 'webp', 'jpg']],
        'desktop_1920' => [1920, 1440, ['avif', 'webp', 'jpg']],
        'thumb' => [320, 320, ['avif', 'webp', 'jpg']],
        'og' => [1200, 630, ['jpg']],
    ]);
})->skip(fn (): bool => ! (new FormatSupport(ImageManager::gd()))->canEncode(MediaFormat::Avif), 'GD without AVIF');

it('uses png as fallback for transparent images', function (): void {
    $plan = planSummary(mediaPlanner()->plan(800, 600, hasAlpha: true));

    expect($plan['og'][2])->toBe(['png'])
        ->and(end($plan['mobile_768'][2]))->toBe('png');
});

it('skips widths wider than the source', function (): void {
    expect(array_keys(planSummary(mediaPlanner()->plan(1000, 500, false))))->toBe(['mobile_480', 'mobile_768', 'thumb', 'og'])
        ->and(array_keys(planSummary(mediaPlanner()->plan(300, 300, false))))->toBe(['thumb', 'og']);
});

it('plans only the requested presets, including on-demand ones', function (): void {
    expect(planSummary(mediaPlanner()->plan(1000, 800, false, ['square'])))->toHaveKeys(['square'])
        ->and(array_keys(planSummary(mediaPlanner()->plan(1000, 800, false, ['square', 'og']))))->toBe(['og', 'square']);
});

it('plans a poster and thumb for animated images', function (): void {
    expect(array_keys(planSummary(mediaPlanner()->planAnimated(2000, 1000, false))))->toBe(['poster', 'thumb'])
        ->and(planSummary(mediaPlanner()->planAnimated(2000, 1000, false))['poster'][0])->toBe(1280)
        ->and(planSummary(mediaPlanner()->planAnimated(400, 300, false))['poster'][0])->toBe(400);
});

it('drops avif when the driver cannot encode it', function (): void {
    $support = new FormatSupport(ImageManager::gd());
    (fn () => $this->cache['avif'] = false)->call($support);

    expect($support->resolve(['avif', 'webp', 'fallback'], false))->toBe([MediaFormat::Webp, MediaFormat::Jpg])
        ->and(planSummary(mediaPlanner($support)->plan(800, 600, false))['mobile_480'][2])->toBe(['webp', 'jpg']);
});
