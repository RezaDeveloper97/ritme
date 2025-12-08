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
        Schema::create('cycle_calculations', function (Blueprint $table) {
            $table->id();
            $table->foreignId('user_id')->constrained()->onDelete('cascade');
            $table->date('calculation_date')->index();
            $table->integer('version')->default(1);
            $table->boolean('is_locked')->default(false);

            // Core cycle data
            $table->integer('cycle_day')->nullable();
            $table->string('phase')->nullable();
            $table->string('subphase')->nullable();
            $table->integer('estimated_ovulation_day')->nullable();
            $table->integer('cycle_length_used')->nullable();

            // Windows and predictions
            $table->boolean('is_fertile_window')->default(false);
            $table->boolean('is_pms_window')->default(false);
            $table->boolean('is_period_tomorrow')->default(false);
            $table->boolean('is_luteal_spotting')->default(false);

            // Scores and factors
            $table->decimal('cycle_score', 5, 4)->nullable();
            $table->decimal('age_factor', 5, 4)->nullable();
            $table->decimal('base_probability', 5, 4)->nullable();
            $table->decimal('symptom_score', 5, 4)->nullable();
            $table->decimal('final_probability', 5, 2)->nullable();

            // Variability and uncertainty
            $table->string('cycle_variability')->nullable();
            $table->integer('uncertainty_range')->nullable();

            // Daily tips and flags (JSON for multi-language)
            $table->json('text_flags')->nullable();
            $table->json('daily_tips')->nullable();

            // Source data reference
            $table->json('source_profile_data')->nullable();
            $table->json('source_daily_log_data')->nullable();

            $table->timestamps();

            // Ensure one calculation per user per date per version
            $table->unique(['user_id', 'calculation_date', 'version']);
        });
    }

    /**
     * Reverse the migrations.
     */
    public function down(): void
    {
        Schema::dropIfExists('cycle_calculations');
    }
};
