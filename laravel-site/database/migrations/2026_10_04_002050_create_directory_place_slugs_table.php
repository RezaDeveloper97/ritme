<?php

declare(strict_types=1);

use Illuminate\Database\Migrations\Migration;
use Illuminate\Database\Schema\Blueprint;
use Illuminate\Support\Facades\Schema;

/*
 * Slug history of places (L5-01): previous slugs of published places, so old URLs 301 to the current one (L5-03).
 */
return new class extends Migration
{
    public function up(): void
    {
        Schema::create('directory_place_slugs', function (Blueprint $table): void {
            $table->id();
            $table->foreignId('place_id')->constrained('directory_places')->cascadeOnDelete();
            $table->string('slug', 191)->unique();
            $table->timestamp('created_at')->nullable();
        });
    }

    public function down(): void
    {
        Schema::dropIfExists('directory_place_slugs');
    }
};
