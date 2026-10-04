<?php

declare(strict_types=1);

use Illuminate\Database\Migrations\Migration;
use Illuminate\Database\Schema\Blueprint;
use Illuminate\Support\Facades\Schema;

/*
 * Districts (neighbourhoods) of a directory city (L5-01). Slugs are unique within their city.
 */
return new class extends Migration
{
    public function up(): void
    {
        Schema::create('directory_districts', function (Blueprint $table): void {
            $table->id();
            $table->foreignId('city_id')->constrained('directory_cities')->cascadeOnDelete();
            $table->string('name', 120);
            $table->string('slug', 191);
            $table->unsignedSmallInteger('sort_order')->default(0);
            $table->timestamps();

            $table->unique(['city_id', 'slug']);
        });
    }

    public function down(): void
    {
        Schema::dropIfExists('directory_districts');
    }
};
