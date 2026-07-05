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
        Schema::create('pregnancy_symptom_logs', function (Blueprint $table) {
            $table->id();
            $table->foreignId('user_id')->constrained()->onDelete('cascade');
            $table->date('log_date');

            // ============ General symptoms ============
            // Nausea
            $table->boolean('has_nausea')->nullable();
            $table->string('nausea_severity')->nullable(); // mild, moderate, severe

            // Vomiting
            $table->boolean('has_vomiting')->nullable();
            $table->string('vomiting_severity')->nullable();

            // Fatigue
            $table->boolean('has_fatigue')->nullable();
            $table->string('fatigue_severity')->nullable();

            // Headache
            $table->boolean('has_headache')->nullable();
            $table->string('headache_severity')->nullable();

            // Dizziness
            $table->boolean('has_dizziness')->nullable();
            $table->string('dizziness_severity')->nullable();

            // Breast pain/sensitivity
            $table->boolean('has_breast_pain')->nullable();
            $table->string('breast_pain_severity')->nullable();

            // ============ Abdominal/Pelvic symptoms ============
            // Lower abdominal pain
            $table->boolean('has_lower_abdominal_pain')->nullable();
            $table->string('lower_abdominal_pain_severity')->nullable();

            // Cramping
            $table->boolean('has_cramping')->nullable();
            $table->string('cramping_severity')->nullable();

            // Back pain
            $table->boolean('has_back_pain')->nullable();
            $table->string('back_pain_severity')->nullable();

            // Pelvic pressure
            $table->boolean('has_pelvic_pressure')->nullable();
            $table->string('pelvic_pressure_severity')->nullable();

            // ============ Sensitive symptoms (always active) ============
            // Spotting
            $table->boolean('has_spotting')->nullable();
            $table->string('spotting_severity')->nullable();

            // Bleeding
            $table->boolean('has_bleeding')->nullable();
            $table->string('bleeding_severity')->nullable();

            // Fluid leakage
            $table->boolean('has_fluid_leakage')->nullable();
            $table->string('fluid_leakage_severity')->nullable();

            // Severe/sudden pain
            $table->boolean('has_severe_sudden_pain')->nullable();
            $table->string('severe_sudden_pain_severity')->nullable();

            // Notes
            $table->text('notes')->nullable();

            $table->timestamps();

            // One log per user per date
            $table->unique(['user_id', 'log_date']);
        });
    }

    /**
     * Reverse the migrations.
     */
    public function down(): void
    {
        Schema::dropIfExists('pregnancy_symptom_logs');
    }
};
