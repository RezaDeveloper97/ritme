<?php

declare(strict_types=1);

use Illuminate\Database\Migrations\Migration;
use Illuminate\Database\Schema\Blueprint;
use Illuminate\Support\Facades\Schema;

/*
 * Services and prices of a place (L5-01), e.g. «آشنایی با آب · ۴۵ دقیقه · ۳۲۰ هزار تومان». `price` is whole Toman
 * as announced by the place; `details` is free text («گروهی», «اعتبار ۶۰ روز»). Ages in months. Bookings (L5-04)
 * reference a service by id.
 */
return new class extends Migration
{
    public function up(): void
    {
        Schema::create('directory_place_services', function (Blueprint $table): void {
            $table->id();
            $table->foreignId('place_id')->constrained('directory_places')->cascadeOnDelete();
            $table->string('name', 191);
            $table->unsignedSmallInteger('duration_minutes')->nullable();
            $table->unsignedInteger('price')->nullable();
            $table->string('price_unit', 60)->nullable();
            $table->string('details', 255)->nullable();
            $table->unsignedSmallInteger('age_min_months')->nullable();
            $table->unsignedSmallInteger('age_max_months')->nullable();
            $table->unsignedSmallInteger('sort_order')->default(0);
            $table->timestamps();

            $table->index(['place_id', 'sort_order']);
        });
    }

    public function down(): void
    {
        Schema::dropIfExists('directory_place_services');
    }
};
