<?php

use Illuminate\Database\Migrations\Migration;
use Illuminate\Database\Schema\Blueprint;
use Illuminate\Support\Facades\Schema;

return new class extends Migration
{
    /**
     * Run the migrations.
     *
     * Twin of backend-go/db/migrations/00128_companion_share_period_days.sql: the owner's consent (off by default) to
     * show her companion the days her period is likely to fall on. Used only by the Go backend: schema only.
     */
    public function up(): void
    {
        if (! Schema::hasColumn('companions', 'share_period_days')) {
            Schema::table('companions', function (Blueprint $table) {
                $table->boolean('share_period_days')->default(false);
            });
        }
    }

    /**
     * Reverse the migrations.
     */
    public function down(): void
    {
        if (Schema::hasColumn('companions', 'share_period_days')) {
            Schema::table('companions', function (Blueprint $table) {
                $table->dropColumn('share_period_days');
            });
        }
    }
};
