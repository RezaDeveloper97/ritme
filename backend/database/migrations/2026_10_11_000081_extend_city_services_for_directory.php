<?php

use Illuminate\Database\Migrations\Migration;
use Illuminate\Database\Schema\Blueprint;
use Illuminate\Support\Facades\DB;
use Illuminate\Support\Facades\Schema;

return new class extends Migration
{
    /**
     * Twin of backend-go/db/migrations/00081_directory.sql: the mother & child directory (CB-DIR-01) on top of
     * bloom's city services (000072). Extends city_service_listings / city_service_bookings and adds services,
     * slots, reviews, reports and business applications, plus the directory seed rows (extra categories,
     * catalog groups dir_amenities / dir_age_ranges) so `make schema-diff` row counts match. Used only by the Go
     * internal/cityservices package. No health field anywhere; no payment fields (MVP).
     */
    public function up(): void
    {
        // Guarded like the goose twin (IF NOT EXISTS), so a database that already carries the columns migrates cleanly.
        if (! Schema::hasColumn('city_service_listings', 'status')) {
            Schema::table('city_service_listings', function (Blueprint $table) {
                $table->string('status', 16)->default('published');
                $table->timestamp('verified_at')->nullable();
                $table->foreignId('owner_user_id')->nullable()->constrained('users')->nullOnDelete();
                $table->decimal('lat', 9, 6)->nullable();
                $table->decimal('lng', 9, 6)->nullable();
                $table->text('about')->nullable();
                $table->json('age_ranges')->nullable();
                $table->json('amenities')->nullable();
                $table->json('weekly_hours')->nullable();
                $table->json('photo_file_ids')->nullable();
                $table->string('website_url', 1000)->nullable();
                $table->unsignedSmallInteger('cancel_free_hours')->nullable();
                $table->string('cancel_policy', 300)->nullable();
                $table->text('rules')->nullable();
                $table->decimal('rating_avg', 3, 2)->nullable();
                $table->unsignedInteger('rating_count')->default(0);

                $table->index(['status', 'city_id', 'category_id']);
                $table->index(['lat', 'lng']);
            });
        }

        if (! Schema::hasTable('city_service_services')) {
            Schema::create('city_service_services', function (Blueprint $table) {
                $table->id();
                $table->foreignId('listing_id')->constrained('city_service_listings')->cascadeOnDelete();
                $table->string('kind', 12)->default('session');
                $table->string('name', 160);
                $table->string('description', 300)->nullable();
                $table->string('format', 12)->nullable();
                $table->unsignedSmallInteger('duration_minutes')->nullable();
                $table->unsignedSmallInteger('age_min_months')->nullable();
                $table->unsignedSmallInteger('age_max_months')->nullable();
                $table->unsignedSmallInteger('capacity')->nullable();
                $table->unsignedInteger('price_toman')->nullable();
                $table->unsignedSmallInteger('package_sessions')->nullable();
                $table->unsignedSmallInteger('package_validity_days')->nullable();
                $table->integer('sort_order')->default(0);
                $table->boolean('is_active')->default(true);
                $table->timestamps();

                $table->index(['listing_id', 'is_active', 'sort_order']);
            });
        }

        if (! Schema::hasTable('city_service_slots')) {
            Schema::create('city_service_slots', function (Blueprint $table) {
                $table->id();
                $table->foreignId('listing_id')->constrained('city_service_listings')->cascadeOnDelete();
                $table->foreignId('service_id')->nullable()->constrained('city_service_services')->cascadeOnDelete();
                $table->dateTime('starts_at');
                $table->dateTime('ends_at')->nullable();
                $table->unsignedSmallInteger('capacity')->default(1);
                $table->unsignedSmallInteger('booked_count')->default(0);
                $table->string('status', 12)->default('open');
                $table->timestamps();

                $table->index(['listing_id', 'starts_at']);
                $table->index(['service_id', 'starts_at']);
            });
        }

        if (! Schema::hasColumn('city_service_bookings', 'code')) {
            Schema::table('city_service_bookings', function (Blueprint $table) {
                $table->string('code', 16)->nullable()->unique();
                $table->foreignId('service_id')->nullable()->constrained('city_service_services')->nullOnDelete();
                $table->foreignId('slot_id')->nullable()->constrained('city_service_slots')->nullOnDelete();
                $table->foreignId('child_id')->nullable()->constrained('children')->nullOnDelete();
                $table->unsignedInteger('price_toman')->nullable();
                $table->timestamp('confirmed_at')->nullable();
                $table->timestamp('cancelled_at')->nullable();
                $table->string('cancelled_by', 8)->nullable();
                $table->timestamp('completed_at')->nullable();
            });
        }

        if (! Schema::hasTable('city_service_reviews')) {
            Schema::create('city_service_reviews', function (Blueprint $table) {
                $table->id();
                $table->foreignId('listing_id')->constrained('city_service_listings')->cascadeOnDelete();
                $table->foreignId('booking_id')->unique()->constrained('city_service_bookings')->cascadeOnDelete();
                $table->foreignId('user_id')->constrained()->cascadeOnDelete();
                $table->unsignedTinyInteger('rating');
                $table->unsignedTinyInteger('score_cleanliness')->nullable();
                $table->unsignedTinyInteger('score_instructor')->nullable();
                $table->unsignedTinyInteger('score_amenities')->nullable();
                $table->string('body', 1000)->nullable();
                $table->string('status', 12)->default('published');
                $table->timestamps();

                $table->index(['listing_id', 'status', 'created_at']);
            });
        }

        if (! Schema::hasTable('city_service_reports')) {
            Schema::create('city_service_reports', function (Blueprint $table) {
                $table->id();
                $table->foreignId('listing_id')->constrained('city_service_listings')->cascadeOnDelete();
                $table->foreignId('user_id')->nullable()->constrained()->nullOnDelete();
                $table->string('reason', 32);
                $table->string('note', 500)->nullable();
                $table->string('status', 12)->default('open');
                $table->timestamp('resolved_at')->nullable();
                $table->timestamps();

                $table->index(['status', 'created_at']);
            });
        }

        if (! Schema::hasTable('city_service_applications')) {
            Schema::create('city_service_applications', function (Blueprint $table) {
                $table->id();
                $table->foreignId('user_id')->constrained()->cascadeOnDelete();
                $table->string('tracking_code', 16)->nullable()->unique();
                $table->string('status', 16)->default('draft');
                $table->unsignedTinyInteger('step')->default(1);
                $table->string('business_name', 160)->nullable();
                $table->json('category_slugs')->nullable();
                $table->foreignId('city_id')->nullable()->constrained('city_service_cities')->nullOnDelete();
                $table->string('district', 120)->nullable();
                $table->string('contact_name', 120)->nullable();
                $table->string('contact_phone', 32)->nullable();
                $table->boolean('owner_confirmed')->default(false);
                $table->decimal('lat', 9, 6)->nullable();
                $table->decimal('lng', 9, 6)->nullable();
                $table->string('address', 300)->nullable();
                $table->text('about')->nullable();
                $table->json('age_ranges')->nullable();
                $table->json('amenities')->nullable();
                $table->json('weekly_hours')->nullable();
                $table->json('photo_file_ids')->nullable();
                $table->json('services')->nullable();
                $table->string('booking_mode', 12)->nullable();
                $table->json('documents')->nullable();
                $table->timestamp('terms_accepted_at')->nullable();
                $table->timestamp('submitted_at')->nullable();
                $table->timestamp('reviewed_at')->nullable();
                $table->string('review_note', 500)->nullable();
                $table->foreignId('listing_id')->nullable()->constrained('city_service_listings')->nullOnDelete();
                $table->timestamps();

                $table->index(['user_id', 'status']);
                $table->index(['status', 'submitted_at']);
            });
        }

        $now = now();
        DB::table('city_service_categories')->insertOrIgnore([
            ['slug' => 'music_art', 'name' => '{"fa":"موسیقی و هنر","en":"Music & art"}', 'icon' => 'mic',
                'sort_order' => 4, 'is_active' => 1, 'created_at' => $now, 'updated_at' => $now],
            ['slug' => 'postpartum_fitness', 'name' => '{"fa":"ورزش پس از زایمان","en":"Postpartum fitness"}', 'icon' => 'walk',
                'sort_order' => 5, 'is_active' => 1, 'created_at' => $now, 'updated_at' => $now],
            ['slug' => 'baby_massage', 'name' => '{"fa":"ماساژ نوزاد","en":"Baby massage"}', 'icon' => 'hand',
                'sort_order' => 6, 'is_active' => 1, 'created_at' => $now, 'updated_at' => $now],
            ['slug' => 'parenting_workshop', 'name' => '{"fa":"کارگاه فرزندپروری","en":"Parenting workshops"}', 'icon' => 'users',
                'sort_order' => 7, 'is_active' => 1, 'created_at' => $now, 'updated_at' => $now],
            ['slug' => 'other', 'name' => '{"fa":"دیگر","en":"Other"}', 'icon' => 'grid',
                'sort_order' => 8, 'is_active' => 1, 'created_at' => $now, 'updated_at' => $now],
        ]);

        $json = fn (?array $v) => $v === null ? null : json_encode($v, JSON_UNESCAPED_UNICODE | JSON_UNESCAPED_SLASHES);
        $row = fn (string $group, string $code, int $sort, array $title, array $meta) => [
            'group' => $group,
            'code' => $code,
            'sort_order' => $sort,
            'is_active' => 1,
            'audiences' => null,
            'title' => $json($title),
            'body' => null,
            'meta' => $json($meta),
            'needs_review' => 0,
            'created_at' => $now,
            'updated_at' => $now,
        ];

        DB::table('catalog_items')->insertOrIgnore([
            $row('dir_amenities', 'nursing_room', 1, ['fa' => 'اتاق شیردهی', 'en' => 'Nursing room'], ['icon' => 'heart']),
            $row('dir_amenities', 'changing_table', 2, ['fa' => 'میز تعویض', 'en' => 'Changing table'], ['icon' => 'box']),
            $row('dir_amenities', 'female_instructor', 3, ['fa' => 'مربی خانم', 'en' => 'Female instructor'], ['icon' => 'female']),
            $row('dir_amenities', 'stroller_parking', 4, ['fa' => 'جای کالسکه', 'en' => 'Stroller parking'], ['icon' => 'home']),
            $row('dir_amenities', 'parking', 5, ['fa' => 'پارکینگ', 'en' => 'Parking'], ['icon' => 'mapPin']),
            $row('dir_amenities', 'elevator', 6, ['fa' => 'آسانسور', 'en' => 'Elevator'], ['icon' => 'arrowR']),
            $row('dir_amenities', 'wheelchair_access', 7, ['fa' => 'دسترسی ویلچر', 'en' => 'Wheelchair access'], ['icon' => 'user']),
            $row('dir_age_ranges', 'pregnancy', 1, ['fa' => 'بارداری', 'en' => 'Pregnancy'], ['pregnancy' => true]),
            $row('dir_age_ranges', 'm0_6', 2, ['fa' => '۰ تا ۶ ماه', 'en' => '0–6 months'], ['min_months' => 0, 'max_months' => 6]),
            $row('dir_age_ranges', 'm6_12', 3, ['fa' => '۶ تا ۱۲ ماه', 'en' => '6–12 months'], ['min_months' => 6, 'max_months' => 12]),
            $row('dir_age_ranges', 'y1_2', 4, ['fa' => '۱ تا ۲ سال', 'en' => '1–2 years'], ['min_months' => 12, 'max_months' => 24]),
            $row('dir_age_ranges', 'y2_4', 5, ['fa' => '۲ تا ۴ سال', 'en' => '2–4 years'], ['min_months' => 24, 'max_months' => 48]),
            $row('dir_age_ranges', 'y4_6', 6, ['fa' => '۴ تا ۶ سال', 'en' => '4–6 years'], ['min_months' => 48, 'max_months' => 72]),
        ]);
    }

    public function down(): void
    {
        DB::table('catalog_items')->where('group', 'dir_amenities')
            ->whereIn('code', ['nursing_room', 'changing_table', 'female_instructor', 'stroller_parking', 'parking', 'elevator', 'wheelchair_access'])->delete();
        DB::table('catalog_items')->where('group', 'dir_age_ranges')
            ->whereIn('code', ['pregnancy', 'm0_6', 'm6_12', 'y1_2', 'y2_4', 'y4_6'])->delete();
        DB::table('city_service_categories')
            ->whereIn('slug', ['music_art', 'postpartum_fitness', 'baby_massage', 'parenting_workshop', 'other'])
            ->whereNotIn('id', DB::table('city_service_listings')->select('category_id'))->delete();

        Schema::dropIfExists('city_service_applications');
        Schema::dropIfExists('city_service_reports');
        Schema::dropIfExists('city_service_reviews');
        Schema::table('city_service_bookings', function (Blueprint $table) {
            $table->dropForeign(['child_id']);
            $table->dropForeign(['slot_id']);
            $table->dropForeign(['service_id']);
            $table->dropUnique(['code']);
            $table->dropColumn(['code', 'service_id', 'slot_id', 'child_id', 'price_toman', 'confirmed_at', 'cancelled_at', 'cancelled_by', 'completed_at']);
        });
        Schema::dropIfExists('city_service_slots');
        Schema::dropIfExists('city_service_services');
        Schema::table('city_service_listings', function (Blueprint $table) {
            $table->dropForeign(['owner_user_id']);
            $table->dropIndex(['status', 'city_id', 'category_id']);
            $table->dropIndex(['lat', 'lng']);
            $table->dropColumn(['status', 'verified_at', 'owner_user_id', 'lat', 'lng', 'about', 'age_ranges', 'amenities',
                'weekly_hours', 'photo_file_ids', 'website_url', 'cancel_free_hours', 'cancel_policy', 'rules', 'rating_avg', 'rating_count']);
        });
    }
};
