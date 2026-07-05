<?php

use Illuminate\Database\Migrations\Migration;
use Illuminate\Database\Schema\Blueprint;
use Illuminate\Support\Facades\Schema;

return new class extends Migration
{
    /**
     * Run the migrations.
     *
     * System-defined daily tasks (وظایف امروز). Completion per user/day is
     * tracked separately in user_task_completions, so templates stay reusable.
     */
    public function up(): void
    {
        Schema::create('task_templates', function (Blueprint $table) {
            $table->id();
            $table->string('key')->unique();             // stable identifier, e.g. log_bbt
            $table->json('title');                        // {fa, en}
            $table->json('description')->nullable();      // {fa, en}
            $table->string('category');                   // App\Enums\TaskCategory
            $table->string('icon')->nullable();           // overrides category default icon
            $table->string('cycle_phase')->nullable();    // null = all phases; else App\Enums\CyclePhase
            $table->boolean('is_active')->default(true);
            $table->integer('sort_order')->default(0);
            $table->timestamps();

            $table->index(['is_active', 'cycle_phase']);
        });
    }

    public function down(): void
    {
        Schema::dropIfExists('task_templates');
    }
};
