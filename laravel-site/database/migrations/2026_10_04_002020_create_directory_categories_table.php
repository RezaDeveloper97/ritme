<?php

declare(strict_types=1);

use Illuminate\Database\Migrations\Migration;
use Illuminate\Database\Schema\Blueprint;
use Illuminate\Support\Facades\Schema;

/*
 * Place categories (L5-01): «استخر مادر و کودک», «خانه بازی»… `schema_type` is the schema.org LocalBusiness subtype
 * (App\Domain\Seo\Schema\Enums\LocalBusinessType value) used for the place page JSON-LD; `icon` is a sprite name.
 */
return new class extends Migration
{
    public function up(): void
    {
        Schema::create('directory_categories', function (Blueprint $table): void {
            $table->id();
            $table->string('name', 120);
            $table->string('slug', 191)->unique();
            $table->text('description')->nullable();
            $table->string('schema_type', 64)->default('LocalBusiness');
            $table->string('icon', 64)->nullable();
            $table->unsignedSmallInteger('sort_order')->default(0);
            $table->boolean('is_active')->default(true);
            $table->timestamps();

            $table->index(['is_active', 'sort_order']);
        });
    }

    public function down(): void
    {
        Schema::dropIfExists('directory_categories');
    }
};
