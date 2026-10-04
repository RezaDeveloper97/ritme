<?php

declare(strict_types=1);

use Illuminate\Database\Migrations\Migration;
use Illuminate\Database\Schema\Blueprint;
use Illuminate\Support\Facades\Schema;

/*
 * «ثبت مجموعه» requests from /directory/join (L5-05), reviewed in the admin (L5-06). Minimal data: the place facts the
 * listing needs plus ONE contact person (name, mobile, optional email) — no IP, user agent or documents (the team asks
 * for licences when it calls). `code` is the public tracking code. `age_groups` (AgeGroup values), `amenity_ids` and
 * `opening_hours` (the directory_places JSON shape) are copied into a Place on approval. Status: pending|approved|rejected.
 */
return new class extends Migration
{
    public function up(): void
    {
        Schema::create('directory_join_requests', function (Blueprint $table): void {
            $table->id();
            $table->string('code', 16)->unique();
            $table->string('status', 16)->default('pending');

            $table->string('name', 191);
            $table->foreignId('category_id')->nullable()->constrained('directory_categories')->nullOnDelete();
            $table->foreignId('city_id')->nullable()->constrained('directory_cities')->nullOnDelete();
            $table->foreignId('district_id')->nullable()->constrained('directory_districts')->nullOnDelete();
            $table->string('contact_name', 100);
            $table->string('mobile', 20);
            $table->string('email', 191)->nullable();

            $table->string('address', 500);
            $table->string('phone', 20)->nullable();
            $table->decimal('latitude', 10, 7)->nullable();
            $table->decimal('longitude', 10, 7)->nullable();
            $table->text('about')->nullable();
            $table->json('age_groups')->nullable();
            $table->json('amenity_ids')->nullable();
            $table->json('opening_hours')->nullable();

            $table->text('services')->nullable();
            $table->string('booking_mode', 16)->default('online');
            $table->timestamp('terms_accepted_at')->nullable();
            $table->timestamps();

            $table->index(['status', 'created_at']);
        });
    }

    public function down(): void
    {
        Schema::dropIfExists('directory_join_requests');
    }
};
