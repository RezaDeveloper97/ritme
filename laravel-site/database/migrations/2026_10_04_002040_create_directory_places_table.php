<?php

declare(strict_types=1);

use Illuminate\Database\Migrations\Migration;
use Illuminate\Database\Schema\Blueprint;
use Illuminate\Support\Facades\Schema;

/*
 * Places of the mother & child directory (L5-01). `opening_hours` is JSON per weekday (`saturday` … `friday`) with a
 * list of {opens, closes} ranges (HH:MM, normalised on save; closes < opens = past midnight). `phones` is a JSON list.
 * Ages are months. `rating_avg`/`rating_count` cache the approved, non-demo reviews and `price_from` the cheapest
 * service (both maintained by observers). `is_demo` marks placeholder businesses from the design seeder.
 * Status: draft|published|suspended.
 */
return new class extends Migration
{
    public function up(): void
    {
        Schema::create('directory_places', function (Blueprint $table): void {
            $table->id();
            $table->string('name', 191);
            $table->string('slug', 191)->unique();
            $table->foreignId('category_id')->constrained('directory_categories')->restrictOnDelete();
            $table->foreignId('city_id')->constrained('directory_cities')->restrictOnDelete();
            $table->foreignId('district_id')->nullable()->constrained('directory_districts')->nullOnDelete();

            $table->string('summary', 300)->nullable();
            $table->text('description')->nullable();
            $table->string('address', 500)->nullable();
            $table->string('postal_code', 20)->nullable();
            $table->decimal('latitude', 10, 7)->nullable();
            $table->decimal('longitude', 10, 7)->nullable();
            $table->json('phones')->nullable();
            $table->string('website', 255)->nullable();

            $table->unsignedSmallInteger('age_min_months')->nullable();
            $table->unsignedSmallInteger('age_max_months')->nullable();
            $table->json('opening_hours')->nullable();
            $table->text('rules')->nullable();
            $table->text('cancellation_policy')->nullable();
            $table->foreignId('cover_media_id')->nullable()->constrained('media')->nullOnDelete();

            $table->string('status', 16)->default('draft');
            $table->boolean('is_verified')->default(false);
            $table->boolean('is_demo')->default(false);
            $table->decimal('rating_avg', 3, 2)->default(0);
            $table->unsignedInteger('rating_count')->default(0);
            $table->unsignedInteger('price_from')->nullable();
            $table->timestamps();

            $table->index(['status', 'city_id', 'category_id']);
            $table->index(['status', 'rating_avg']);
            $table->index(['status', 'price_from']);
        });
    }

    public function down(): void
    {
        Schema::dropIfExists('directory_places');
    }
};
