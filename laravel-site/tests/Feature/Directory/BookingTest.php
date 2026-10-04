<?php

declare(strict_types=1);

use App\Domain\Contact\Support\FormTimer;
use App\Domain\Directory\Booking\Contracts\SmsSender;
use App\Domain\Directory\Booking\Enums\BookingStatus;
use App\Domain\Directory\Booking\Enums\TimeWindow;
use App\Domain\Directory\Booking\Models\BookingRequest;
use App\Domain\Directory\Booking\Support\BookingCode;
use App\Domain\Directory\Booking\Support\BookingDays;
use App\Domain\Directory\Booking\Support\MobileMask;
use App\Domain\Directory\Models\Place;
use App\Domain\Directory\Models\PlaceService;
use App\Domain\Directory\Support\OpeningHours;
use App\Domain\Settings\Actions\UpdateSettings;
use App\Domain\Settings\Enums\SettingGroup;
use App\Http\Controllers\Directory\BookingController;
use App\Http\Middleware\PageCache;
use App\Http\Requests\BookingRequest as BookingForm;
use App\Notifications\BookingRequestForPlace;
use App\Notifications\BookingRequestReceived;
use App\Notifications\Channels\SmsChannel;
use Carbon\CarbonImmutable;
use Database\Seeders\DirectorySeeder;
use Database\Seeders\SettingsSeeder;
use Illuminate\Notifications\AnonymousNotifiable;
use Illuminate\Support\Facades\Notification;
use Illuminate\Support\Facades\Route;

/** Sunday 2026-10-04 10:00 Tehran; the demo pool (ab-pari) is closed on Fridays (2026-10-09). */
const BOOKING_NOW = '2026-10-04 10:00:00';

beforeEach(function (): void {
    $this->seed([SettingsSeeder::class, DirectorySeeder::class]);
    config(['app.url' => 'https://ritme.test']);
    $this->travelTo(CarbonImmutable::parse(BOOKING_NOW, 'Asia/Tehran'));
    Notification::fake();
});

function bookingPlace(): Place
{
    return Place::query()->where('slug', DirectorySeeder::DEMO_PLACE_SLUG)->firstOrFail();
}

/**
 * A complete, valid booking post with a time-trap token issued `$age` seconds ago.
 *
 * @param  array<string, mixed>  $overrides
 * @return array<string, mixed>
 */
function bookingForm(array $overrides = [], int $age = 30): array
{
    $now = CarbonImmutable::now();
    test()->travelTo($now->subSeconds($age));
    $token = app(FormTimer::class)->issue();
    test()->travelTo($now);

    return [
        'service' => PlaceService::query()->where('place_id', bookingPlace()->id)->orderBy('sort_order')->orderBy('id')->firstOrFail()->id,
        'date' => '2026-10-07',
        'time_window' => 'evening',
        'name' => '  سارا   رحیمی ',
        'mobile' => '+98 912 ۱۲۳ ۴۵۶۷',
        'child_age' => '۱۴',
        'child_age_unit' => 'month',
        'note' => 'اولین جلسه است.',
        BookingForm::HONEYPOT => '',
        BookingForm::TIMER => $token,
        ...$overrides,
    ];
}

function bookUrl(): string
{
    return '/directory/place/'.DirectorySeeder::DEMO_PLACE_SLUG.'/book';
}

it('renders the booking request form in the #book slot of the place page: services, open days from tomorrow, windows, anti-spam fields', function (): void {
    $html = (string) $this->get('/directory/place/'.DirectorySeeder::DEMO_PLACE_SLUG)->assertOk()->getContent();

    preg_match('~<aside id="book".*?</aside>~s', $html, $m);
    $panel = $m[0] ?? '';

    expect($panel)->toContain('data-booking-slot')
        ->and($panel)->toContain('action="'.route('directory.place.book', [DirectorySeeder::DEMO_PLACE_SLUG]).'"')
        ->and($panel)->toContain('name="_token"')
        ->and($panel)->toContain('name="'.BookingForm::TIMER.'"')
        ->and($panel)->toContain('name="'.BookingForm::HONEYPOT.'"')
        ->and($panel)->toContain('آشنایی با آب · ۴۵ دقیقه')
        ->and(substr_count($panel, 'name="date"'))->toBe(BookingDays::OFFERED)
        ->and($panel)->not->toContain('value="2026-10-04"')   // today: requests start tomorrow
        ->and($panel)->toContain('value="2026-10-05"')
        ->and($panel)->not->toContain('value="2026-10-09"')   // Friday: closed
        ->and(substr_count($panel, 'name="time_window"'))->toBe(count(TimeWindow::cases()))
        ->and($panel)->toContain('مهر ۱۴۰۵')
        ->and($panel)->not->toContain(__('directory.place.booking.soon'))
        ->and($html)->not->toMatch('/\sstyle\s*=/i')
        ->and(substr_count($html, '<h1'))->toBe(1);
});

it('keeps the form working on a page-cache HIT (fresh CSRF token per visitor, cached time-trap token)', function (): void {
    $url = '/directory/place/'.DirectorySeeder::DEMO_PLACE_SLUG;
    $this->get($url)->assertOk();
    $hit = $this->get($url)->assertOk();

    expect($hit->headers->get(PageCache::HEADER))->toBe('HIT')
        ->and((string) $hit->getContent())->toContain('name="'.BookingForm::TIMER.'"')
        ->and((string) $hit->getContent())->not->toContain('__RT_PAGE_CACHE_CSRF__');
});

it('stores a request with normalised data, notifies the team and redirects (PRG) to the booked page', function (): void {
    expect(Route::getRoutes()->getByName('directory.place.book')?->getActionName())->toBe(BookingController::class.'@store');
    app(UpdateSettings::class)->handle(SettingGroup::Contact, ['support_email' => 'support@ritme.test']);

    $response = $this->post(bookUrl(), bookingForm());

    $booking = BookingRequest::query()->sole();
    $response->assertRedirect(route('directory.booked', [$booking->code]));

    expect($booking->code)->toMatch(BookingCode::PATTERN)
        ->and($booking->status)->toBe(BookingStatus::New)
        ->and($booking->place_id)->toBe(bookingPlace()->id)
        ->and($booking->place_name)->toBe('استخر مادر و کودک آب‌پری')
        ->and($booking->service_name)->toBe('آشنایی با آب')
        ->and($booking->service_price)->toBe(320_000)
        ->and($booking->preferred_date->format('Y-m-d'))->toBe('2026-10-07')
        ->and($booking->time_window)->toBe(TimeWindow::Evening)
        ->and($booking->parent_name)->toBe('سارا رحیمی')
        ->and($booking->mobile)->toBe('09121234567')
        ->and($booking->child_age_months)->toBe(14)
        ->and($booking->note)->toBe('اولین جلسه است.');

    Notification::assertSentTo(new AnonymousNotifiable, BookingRequestReceived::class, function (BookingRequestReceived $n, array $channels, AnonymousNotifiable $to): bool {
        $mail = $n->toMail($to)->render();

        return $to->routes['mail'] === 'support@ritme.test' && ! str_contains((string) $mail, '09121234567');
    });
    // Demo places have no phones and never get the place SMS.
    Notification::assertNotSentTo(new AnonymousNotifiable, BookingRequestForPlace::class);
});

it('accepts a typed Jalali date and years for the child age, and works without a service when the place lists none', function (): void {
    $form = bookingForm(['service' => '', 'date' => '۱۴۰۵/۷/۲۰', 'child_age' => '3', 'child_age_unit' => 'year']);
    PlaceService::query()->where('place_id', bookingPlace()->id)->get()->each->delete();

    $this->post(bookUrl(), $form)->assertRedirect();

    $booking = BookingRequest::query()->sole();
    expect($booking->preferred_date->format('Y-m-d'))->toBe('2026-10-12')
        ->and($booking->service_id)->toBeNull()
        ->and($booking->child_age_months)->toBe(36);
});

it('sends the place an SMS through the SmsSender contract when it lists a mobile (not for demo places)', function (): void {
    $place = bookingPlace();
    $place->forceFill(['is_demo' => false, 'phones' => ['021-22334455', '0935 111 2233']])->save();

    $this->post(bookUrl(), bookingForm())->assertRedirect();

    Notification::assertSentTo(new AnonymousNotifiable, BookingRequestForPlace::class, function (BookingRequestForPlace $n, array $channels, AnonymousNotifiable $to): bool {
        $text = $n->toSms($to);

        return $channels === [SmsChannel::class] && $to->routes['sms'] === '09351112233'
            && str_contains($text, '09121234567') && str_contains($text, 'سن کودک: ۱۴ ماه') && str_contains($text, BookingRequest::query()->sole()->code);
    });

    $sender = new class implements SmsSender
    {
        /** @var list<array{string, string}> */
        public array $sent = [];

        public function send(string $mobile, string $text): void
        {
            $this->sent[] = [$mobile, $text];
        }
    };
    app()->instance(SmsSender::class, $sender);
    app(SmsChannel::class)->send((new AnonymousNotifiable)->route('sms', '09351112233'), new BookingRequestForPlace(BookingRequest::query()->sole()));
    expect($sender->sent)->toHaveCount(1)->and($sender->sent[0][0])->toBe('09351112233');
});

it('shows the booked page: request copy, one h1, noindex, no-store, masked mobile, no note, never page-cached', function (): void {
    $this->post(bookUrl(), bookingForm());
    $booking = BookingRequest::query()->sole();

    $response = $this->get('/directory/booked/'.$booking->code)->assertOk();
    $html = (string) $response->getContent();

    expect($response->headers->get('Cache-Control'))->toContain('no-store')
        ->and($html)->toContain('<meta name="robots" content="noindex')
        ->and(substr_count($html, '<h1'))->toBe(1)
        ->and($html)->toContain('درخواست رزروت ثبت شد')
        ->and($html)->toContain($booking->code)
        ->and($html)->toContain('آشنایی با آب')
        ->and($html)->toContain('چهارشنبه ۱۵ مهر ۱۴۰۵')
        ->and($html)->toContain('عصر (۱۶ تا ۲۰)')
        ->and($html)->toContain('کودک ۱۴ ماه')
        ->and($html)->toContain(MobileMask::mask('09121234567'))
        ->and($html)->not->toContain('09121234567')
        ->and($html)->not->toContain(fa_digits('09121234567'))
        ->and($html)->not->toContain('اولین جلسه است')
        ->and($html)->not->toContain('پرداخت شد')
        ->and($html)->toContain('id="download"')
        ->and($html)->toContain('کلاه شنا و پوشک مخصوص آب لازم است')
        ->and($html)->not->toMatch('/\sstyle\s*=/i');

    // Lower-case codes find the same request; the page is never stored by the full-page cache.
    $this->get('/directory/booked/'.strtolower($booking->code))->assertOk();
    expect($this->get('/directory/booked/'.$booking->code)->headers->get(PageCache::HEADER))->not->toBe('HIT');

    $booking->forceFill(['status' => BookingStatus::Confirmed])->save();
    $this->get('/directory/booked/'.$booking->code)->assertOk()->assertSee('رزروت تأیید شد');
});

it('answers unknown codes with 404', function (): void {
    $this->get('/directory/booked/'.BookingCode::generate())->assertNotFound();
    $this->get('/directory/booked/nope')->assertNotFound();
});

it('answers spam exactly like a real submit and stores nothing', function (array $overrides, int $age): void {
    app(UpdateSettings::class)->handle(SettingGroup::Contact, ['support_email' => 'support@ritme.test']);

    $response = $this->post(bookUrl(), bookingForm($overrides, $age));

    $location = (string) $response->headers->get('Location');
    expect($response->getStatusCode())->toBe(302)
        ->and($location)->toMatch('~/directory/booked/[2-9A-HJKMNP-Z]{4}-[2-9A-HJKMNP-Z]{4}-[2-9A-HJKMNP-Z]{4}$~')
        ->and(BookingRequest::query()->count())->toBe(0);
    Notification::assertNothingSent();

    $this->get($location)->assertOk()->assertSee('درخواست رزروت ثبت شد')->assertSee('آشنایی با آب');
    $this->get($location)->assertNotFound(); // the echo lives for one request only
})->with([
    'honeypot' => [[BookingForm::HONEYPOT => 'https://spam.example'], 30],
    'too fast' => [[], 1],
    'missing token' => [[BookingForm::TIMER => ''], 30],
    'tampered token' => [[BookingForm::TIMER => 'abc'], 30],
]);

it('rejects invalid input with Persian messages in the booking error bag and keeps the input', function (array $overrides, string $field, string $message): void {
    $overrides = array_map(static fn (mixed $v): mixed => $v instanceof Closure ? $v() : $v, $overrides);
    $response = $this->post(bookUrl(), bookingForm($overrides));

    $response->assertRedirect(route('directory.place', [DirectorySeeder::DEMO_PLACE_SLUG]).'#book')
        ->assertSessionHasErrors([$field => $message], null, 'booking')
        ->assertSessionHasInput('mobile');
    expect(BookingRequest::query()->count())->toBe(0);
})->with([
    'service of another place' => [['service' => fn (): int => PlaceService::query()->where('place_id', '!=', bookingPlace()->id)->value('id')], 'service', 'یکی از خدمات این مجموعه را انتخاب کن.'],
    'no service' => [['service' => ''], 'service', 'یکی از خدمات این مجموعه را انتخاب کن.'],
    'past day' => [['date' => '2026-10-03'], 'date', 'یک روز از امروز تا دو ماه آینده انتخاب کن که مجموعه باز باشد.'],
    'closed friday' => [['date' => '2026-10-09'], 'date', 'یک روز از امروز تا دو ماه آینده انتخاب کن که مجموعه باز باشد.'],
    'beyond horizon' => [['date' => '2026-12-31'], 'date', 'یک روز از امروز تا دو ماه آینده انتخاب کن که مجموعه باز باشد.'],
    'garbage day' => [['date' => '1405/13/40'], 'date', 'یک روز از امروز تا دو ماه آینده انتخاب کن که مجموعه باز باشد.'],
    'no window' => [['time_window' => 'midnight'], 'time_window', 'بازه زمانی دلخواه را انتخاب کن.'],
    'landline' => [['mobile' => '021-22334455'], 'mobile', 'یک شماره همراه درست بنویس؛ مثل ۰۹۱۲۱۲۳۴۵۶۷.'],
    'no name' => [['name' => ' '], 'name', 'نامت را بنویس (حداکثر ۱۰۰ حرف).'],
    'too old' => [['child_age' => '19', 'child_age_unit' => 'year'], 'child_age', 'سن کودک را به ماه یا سال بنویس (تا ۱۸ سال).'],
    'long note' => [['note' => str_repeat('ا', 501)], 'note', 'توضیح حداکثر ۵۰۰ حرف باشد.'],
]);

it('asks to send again when the form token has expired (no silent drop)', function (): void {
    $this->post(bookUrl(), bookingForm([], FormTimer::MAX_SECONDS + 60))
        ->assertSessionHasErrors([BookingForm::TIMER => 'این فرم مدت زیادی باز مانده بود؛ لطفاً دوباره بفرست.'], null, 'booking');
    expect(BookingRequest::query()->count())->toBe(0);

    $html = (string) $this->get('/directory/place/'.DirectorySeeder::DEMO_PLACE_SLUG)->getContent();
    expect($html)->toContain('این فرم مدت زیادی باز مانده بود');
});

it('404s posts for unknown places', function (): void {
    $this->post('/directory/place/nope/book', bookingForm())->assertNotFound();
});

it('rate limits booking posts per IP and per mobile', function (): void {
    for ($i = 0; $i < 5; $i++) {
        $this->post(bookUrl(), bookingForm(['mobile' => '0912000000'.$i]))->assertRedirect();
    }
    $this->post(bookUrl(), bookingForm(['mobile' => '09120000009']))->assertStatus(429);

    // Another IP, same mobile: the per-mobile daily limit (5) applies across IPs.
    $this->travel(11)->minutes();
    for ($i = 0; $i < 5; $i++) {
        $this->withServerVariables(['REMOTE_ADDR' => '10.0.0.'.($i + 2)])->post(bookUrl(), bookingForm(['mobile' => '09129999999']))->assertRedirect();
    }
    $this->withServerVariables(['REMOTE_ADDR' => '10.0.0.50'])->post(bookUrl(), bookingForm(['mobile' => '۰۹۱۲۹۹۹۹۹۹۹']))->assertStatus(429);
});

it('offers and accepts days from the opening hours only (unknown hours = any day)', function (): void {
    $hours = OpeningHours::fromArray(['saturday' => [['opens' => '09:00', 'closes' => '12:00']], 'sunday' => [], 'monday' => [], 'tuesday' => [], 'wednesday' => [], 'thursday' => [], 'friday' => []]);
    $offered = array_map(static fn (CarbonImmutable $d): string => $d->format('Y-m-d'), BookingDays::offered($hours, null, 3));

    expect($offered)->toBe(['2026-10-10', '2026-10-17', '2026-10-24'])
        ->and(BookingDays::isBookable(CarbonImmutable::parse('2026-10-04', 'Asia/Tehran'), OpeningHours::fromArray([])))->toBeTrue()
        ->and(BookingForm::parseDate('۱۴۰۵/۰۷/۱۲')?->format('Y-m-d'))->toBe('2026-10-04')
        ->and(BookingForm::parseDate('2026-02-30'))->toBeNull()
        ->and(BookingCode::generate())->toMatch(BookingCode::PATTERN)
        ->and(MobileMask::mask('09121234567'))->toBe('۰۹۱۲•••••۶۷');
});
