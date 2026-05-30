<?php

use Illuminate\Database\Migrations\Migration;
use Illuminate\Database\Schema\Blueprint;
use Illuminate\Support\Facades\Schema;

return new class extends Migration
{
    /**
     * Run the migrations.
     *
     * Adds the vital-sign and energy fields surfaced by the home page
     * "وضعیت امروز" (vitals) and "خلاصه هفته" (energy) sections.
     */
    public function up(): void
    {
        Schema::table('daily_health_logs', function (Blueprint $table) {
            // 6. علائم حیاتی (Vital Signs) — extended
            $table->unsignedSmallInteger('heart_rate')->nullable()->after('basal_body_temperature'); // bpm
            $table->unsignedSmallInteger('systolic_pressure')->nullable()->after('heart_rate'); // mmHg (فشار سیستولیک)
            $table->unsignedSmallInteger('diastolic_pressure')->nullable()->after('systolic_pressure'); // mmHg (فشار دیاستولیک)
            $table->decimal('blood_sugar', 5, 1)->nullable()->after('diastolic_pressure'); // mg/dl (قند خون)

            // انرژی (Energy) — self reported level
            $table->string('energy_level')->nullable()->after('blood_sugar'); // very_low, low, medium, high, very_high
        });
    }

    /**
     * Reverse the migrations.
     */
    public function down(): void
    {
        Schema::table('daily_health_logs', function (Blueprint $table) {
            $table->dropColumn([
                'heart_rate',
                'systolic_pressure',
                'diastolic_pressure',
                'blood_sugar',
                'energy_level',
            ]);
        });
    }
};
