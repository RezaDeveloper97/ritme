<?php

use Illuminate\Database\Migrations\Migration;
use Illuminate\Database\Schema\Blueprint;
use Illuminate\Support\Facades\DB;
use Illuminate\Support\Facades\Schema;

return new class extends Migration
{
    /**
     * Twin of backend-go/db/migrations/00072_city_services.sql: admin-managed city
     * services (classes, pools, playhouses) and simple booking requests (bloom
     * B-N10-05, D-92). Used only by the Go internal/cityservices package. A
     * booking never carries a health field.
     */
    public function up(): void
    {
        Schema::create('city_service_cities', function (Blueprint $table) {
            $table->id();
            $table->string('slug', 64)->unique();
            $table->json('name');
            $table->integer('sort_order')->default(0);
            $table->boolean('is_active')->default(true);
            $table->timestamps();

            $table->index(['is_active', 'sort_order']);
        });

        Schema::create('city_service_categories', function (Blueprint $table) {
            $table->id();
            $table->string('slug', 64)->unique();
            $table->json('name');
            $table->string('icon', 32)->nullable();
            $table->integer('sort_order')->default(0);
            $table->boolean('is_active')->default(true);
            $table->timestamps();

            $table->index(['is_active', 'sort_order']);
        });

        Schema::create('city_service_listings', function (Blueprint $table) {
            $table->id();
            $table->foreignId('category_id')->constrained('city_service_categories');
            $table->foreignId('city_id')->constrained('city_service_cities');
            $table->string('name', 160);
            $table->string('district', 120)->nullable();
            $table->string('address', 300)->nullable();
            $table->string('phone', 32)->nullable();
            $table->text('description')->nullable();
            $table->string('price_text', 120)->nullable();
            $table->string('hours_text', 200)->nullable();
            $table->json('photos')->nullable();
            $table->string('booking_mode', 12)->default('phone');
            $table->string('booking_url', 1000)->nullable();
            $table->integer('sort_order')->default(0);
            $table->boolean('is_active')->default(true);
            $table->timestamps();

            $table->index(['city_id', 'category_id', 'is_active']);
        });

        Schema::create('city_service_bookings', function (Blueprint $table) {
            $table->id();
            $table->foreignId('user_id')->constrained()->cascadeOnDelete();
            $table->foreignId('listing_id')->constrained('city_service_listings');
            $table->string('contact_name', 120);
            $table->string('contact_phone', 32);
            $table->date('preferred_date');
            $table->string('child_age_range', 16)->nullable();
            $table->string('note', 500)->nullable();
            $table->string('status', 16)->default('requested');
            $table->string('status_note', 300)->nullable();
            $table->timestamps();

            $table->index(['user_id', 'created_at']);
            $table->index(['status', 'created_at']);
        });

        $now = now();
        DB::table('city_service_categories')->insertOrIgnore([
            ['slug' => 'class', 'name' => '{"fa":"کلاس","en":"Classes"}', 'icon' => 'book',
                'sort_order' => 1, 'is_active' => 1, 'created_at' => $now, 'updated_at' => $now],
            ['slug' => 'pool', 'name' => '{"fa":"استخر","en":"Pools"}', 'icon' => 'drop',
                'sort_order' => 2, 'is_active' => 1, 'created_at' => $now, 'updated_at' => $now],
            ['slug' => 'playhouse', 'name' => '{"fa":"خانه بازی","en":"Playhouses"}', 'icon' => 'home',
                'sort_order' => 3, 'is_active' => 1, 'created_at' => $now, 'updated_at' => $now],
        ]);
    }

    public function down(): void
    {
        Schema::dropIfExists('city_service_bookings');
        Schema::dropIfExists('city_service_listings');
        Schema::dropIfExists('city_service_categories');
        Schema::dropIfExists('city_service_cities');
    }
};
