<?php

use Illuminate\Database\Migrations\Migration;
use Illuminate\Database\Schema\Blueprint;
use Illuminate\Support\Facades\Schema;

return new class extends Migration
{
    /**
     * Adds the ورزش (exercise) section of the daily log: what the user did,
     * for how long, and how hard it felt.
     */
    public function up(): void
    {
        Schema::table('daily_health_logs', function (Blueprint $table) {
            $table->string('exercise_type')->nullable()->after('sleep_quality'); // walking, running, …
            $table->unsignedSmallInteger('exercise_duration')->nullable()->after('exercise_type'); // minutes
            $table->string('exercise_intensity')->nullable()->after('exercise_duration'); // low, medium, high
        });
    }

    public function down(): void
    {
        Schema::table('daily_health_logs', function (Blueprint $table) {
            $table->dropColumn([
                'exercise_type',
                'exercise_duration',
                'exercise_intensity',
            ]);
        });
    }
};
