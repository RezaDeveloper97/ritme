<?php

declare(strict_types=1);

use App\Domain\Settings\Data\EnamadSeal;
use App\Domain\Settings\Data\LegalSettings;

it('reads id and Code from the official snippet, the seal link or the image url', function (string $input): void {
    $seal = EnamadSeal::parse($input);

    expect($seal)->not->toBeNull()
        ->and($seal?->linkUrl())->toBe('https://trustseal.enamad.ir/?id=7913549&Code=ndNb0Esw0otBNJ0rj6PN')
        ->and($seal?->imageUrl())->toBe('https://trustseal.enamad.ir/logo.aspx?id=7913549&Code=ndNb0Esw0otBNJ0rj6PN');
})->with([
    'snippet' => "<a referrerpolicy='origin' target='_blank' href='https://trustseal.enamad.ir/?id=7913549&amp;Code=ndNb0Esw0otBNJ0rj6PN'><img referrerpolicy='origin' src='https://trustseal.enamad.ir/logo.aspx?id=7913549&amp;Code=ndNb0Esw0otBNJ0rj6PN' alt='' code='ndNb0Esw0otBNJ0rj6PN'></a>",
    'link' => 'https://trustseal.enamad.ir/?id=7913549&Code=ndNb0Esw0otBNJ0rj6PN',
    'image' => 'https://trustseal.enamad.ir/logo.aspx?id=7913549&Code=ndNb0Esw0otBNJ0rj6PN',
]);

it('returns null without both id and Code, and never lets markup into the urls', function (): void {
    expect(EnamadSeal::parse('https://trustseal.enamad.ir/?id=1'))->toBeNull()
        ->and(EnamadSeal::parse('[اینماد]'))->toBeNull()
        ->and(EnamadSeal::parse('?id=12"><script>&Code=abc"onerror=x'))->toBeNull();
});

it('prefers the HTML field, then the code field', function (): void {
    $legal = LegalSettings::fromArray(['enamad_html' => '', 'enamad_code' => 'https://trustseal.enamad.ir/?id=5&Code=abcdefgh12']);

    expect(EnamadSeal::fromSettings($legal)?->linkUrl())->toBe('https://trustseal.enamad.ir/?id=5&Code=abcdefgh12');
});
