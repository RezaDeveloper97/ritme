<?php

declare(strict_types=1);

use Illuminate\Database\Migrations\Migration;
use Illuminate\Database\Schema\Blueprint;
use Illuminate\Support\Facades\Schema;

/*
 * Amenities of places (L5-01): «اتاق شیردهی», «مربی خانم»… `icon` is a sprite icon name; `is_filter` puts the
 * amenity into the listing's filter chips.
 */
return new class extends Migration
{
    public function up(): void
    {
        Schema::create('directory_amenities', function (Blueprint $table): void {
            $table->id();
            $table->string('name', 120);
            $table->string('slug', 191)->unique();
            $table->string('icon', 64)->nullable();
            $table->boolean('is_filter')->default(false);
            $table->unsignedSmallInteger('sort_order')->default(0);
            $table->timestamps();
        });
    }

    public function down(): void
    {
        Schema::dropIfExists('directory_amenities');
    }
};
