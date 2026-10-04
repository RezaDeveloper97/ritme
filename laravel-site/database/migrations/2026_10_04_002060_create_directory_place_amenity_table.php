<?php

declare(strict_types=1);

use Illuminate\Database\Migrations\Migration;
use Illuminate\Database\Schema\Blueprint;
use Illuminate\Support\Facades\Schema;

/*
 * Place ↔ amenity pivot (L5-01). Written through App\Domain\Directory\Actions\SyncPlaceAmenities (bumps caches).
 */
return new class extends Migration
{
    public function up(): void
    {
        Schema::create('directory_place_amenity', function (Blueprint $table): void {
            $table->foreignId('place_id')->constrained('directory_places')->cascadeOnDelete();
            $table->foreignId('amenity_id')->constrained('directory_amenities')->cascadeOnDelete();

            $table->primary(['place_id', 'amenity_id']);
            $table->index('amenity_id');
        });
    }

    public function down(): void
    {
        Schema::dropIfExists('directory_place_amenity');
    }
};
