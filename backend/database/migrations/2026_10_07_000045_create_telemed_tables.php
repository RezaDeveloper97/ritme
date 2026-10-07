<?php

use Illuminate\Database\Migrations\Migration;
use Illuminate\Database\Schema\Blueprint;
use Illuminate\Support\Facades\DB;
use Illuminate\Support\Facades\Schema;

return new class extends Migration
{
    /**
     * Run the migrations.
     *
     * Twin of backend-go/db/migrations/00045_telemed.sql
     * (docs/go-migration/migrations.md): the doctors directory «پزشکان و
     * ماماها» (bloom B-N7-02) — admin-managed doctors / midwives, accepted
     * insurers, visit types (price in rials), weekly availability, time off
     * and reviews, used only by the Go internal/telemed package, plus the
     * telemed_specialties catalog rows (fa + en) so `make schema-diff` row
     * counts match. No model or routes here.
     */
    public function up(): void
    {
        Schema::create('telemed_doctors', function (Blueprint $table) {
            $table->id();
            $table->string('kind', 12);                          // doctor|midwife
            $table->json('name');
            $table->json('headline')->nullable();
            $table->json('bio')->nullable();
            $table->string('specialty', 64);                     // catalog telemed_specialties code
            $table->string('city', 64)->nullable();              // catalog telemed_cities code
            $table->string('licence_no', 32);
            $table->unsignedTinyInteger('experience_years')->nullable();
            $table->string('photo_path')->nullable();
            $table->unsignedSmallInteger('response_minutes')->nullable();
            $table->unsignedInteger('visits_count')->default(0);
            $table->unsignedInteger('rating_sum')->default(0);
            $table->unsignedInteger('rating_count')->default(0);
            $table->unsignedInteger('positive_count')->default(0);
            $table->boolean('is_active')->default(true);
            $table->integer('sort_order')->default(0);
            $table->foreignId('admin_id')->nullable()->unique()->constrained('admins')->nullOnDelete();
            $table->timestamps();

            $table->index(['is_active', 'sort_order']);
            $table->index('specialty');
            $table->index('city');
        });

        Schema::create('telemed_doctor_insurers', function (Blueprint $table) {
            $table->foreignId('doctor_id')->constrained('telemed_doctors')->cascadeOnDelete();
            $table->string('insurer', 64);                       // catalog telemed_insurers code

            $table->primary(['doctor_id', 'insurer']);
            $table->index('insurer');
        });

        Schema::create('telemed_visit_types', function (Blueprint $table) {
            $table->id();
            $table->foreignId('doctor_id')->constrained('telemed_doctors')->cascadeOnDelete();
            $table->string('mode', 12);                          // video|phone|in_person
            $table->unsignedSmallInteger('duration_minutes');
            $table->unsignedBigInteger('price_rials');
            $table->json('note')->nullable();
            $table->json('address')->nullable();
            $table->boolean('is_active')->default(true);
            $table->timestamps();

            $table->unique(['doctor_id', 'mode']);
        });

        Schema::create('telemed_availability_rules', function (Blueprint $table) {
            $table->id();
            $table->foreignId('doctor_id')->constrained('telemed_doctors')->cascadeOnDelete();
            $table->unsignedTinyInteger('weekday');              // 0 = Sunday … 6 = Saturday
            $table->unsignedSmallInteger('start_minute');
            $table->unsignedSmallInteger('end_minute');
            $table->unsignedSmallInteger('slot_minutes');
            $table->json('modes')->nullable();
            $table->timestamps();

            $table->index(['doctor_id', 'weekday']);
        });

        Schema::create('telemed_time_off', function (Blueprint $table) {
            $table->id();
            $table->foreignId('doctor_id')->constrained('telemed_doctors')->cascadeOnDelete();
            $table->dateTime('starts_at');
            $table->dateTime('ends_at');
            $table->string('note', 190)->nullable();
            $table->timestamps();

            $table->index(['doctor_id', 'ends_at']);
        });

        Schema::create('telemed_reviews', function (Blueprint $table) {
            $table->id();
            $table->foreignId('doctor_id')->constrained('telemed_doctors')->cascadeOnDelete();
            $table->foreignId('user_id')->constrained()->cascadeOnDelete();
            $table->unsignedBigInteger('booking_id')->nullable()->unique();
            $table->unsignedTinyInteger('rating');
            $table->string('body', 1000)->nullable();
            $table->boolean('is_visible')->default(true);
            $table->timestamps();

            $table->index(['doctor_id', 'is_visible', 'created_at']);
            $table->index('user_id');
        });

        $now = now();
        $row = fn (string $code, int $sort, array $title) => [
            'group' => 'telemed_specialties',
            'code' => $code,
            'sort_order' => $sort,
            'is_active' => 1,
            'audiences' => null,
            'title' => json_encode($title, JSON_UNESCAPED_UNICODE | JSON_UNESCAPED_SLASHES),
            'body' => null,
            'meta' => null,
            'needs_review' => 1,
            'created_at' => $now,
            'updated_at' => $now,
        ];
        DB::table('catalog_items')->insertOrIgnore([
            $row('gynecology', 1, ['fa' => 'زنان و زایمان', 'en' => 'Obstetrics & gynaecology']),
            $row('midwifery', 2, ['fa' => 'ماما', 'en' => 'Midwife']),
            $row('dermatology', 3, ['fa' => 'پوست', 'en' => 'Dermatology']),
            $row('nutrition', 4, ['fa' => 'تغذیه', 'en' => 'Nutrition']),
            $row('psychology', 5, ['fa' => 'روان‌شناسی', 'en' => 'Psychology']),
            $row('general', 6, ['fa' => 'پزشک عمومی', 'en' => 'General practice']),
        ]);
    }

    /**
     * Reverse the migrations.
     */
    public function down(): void
    {
        DB::table('catalog_items')->where('group', 'telemed_specialties')
            ->whereIn('code', ['gynecology', 'midwifery', 'dermatology', 'nutrition', 'psychology', 'general'])
            ->whereColumn('updated_at', 'created_at')
            ->delete();
        Schema::dropIfExists('telemed_reviews');
        Schema::dropIfExists('telemed_time_off');
        Schema::dropIfExists('telemed_availability_rules');
        Schema::dropIfExists('telemed_visit_types');
        Schema::dropIfExists('telemed_doctor_insurers');
        Schema::dropIfExists('telemed_doctors');
    }
};
