<?php

use Illuminate\Database\Migrations\Migration;
use Illuminate\Database\Schema\Blueprint;
use Illuminate\Support\Facades\Schema;

return new class extends Migration
{
    /**
     * Run the migrations.
     *
     * Twin of backend-go/db/migrations/00062_challenges_v2.sql
     * (docs/go-migration/migrations.md): challenges v2 — program settings,
     * per-day tasks, enrollments, task completions and the once-only reward
     * grant (bloom B-N9-07, D-87). Used only by the Go internal/challenges
     * and internal/admin packages. No model or routes.
     */
    public function up(): void
    {
        Schema::create('challenge_programs', function (Blueprint $table) {
            $table->unsignedBigInteger('challenge_id')->primary();
            $table->unsignedSmallInteger('duration_days');
            $table->string('start_rule', 16)->default('anytime');
            $table->date('starts_on')->nullable();
            $table->string('reminder_time', 5)->nullable();
            $table->json('eligibility')->nullable();
            $table->string('reward_badge_code', 64)->nullable();
            $table->json('reward_badge_title')->nullable();
            $table->unsignedSmallInteger('reward_plus_days')->default(0);
            $table->unsignedSmallInteger('completion_min_days')->nullable();
            $table->timestamps();

            $table->foreign('challenge_id')->references('id')->on('challenges')->cascadeOnDelete();
        });

        Schema::create('challenge_tasks', function (Blueprint $table) {
            $table->id();
            $table->unsignedBigInteger('challenge_id');
            $table->unsignedSmallInteger('day_from')->default(1);
            $table->unsignedSmallInteger('day_to')->default(1);
            $table->json('title');
            $table->integer('sort_order')->default(0);
            $table->timestamps();

            $table->index(['challenge_id', 'sort_order']);
            $table->foreign('challenge_id')->references('id')->on('challenges')->cascadeOnDelete();
        });

        Schema::create('challenge_enrollments', function (Blueprint $table) {
            $table->id();
            $table->unsignedBigInteger('user_id');
            $table->unsignedBigInteger('challenge_id');
            $table->date('started_on');
            $table->string('status', 16)->default('active'); // active | completed
            $table->timestamp('completed_at')->nullable();
            $table->timestamps();

            $table->unique(['user_id', 'challenge_id']);
            $table->index(['challenge_id', 'status']);
            $table->foreign('challenge_id')->references('id')->on('challenges')->cascadeOnDelete();
            $table->foreign('user_id')->references('id')->on('users')->cascadeOnDelete();
        });

        Schema::create('challenge_task_completions', function (Blueprint $table) {
            $table->id();
            $table->unsignedBigInteger('enrollment_id');
            $table->unsignedBigInteger('task_id');
            $table->unsignedSmallInteger('day_number');
            $table->timestamp('completed_at')->nullable();

            $table->unique(['enrollment_id', 'task_id', 'day_number'], 'challenge_task_completions_unique');
            $table->foreign('enrollment_id')->references('id')->on('challenge_enrollments')->cascadeOnDelete();
            $table->foreign('task_id')->references('id')->on('challenge_tasks')->cascadeOnDelete();
        });

        Schema::create('challenge_reward_grants', function (Blueprint $table) {
            $table->id();
            $table->unsignedBigInteger('user_id');
            $table->unsignedBigInteger('challenge_id');
            $table->unsignedBigInteger('enrollment_id');
            $table->string('badge_code', 64)->nullable();
            $table->json('badge_title')->nullable();
            $table->unsignedSmallInteger('plus_days')->default(0);
            $table->unsignedBigInteger('subscription_id')->nullable();
            $table->timestamp('granted_at')->nullable();

            $table->unique(['user_id', 'challenge_id']);
            $table->unique('enrollment_id');
            $table->foreign('challenge_id')->references('id')->on('challenges')->cascadeOnDelete();
            $table->foreign('enrollment_id')->references('id')->on('challenge_enrollments')->cascadeOnDelete();
            $table->foreign('user_id')->references('id')->on('users')->cascadeOnDelete();
        });
    }

    /**
     * Reverse the migrations.
     */
    public function down(): void
    {
        Schema::dropIfExists('challenge_reward_grants');
        Schema::dropIfExists('challenge_task_completions');
        Schema::dropIfExists('challenge_enrollments');
        Schema::dropIfExists('challenge_tasks');
        Schema::dropIfExists('challenge_programs');
    }
};
