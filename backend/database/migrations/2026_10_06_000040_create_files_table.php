<?php

use Illuminate\Database\Migrations\Migration;
use Illuminate\Database\Schema\Blueprint;
use Illuminate\Support\Facades\Schema;

return new class extends Migration
{
    /**
     * Run the migrations.
     *
     * Twin of backend-go/db/migrations/00040_files.sql
     * (docs/go-migration/migrations.md): generic encrypted file storage
     * (canvas-build CB-CORE-05) — one row per stored file (record / claim
     * documents, place photos / licences, product images), used only by the
     * Go internal/files package. The bytes are encrypted on disk by Go. No
     * model or routes here.
     */
    public function up(): void
    {
        Schema::create('files', function (Blueprint $table) {
            $table->id();
            $table->foreignId('user_id')->nullable()->constrained('users')->cascadeOnDelete(); // NULL = Ritme team (public purposes)
            $table->string('purpose', 32);                        // record_document|claim_document|place_photo|place_licence|product_image
            $table->string('visibility', 8);                      // private|public
            $table->string('mime', 32);
            $table->unsignedInteger('size_bytes');
            $table->char('sha256', 64);
            $table->string('path', 255);                          // relative to STORAGE_PATH, encrypted file
            $table->timestamps();

            $table->index(['user_id', 'purpose']);
        });
    }

    /**
     * Reverse the migrations.
     */
    public function down(): void
    {
        Schema::dropIfExists('files');
    }
};
