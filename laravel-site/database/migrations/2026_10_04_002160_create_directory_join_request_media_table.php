<?php

declare(strict_types=1);

use Illuminate\Database\Migrations\Migration;
use Illuminate\Database\Schema\Blueprint;
use Illuminate\Support\Facades\Schema;

/*
 * Photos uploaded with a join request (L5-05), in upload order (the first one becomes the cover on approval). The files
 * went through the media pipeline (StoreMedia: sniffed type, size/pixel limits, metadata stripped, variants queued).
 * Has its own `id` because FindMediaUsages reads `id` from every registered media column's table.
 */
return new class extends Migration
{
    public function up(): void
    {
        Schema::create('directory_join_request_media', function (Blueprint $table): void {
            $table->id();
            $table->foreignId('join_request_id')->constrained('directory_join_requests')->cascadeOnDelete();
            $table->foreignId('media_id')->constrained('media')->cascadeOnDelete();
            $table->unsignedSmallInteger('sort_order')->default(0);

            $table->unique(['join_request_id', 'media_id']);
            $table->index('media_id');
        });
    }

    public function down(): void
    {
        Schema::dropIfExists('directory_join_request_media');
    }
};
