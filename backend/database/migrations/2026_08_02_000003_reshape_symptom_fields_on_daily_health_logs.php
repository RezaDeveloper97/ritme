<?php

use Illuminate\Database\Migrations\Migration;
use Illuminate\Database\Schema\Blueprint;
use Illuminate\Support\Facades\Schema;

return new class extends Migration
{
    /**
     * Reshapes several daily-log answers that only recorded presence into the
     * graded / single-choice questions the form now asks.
     *
     * The superseded columns (`has_clots`, `vaginal_burning`, `vaginal_itching`,
     * `sexual_activities` entries for desire and intercourse) are deliberately
     * left in place: they hold real user answers, and the engines still read
     * them alongside the new fields. Nothing is back-filled — inventing an
     * intensity for a plain "yes" would be fabricated health data.
     */
    public function up(): void
    {
        Schema::table('daily_health_logs', function (Blueprint $table) {
            $table->string('clots_amount')->nullable()->after('has_clots'); // none, low, medium, high
            $table->string('vaginal_burning_intensity')->nullable()->after('vaginal_burning'); // low, medium, high
            $table->string('vaginal_itching_intensity')->nullable()->after('vaginal_itching'); // low, medium, high
            $table->string('sexual_desire')->nullable()->after('sexual_activities'); // lower, normal, higher
            $table->string('intercourse_type')->nullable()->after('sexual_desire'); // protected, unprotected
        });
    }

    public function down(): void
    {
        Schema::table('daily_health_logs', function (Blueprint $table) {
            $table->dropColumn([
                'clots_amount',
                'vaginal_burning_intensity',
                'vaginal_itching_intensity',
                'sexual_desire',
                'intercourse_type',
            ]);
        });
    }
};
