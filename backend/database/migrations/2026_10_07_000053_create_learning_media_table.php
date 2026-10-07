<?php

use Illuminate\Database\Migrations\Migration;
use Illuminate\Database\Schema\Blueprint;
use Illuminate\Support\Facades\Schema;

return new class extends Migration
{
    /**
     * Run the migrations.
     *
     * Twin of backend-go/db/migrations/00053_learning_media.sql
     * (docs/go-migration/migrations.md): resumable lesson media uploads and
     * the files served by signed playback URLs (bloom B-N8-02), used only by
     * the Go internal/media package. No model or routes here.
     */
    public function up(): void
    {
        Schema::create('learning_media', function (Blueprint $table) {
            $table->id();
            $table->foreignId('instructor_id')->constrained('learning_instructors')->cascadeOnDelete();
            $table->foreignId('lesson_id')->nullable()->constrained('learning_lessons')->nullOnDelete();
            $table->string('kind', 8);                          // video|audio|pdf
            $table->string('mime', 40)->nullable();
            $table->unsignedBigInteger('size_bytes');
            $table->unsignedBigInteger('offset_bytes')->default(0);
            $table->string('path', 120);
            $table->string('status', 12)->default('uploading'); // uploading|processing|ready|failed
            $table->dateTime('upload_expires_at')->nullable();
            $table->dateTime('completed_at')->nullable();
            $table->timestamps();

            $table->index(['instructor_id', 'status']);
            $table->index('lesson_id');
            $table->index(['status', 'upload_expires_at']);
        });
    }

    /**
     * Reverse the migrations.
     */
    public function down(): void
    {
        Schema::dropIfExists('learning_media');
    }
};
