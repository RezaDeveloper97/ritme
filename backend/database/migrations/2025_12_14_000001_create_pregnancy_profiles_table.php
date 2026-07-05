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
        Schema::create('pregnancy_profiles', function (Blueprint $table) {
            $table->id();
            $table->foreignId('user_id')->constrained()->onDelete('cascade');

            // Mode flags
            $table->boolean('pregnancy_mode')->default(true);
            $table->boolean('cycle_mode')->default(false);
            $table->boolean('is_locked')->default(false);

            // Pregnancy age source and calculation data
            $table->string('age_source')->nullable(); // lmp, ultrasound, manual
            $table->string('confidence_level')->nullable(); // high, medium, low

            // LMP-based data
            $table->date('lmp_date')->nullable(); // Last Menstrual Period date

            // Ultrasound-based data
            $table->date('ultrasound_date')->nullable();
            $table->integer('ultrasound_weeks')->nullable();
            $table->integer('ultrasound_days')->nullable();

            // Manual entry data
            $table->integer('manual_weeks')->nullable();
            $table->integer('manual_days')->nullable();
            $table->date('manual_entry_date')->nullable(); // When manual entry was made

            // Calculated data
            $table->date('estimated_due_date')->nullable(); // EDD
            $table->date('estimated_conception_date')->nullable();
            $table->integer('uncertainty_days')->default(3); // ±3 days by default

            // Baseline risk profile - Pregnancy history (optional)
            $table->boolean('has_miscarriage_history')->nullable();
            $table->boolean('has_high_risk_history')->nullable();

            // Pre-existing conditions (stored as JSON array)
            $table->json('pre_existing_conditions')->nullable();

            // Blood information (optional)
            $table->string('blood_type')->nullable(); // A, B, AB, O
            $table->string('rh_factor')->nullable(); // positive, negative
            $table->boolean('rh_negative_care_flag')->default(false);

            // Fetal movement tracking
            $table->date('first_fetal_movement_date')->nullable();
            $table->boolean('fetal_movement_felt')->default(false);

            // Onboarding status
            $table->boolean('onboarding_completed')->default(false);
            $table->timestamp('onboarding_completed_at')->nullable();

            $table->timestamps();

            // One pregnancy profile per user
            $table->unique('user_id');
        });
    }

    /**
     * Reverse the migrations.
     */
    public function down(): void
    {
        Schema::dropIfExists('pregnancy_profiles');
    }
};
