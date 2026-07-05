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
        Schema::create('pregnancy_weekly_logs', function (Blueprint $table) {
            $table->id();
            $table->foreignId('user_id')->constrained()->onDelete('cascade');
            $table->date('log_date');
            $table->integer('pregnancy_week'); // Week number (1-40)

            // ============ Physical status ============
            // Weight
            $table->decimal('weight', 5, 2)->nullable(); // in kg

            // Swelling (stored as JSON array of locations: feet, hands, face)
            $table->json('swelling_locations')->nullable();
            $table->boolean('has_swelling')->nullable();

            // Shortness of breath
            $table->boolean('has_shortness_of_breath')->nullable();

            // ============ Blood pressure ============
            $table->boolean('has_blood_pressure_device')->default(false);
            $table->integer('systolic_pressure')->nullable(); // Upper number
            $table->integer('diastolic_pressure')->nullable(); // Lower number

            // ============ Blood sugar ============
            $table->decimal('fasting_blood_sugar', 5, 2)->nullable();
            $table->decimal('post_meal_blood_sugar', 5, 2)->nullable();

            // ============ Mental/Emotional monitoring ============
            $table->string('overall_mood')->nullable(); // good, moderate, poor
            $table->boolean('has_anxiety')->nullable();
            $table->string('anxiety_severity')->nullable();
            $table->boolean('has_mood_swings')->nullable();
            $table->string('mood_swings_severity')->nullable();
            $table->boolean('has_depression_feelings')->nullable();
            $table->string('depression_severity')->nullable();

            // Notes
            $table->text('notes')->nullable();

            $table->timestamps();

            // One log per user per week
            $table->unique(['user_id', 'pregnancy_week']);
        });
    }

    /**
     * Reverse the migrations.
     */
    public function down(): void
    {
        Schema::dropIfExists('pregnancy_weekly_logs');
    }
};
