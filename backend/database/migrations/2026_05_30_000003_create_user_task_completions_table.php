<?php

use Illuminate\Database\Migrations\Migration;
use Illuminate\Database\Schema\Blueprint;
use Illuminate\Support\Facades\Schema;

return new class extends Migration
{
    /**
     * Run the migrations.
     *
     * One row per (user, task, day) when the user checks a task off.
     * Absence of a row means "not completed".
     */
    public function up(): void
    {
        Schema::create('user_task_completions', function (Blueprint $table) {
            $table->id();
            $table->foreignId('user_id')->constrained()->cascadeOnDelete();
            $table->foreignId('task_template_id')->constrained('task_templates')->cascadeOnDelete();
            $table->date('completion_date');
            $table->timestamp('completed_at')->nullable();
            $table->timestamps();

            $table->unique(['user_id', 'task_template_id', 'completion_date'], 'user_task_unique_per_day');
            $table->index(['user_id', 'completion_date']);
        });
    }

    public function down(): void
    {
        Schema::dropIfExists('user_task_completions');
    }
};
