<?php

declare(strict_types=1);

use Illuminate\Database\Migrations\Migration;
use Illuminate\Database\Schema\Blueprint;
use Illuminate\Support\Facades\Schema;

/*
 * Admin-edited copy of the directory landing pages (L5-01; used by L5-02, edited in L5-06): one row per city
 * (`category_id` null → `/directory/{city}`) or city × category (`/directory/{city}/{category}`). Empty fields fall
 * back to App\Domain\Directory\Support\LandingCopy templates. Uniqueness of the city-only row is enforced in code
 * (NULL is never equal in a unique index).
 */
return new class extends Migration
{
    public function up(): void
    {
        Schema::create('directory_landings', function (Blueprint $table): void {
            $table->id();
            $table->foreignId('city_id')->constrained('directory_cities')->cascadeOnDelete();
            $table->foreignId('category_id')->nullable()->constrained('directory_categories')->cascadeOnDelete();
            $table->string('h1', 191)->nullable();
            $table->string('meta_title', 191)->nullable();
            $table->string('meta_description', 320)->nullable();
            $table->text('intro')->nullable();
            $table->timestamps();

            $table->unique(['city_id', 'category_id']);
        });
    }

    public function down(): void
    {
        Schema::dropIfExists('directory_landings');
    }
};
