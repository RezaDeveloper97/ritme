<?php

declare(strict_types=1);

use App\Domain\Seo\Support\Robots;

it('defaults to index,follow with large previews', function (): void {
    expect((string) Robots::default())->toBe('index,follow,max-image-preview:large,max-snippet:-1,max-video-preview:-1')
        ->and((string) Robots::parse(null))->toBe('index,follow');
});

it('parses stored directives', function (string $stored, string $rendered, bool $index): void {
    $robots = Robots::parse($stored);

    expect((string) $robots)->toBe($rendered)->and($robots->index)->toBe($index);
})->with([
    ['noindex, follow', 'noindex,follow', false],
    ['NOINDEX,NOFOLLOW', 'noindex,nofollow', false],
    ['none', 'noindex,nofollow', false],
    ['index,follow,max-image-preview:large', 'index,follow,max-image-preview:large', true],
    ['nofollow', 'index,nofollow', true],
]);

it('drops preview directives once noindexed', function (): void {
    expect((string) Robots::default()->withNoindex())->toBe('noindex,follow')
        ->and((string) Robots::parse('nofollow')->withNoindex())->toBe('noindex,nofollow');
});
