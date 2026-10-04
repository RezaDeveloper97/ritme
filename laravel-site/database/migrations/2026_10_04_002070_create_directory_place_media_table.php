<?php

declare(strict_types=1);

use Illuminate\Database\Migrations\Migration;
use Illuminate\Database\Schema\Blueprint;
use Illuminate\Support\Facades\Schema;

/*
 * Ordered gallery of a place (L5-01). Written through App\Domain\Directory\Actions\SyncPlaceGallery. Has its own `id`
 * because FindMediaUsages reads `id` from every registered media column's table.
 */
return new class extends Migration
{
    public function up(): void
    {
        Schema::create('directory_place_media', function (Blueprint $table): void {
            $table->id();
            $table->foreignId('place_id')->constrained('directory_places')->cascadeOnDelete();
            $table->foreignId('media_id')->constrained('media')->cascadeOnDelete();
            $table->unsignedSmallInteger('sort_order')->default(0);

            $table->unique(['place_id', 'media_id']);
            $table->index('media_id');
        });
    }

    public function down(): void
    {
        Schema::dropIfExists('directory_place_media');
    }
};
