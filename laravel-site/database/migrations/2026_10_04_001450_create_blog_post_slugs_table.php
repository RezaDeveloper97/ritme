<?php

declare(strict_types=1);

use Illuminate\Database\Migrations\Migration;
use Illuminate\Database\Schema\Blueprint;
use Illuminate\Support\Facades\Schema;

/*
 * Slug history of posts (L4-01): every previous slug, so old URLs 301 to the current one (L4-03). A slug is either
 * a post's current slug or one history row, never both (enforced by PostSlugger).
 */
return new class extends Migration
{
    public function up(): void
    {
        Schema::create('blog_post_slugs', function (Blueprint $table): void {
            $table->id();
            $table->foreignId('post_id')->constrained('blog_posts')->cascadeOnDelete();
            $table->string('slug', 191)->unique();
            $table->timestamp('created_at')->nullable();
        });
    }

    public function down(): void
    {
        Schema::dropIfExists('blog_post_slugs');
    }
};
