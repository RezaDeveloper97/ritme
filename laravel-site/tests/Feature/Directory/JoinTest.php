<?php

declare(strict_types=1);

use App\Domain\Contact\Support\FormTimer;
use App\Domain\Directory\Join\Enums\BookingMode;
use App\Domain\Directory\Join\Enums\JoinRequestStatus;
use App\Domain\Directory\Join\Models\JoinRequest;
use App\Domain\Directory\Join\Support\JoinForm;
use App\Domain\Directory\Models\Amenity;
use App\Domain\Directory\Models\City;
use App\Domain\Directory\Models\District;
use App\Domain\Directory\Models\PlaceCategory;
use App\Domain\Media\Actions\FindMediaUsages;
use App\Domain\Media\Models\Media;
use App\Domain\Settings\Actions\UpdateSettings;
use App\Domain\Settings\Enums\SettingGroup;
use App\Http\Controllers\Directory\JoinController;
use App\Notifications\JoinRequestReceived;
use Database\Seeders\DirectorySeeder;
use Database\Seeders\FaqSeeder;
use Database\Seeders\SettingsSeeder;
use Illuminate\Http\UploadedFile;
use Illuminate\Notifications\AnonymousNotifiable;
use Illuminate\Support\Facades\Notification;
use Illuminate\Support\Facades\Route;
use Illuminate\Support\Facades\Storage;
use Tests\Feature\Media\MediaFixtures;

beforeEach(function (): void {
    $this->seed([SettingsSeeder::class, FaqSeeder::class, DirectorySeeder::class]);
    config(['app.url' => 'https://ritme.test']);
    Storage::fake('public');
    Notification::fake();
});

/** @return array<string, mixed> */
function joinGraph(string $html): array
{
    preg_match('~<script type="application/ld\+json"[^>]*>(.*?)</script>~s', $html, $m);

    return json_decode($m[1] ?? '{}', true, 512, JSON_THROW_ON_ERROR);
}

function joinPhoto(string $name = 'space.jpg', int $width = 900, int $height = 600): UploadedFile
{
    return new UploadedFile(MediaFixtures::jpeg($width, $height, name: pathinfo($name, PATHINFO_FILENAME)), $name, 'image/jpeg', null, true);
}

/**
 * A complete, valid join post with a time-trap token issued `$age` seconds ago.
 *
 * @param  array<string, mixed>  $overrides
 * @return array<string, mixed>
 */
function joinForm(array $overrides = [], int $age = 30): array
{
    test()->travel(-$age)->seconds();
    $token = app(FormTimer::class)->issue();
    test()->travelBack();

    $city = City::query()->firstOrFail();
    $district = District::query()->where('city_id', $city->id)->firstOrFail();

    return [
        'name' => 'استخر مادر و کودک نیلوفر',
        'category' => PlaceCategory::query()->firstOrFail()->id,
        'city' => $city->id,
        'district' => $district->id,
        'contact_name' => 'سارا رحیمی',
        'mobile' => '۰۹۱۲ ۱۲۳ ۴۵۶۷',
        'email' => 'Sara@Example.com',
        'address' => 'تهران، خیابان ولیعصر، کوچه نهم، پلاک ۱۲',
        'phone' => '۰۲۱-۲۲۳۳۴۴۵۵',
        'latitude' => '۳۵.۷۵۶۱',
        'longitude' => '51.4100',
        'about' => 'استخر آب گرم با مربی خانم.',
        'ages' => ['6-12m', '1-2y', '2-4y'],
        'amenities' => [Amenity::query()->firstOrFail()->id],
        'hours' => [
            'saturday' => ['open' => '1', 'opens' => '09:00', 'closes' => '20:00'],
            'thursday' => ['open' => '1', 'opens' => '۰۹:۰۰', 'closes' => '14:00'],
            'friday' => ['opens' => '09:00', 'closes' => '14:00'],
        ],
        'services' => 'آشنایی با آب، ۴۵ دقیقه، ۳۲۰ هزار تومان',
        'booking_mode' => 'phone',
        'license' => '1',
        'terms' => '1',
        'website' => '',
        'form_token' => $token,
        ...$overrides,
    ];
}

it('serves the business landing with one h1, its FAQ group, FAQPage + breadcrumbs JSON-LD and join links', function (): void {
    expect(Route::getRoutes()->getByName('directory.business')?->getActionName())->toBe(JoinController::class.'@business');

    $html = $this->get('/directory/business')->assertOk()
        ->assertSee('مجموعه‌ات را به مادرهای شهرت معرفی کن')
        ->assertSee('ثبت مجموعه چقدر طول می‌کشد؟')
        ->assertSee('id="faq"', false)
        ->assertSee('href="'.route('directory.join').'"', false)
        ->assertDontSee('۴٫۸')
        ->getContent();

    expect(substr_count((string) $html, '<h1'))->toBe(1);
    $types = array_column(joinGraph((string) $html)['@graph'] ?? [], '@type');
    expect($types)->toContain('FAQPage')->toContain('BreadcrumbList');
});

it('renders the join form as one long multipart form with four steps, the stepper module and no-store', function (): void {
    $response = $this->get('/directory/join')->assertOk();
    $html = (string) $response->getContent();

    expect($response->headers->get('Cache-Control'))->toContain('no-store')
        ->and(substr_count($html, '<h1'))->toBe(1)
        ->and(substr_count($html, 'data-step="'))->toBe(JoinForm::STEPS)
        ->and($html)->toContain('data-module="stepper"')
        ->and($html)->toContain('enctype="multipart/form-data"')
        ->and($html)->toContain('name="photos[]"')
        ->and($html)->toContain('name="hours[friday][open]"')
        ->and($html)->toContain('name="form_token"')
        ->and($html)->not->toContain('style="');
});

it('marks join and done noindex and keeps the business page indexable in production', function (): void {
    config(['app.env' => 'production']);

    expect($this->get('/directory/business')->getContent())->toContain('name="robots" content="index,follow')
        ->and($this->get('/directory/join')->getContent())->toContain('noindex')
        ->and($this->get('/directory/join/done')->getContent())->toContain('noindex');
});

it('stores a full submit with photos as a pending request with optimised media and notifies partnerships', function (): void {
    app(UpdateSettings::class)->handle(SettingGroup::Contact, ['partnership_email' => 'partners@ritme.test']);

    $response = $this->post('/directory/join', joinForm(['photos' => [joinPhoto('a.jpg', 1000, 700), joinPhoto('b.jpg', 800, 800)]]));

    $request = JoinRequest::query()->with('photos')->sole();
    $response->assertRedirect(route('directory.join.done'));

    expect($request->status)->toBe(JoinRequestStatus::Pending)
        ->and($request->code)->toMatch('/^\d{6}$/')
        ->and($request->mobile)->toBe('09121234567')
        ->and($request->email)->toBe('sara@example.com')
        ->and($request->latitude)->toBe(35.7561)
        ->and($request->booking_mode)->toBe(BookingMode::Phone)
        ->and($request->age_groups)->toBe(['6-12m', '1-2y', '2-4y'])
        ->and($request->ageRange()->label())->toBe('۶ ماه تا ۴ سال')
        ->and($request->opening_hours['saturday'])->toBe([['opens' => '09:00', 'closes' => '20:00']])
        ->and($request->opening_hours['thursday'])->toBe([['opens' => '09:00', 'closes' => '14:00']])
        ->and($request->opening_hours['friday'])->toBe([])
        ->and($request->terms_accepted_at)->not->toBeNull()
        ->and($request->photos)->toHaveCount(2);

    $media = $request->photos->first();
    assert($media instanceof Media);
    expect($media->optimized_at)->not->toBeNull()
        ->and($media->variants)->not->toBeEmpty()
        ->and($media->alt)->toContain('استخر مادر و کودک نیلوفر');
    expect(app(FindMediaUsages::class)->used([$media->id]))->toContain($media->id);

    Notification::assertSentTo(new AnonymousNotifiable, JoinRequestReceived::class,
        fn (JoinRequestReceived $n, array $channels, AnonymousNotifiable $to): bool => $to->routes['mail'] === 'partners@ritme.test');

    $this->get('/directory/join/done')->assertOk()
        ->assertSee('کد پیگیری '.fa_digits($request->code))
        ->assertSee('استخر مادر و کودک نیلوفر');
});

it('rejects invalid input with Persian messages, keeps old input and stores nothing', function (): void {
    $this->from('/directory/join')
        ->post('/directory/join', joinForm(['name' => '', 'mobile' => '12345', 'terms' => null, 'hours' => ['monday' => ['open' => '1', 'opens' => '', 'closes' => '']]]))
        ->assertRedirect(route('directory.join').'#join-form')
        ->assertSessionHasErrors(['name' => 'نام مجموعه را بنویس (۲ تا ۱۲۰ حرف).', 'mobile', 'terms', 'hours.monday.opens']);

    expect(JoinRequest::query()->count())->toBe(0);

    $html = (string) $this->get('/directory/join')->getContent();
    expect($html)->toContain('درخواست ثبت نشد')->toContain('data-start="1"');
});

it('rejects a district of another city, a non-image photo and more than the allowed photos', function (): void {
    $other = City::query()->create(['name' => 'کرج', 'slug' => 'karaj', 'is_active' => true]);
    $foreign = District::query()->create(['city_id' => $other->id, 'name' => 'گوهردشت', 'slug' => 'gohardasht']);

    $this->post('/directory/join', joinForm(['district' => $foreign->id]))->assertSessionHasErrors('district');

    $fake = new UploadedFile(MediaFixtures::text(), 'notes.jpg', 'image/jpeg', null, true);
    $this->post('/directory/join', joinForm(['photos' => [$fake]]))->assertSessionHasErrors('photos.0');

    $many = array_map(static fn (int $i): UploadedFile => UploadedFile::fake()->image("p{$i}.jpg", 10, 10), range(1, JoinForm::PHOTOS_MAX + 1));
    $this->post('/directory/join', joinForm(['photos' => $many]))->assertSessionHasErrors('photos');

    expect(JoinRequest::query()->count())->toBe(0)->and(Media::query()->count())->toBe(0);
});

it('answers spam (honeypot or too-fast post) exactly like a real submit and stores nothing', function (): void {
    $this->post('/directory/join', joinForm(['website' => 'http://spam.example', 'photos' => [joinPhoto()]]))
        ->assertRedirect(route('directory.join.done'))->assertSessionHasNoErrors();
    $this->post('/directory/join', joinForm([], age: 0))
        ->assertRedirect(route('directory.join.done'))->assertSessionHasNoErrors();

    expect(JoinRequest::query()->count())->toBe(0)->and(Media::query()->count())->toBe(0);
    Notification::assertNothingSent();
});

it('rate limits join posts per IP', function (): void {
    for ($i = 0; $i < 3; $i++) {
        $this->post('/directory/join', joinForm(['website' => 'x']))->assertRedirect();
    }

    $this->post('/directory/join', joinForm(['website' => 'x']))->assertStatus(429);
});

it('shows the done page without a code on a direct visit, never cached', function (): void {
    $response = $this->get('/directory/join/done')->assertOk()->assertSee('درخواستت رسید')->assertDontSee('کد پیگیری');

    expect($response->headers->get('Cache-Control'))->toContain('no-store')
        ->and(substr_count((string) $response->getContent(), '<h1'))->toBe(1);
});
