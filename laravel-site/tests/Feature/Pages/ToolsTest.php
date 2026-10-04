<?php

declare(strict_types=1);

use App\Http\Controllers\ToolsController;
use Illuminate\Support\Facades\Route;

/** @return array<string, mixed> */
function toolsJsonLd(string $html): array
{
    preg_match('~<script type="application/ld\+json"[^>]*>(.*?)</script>~s', $html, $m);

    return json_decode($m[1] ?? '{}', true, 512, JSON_THROW_ON_ERROR);
}

it('serves /tools from ToolsController', function (): void {
    expect(Route::getRoutes()->getByName('tools')?->getActionName())->toBe(ToolsController::class);
});

it('renders the design sections with one h1 and the stable anchors', function (): void {
    $html = $this->get('/tools')->assertOk()->getContent();

    expect(substr_count($html, '<h1'))->toBe(1)
        ->and($html)->toContain('محاسبه‌گرها و چک‌لیست‌ها، بدون ثبت‌نام')
        ->toContain('id="due-date"')
        ->toContain('id="fertility"')
        ->toContain('id="hospital-bag"')
        ->toContain('id="sisemoni"')
        ->toContain('id="download"')
        ->toContain('آماده باش، بدون استرس')
        ->toContain('قبل از خرید، این‌ها را بدان')
        ->toContain('همه ابزارها در اپ ریتمی')
        ->toContain('روش پیشگیری از بارداری نیست')
        ->toContain('تشخیص پزشکی نیست')
        ->toContain('data-module="calculators"')
        ->not->toContain('style="')
        ->not->toContain('aria-label="lmp"')
        ->not->toContain('aria-label="cycle"');
});

it('labels every calculator input in Persian', function (): void {
    $html = $this->get('/tools')->getContent();

    foreach (['due-date', 'fertility'] as $id) {
        expect($html)->toContain('for="'.$id.'-lmp"')->toContain('id="'.$id.'-lmp"')
            ->toContain('for="'.$id.'-cycle"')->toContain('id="'.$id.'-cycle"');
    }
    expect($html)->toContain('اولین روز آخرین پریود')->toContain('طول معمول چرخه');
});

it('shows the example results on the plain page without date-dependent text', function (): void {
    $html = $this->get('/tools')->getContent();

    expect($html)->toContain('۱۷ بهمن ۱۴۰۵')
        ->toContain('بازه معمول: ۳ بهمن تا ۱ اسفند ۱۴۰۵')
        ->toContain('۱۲ تا ۱۷ مهر ۱۴۰۵');

    preg_match_all('~data-calc-detail[^>]*>([^<]*)<~u', $html, $details);
    expect($details[1])->toHaveCount(2)->and(implode(' ', $details[1]))->not->toContain('از بارداری گذشته');
});

it('has the design title, description, canonical, robots and WebApplication nodes', function (): void {
    $html = $this->get('/tools')->getContent();

    expect($html)->toContain('<title>ابزارهای رایگان: محاسبه تاریخ زایمان و روزهای باروری — ریتمی</title>')
        ->toMatch('~<link rel="canonical" href="https?://[^"/]+/tools">~');

    $graph = toolsJsonLd($html)['@graph'] ?? [];
    $apps = array_values(array_filter($graph, fn (array $node): bool => ($node['@type'] ?? null) === 'WebApplication'));

    expect(array_column($graph, '@type'))->toContain('WebPage', 'BreadcrumbList')->not->toContain('FAQPage', 'AggregateRating')
        ->and($apps)->toHaveCount(2)
        ->and($apps[0]['url'])->toEndWith('/tools#due-date')
        ->and($apps[0]['offers']['price'])->toBe(0)
        ->and($apps[0])->not->toHaveKey('aggregateRating');
});

it('computes a submitted due date without JS, noindex with the canonical on /tools', function (): void {
    $this->travelTo(new DateTimeImmutable('2026-10-04 10:00', new DateTimeZone('Asia/Tehran')));

    $response = $this->get('/tools?calc=due&lmp=1405/02/12&cycle=28')->assertOk();
    $html = $response->getContent();

    expect($html)->toContain('الان حدود ۲۲ هفته و ۱ روز از بارداری گذشته · بازه معمول: ۳ بهمن تا ۱ اسفند ۱۴۰۵')
        ->toContain('value="1405/02/12"')
        ->toContain('noindex')
        ->toMatch('~<link rel="canonical" href="https?://[^"/]+/tools">~');
    expect($response->headers->get('X-Page-Cache'))->not->toBe('HIT');
});

it('computes a submitted fertility window and keeps the warning above the result', function (): void {
    $html = $this->get('/tools?calc=fert&lmp='.urlencode('۲ مهر ۱۴۰۵').'&cycle=28')->assertOk()->getContent();

    expect($html)->toContain('۱۱ تا ۱۶ مهر ۱۴۰۵')
        ->toContain('تخمک‌گذاری احتمالی: حدود ۱۶ مهر · پریود بعدی احتمالاً ۳۰ مهر ۱۴۰۵');

    $form = substr($html, (int) strpos($html, 'id="fertility"'));
    expect(strpos($form, 'روش پیشگیری از بارداری نیست'))->toBeLessThan(strpos($form, '۱۱ تا ۱۶ مهر ۱۴۰۵'));
});

it('shows Persian validation messages and marks the wrong field', function (): void {
    $html = $this->get('/tools?calc=due&lmp=abc&cycle=28')->assertOk()->getContent();
    expect($html)->toContain('تاریخ را مثل ۱۴۰۵/۰۲/۱۲ یا «۱۲ اردیبهشت ۱۴۰۵» بنویس.')->toContain('aria-invalid="true"');

    $html = $this->get('/tools?calc=fert&lmp=1405/07/01&cycle=60')->getContent();
    expect($html)->toContain('طول چرخه معمولاً بین ۲۱ تا ۴۵ روز است');

    $this->travelTo(new DateTimeImmutable('2026-10-04 10:00', new DateTimeZone('Asia/Tehran')));
    expect($this->get('/tools?calc=due&lmp=1405/08/01')->getContent())->toContain('نمی‌تواند بعد از امروز باشد');
});

it('ignores unknown calculators and array parameters', function (): void {
    $this->get('/tools?calc=x&lmp=1405/02/12')->assertOk()->assertSee('۱۷ بهمن ۱۴۰۵', false);
    $this->get('/tools?calc[]=due&lmp[]=1')->assertOk();
    $this->get('/tools?calc=due&lmp[]=1&cycle[]=2')->assertOk()->assertSee('تاریخ را مثل', false);
});
