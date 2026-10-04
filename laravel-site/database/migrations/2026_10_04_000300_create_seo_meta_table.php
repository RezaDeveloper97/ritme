<?php

declare(strict_types=1);

use Illuminate\Database\Migrations\Migration;
use Illuminate\Database\Schema\Blueprint;
use Illuminate\Support\Facades\Schema;

/*
 * Per-page SEO overrides. A row belongs either to a model (morph `seoable`, via the HasSeo trait) or to a static
 * page (keyed by `route_name`). Every column is optional: null means "inherit" (settings defaults → page → overrides).
 */
return new class extends Migration
{
    public function up(): void
    {
        Schema::create('seo_meta', function (Blueprint $table): void {
            $table->id();
            $table->string('seoable_type', 191)->nullable();
            $table->unsignedBigInteger('seoable_id')->nullable();
            $table->string('route_name', 191)->nullable()->unique();

            $table->string('title', 255)->nullable();
            $table->string('description', 500)->nullable();
            $table->string('canonical_url', 2048)->nullable();
            $table->string('robots', 191)->nullable();

            $table->string('og_title', 255)->nullable();
            $table->string('og_description', 500)->nullable();
            $table->unsignedBigInteger('og_media_id')->nullable(); // media table arrives in L2-01
            $table->string('og_type', 32)->nullable();
            $table->string('twitter_card', 32)->nullable();

            $table->string('focus_keyword', 191)->nullable();
            $table->json('schema_overrides')->nullable();

            $table->boolean('sitemap_include')->default(true);
            $table->decimal('sitemap_priority', 2, 1)->nullable();
            $table->string('sitemap_changefreq', 16)->nullable();

            $table->timestamps();

            $table->unique(['seoable_type', 'seoable_id']);
        });
    }

    public function down(): void
    {
        Schema::dropIfExists('seo_meta');
    }
};
