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
        Schema::create('pregnancy_alerts', function (Blueprint $table) {
            $table->id();
            $table->foreignId('user_id')->constrained()->onDelete('cascade');

            // Alert details
            $table->string('alert_level'); // info, warning, emergency
            $table->string('alert_type'); // symptom_based, missing_data, routine
            $table->string('title');
            $table->text('message');

            // Context
            $table->integer('pregnancy_week')->nullable();
            $table->json('trigger_symptoms')->nullable(); // Symptoms that triggered the alert
            $table->json('medical_history_flags')->nullable(); // Relevant medical history

            // Status
            $table->boolean('is_read')->default(false);
            $table->boolean('is_dismissed')->default(false);
            $table->timestamp('read_at')->nullable();
            $table->timestamp('dismissed_at')->nullable();

            // Recommended actions
            $table->json('recommended_actions')->nullable();

            $table->timestamps();

            // Index for efficient queries
            $table->index(['user_id', 'is_read']);
            $table->index(['user_id', 'alert_level']);
        });
    }

    /**
     * Reverse the migrations.
     */
    public function down(): void
    {
        Schema::dropIfExists('pregnancy_alerts');
    }
};
