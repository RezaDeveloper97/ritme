<?php

declare(strict_types=1);

use Illuminate\Database\Migrations\Migration;
use Illuminate\Database\Schema\Blueprint;
use Illuminate\Support\Facades\Schema;

/*
 * Media library (L2-01). One row per uploaded file; the stored original lives at {disk}:{directory}/{filename},
 * optimised variants are listed in `variants` as {name: {format: {w, h, path, size}}} with paths relative to the disk.
 * `hash` (sha256 of the uploaded bytes) dedupes re-uploads.
 */
return new class extends Migration
{
    public function up(): void
    {
        Schema::create('media', function (Blueprint $table): void {
            $table->id();
            $table->string('disk', 32);
            $table->string('directory', 191);
            $table->string('filename', 191);
            $table->string('original_name', 255)->nullable();
            $table->string('mime', 64);
            $table->unsignedInteger('size');
            $table->unsignedInteger('width')->nullable();
            $table->unsignedInteger('height')->nullable();

            $table->string('alt', 255)->nullable();
            $table->string('title', 255)->nullable();
            $table->text('caption')->nullable();

            $table->decimal('focal_x', 5, 4)->default(0.5);
            $table->decimal('focal_y', 5, 4)->default(0.5);
            $table->string('dominant_color', 7)->nullable();
            $table->text('lqip')->nullable();
            $table->json('variants')->nullable();
            $table->timestamp('optimized_at')->nullable();

            $table->foreignId('uploaded_by')->nullable()->constrained('users')->nullOnDelete();
            $table->char('hash', 64)->unique();
            $table->timestamps();

            $table->index('created_at');
        });
    }

    public function down(): void
    {
        Schema::dropIfExists('media');
    }
};
