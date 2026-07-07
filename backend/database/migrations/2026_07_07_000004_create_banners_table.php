<?php

use Illuminate\Database\Migrations\Migration;
use Illuminate\Database\Schema\Blueprint;
use Illuminate\Support\Facades\Schema;

return new class extends Migration
{
    /**
     * Run the migrations.
     *
     * Promotional banners shown as a swipeable slideshow in fixed home-page
     * slots (App\Enums\BannerPosition). Each banner may carry an internal or
     * external link and is calendar-scheduled via starts_at/ends_at, so it only
     * appears while active and within its window.
     */
    public function up(): void
    {
        Schema::create('banners', function (Blueprint $table) {
            $table->id();
            $table->json('title')->nullable();          // {fa, en} — optional caption/alt
            $table->string('image_path');               // stored on the public disk
            $table->string('position')->default('home_top'); // App\Enums\BannerPosition
            $table->string('link_url', 1000)->nullable();
            $table->string('link_type')->nullable();     // App\Enums\BannerLinkType (null = no link)
            $table->timestamp('starts_at')->nullable();  // null = no lower bound
            $table->timestamp('ends_at')->nullable();    // null = no upper bound
            $table->boolean('is_active')->default(true);
            $table->integer('sort_order')->default(0);
            $table->timestamps();

            $table->index(['position', 'is_active', 'starts_at', 'ends_at']);
        });
    }

    public function down(): void
    {
        Schema::dropIfExists('banners');
    }
};
