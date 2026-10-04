<?php

declare(strict_types=1);

use Illuminate\Database\Migrations\Migration;
use Illuminate\Database\Schema\Blueprint;
use Illuminate\Support\Facades\Schema;

/*
 * Visitor reviews of places (L5-01), moderated: pending|approved|rejected. `aspects` is optional JSON of 1–5 scores
 * per aspect (cleanliness, staff, facilities, access). `is_demo` marks seeded design samples: they may be displayed
 * (labelled) but never count towards the rating aggregate. `ip_hash` is a salted hash for spam checks, never the IP.
 */
return new class extends Migration
{
    public function up(): void
    {
        Schema::create('directory_reviews', function (Blueprint $table): void {
            $table->id();
            $table->foreignId('place_id')->constrained('directory_places')->cascadeOnDelete();
            $table->string('author_name', 80);
            $table->unsignedTinyInteger('rating');
            $table->json('aspects')->nullable();
            $table->text('body');
            $table->string('status', 16)->default('pending');
            $table->boolean('is_demo')->default(false);
            $table->string('ip_hash', 64)->nullable();
            $table->timestamp('approved_at')->nullable();
            $table->timestamps();

            $table->index(['place_id', 'status', 'created_at']);
            $table->index(['status', 'created_at']);
        });
    }

    public function down(): void
    {
        Schema::dropIfExists('directory_reviews');
    }
};
