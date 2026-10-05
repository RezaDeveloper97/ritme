<?php

declare(strict_types=1);

use App\Domain\Directory\Booking\Enums\BookingStatus;
use App\Domain\Directory\Booking\Enums\TimeWindow;
use App\Domain\Directory\Booking\Models\BookingRequest;
use App\Domain\Directory\Booking\Support\BookingCode;
use App\Domain\Directory\Enums\PlaceStatus;
use App\Domain\Directory\Enums\ReviewStatus;
use App\Domain\Directory\Join\Actions\ApproveJoinRequest;
use App\Domain\Directory\Join\Actions\SubmitJoinRequest;
use App\Domain\Directory\Join\Enums\BookingMode;
use App\Domain\Directory\Join\Enums\JoinRequestStatus;
use App\Domain\Directory\Join\Models\JoinRequest;
use App\Domain\Directory\Models\Amenity;
use App\Domain\Directory\Models\City;
use App\Domain\Directory\Models\District;
use App\Domain\Directory\Models\Landing;
use App\Domain\Directory\Models\Place;
use App\Domain\Directory\Models\PlaceCategory;
use App\Domain\Directory\Models\PlaceReview;
use App\Domain\Media\Actions\StoreMedia;
use App\Domain\Media\Data\MediaUpload;
use App\Domain\Media\Models\Media;
use App\Domain\Seo\Models\SeoMeta;
use App\Domain\Seo\Schema\Enums\LocalBusinessType;
use App\Filament\Auth\AdminRole;
use App\Filament\Resources\Directory\Amenities\AmenityResource;
use App\Filament\Resources\Directory\Bookings\BookingRequestResource;
use App\Filament\Resources\Directory\Bookings\Pages\ListBookingRequests;
use App\Filament\Resources\Directory\Bookings\Pages\ViewBookingRequest;
use App\Filament\Resources\Directory\Categories\Pages\CreatePlaceCategory;
use App\Filament\Resources\Directory\Categories\PlaceCategoryResource;
use App\Filament\Resources\Directory\Cities\CityResource;
use App\Filament\Resources\Directory\Cities\Pages\EditCity;
use App\Filament\Resources\Directory\JoinRequests\JoinRequestResource;
use App\Filament\Resources\Directory\JoinRequests\Pages\ListJoinRequests;
use App\Filament\Resources\Directory\JoinRequests\Pages\ViewJoinRequest;
use App\Filament\Resources\Directory\Landings\LandingResource;
use App\Filament\Resources\Directory\Landings\Pages\CreateLanding;
use App\Filament\Resources\Directory\Places\Pages\CreatePlace;
use App\Filament\Resources\Directory\Places\Pages\EditPlace;
use App\Filament\Resources\Directory\Places\PlaceResource;
use App\Filament\Resources\Directory\Reviews\Pages\ListPlaceReviews;
use App\Filament\Resources\Directory\Reviews\PlaceReviewResource;
use App\Models\User;
use Database\Seeders\AdminRolesSeeder;
use Database\Seeders\SettingsSeeder;
use Filament\Actions\Testing\TestAction;
use Filament\Facades\Filament;
use Illuminate\Database\QueryException;
use Illuminate\Support\Facades\Storage;
use Livewire\Livewire;
use Spatie\Activitylog\Models\Activity;
use Tests\Feature\Admin\AdminMfa;
use Tests\Feature\Media\MediaFixtures;

function dirAdmin(?AdminRole $role = AdminRole::DirectoryManager): User
{
    $user = User::factory()->create();
    if ($role !== null) {
        $user->assignRole($role->value);
    }

    return AdminMfa::enrol($user); // PII roles must have MFA (L9-04b)
}

function dirAdminPhoto(string $alt, int $width = 1200, ?string $disk = null): Media
{
    return app(StoreMedia::class)->handle(new MediaUpload(MediaFixtures::jpeg($width, 800), $alt.'.jpg', alt: $alt, disk: $disk));
}

/**
 * A pending join request as SubmitJoinRequest stores it, with two photos on the private pending disk.
 */
function dirAdminJoinRequest(PlaceCategory $category, City $city, ?District $district = null): JoinRequest
{
    $parking = Amenity::factory()->create(['name' => 'پارکینگ']);
    $request = JoinRequest::query()->create([
        'code' => '482913',
        'name' => 'استخر کودک دلفین',
        'category_id' => $category->id,
        'city_id' => $city->id,
        'district_id' => $district?->id,
        'contact_name' => 'مریم احمدی',
        'mobile' => '09121234567',
        'email' => 'owner@example.com',
        'address' => 'تهران، خیابان آزمایشی، پلاک ۱۲',
        'phone' => '02122223333',
        'latitude' => 35.7,
        'longitude' => 51.4,
        'about' => "استخر آب گرم برای نوزادان و کودکان.\nمربیان دارای مدرک.",
        'age_groups' => ['6-12m', '1-2y', '2-4y'],
        'amenity_ids' => [$parking->id, 999_999],
        'opening_hours' => ['saturday' => [['opens' => '09:00', 'closes' => '18:00']], 'friday' => []],
        'services' => "- آشنایی با آب، ۴۵ دقیقه\n\nشنای مادر و نوزاد",
        'booking_mode' => BookingMode::Phone,
        'terms_accepted_at' => now(),
    ]);
    $request->photos()->sync([
        dirAdminPhoto('نمای استخر', disk: SubmitJoinRequest::PHOTO_DISK)->id => ['sort_order' => 0],
        dirAdminPhoto('رختکن', 1000, SubmitJoinRequest::PHOTO_DISK)->id => ['sort_order' => 1],
    ]);

    return $request;
}

function dirAdminBooking(Place $place, string $status = 'new'): BookingRequest
{
    return BookingRequest::query()->create([
        'code' => BookingCode::generate(), // unique: a 4-hex random prefix collided now and then
        'status' => $status,
        'place_id' => $place->id,
        'place_name' => $place->name,
        'preferred_date' => '2026-10-07',
        'time_window' => TimeWindow::Evening,
        'parent_name' => 'سارا رحیمی',
        'mobile' => '09121234567',
        'child_age_months' => 14,
        'note' => 'اولین جلسه است.',
    ]);
}

beforeEach(function (): void {
    Storage::fake('public');
    Storage::fake('pending', ['url' => '/admin/pending-media']);
    $this->seed([SettingsSeeder::class, AdminRolesSeeder::class]);
    Filament::setCurrentPanel('admin');
});

it('lets directory managers and super-admins into every directory resource and denies everyone else', function (): void {
    $place = Place::factory()->create();
    $urls = [
        PlaceResource::getUrl('index'), PlaceResource::getUrl('create'), PlaceResource::getUrl('edit', ['record' => $place]),
        PlaceCategoryResource::getUrl('index'), CityResource::getUrl('index'), AmenityResource::getUrl('index'),
        LandingResource::getUrl('index'), PlaceReviewResource::getUrl('index'), BookingRequestResource::getUrl('index'),
        JoinRequestResource::getUrl('index'),
    ];

    $this->get($urls[0])->assertRedirect('/admin/login');

    foreach ([AdminRole::DirectoryManager, AdminRole::SuperAdmin] as $role) {
        $user = dirAdmin($role);
        foreach ($urls as $url) {
            $this->actingAs($user)->get($url)->assertOk();
        }
    }

    foreach ([AdminRole::Editor, AdminRole::SeoManager, AdminRole::ShopManager, AdminRole::Support, null] as $role) {
        $user = dirAdmin($role);
        foreach ([$urls[0], $urls[7], $urls[8], $urls[9], BookingRequestResource::getUrl('export')] as $url) {
            $this->actingAs($user)->get($url)->assertForbidden();
        }
    }

    $manager = dirAdmin();
    $review = PlaceReview::factory()->create(['place_id' => $place->id]);
    $booking = dirAdminBooking($place);
    expect($manager->can('create', PlaceReview::class))->toBeFalse()
        ->and($manager->can('update', $review))->toBeTrue()
        ->and($manager->can('create', BookingRequest::class))->toBeFalse()
        ->and($manager->can('delete', $booking))->toBeFalse()
        ->and($manager->can('export', BookingRequest::class))->toBeTrue()
        ->and($manager->can('create', JoinRequest::class))->toBeFalse();
});

it('converts a join request into a draft place with its data and photos, then publishes it onto /directory', function (): void {
    $this->actingAs($manager = dirAdmin());
    $category = PlaceCategory::factory()->create(['name' => 'استخر کودک']);
    $city = City::factory()->create(['name' => 'تهران', 'slug' => 'tehran']);
    $district = District::factory()->create(['city_id' => $city->id, 'name' => 'ونک']);
    $request = dirAdminJoinRequest($category, $city, $district);
    $photoIds = $request->photos()->pluck('media.id')->map(intval(...))->all();

    $this->get('/directory')->assertOk()->assertDontSee('استخر کودک دلفین');

    Livewire::test(ListJoinRequests::class)->assertCanSeeTableRecords([$request])->assertDontSee('09121234567');

    Livewire::test(ViewJoinRequest::class, ['record' => $request->getRouteKey()])
        ->assertSee('مریم احمدی')
        ->callAction('approve', ['category_id' => $category->id, 'city_id' => $city->id])
        ->assertHasNoActionErrors();

    $place = Place::query()->where('name', 'استخر کودک دلفین')->firstOrFail();

    // L9-04b (F17): approval moved the photos to the public media disk (+ variants) and removed the pending files.
    foreach (Media::query()->whereKey($photoIds)->get() as $photo) {
        expect($photo->disk)->toBe('public')
            ->and($photo->optimized_at)->not->toBeNull()
            ->and($photo->variants)->not->toBeEmpty()
            ->and(Storage::disk('public')->exists($photo->path()))->toBeTrue()
            ->and(Storage::disk('pending')->exists($photo->path()))->toBeFalse();
    }
    expect(Storage::disk('pending')->allFiles())->toBe([]);

    expect($request->refresh()->status)->toBe(JoinRequestStatus::Approved)
        ->and($place->status)->toBe(PlaceStatus::Draft)
        ->and($place->category_id)->toBe($category->id)
        ->and($place->district_id)->toBe($district->id)
        ->and($place->phones)->toBe(['02122223333'])
        ->and($place->booking_mode)->toBe(BookingMode::Phone)
        ->and($place->age_min_months)->toBe(6)
        ->and($place->age_max_months)->toBe(48)
        ->and($place->opening_hours)->toBe(['saturday' => [['opens' => '09:00', 'closes' => '18:00']], 'friday' => []])
        ->and($place->summary)->toBe('استخر آب گرم برای نوزادان و کودکان. مربیان دارای مدرک.')
        ->and($place->cover_media_id)->toBe($photoIds[0])
        ->and($place->gallery()->pluck('media.id')->map(intval(...))->all())->toBe([$photoIds[1]])
        ->and($place->amenities()->pluck('name')->all())->toBe(['پارکینگ'])
        ->and($place->services()->pluck('name')->all())->toBe(['آشنایی با آب، ۴۵ دقیقه', 'شنای مادر و نوزاد'])
        ->and((string) $place->description)->not->toContain('09121234567')->not->toContain('owner@example.com');

    $approved = Activity::query()->where('log_name', 'directory')->where('description', 'directory.join.approved')->firstOrFail();
    expect($approved->causer_id)->toBe($manager->id)
        ->and($approved->subject_id)->toBe($request->id)
        ->and($approved->properties['place_id'])->toBe($place->id);

    // Not public while it is a draft.
    $this->get('/directory')->assertDontSee('استخر کودک دلفین');
    $this->get(route('directory.place', [$place->slug]))->assertNotFound();

    Livewire::test(EditPlace::class, ['record' => $place->getRouteKey()])
        ->callAction('publish')
        ->assertHasNoActionErrors();

    expect($place->refresh()->status)->toBe(PlaceStatus::Published)
        ->and(Activity::query()->where('description', 'directory.place.status')->where('subject_id', $place->id)->where('causer_id', $manager->id)->exists())->toBeTrue();

    $this->get('/directory')->assertOk()->assertSee('استخر کودک دلفین');
    $page = (string) $this->get(route('directory.place', [$place->slug]))->assertOk()->getContent();
    // Phone-only place: no booking request form, the number instead.
    expect($page)->not->toContain('name="time_window"')->toContain('tel:02122223333');
    $this->post(route('directory.place.book', [$place->slug]), [])->assertNotFound();

    // A converted request cannot be converted twice.
    Livewire::test(ViewJoinRequest::class, ['record' => $request->getRouteKey()])->assertActionHidden('approve');
});

it('streams pending join photos only to admins who may review them (F17)', function (): void {
    $request = dirAdminJoinRequest(PlaceCategory::factory()->create(), City::factory()->create());
    $photo = $request->photos()->firstOrFail();
    $url = Storage::disk('pending')->url($photo->path());

    expect($url)->toBe('/admin/pending-media/'.$photo->path())
        ->and(Storage::disk('public')->allFiles())->toBe([]);

    // Guests: the panel login, never the file.
    $this->get($url)->assertRedirect('/admin/login');

    foreach ([AdminRole::DirectoryManager, AdminRole::SuperAdmin, AdminRole::Editor] as $role) {
        $response = $this->actingAs(dirAdmin($role))->get($url)->assertOk()
            ->assertHeader('Content-Type', 'image/jpeg')
            ->assertHeader('X-Content-Type-Options', 'nosniff')
            ->assertHeader('X-Robots-Tag', 'noindex, nofollow');
        expect((string) $response->headers->get('Cache-Control'))->toContain('no-store')->toContain('private')
            ->and((string) $response->headers->get('Content-Security-Policy'))->toContain('sandbox')
            ->and($response->streamedContent())->toBe(Storage::disk('pending')->get($photo->path()));
    }

    // Support reviews neither join requests nor the media library; a user without a role is no admin at all.
    $this->actingAs(dirAdmin(AdminRole::Support))->get($url)->assertForbidden();
    $this->actingAs(dirAdmin(null))->get($url)->assertForbidden();

    $manager = dirAdmin();
    $this->actingAs($manager)->get('/admin/pending-media/'.dirname($photo->path()).'/missing.jpg')->assertNotFound();
    $this->actingAs($manager)->get('/admin/pending-media/../../.env')->assertNotFound();
    Storage::disk('pending')->put('2026/10/abcdefghij/note.txt', 'not an image');
    $this->actingAs($manager)->get('/admin/pending-media/2026/10/abcdefghij/note.txt')->assertNotFound();

    // The review page previews them through that route.
    $this->actingAs($manager);
    Livewire::test(ViewJoinRequest::class, ['record' => $request->getRouteKey()])->assertSee('/admin/pending-media/', false);
});

it('keeps the photos pending and leaves no public copies when an approval fails', function (): void {
    $this->actingAs(dirAdmin());
    $category = PlaceCategory::factory()->create();
    $request = dirAdminJoinRequest($category, City::factory()->create());
    $photos = $request->photos()->get();
    // A copy that cannot be written (the second photo's target path is taken) aborts everything.
    Storage::disk('public')->put($photos[1]->path(), 'occupied');

    expect(fn () => app(ApproveJoinRequest::class)->handle($request))->toThrow(RuntimeException::class);

    expect($request->refresh()->status)->toBe(JoinRequestStatus::Pending)
        ->and(Place::query()->count())->toBe(0)
        ->and(Media::query()->whereKey($photos->modelKeys())->pluck('disk')->unique()->all())->toBe([SubmitJoinRequest::PHOTO_DISK])
        ->and(Storage::disk('pending')->exists($photos[0]->path()))->toBeTrue()
        ->and(Storage::disk('pending')->exists($photos[1]->path()))->toBeTrue()
        ->and(Storage::disk('public')->allFiles())->toBe([$photos[1]->path()]); // only the blocker, no copy of photo 1

    // A failure inside the transaction rolls the disk switch back and removes the copies as well.
    Storage::disk('public')->delete($photos[1]->path());
    expect(fn () => app(ApproveJoinRequest::class)->handle($request, 999_999))->toThrow(QueryException::class); // unknown category (FK)
    expect($request->refresh()->status)->toBe(JoinRequestStatus::Pending)
        ->and(Media::query()->whereKey($photos->modelKeys())->pluck('disk')->unique()->all())->toBe([SubmitJoinRequest::PHOTO_DISK])
        ->and(Storage::disk('public')->allFiles())->toBe([])
        ->and(Storage::disk('pending')->allFiles())->toHaveCount(2);
});

it('promotes a pending photo when the same image is uploaded to the public library', function (): void {
    $pending = dirAdminPhoto('نمای استخر', disk: SubmitJoinRequest::PHOTO_DISK);
    expect($pending->disk)->toBe(SubmitJoinRequest::PHOTO_DISK);

    $public = dirAdminPhoto('نمای استخر');

    expect($public->id)->toBe($pending->id)
        ->and($public->disk)->toBe('public')
        ->and($public->optimized_at)->not->toBeNull()
        ->and(Storage::disk('public')->exists($public->path()))->toBeTrue()
        ->and(Storage::disk('pending')->allFiles())->toBe([]);
});

it('rejects a join request with an activity-log entry', function (): void {
    $this->actingAs($manager = dirAdmin());
    $request = dirAdminJoinRequest(PlaceCategory::factory()->create(), City::factory()->create());

    Livewire::test(ListJoinRequests::class)
        ->callAction(TestAction::make('reject')->table($request))
        ->assertHasNoActionErrors();

    expect($request->refresh()->status)->toBe(JoinRequestStatus::Rejected)
        ->and(Place::query()->count())->toBe(0)
        ->and(Activity::query()->where('description', 'directory.join.rejected')->where('causer_id', $manager->id)->exists())->toBeTrue();
});

it('creates a place with services, opening hours, amenities, gallery and SEO, as a draft', function (): void {
    $this->actingAs(dirAdmin());
    $category = PlaceCategory::factory()->create();
    $city = City::factory()->create();
    $district = District::factory()->create(['city_id' => $city->id]);
    $amenities = Amenity::factory()->count(2)->create();
    $cover = dirAdminPhoto('کاور');
    $photo = dirAdminPhoto('سالن', 1000);

    Livewire::test(CreatePlace::class)
        ->fillForm([
            'name' => 'مهد کودک باران',
            'category_id' => $category->id,
            'city_id' => $city->id,
        ])
        ->fillForm([
            'district_id' => $district->id,
            'summary' => 'مهدکودک با فضای باز.',
            'phones' => ['۰۲۱ ۸۸۸۸ ۷۷۷۷'],
            'booking_mode' => BookingMode::Online->value,
            'amenity_ids' => [$amenities[1]->id],
            'hours' => [
                'saturday' => ['state' => 'open', 'ranges' => [['opens' => '08:00', 'closes' => '16:00']]],
                'friday' => ['state' => 'closed', 'ranges' => []],
            ],
            'services' => [
                ['name' => 'ثبت‌نام ماهانه', 'price' => 4_500_000, 'price_unit' => 'ماهانه'],
                ['name' => 'کلاس نقاشی', 'price' => 900_000],
            ],
            'cover_media_id' => $cover->id,
            'gallery_items' => [['media_id' => $photo->id]],
            'seoMeta.title' => 'مهد کودک باران در ونک',
        ])
        ->call('create')
        ->assertHasNoFormErrors();

    $place = Place::query()->where('name', 'مهد کودک باران')->firstOrFail();
    expect($place->status)->toBe(PlaceStatus::Draft)
        ->and($place->slug)->not->toBe('')
        ->and($place->phones)->toBe(['021 8888 7777'])
        ->and($place->opening_hours)->toBe(['saturday' => [['opens' => '08:00', 'closes' => '16:00']], 'friday' => []])
        ->and($place->amenities()->pluck('directory_amenities.id')->map(intval(...))->all())->toBe([$amenities[1]->id])
        ->and($place->gallery()->pluck('media.id')->map(intval(...))->all())->toBe([$photo->id])
        ->and($place->services()->count())->toBe(2)
        ->and($place->refresh()->price_from)->toBe(900_000)
        ->and(SeoMeta::query()->where('seoable_id', $place->id)->value('title'))->toBe('مهد کودک باران در ونک')
        ->and(Activity::query()->where('description', 'directory.place.created')->where('subject_id', $place->id)->exists())->toBeTrue();

    // The edit form round-trips the hours editor, amenities and gallery.
    Livewire::test(EditPlace::class, ['record' => $place->getRouteKey()])
        ->assertSchemaStateSet([
            'hours.saturday.state' => 'open',
            'hours.friday.state' => 'closed',
            'hours.monday.state' => 'unknown',
            'amenity_ids' => [$amenities[1]->id],
        ])
        ->fillForm(['hours.monday' => ['state' => 'open', 'ranges' => [['opens' => '22:00', 'closes' => '02:00']]]])
        ->call('save')
        ->assertHasNoFormErrors();

    expect($place->refresh()->opening_hours['monday'] ?? null)->toBe([['opens' => '22:00', 'closes' => '02:00']])
        ->and($place->status)->toBe(PlaceStatus::Draft);
});

it('moderates reviews one by one and in bulk, recalculating the rating and logging every change', function (): void {
    $this->actingAs($manager = dirAdmin());
    $place = Place::factory()->published()->create();
    $a = PlaceReview::factory()->create(['place_id' => $place->id, 'rating' => 5, 'status' => ReviewStatus::Pending]);
    $b = PlaceReview::factory()->create(['place_id' => $place->id, 'rating' => 3, 'status' => ReviewStatus::Pending]);
    $c = PlaceReview::factory()->create(['place_id' => $place->id, 'rating' => 1, 'status' => ReviewStatus::Pending]);

    Livewire::test(ListPlaceReviews::class)
        ->assertCanSeeTableRecords([$a, $b, $c])
        ->selectTableRecords([$a->id, $b->id])
        ->callAction(TestAction::make('approveBulk')->table()->bulk())
        ->assertHasNoActionErrors();

    expect($place->refresh()->rating_count)->toBe(2)
        ->and($place->rating_avg)->toBe(4.0);

    Livewire::test(ListPlaceReviews::class)
        ->callAction(TestAction::make('reject')->table($c))
        ->assertHasNoActionErrors();

    expect($c->refresh()->status)->toBe(ReviewStatus::Rejected)
        ->and(Activity::query()->where('description', 'directory.review.status')->where('causer_id', $manager->id)->count())->toBe(3);
});

it('moves bookings along the status flow only, masks the mobile in lists and exports a minimal CSV', function (): void {
    $this->actingAs($manager = dirAdmin());
    $place = Place::factory()->published()->create(['name' => 'استخر آبی']);
    $booking = dirAdminBooking($place);
    $done = dirAdminBooking($place, 'done');

    Livewire::test(ListBookingRequests::class)
        ->assertCanSeeTableRecords([$booking])
        ->assertCanNotSeeTableRecords([$done])
        ->assertSee('۰۹۱۲•••••۶۷')
        ->assertDontSee('09121234567')
        ->assertActionVisible(TestAction::make('status_confirmed')->table($booking))
        ->assertActionHidden(TestAction::make('status_done')->table($booking))
        ->callAction(TestAction::make('status_confirmed')->table($booking))
        ->assertHasNoActionErrors();

    expect($booking->refresh()->status)->toBe(BookingStatus::Confirmed);

    Livewire::test(ViewBookingRequest::class, ['record' => $booking->getRouteKey()])
        ->assertSee('tel:09121234567', escape: false)
        ->assertActionHidden('status_confirmed')
        ->callAction('status_done')
        ->assertHasNoActionErrors();

    expect($booking->refresh()->status)->toBe(BookingStatus::Done)
        ->and(Activity::query()->where('description', 'directory.booking.status')->where('subject_id', $booking->id)->count())->toBe(2);

    // Status page of the parent follows the admin change.
    $this->get('/directory/booked/'.$booking->code)->assertOk();

    $csv = $this->get(BookingRequestResource::getUrl('export', ['status' => 'done']))
        ->assertOk()
        ->assertHeader('Cache-Control', 'no-store, private')
        ->streamedContent();

    expect($csv)->toStartWith("\xEF\xBB\xBF")
        ->toContain('استخر آبی')
        ->toContain('0912•••••67')
        ->not->toContain('09121234567')
        ->not->toContain('اولین جلسه است')
        ->and(substr_count(trim($csv), "\n"))->toBe(2)
        ->and(Activity::query()->where('description', 'directory.booking.exported')->where('causer_id', $manager->id)->exists())->toBeTrue();
});

it('edits taxonomy and landing texts: schema type and icon on categories, districts on cities, one text per city × category', function (): void {
    $this->actingAs(dirAdmin());

    Livewire::test(CreatePlaceCategory::class)
        ->fillForm(['name' => 'مهدکودک', 'schema_type' => LocalBusinessType::ChildCare->value, 'icon' => 'heart'])
        ->call('create')
        ->assertHasNoFormErrors();
    $category = PlaceCategory::query()->where('name', 'مهدکودک')->firstOrFail();
    expect($category->schema_type)->toBe(LocalBusinessType::ChildCare)->and($category->slug)->not->toBe('');

    Livewire::test(CreatePlaceCategory::class)
        ->fillForm(['name' => 'دیگر', 'schema_type' => LocalBusinessType::Store->value, 'icon' => 'not-an-icon'])
        ->call('create')
        ->assertHasFormErrors(['icon']);

    $city = City::factory()->create();
    Livewire::test(EditCity::class, ['record' => $city->getRouteKey()])
        ->fillForm(['districts' => [['name' => 'سعادت‌آباد'], ['name' => 'ونک']]])
        ->call('save')
        ->assertHasNoFormErrors();
    expect($city->districts()->pluck('name')->all())->toBe(['سعادت‌آباد', 'ونک']);

    Livewire::test(CreateLanding::class)
        ->fillForm(['city_id' => $city->id, 'category_id' => $category->id, 'h1' => 'بهترین مهدها؟ نه — مهدهای نزدیک تو', 'intro' => 'متن معرفی.'])
        ->call('create')
        ->assertHasNoFormErrors();
    expect(Landing::query()->where('city_id', $city->id)->where('category_id', $category->id)->value('intro'))->toBe('متن معرفی.');

    Livewire::test(CreateLanding::class)
        ->fillForm(['city_id' => $city->id, 'category_id' => $category->id, 'h1' => 'تکراری'])
        ->call('create')
        ->assertHasFormErrors(['city_id']);

    // A city with places cannot be deleted.
    Place::factory()->create(['city_id' => $city->id, 'category_id' => $category->id]);
    Livewire::test(EditCity::class, ['record' => $city->getRouteKey()])->assertActionDisabled('delete');
});
