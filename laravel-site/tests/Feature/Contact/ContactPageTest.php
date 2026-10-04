<?php

declare(strict_types=1);

use App\Domain\Contact\Enums\ContactMessageStatus;
use App\Domain\Contact\Enums\ContactTopic;
use App\Domain\Contact\Models\ContactMessage;
use App\Domain\Contact\Support\FormTimer;
use App\Domain\Settings\Actions\UpdateSettings;
use App\Domain\Settings\Enums\SettingGroup;
use App\Http\Controllers\ContactController;
use App\Http\Middleware\PageCache;
use App\Notifications\ContactMessageReceived;
use Database\Seeders\FaqSeeder;
use Database\Seeders\SettingsSeeder;
use Illuminate\Contracts\Queue\ShouldQueue;
use Illuminate\Notifications\AnonymousNotifiable;
use Illuminate\Notifications\ChannelManager;
use Illuminate\Support\Facades\Exceptions;
use Illuminate\Support\Facades\Notification;
use Illuminate\Support\Facades\Route;

beforeEach(function (): void {
    $this->seed([SettingsSeeder::class, FaqSeeder::class]);
    config(['app.url' => 'https://ritme.test']);
    Notification::fake();
});

/** @return array<string, mixed> */
function contactGraph(string $html): array
{
    preg_match('~<script type="application/ld\+json"[^>]*>(.*?)</script>~s', $html, $m);

    return json_decode($m[1] ?? '{}', true, 512, JSON_THROW_ON_ERROR);
}

/** @return array<string, mixed> */
function contactNode(string $html, string $idSuffix): array
{
    foreach (contactGraph($html)['@graph'] ?? [] as $node) {
        if (str_ends_with((string) ($node['@id'] ?? ''), $idSuffix)) {
            return $node;
        }
    }

    return [];
}

/**
 * A form post with a time-trap token issued `$age` seconds ago.
 *
 * @param  array<string, string>  $overrides
 * @return array<string, string>
 */
function contactForm(array $overrides = [], int $age = 10): array
{
    test()->travel(-$age)->seconds();
    $token = app(FormTimer::class)->issue();
    test()->travelBack();

    return [
        'topic' => 'support',
        'name' => 'مریم',
        'contact' => 'Maryam@Example.com',
        'message' => 'سلام، یادآور قرص در اپ برایم نمی‌آید.',
        'website' => '',
        'form_token' => $token,
        ...$overrides,
    ];
}

function useSupportEmail(string $email = 'support@ritme.test'): void
{
    app(UpdateSettings::class)->handle(SettingGroup::Contact, ['support_email' => $email]);
}

it('serves /contact from its controller with one h1, SEO tags, ContactPage JSON-LD and the contact FAQ', function (): void {
    expect(Route::getRoutes()->getByName('contact')?->getActionName())->toBe(ContactController::class.'@show');

    $response = $this->get('/contact')->assertOk();
    $html = $response->getContent();

    expect(substr_count($html, '<h1'))->toBe(1)
        ->and($html)->toContain('حرفت را بشنویم')
        ->toContain('rel="canonical"')
        ->toContain('<meta name="description"')
        ->toContain('اطلاعات سلامت را در این فرم ننویس')
        ->toContain('رمز یا قفل اپ را فراموش کرده‌ام')
        ->toContain('همه سؤال‌ها')
        ->toContain('href="tel:115"')
        ->toContain('name="_token"')
        ->toContain('name="website"')
        ->toContain('name="form_token"')
        ->toContain('action="'.route('contact.store').'"')
        ->not->toContain('style="')
        ->not->toContain('href="#"');

    foreach (ContactTopic::cases() as $topic) {
        expect($html)->toContain('value="'.$topic->value.'"')->toContain($topic->label());
    }

    preg_match('~<title>(.*?)</title>~u', $html, $title);
    preg_match('~<meta name="description" content="([^"]*)"~u', $html, $description);
    expect(mb_strlen(html_entity_decode($title[1] ?? '')))->toBeGreaterThanOrEqual(30)->toBeLessThanOrEqual(60)
        ->and(mb_strlen(html_entity_decode($description[1] ?? '')))->toBeGreaterThanOrEqual(70)->toBeLessThanOrEqual(160)
        ->and(contactNode($html, '#webpage')['@type'] ?? null)->toBe('ContactPage')
        ->and(contactNode($html, '#faq')['@type'] ?? null)->toBe('FAQPage')
        ->and(contactNode($html, '#faq')['mainEntity'] ?? [])->toHaveCount(3);
});

it('is never page-cached, so every visitor gets a fresh time-trap token and their own CSRF token', function (): void {
    $first = $this->get('/contact');
    $first->assertHeader(PageCache::HEADER, 'BYPASS');
    expect((string) $first->headers->get('Cache-Control'))->toContain('no-store');

    $second = $this->get('/contact');
    $second->assertHeader(PageCache::HEADER, 'BYPASS');
    expect($second->getContent())->toContain('value="'.session()->token().'"');
});

it('lists the configured mailboxes as Organization contactPoints and links them', function (): void {
    useSupportEmail();
    app(UpdateSettings::class)->handle(SettingGroup::Contact, ['partnership_email' => 'biz@ritme.test', 'phone' => '۰۲۱-۱۲۳۴۵۶۷۸']);
    app(UpdateSettings::class)->handle(SettingGroup::Legal, ['data_protection_email' => 'dpo@ritme.test']);

    $html = $this->get('/contact')->getContent();
    $points = contactNode($html, '#organization')['contactPoint'] ?? [];

    expect($html)->toContain('href="mailto:support@ritme.test"')
        ->toContain('href="mailto:biz@ritme.test"')
        ->toContain('href="tel:02112345678"')
        ->and(array_column($points, 'contactType'))->toBe(['customer support', 'partnerships', 'data protection'])
        ->and($points[0]['email'] ?? null)->toBe('support@ritme.test')
        ->and($points[0]['telephone'] ?? null)->toBe('02112345678');
});

it('stores a valid message, queues the notification to the support email and redirects with a flash (PRG)', function (): void {
    useSupportEmail();

    $this->post('/contact', contactForm())
        ->assertRedirect(route('contact').'#contact-form')
        ->assertSessionHas(ContactController::FLASH, 'sent');

    $message = ContactMessage::query()->sole();
    expect($message->topic)->toBe(ContactTopic::Support)
        ->and($message->name)->toBe('مریم')
        ->and($message->email)->toBe('maryam@example.com')
        ->and($message->phone)->toBeNull()
        ->and($message->status)->toBe(ContactMessageStatus::Unread)
        ->and(new ContactMessageReceived($message))->toBeInstanceOf(ShouldQueue::class);

    Notification::assertSentOnDemand(ContactMessageReceived::class, static function (ContactMessageReceived $n, array $channels, AnonymousNotifiable $notifiable) use ($message): bool {
        return $notifiable->routes['mail'] === 'support@ritme.test' && $n->contactMessage->is($message);
    });

    // The flash state renders the success message and is not cached either.
    $this->get('/contact')->assertSee('پیامت رسید')->assertHeader(PageCache::HEADER, 'BYPASS');
});

it('normalises a Persian mobile number and keeps the mail data-minimal', function (): void {
    useSupportEmail();

    $this->post('/contact', contactForm(['contact' => '+۹۸ ۹۱۲ ۳۴۵ ۶۷۸۹', 'topic' => 'media']))->assertSessionHasNoErrors();

    $message = ContactMessage::query()->sole();
    expect($message->phone)->toBe('09123456789')
        ->and($message->email)->toBeNull()
        ->and($message->topic)->toBe(ContactTopic::Media);

    $mail = (new ContactMessageReceived($message))->toMail(new AnonymousNotifiable);
    $text = implode("\n", [$mail->subject, ...$mail->introLines, ...$mail->outroLines]);
    expect($text)->toContain('رسانه')->toContain('مریم')
        ->not->toContain('یادآور قرص')
        ->not->toContain('09123456789');
});

it('routes privacy requests to the data-protection email and partnership to the partnership email', function (): void {
    useSupportEmail();
    app(UpdateSettings::class)->handle(SettingGroup::Legal, ['data_protection_email' => 'dpo@ritme.test']);
    app(UpdateSettings::class)->handle(SettingGroup::Contact, ['partnership_email' => 'biz@ritme.test']);

    $this->post('/contact', contactForm(['topic' => 'privacy']));
    $this->post('/contact', contactForm(['topic' => 'partnership']));

    $routes = [];
    Notification::assertSentOnDemandTimes(ContactMessageReceived::class, 2);
    Notification::assertSentOnDemand(ContactMessageReceived::class, static function ($n, $c, AnonymousNotifiable $notifiable) use (&$routes): bool {
        $routes[] = $notifiable->routes['mail'];

        return true;
    });
    expect($routes)->toContain('dpo@ritme.test')->toContain('biz@ritme.test');
});

it('still stores the message when no valid support email is configured', function (): void {
    // Seeded settings keep the design placeholder «[ایمیل پشتیبانی]».
    $this->post('/contact', contactForm())->assertSessionHas(ContactController::FLASH, 'sent');

    expect(ContactMessage::query()->count())->toBe(1);
    Notification::assertNothingSent();
});

it('drops spam silently with the very same response', function (array $overrides, int $age): void {
    useSupportEmail();

    $this->post('/contact', contactForm($overrides, $age))
        ->assertRedirect(route('contact').'#contact-form')
        ->assertSessionHas(ContactController::FLASH, 'sent')
        ->assertSessionHasNoErrors();

    expect(ContactMessage::query()->count())->toBe(0);
    Notification::assertNothingSent();
})->with([
    'honeypot filled' => [['website' => 'https://spam.example'], 10],
    'posted too fast' => [[], 1],
    'missing token' => [['form_token' => ''], 10],
    'tampered token' => [['form_token' => 'not-a-token'], 10],
    'honeypot with invalid data' => [['website' => 'x', 'contact' => 'nope', 'message' => ''], 10],
]);

it('asks an honest visitor with a day-old form to send again', function (): void {
    $this->from('/contact')->post('/contact', contactForm([], FormTimer::MAX_SECONDS + 60))
        ->assertRedirect(route('contact').'#contact-form')
        ->assertSessionHasErrors(['form_token' => __('contact.validation.expired')]);

    expect(ContactMessage::query()->count())->toBe(0);
});

it('validates with Persian messages and keeps the input', function (): void {
    $this->post('/contact', contactForm(['topic' => 'sales', 'name' => '', 'contact' => '12345', 'message' => 'کوتاه']))
        ->assertRedirect(route('contact').'#contact-form')
        ->assertSessionHasErrors([
            'topic' => 'یکی از موضوع‌های فهرست را انتخاب کن.',
            'name' => 'نامت را بنویس.',
            'contact' => __('contact.validation.contact_format'),
            'message' => 'پیام کمی کوتاه است؛ دست‌کم ۱۰ نویسه بنویس.',
        ])
        ->assertSessionHasInput('message', 'کوتاه');

    $html = $this->get('/contact')->getContent();
    expect($html)->toContain('نامت را بنویس.')
        ->toContain('aria-invalid="true"')
        ->toContain('چند مورد را درست کن')
        ->and(ContactMessage::query()->count())->toBe(0);
});

it('rate limits posts per IP without a captcha', function (): void {
    for ($i = 0; $i < 3; $i++) {
        $this->post('/contact', contactForm())->assertRedirect();
    }

    $this->post('/contact', contactForm())->assertStatus(429);
    expect(ContactMessage::query()->count())->toBe(3);
});

it('renders the queued mail with a link to the admin inbox when the queue runs', function (): void {
    Notification::swap(new ChannelManager(app())); // undo the fake of beforeEach
    Exceptions::fake();
    config(['mail.default' => 'array', 'queue.default' => 'sync']);
    useSupportEmail();

    $this->post('/contact', contactForm())->assertSessionHas(ContactController::FLASH, 'sent');

    Exceptions::assertNothingReported();
    $sent = app('mail.manager')->mailer('array')->getSymfonyTransport()->messages();
    expect($sent)->toHaveCount(1);
    $body = $sent[0]->getOriginalMessage()->getHtmlBody();
    expect($sent[0]->getEnvelope()->getRecipients()[0]->getAddress())->toBe('support@ritme.test')
        ->and($body)->toContain('/admin/contact-messages/'.ContactMessage::query()->sole()->id)
        ->not->toContain('یادآور قرص');
});
