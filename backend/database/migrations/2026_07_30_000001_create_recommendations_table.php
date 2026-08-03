<?php

use Illuminate\Database\Migrations\Migration;
use Illuminate\Database\Schema\Blueprint;
use Illuminate\Support\Facades\Schema;

/**
 * Editable storage for the "توصیه‌های امروز" daily recommendations that used to
 * be hardcoded in HealthDataEngine::getPhaseBasedTips()/getSymptomBasedTips().
 *
 * A row targets a cycle phase (null = every phase), optionally narrows to a set
 * of sub-phases, and may additionally require a logged symptom
 * (`symptom_trigger`). Resolution lives in
 * {@see App\Services\HealthEngine\RecommendationRepository}.
 */
return new class extends Migration
{
    public function up(): void
    {
        Schema::create('recommendations', function (Blueprint $table) {
            $table->id();
            // Stable key for the rows shipped by RecommendationSeeder, so
            // re-seeding updates them instead of duplicating. Admin-created
            // rows leave it null.
            $table->string('key')->nullable()->unique();
            $table->string('type')->default('general');   // RecommendationType
            $table->json('title')->nullable();            // {fa, en} — overrides the category label
            $table->json('text');                         // {fa, en}
            $table->string('cycle_phase')->nullable();    // CyclePhase, null = every phase
            $table->json('cycle_subphases')->nullable();  // CycleSubphase[], empty = every sub-phase
            $table->string('symptom_trigger')->nullable(); // RecommendationTrigger, null = not symptom-gated
            $table->boolean('is_active')->default(true);
            $table->integer('sort_order')->default(0);
            $table->timestamps();

            $table->index(['is_active', 'cycle_phase']);
        });
    }

    public function down(): void
    {
        Schema::dropIfExists('recommendations');
    }
};
