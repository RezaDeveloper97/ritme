<?php

use Illuminate\Database\Migrations\Migration;
use Illuminate\Database\Schema\Blueprint;
use Illuminate\Support\Facades\Schema;

return new class extends Migration
{
    /**
     * Twin of backend-go/db/migrations/00074_banner_ads.sql: sponsored home slots on the
     * banner system (bloom B-N10-02, D-93) and their per-day aggregate counters
     * (no user id). Used only by the Go internal/bannerads package.
     */
    public function up(): void
    {
        // Guarded like the goose twin (IF NOT EXISTS), so a database that already carries the columns migrates cleanly.
        if (! Schema::hasColumn('banners', 'kind')) {
            Schema::table('banners', function (Blueprint $table) {
                $table->string('kind', 16)->default('banner')->after('sort_order');
                $table->string('advertiser', 120)->nullable()->after('kind');
                $table->json('cta_label')->nullable()->after('advertiser');
                $table->json('modes')->nullable()->after('cta_label');
                $table->index(['kind', 'position', 'is_active']);
            });
        }

        if (! Schema::hasTable('banner_daily_stats')) {
            Schema::create('banner_daily_stats', function (Blueprint $table) {
                $table->foreignId('banner_id')->constrained()->cascadeOnDelete();
                $table->date('day');
                $table->unsignedInteger('impressions')->default(0);
                $table->unsignedInteger('clicks')->default(0);
                $table->primary(['banner_id', 'day']);
            });
        }
    }

    public function down(): void
    {
        Schema::dropIfExists('banner_daily_stats');
        Schema::table('banners', function (Blueprint $table) {
            $table->dropIndex(['kind', 'position', 'is_active']);
            $table->dropColumn(['modes', 'cta_label', 'advertiser', 'kind']);
        });
    }
};
