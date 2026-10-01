<?php

use Illuminate\Database\Migrations\Migration;
use Illuminate\Database\Schema\Blueprint;
use Illuminate\Support\Facades\Schema;

return new class extends Migration
{
    /**
     * Run the migrations.
     *
     * Twin of backend-go/db/migrations/00014_life_profiles.sql
     * (docs/go-migration/migrations.md): onboarding v2 answers and the
     * six life-stage modes, written by the Go-only /api/v1/onboarding and
     * /api/v1/profile/life-stage endpoints (B-N2-01). Schema only — no
     * models or routes here. Existing users get no row (legacy mode).
     */
    public function up(): void
    {
        Schema::create('user_life_profiles', function (Blueprint $table) {
            $table->id();
            $table->foreignId('user_id')->unique()->constrained()->cascadeOnDelete();
            $table->string('gender', 16)->nullable();            // female | male
            $table->string('life_mode', 32)->nullable();         // cycle | ttc | pregnancy | postpartum | menopause | teen
            $table->boolean('ivf_iui')->default(false);
            $table->boolean('track_contraception')->default(false);
            $table->json('chronic_illnesses')->nullable();
            $table->json('gyn_conditions')->nullable();
            $table->json('medications')->nullable();
            $table->string('menopause_stage', 16)->nullable();   // peri | meno | post | unsure
            $table->date('menopause_last_period')->nullable();
            $table->boolean('menopause_surgical')->nullable();
            $table->boolean('menopause_hrt')->nullable();
            $table->timestamp('onboarding_started_at')->nullable();
            $table->timestamp('onboarding_completed_at')->nullable();
            $table->timestamps();
        });
    }

    /**
     * Reverse the migrations.
     */
    public function down(): void
    {
        Schema::dropIfExists('user_life_profiles');
    }
};
