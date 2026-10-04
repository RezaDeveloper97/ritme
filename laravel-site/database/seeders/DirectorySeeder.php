<?php

declare(strict_types=1);

namespace Database\Seeders;

use App\Domain\Directory\Actions\SyncPlaceAmenities;
use App\Domain\Directory\Enums\PlaceStatus;
use App\Domain\Directory\Enums\ReviewStatus;
use App\Domain\Directory\Models\Amenity;
use App\Domain\Directory\Models\City;
use App\Domain\Directory\Models\District;
use App\Domain\Directory\Models\Landing;
use App\Domain\Directory\Models\Place;
use App\Domain\Directory\Models\PlaceCategory;
use App\Domain\Directory\Models\PlaceReview;
use App\Domain\Directory\Models\PlaceService;
use App\Domain\Seo\Schema\Enums\LocalBusinessType;
use Illuminate\Database\Seeder;

/**
 * DEMO directory content from the design (directory.html, directory-place.html): Tehran with five districts, the eight
 * category chips, amenities, and the six sample places incl. «استخر مادر و کودک آب‌پری» (`ab-pari`). Not called from
 * DatabaseSeeder — demo data never reaches production by accident:
 *
 *     php artisan db:seed --class=DirectorySeeder
 *
 * Every place is `is_demo` (placeholder business: invented name, `[...]` address, district-centre coordinates, no
 * phone). The two sample reviews are `is_demo` too: they may be shown labelled as samples but never count towards a
 * rating — the design's «۴٫۸ (۱۲۶)» is NOT seeded, so no place has an invented rating. Idempotent: rows are matched by
 * slug and existing places are left untouched. Copy follows the content red lines.
 */
final class DirectorySeeder extends Seeder
{
    public const DEMO_PLACE_SLUG = 'ab-pari';

    public function run(): void
    {
        $city = City::query()->firstOrCreate(['slug' => 'tehran'], ['name' => 'تهران', 'province' => 'تهران', 'sort_order' => 1]);
        $districts = $this->districts($city);
        $categories = $this->categories();
        $amenities = $this->amenities();

        Landing::query()->firstOrCreate(
            ['city_id' => $city->id, 'category_id' => null],
            ['intro' => 'کلاس مادر و کودک، استخر، خانه بازی و کارگاه‌ها را ببین، صفحه هر مجموعه را بخوان و همین‌جا وقت رزرو کن.'],
        );

        $sync = app(SyncPlaceAmenities::class);
        foreach ($this->places() as $data) {
            if (Place::query()->where('slug', $data['slug'])->exists()) {
                continue;
            }

            $place = new Place([
                'name' => $data['name'],
                'slug' => $data['slug'],
                'category_id' => $categories[$data['category']]->id,
                'city_id' => $city->id,
                'district_id' => $districts[$data['district']]->id,
                'summary' => $data['summary'],
                'description' => $data['description'],
                'address' => '[آدرس کامل مجموعه]',
                'latitude' => $data['lat'],
                'longitude' => $data['lng'],
                'phones' => [],
                'age_min_months' => $data['age'][0],
                'age_max_months' => $data['age'][1],
                'opening_hours' => $data['hours'],
                'rules' => $data['rules'] ?? null,
                'cancellation_policy' => $data['cancellation'] ?? null,
                'status' => PlaceStatus::Published,
                'is_verified' => $data['verified'],
                'is_demo' => true,
            ]);
            $place->save();

            foreach ($data['services'] as $i => $service) {
                PlaceService::query()->create(['place_id' => $place->id, 'sort_order' => $i, ...$service]);
            }

            $sync->handle($place, array_map(static fn (string $slug): int => $amenities[$slug]->id, $data['amenities']));

            foreach ($data['reviews'] ?? [] as $review) {
                PlaceReview::query()->create(['place_id' => $place->id, 'status' => ReviewStatus::Approved, 'is_demo' => true, ...$review]);
            }
        }
    }

    /**
     * @return array<string, District>
     */
    private function districts(City $city): array
    {
        $rows = ['vanak' => 'ونک', 'yousefabad' => 'یوسف‌آباد', 'jordan' => 'جردن', 'saadatabad' => 'سعادت‌آباد', 'zafaranieh' => 'زعفرانیه'];
        $districts = [];
        $order = 0;
        foreach ($rows as $slug => $name) {
            $districts[$slug] = District::query()->firstOrCreate(['city_id' => $city->id, 'slug' => $slug], ['name' => $name, 'sort_order' => ++$order]);
        }

        return $districts;
    }

    /**
     * The category chips of directory.html with their schema.org LocalBusiness subtype.
     *
     * @return array<string, PlaceCategory>
     */
    private function categories(): array
    {
        $rows = [
            'mother-child-class' => ['کلاس مادر و کودک', LocalBusinessType::ChildCare, 'users'],
            'pool' => ['استخر مادر و کودک', LocalBusinessType::SportsActivityLocation, 'waves'],
            'playhouse' => ['خانه بازی', LocalBusinessType::ChildCare, 'home'],
            'music-art' => ['موسیقی و هنر', LocalBusinessType::LocalBusiness, 'music'],
            'postpartum-exercise' => ['ورزش پس از زایمان', LocalBusinessType::ExerciseGym, 'exercise'],
            'baby-massage' => ['ماساژ نوزاد', LocalBusinessType::HealthAndBeautyBusiness, 'hand'],
            'parenting-workshop' => ['کارگاه فرزندپروری', LocalBusinessType::LocalBusiness, 'book'],
            'outdoor' => ['پارک و فضای باز', LocalBusinessType::LocalBusiness, 'tree'],
        ];

        $categories = [];
        $order = 0;
        foreach ($rows as $slug => [$name, $type, $icon]) {
            $categories[$slug] = PlaceCategory::query()->firstOrCreate(
                ['slug' => $slug],
                ['name' => $name, 'schema_type' => $type, 'icon' => $icon, 'sort_order' => ++$order],
            );
        }

        return $categories;
    }

    /**
     * @return array<string, Amenity>
     */
    private function amenities(): array
    {
        $rows = [
            'female-coach' => ['مربی خانم', 'female', true],
            'nursing-room' => ['اتاق شیردهی', 'bottle', true],
            'online-booking' => ['رزرو آنلاین', 'calendar', true],
            'warm-water' => ['آب گرم', 'thermometer', false],
            'changing-table' => ['میز تعویض', 'check', false],
            'stroller-parking' => ['جای کالسکه', 'check', false],
            'parking' => ['پارکینگ', 'parking', false],
            'small-groups' => ['گروه‌های کوچک', 'users', false],
        ];

        $amenities = [];
        $order = 0;
        foreach ($rows as $slug => [$name, $icon, $filter]) {
            $amenities[$slug] = Amenity::query()->firstOrCreate(
                ['slug' => $slug],
                ['name' => $name, 'icon' => $icon, 'is_filter' => $filter, 'sort_order' => ++$order],
            );
        }

        return $amenities;
    }

    /**
     * @return array<string, list<array{opens: string, closes: string}>>
     */
    private static function hours(string $opens, string $closes, ?string $thursdayCloses = null): array
    {
        $day = [['opens' => $opens, 'closes' => $closes]];

        return [
            'saturday' => $day, 'sunday' => $day, 'monday' => $day, 'tuesday' => $day, 'wednesday' => $day,
            'thursday' => $thursdayCloses === null ? [] : [['opens' => $opens, 'closes' => $thursdayCloses]],
            'friday' => [],
        ];
    }

    /**
     * @return list<array<string, mixed>>
     */
    private function places(): array
    {
        return [
            [
                'slug' => self::DEMO_PLACE_SLUG,
                'name' => 'استخر مادر و کودک آب‌پری',
                'category' => 'pool',
                'district' => 'vanak',
                'verified' => true,
                'lat' => 35.7575,
                'lng' => 51.41,
                'age' => [6, 48],
                'summary' => 'استخر سرپوشیده مادر و کودک با آب گرم و کلاس‌های گروهی کوچک.',
                'description' => 'استخر سرپوشیده ویژه مادر و کودک با آب گرم و کلاس‌های گروهی کوچک. هر کلاس حداکثر ۶ مادر و کودک دارد و مربی کنار آب همراهتان است. رختکن خانوادگی، اتاق شیردهی و جای کالسکه دارد.',
                'hours' => self::hours('09:00', '20:00', '14:00'),
                'rules' => "کلاه شنا و پوشک مخصوص آب لازم است\n۱۵ دقیقه قبل از شروع کلاس برسید",
                'cancellation' => 'لغو رایگان تا ۲۴ ساعت قبل؛ بعد از آن [شرایط لغو مجموعه]',
                'amenities' => ['female-coach', 'warm-water', 'nursing-room', 'changing-table', 'stroller-parking', 'parking', 'small-groups', 'online-booking'],
                'services' => [
                    ['name' => 'آشنایی با آب', 'duration_minutes' => 45, 'price' => 320_000, 'price_unit' => 'هر جلسه', 'details' => 'گروهی', 'age_min_months' => 6, 'age_max_months' => 18],
                    ['name' => 'شنای مادر و کودک', 'duration_minutes' => 45, 'price' => 350_000, 'price_unit' => 'هر جلسه', 'age_min_months' => 18, 'age_max_months' => 48],
                    ['name' => 'بسته ۸ جلسه', 'price' => 2_400_000, 'details' => 'اعتبار ۶۰ روز'],
                    ['name' => 'جلسه خصوصی', 'duration_minutes' => 30, 'price' => 650_000, 'price_unit' => 'هر جلسه', 'details' => 'با یک مربی'],
                ],
                'reviews' => [
                    ['author_name' => 'مادر رها', 'rating' => 5, 'aspects' => ['cleanliness' => 5, 'staff' => 5, 'facilities' => 5, 'access' => 4],
                        'body' => 'مربی خیلی صبور بود و آب واقعاً گرم. اتاق شیردهی تمیز و آرام بود.', 'approved_at' => now()->subWeeks(2)],
                    ['author_name' => 'مادر نیلا', 'rating' => 4, 'aspects' => ['cleanliness' => 4, 'staff' => 5, 'facilities' => 4, 'access' => 3],
                        'body' => 'کلاس خوب بود؛ فقط پارکینگ ساعت ۵ عصر شلوغ است.', 'approved_at' => now()->subMonth()],
                ],
            ],
            [
                'slug' => 'tab-tab',
                'name' => 'خانه بازی تاب‌تاب',
                'category' => 'playhouse',
                'district' => 'yousefabad',
                'verified' => false,
                'lat' => 35.7297,
                'lng' => 51.4069,
                'age' => [12, 72],
                'summary' => 'خانه بازی سرپوشیده با بازی‌های حرکتی و گوشه آرام برای کودکان کوچک‌تر.',
                'description' => 'فضای بازی سرپوشیده با وسایل حرکتی نرم، گوشه کتاب و بازی آرام برای کودکان کوچک‌تر و جای نشستن برای همراه.',
                'hours' => self::hours('10:00', '21:00', '21:00'),
                'amenities' => ['nursing-room', 'changing-table', 'stroller-parking'],
                'services' => [
                    ['name' => 'ورود یک ساعته', 'duration_minutes' => 60, 'price' => 180_000, 'price_unit' => 'هر کودک', 'age_min_months' => 12, 'age_max_months' => 72],
                ],
            ],
            [
                'slug' => 'ghoncheh',
                'name' => 'کلاس حرکت و بازی غنچه',
                'category' => 'mother-child-class',
                'district' => 'jordan',
                'verified' => true,
                'lat' => 35.7686,
                'lng' => 51.417,
                'age' => [12, 36],
                'summary' => 'کلاس‌های گروهی حرکت و بازی برای کودک و همراهش.',
                'description' => 'کلاس‌های گروهی کوچک با بازی‌های حرکتی، ریتم و تعامل؛ مادر یا همراه کودک در تمام کلاس کنار اوست.',
                'hours' => self::hours('09:00', '18:00', '13:00'),
                'amenities' => ['female-coach', 'small-groups', 'online-booking'],
                'services' => [
                    ['name' => 'کلاس حرکت و بازی', 'duration_minutes' => 60, 'price' => 240_000, 'price_unit' => 'هر جلسه', 'details' => 'گروهی', 'age_min_months' => 12, 'age_max_months' => 36],
                ],
            ],
            [
                'slug' => 'naghmeh',
                'name' => 'موسیقی کودک نغمه',
                'category' => 'music-art',
                'district' => 'saadatabad',
                'verified' => false,
                'lat' => 35.7813,
                'lng' => 51.3756,
                'age' => [18, 60],
                'summary' => 'آشنایی با ریتم و آواز در کلاس‌های گروهی کوچک.',
                'description' => 'کلاس‌های موسیقی کودک با ساز‌های کوبه‌ای ساده، آواز و بازی‌های ریتمیک در گروه‌های کوچک.',
                'hours' => self::hours('10:00', '19:00'),
                'amenities' => ['small-groups'],
                'services' => [
                    ['name' => 'کلاس موسیقی کودک', 'duration_minutes' => 45, 'price' => 260_000, 'price_unit' => 'هر جلسه', 'age_min_months' => 18, 'age_max_months' => 60],
                ],
            ],
            [
                'slug' => 'aramesh',
                'name' => 'یوگای مادر و نوزاد آرامش',
                'category' => 'postpartum-exercise',
                'district' => 'zafaranieh',
                'verified' => true,
                'lat' => 35.8,
                'lng' => 51.414,
                'age' => [2, 12],
                'summary' => 'حرکات ملایم بدنی برای مادر بعد از زایمان، همراه نوزاد.',
                'description' => 'کلاس‌های حرکت ملایم برای دوره پس از زایمان، همراه نوزاد. پیش از شروع ورزش بعد از زایمان با پزشک یا ماما مشورت کن.',
                'hours' => self::hours('09:00', '17:00', '12:00'),
                'amenities' => ['female-coach', 'nursing-room', 'changing-table'],
                'services' => [
                    ['name' => 'یوگای مادر و نوزاد', 'duration_minutes' => 60, 'price' => 220_000, 'price_unit' => 'هر جلسه', 'details' => 'گروهی'],
                ],
            ],
            [
                'slug' => 'navazesh',
                'name' => 'ماساژ نوزاد نوازش',
                'category' => 'baby-massage',
                'district' => 'vanak',
                'verified' => false,
                'lat' => 35.756,
                'lng' => 51.408,
                'age' => [1, 12],
                'summary' => 'آموزش ماساژ نوزاد به مادر و پدر در جلسه‌های کوچک.',
                'description' => 'در این کارگاه‌ها مادر یا پدر حرکات ساده ماساژ نوزاد را یاد می‌گیرد. اگر نوزاد بیماری یا مشکل پوستی دارد، قبل از شروع با پزشک کودک مشورت کن.',
                'hours' => self::hours('10:00', '18:00', '13:00'),
                'amenities' => ['female-coach', 'nursing-room', 'changing-table', 'online-booking'],
                'services' => [
                    ['name' => 'جلسه آموزش ماساژ نوزاد', 'duration_minutes' => 60, 'price' => 280_000, 'price_unit' => 'هر جلسه', 'age_min_months' => 1, 'age_max_months' => 12],
                ],
            ],
        ];
    }
}
