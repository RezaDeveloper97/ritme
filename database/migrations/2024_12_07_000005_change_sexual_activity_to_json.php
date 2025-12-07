<?php

use Illuminate\Database\Migrations\Migration;
use Illuminate\Database\Schema\Blueprint;
use Illuminate\Support\Facades\Schema;

return new class extends Migration
{
    /**
     * Run the migrations.
     */
    public function up(): void
    {
        Schema::table('daily_health_logs', function (Blueprint $table) {
            // Drop the old column
            $table->dropColumn('sexual_activity_level');
        });

        Schema::table('daily_health_logs', function (Blueprint $table) {
            // Add new column as JSON for multiple selections
            $table->json('sexual_activities')->nullable()->after('sleep_quality');
        });
    }

    /**
     * Reverse the migrations.
     */
    public function down(): void
    {
        Schema::table('daily_health_logs', function (Blueprint $table) {
            $table->dropColumn('sexual_activities');
        });

        Schema::table('daily_health_logs', function (Blueprint $table) {
            $table->string('sexual_activity_level')->nullable()->after('sleep_quality');
        });
    }
};
