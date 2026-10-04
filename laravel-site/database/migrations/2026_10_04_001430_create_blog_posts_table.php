<?php

declare(strict_types=1);

use Illuminate\Database\Migrations\Migration;
use Illuminate\Database\Schema\Blueprint;
use Illuminate\Support\Facades\Schema;

/*
 * Magazine posts (L4-01). `body` and `sources` hold sanitised HTML (sanitised on save); `reading_time` is minutes
 * from a Persian word count; `updated_content_at` is the reader-visible "updated" date (dateModified), unlike
 * `updated_at`. `views` is flushed in batches from a cache counter. Status: draft|scheduled|published|archived.
 */
return new class extends Migration
{
    public function up(): void
    {
        Schema::create('blog_posts', function (Blueprint $table): void {
            $table->id();
            $table->string('title', 255);
            $table->string('slug', 191)->unique();
            $table->text('excerpt')->nullable();
            $table->longText('body');
            $table->text('sources')->nullable();

            $table->foreignId('cover_media_id')->nullable()->constrained('media')->nullOnDelete();
            $table->foreignId('cover_mobile_media_id')->nullable()->constrained('media')->nullOnDelete();
            $table->foreignId('category_id')->nullable()->constrained('blog_categories')->nullOnDelete();
            $table->foreignId('author_id')->nullable()->constrained('blog_authors')->nullOnDelete();
            $table->foreignId('reviewer_id')->nullable()->constrained('blog_authors')->nullOnDelete();
            $table->timestamp('reviewed_at')->nullable();

            $table->string('life_stage', 16)->nullable();
            $table->string('status', 16)->default('draft');
            $table->timestamp('published_at')->nullable();
            $table->timestamp('updated_content_at')->nullable();
            $table->unsignedSmallInteger('reading_time')->default(1);
            $table->unsignedInteger('word_count')->default(0);
            $table->boolean('is_featured')->default(false);
            $table->unsignedBigInteger('views')->default(0);
            $table->timestamps();

            $table->index(['status', 'published_at']);
            $table->index(['category_id', 'status', 'published_at']);
            $table->index(['life_stage', 'status', 'published_at']);
            $table->index(['is_featured', 'status', 'published_at']);
            $table->index(['author_id', 'status', 'published_at']);
        });
    }

    public function down(): void
    {
        Schema::dropIfExists('blog_posts');
    }
};
