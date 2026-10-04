<?php

declare(strict_types=1);

use App\Domain\Settings\Contracts\SettingsRepository;
use App\Domain\Settings\Enums\SettingGroup;
use App\Http\Controllers\AboutController;
use App\Http\Controllers\PrivacyController;
use App\Http\Controllers\SocialResponsibilityController;
use App\Http\Controllers\TermsController;
use Illuminate\Support\Facades\Route;

/** @return array<string, mixed> */
function infoPageGraph(string $html): array
{
    preg_match('~<script type="application/ld\+json"[^>]*>(.*?)</script>~s', $html, $m);

    return json_decode($m[1] ?? '{}', true, 512, JSON_THROW_ON_ERROR);
}

/** @return array<string, mixed> */
function infoPageNode(string $html, string $idSuffix): array
{
    foreach (infoPageGraph($html)['@graph'] ?? [] as $node) {
        if (str_ends_with((string) ($node['@id'] ?? ''), $idSuffix)) {
            return $node;
        }
    }

    return [];
}

dataset('info pages', [
    'about' => ['about', '/about', AboutController::class],
    'social' => ['social-responsibility', '/social-responsibility', SocialResponsibilityController::class],
    'privacy' => ['privacy', '/privacy', PrivacyController::class],
    'terms' => ['terms', '/terms', TermsController::class],
]);

it('serves each info page from its controller with one h1, SEO tags and no dead links', function (string $name, string $path, string $controller): void {
    expect(Route::getRoutes()->getByName($name)?->getActionName())->toBe($controller);

    $html = $this->get($path)->assertOk()->getContent();

    expect(substr_count($html, '<h1'))->toBe(1)
        ->and($html)->toContain('rel="canonical"')
        ->toContain('<meta name="description"')
        ->not->toContain('style="')
        ->not->toContain('href="#"')
        ->not->toContain('در حال آماده شدن');

    preg_match('~<title>(.*?)</title>~u', $html, $title);
    preg_match('~<meta name="description" content="([^"]*)"~u', $html, $description);
    expect(mb_strlen(html_entity_decode($title[1] ?? '')))->toBeGreaterThanOrEqual(30)->toBeLessThanOrEqual(60)
        ->and(mb_strlen(html_entity_decode($description[1] ?? '')))->toBeGreaterThanOrEqual(70)->toBeLessThanOrEqual(160);

    $types = array_column(infoPageGraph($html)['@graph'] ?? [], '@type');
    expect($types)->toContain('Organization', 'WebSite', 'BreadcrumbList');
})->with('info pages');

it('renders the about page as an AboutPage with the design sections and the review-policy anchor', function (): void {
    $html = $this->get('/about')->getContent();

    expect($html)->toContain('صدای زن')
        ->toContain('از یک سؤال ساده شروع شد')
        ->toContain('چطور تصمیم می‌گیریم')
        ->toContain('کارهایی که هرگز نمی‌کنیم')
        ->toContain('آدم‌های پشت ریتمی')
        ->toContain('id="review-policy"')
        ->toContain('شورای علمی')
        ->toContain('آگاهی، حق همه زنان ایران است')
        ->toContain(route('social-responsibility'))
        ->toContain(route('contact'))
        ->and(infoPageNode($html, '#webpage')['@type'] ?? null)->toBe('AboutPage');
});

it('shows no donation amounts, a contact CTA instead, and hides the transparency link while it is unset', function (): void {
    $html = $this->get('/social-responsibility')->getContent();

    expect($html)->toContain('از ریتمی حمایت کن')
        ->toContain('تماس برای حمایت')
        ->toContain('همیشه رایگان')
        ->toContain('id="download"')
        ->toContain('گزارش شفافیت')
        ->not->toContain('[مبلغ')
        ->not->toContain('مشاهده گزارش');
});

it('shows the transparency report link once a URL is configured', function (): void {
    __('social.seo.title'); // load the group first: addLines() on an unloaded group would replace it
    app('translator')->addLines(['social.transparency.url' => 'https://ritme.example/report.pdf'], 'fa');

    expect($this->get('/social-responsibility')->getContent())
        ->toContain('مشاهده گزارش')
        ->toContain('https://ritme.example/report.pdf');
});

it('carries the full privacy policy, the last-updated date and dateModified', function (): void {
    $html = $this->get('/privacy')->getContent();
    $updated = PrivacyController::date((string) __('privacy.updated_at'));

    expect($updated)->not->toBeNull()
        ->and($html)->toContain('<details id="policy"')
        ->toContain('خواندن متن کامل')
        ->toContain('کوکی‌ها و ردیاب‌ها')
        ->toContain('پیش‌فرض همه‌چیز خاموش است')
        ->toContain('datetime="'.$updated?->format('Y-m-d').'"')
        ->toContain(jdate($updated, 'j F Y'))
        ->toContain('[ایمیل مسئول حفاظت از داده]')
        ->and(infoPageNode($html, '#webpage')['dateModified'] ?? null)->toBe($updated?->toAtomString());
});

it('uses the data-protection email from LegalSettings', function (): void {
    app(SettingsRepository::class)->put(SettingGroup::Legal, ['data_protection_email' => 'privacy@ritme.example']);

    expect($this->get('/privacy')->getContent())
        ->toContain('mailto:privacy@ritme.example')
        ->not->toContain('[ایمیل مسئول حفاظت از داده]');
});

it('renders the terms in the privacy layout with settings tokens resolved', function (): void {
    app(SettingsRepository::class)->put(SettingGroup::Contact, ['support_email' => 'help@ritme.example']);

    $html = $this->get('/terms')->getContent();

    expect($html)->toContain('شرایط استفاده')
        ->toContain('help@ritme.example')
        ->toContain('۱۱۵')
        ->toContain(route('privacy'))
        ->not->toContain(':support_email')
        ->not->toContain(':emergency')
        ->and(infoPageNode($html, '#webpage')['dateModified'] ?? null)->not->toBeNull();
});
