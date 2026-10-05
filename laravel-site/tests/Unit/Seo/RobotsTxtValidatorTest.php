<?php

declare(strict_types=1);

use App\Domain\Seo\Indexing\RobotsTxtValidator;

it('accepts empty rules (the built-in default is used)', function (string $rules): void {
    expect((new RobotsTxtValidator)->errors($rules))->toBe([]);
})->with(['empty' => [''], 'blank lines' => ["  \r\n \n"]]);

it('rejects oversized and non-UTF-8 files before parsing', function (): void {
    $validator = new RobotsTxtValidator;

    expect($validator->errors("User-agent: *\nDisallow: /".str_repeat('a', RobotsTxtValidator::MAX_BYTES)))
        ->toBe(['حجم robots.txt نباید بیشتر از ۳۲ کیلوبایت باشد.'])
        ->and($validator->errors("User-agent: *\nDisallow: /\xC3\x28"))->toBe(['robots.txt باید متن UTF-8 باشد.']);
});

it('reports one Persian error per broken line with its number', function (string $rules, string $expected): void {
    expect((new RobotsTxtValidator)->errors($rules))->toBe([$expected]);
})->with([
    'invalid agent' => ["User-agent: bad<agent>\nDisallow: /x", 'خط ۱: نام ربات (User-agent) نامعتبر است.'],
    'rule before agent' => ["Disallow: /x\nUser-agent: *", 'خط ۱: هر قانون باید بعد از یک خط User-agent بیاید.'],
    'crawl-delay before agent' => ["Crawl-delay: 2\nUser-agent: *", 'خط ۱: هر قانون باید بعد از یک خط User-agent بیاید.'],
    'crawl-delay out of range' => ["User-agent: *\nCrawl-delay: 61", 'خط ۲: Crawl-delay باید عددی بین ۰ تا ۶۰ باشد.'],
    'empty clean-param' => ["User-agent: Yandex\nClean-param:", 'خط ۲: Clean-param باید مقدار داشته باشد و بعد از User-agent بیاید.'],
    'path with spaces' => ["User-agent: *\nDisallow: /a b", 'خط ۲: مسیر باید با «/» یا «*» شروع شود و فاصله نداشته باشد.'],
    'unknown field' => ["User-agent: *\nHost: ritme.ir", 'خط ۲: دستور ناشناخته «host». دستورهای مجاز: User-agent، Allow، Disallow، Crawl-delay، Clean-param.'],
]);

it('accepts Clean-param and starts a new group after rules', function (): void {
    $rules = "User-agent: Yandex\nClean-param: utm_source /blog/\nUser-agent: Googlebot\nDisallow: /search";

    expect((new RobotsTxtValidator)->errors($rules))->toBe([]);
});

it('never lets every robot be shut out of the whole site', function (string $rules): void {
    expect(implode(' ', (new RobotsTxtValidator)->errors($rules)))->toContain('کل سایت را از نتایج جست‌وجو حذف می‌کند');
})->with([
    'slash' => ["User-agent: *\nDisallow: /"],
    'wildcard' => ["User-agent: Googlebot\nUser-agent: *\nDisallow: /*"],
]);

it('allows Disallow: / for a single named robot', function (): void {
    expect((new RobotsTxtValidator)->errors("User-agent: BadBot\nDisallow: /"))->toBe([]);
});

it('requires at least one User-agent line', function (): void {
    expect((new RobotsTxtValidator)->errors('# only a comment'))->toBe(['دست‌کم یک خط User-agent لازم است.']);
});
