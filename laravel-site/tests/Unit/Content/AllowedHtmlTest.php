<?php

declare(strict_types=1);

use App\Domain\Content\Support\AllowedHtml;

it('keeps an allowed https link and strips unsafe attributes', function (): void {
    $html = AllowedHtml::clean('<a href="https://trustseal.enamad.ir/?id=1" target="_top" onclick="x()" style="color:red" referrerpolicy="origin">نماد</a>', ['a', 'img']);

    expect($html)->toBe('<a href="https://trustseal.enamad.ir/?id=1" target="_blank" rel="noopener nofollow">نماد</a>');
});

it('allows only http and https links', function (string $href): void {
    expect(AllowedHtml::clean('<a href="'.$href.'">x</a>', ['a']))->toBe('x');
})->with(['javascript:alert(1)', 'data:text/html,hi', '//evil.example/x', '/relative', 'mailto:a@b.c', '']);

it('drops external, protocol-relative and data images but keeps same-origin ones', function (): void {
    expect(AllowedHtml::clean('<img src="https://trustseal.enamad.ir/logo.png">', ['img']))->toBe('')
        ->and(AllowedHtml::clean('<img src="//cdn.example/x.png">', ['img']))->toBe('')
        ->and(AllowedHtml::clean('<img src="data:image/png;base64,AAAA">', ['img']))->toBe('')
        ->and(AllowedHtml::clean('<img src="/media/enamad.png" width="120" height="abc" onerror="x()">', ['img']))
        ->toBe('<img src="/media/enamad.png" width="120" alt="">');
});

it('gives a link whose external image was removed a fallback text', function (): void {
    $html = AllowedHtml::clean('<a href="https://trustseal.enamad.ir/?id=1"><img src="https://trustseal.enamad.ir/logo.png"></a>', ['a', 'img'], 'نماد اعتماد');

    expect($html)->toBe('<a href="https://trustseal.enamad.ir/?id=1" rel="noopener nofollow">نماد اعتماد</a>')
        ->not->toContain('<img');
});

it('drops scripts with their content and unwraps tags outside the allow-list', function (): void {
    $html = AllowedHtml::clean('<div><b>متن</b><script>alert(1)</script><iframe src="https://x.example"></iframe><!-- c --></div>', ['a', 'img']);

    expect($html)->toBe('متن');
});

it('returns an empty string for empty input or an empty allow-list', function (): void {
    expect(AllowedHtml::clean(null, ['a']))->toBe('')
        ->and(AllowedHtml::clean('  ', ['a']))->toBe('')
        ->and(AllowedHtml::clean('<a href="https://x.example">x</a>', []))->toBe('');
});
