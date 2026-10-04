<?php

declare(strict_types=1);

use Illuminate\Database\Migrations\Migration;
use Illuminate\Database\Schema\Blueprint;
use Illuminate\Support\Facades\Schema;

/*
 * Cities of the mother & child directory (L5-01). `slug` is the first URL segment of the landing pages
 * (`/directory/{city}`, `/directory/{city}/{category}`); Persian slugs are allowed.
 */
return new class extends Migration
{
    public function up(): void
    {
        Schema::create('directory_cities', function (Blueprint $table): void {
            $table->id();
            $table->string('name', 120);
            $table->string('slug', 191)->unique();
            $table->string('province', 120)->nullable();
            $table->unsignedSmallInteger('sort_order')->default(0);
            $table->boolean('is_active')->default(true);
            $table->timestamps();

            $table->index(['is_active', 'sort_order']);
        });
    }

    public function down(): void
    {
        Schema::dropIfExists('directory_cities');
    }
};
