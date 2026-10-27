<?php

use Illuminate\Database\Migrations\Migration;
use Illuminate\Database\Schema\Blueprint;
use Illuminate\Support\Facades\Schema;

return new class extends Migration
{
    /**
     * Run the migrations.
     *
     * Twin of backend-go/db/migrations/00109_engine_snapshots.sql (docs/go-migration/migrations.md, T-M9-39, D-137):
     * the nightly engine output per active user and day, and the per-cycle prediction log. Used only by the Go backend:
     * schema only, no model or routes here.
     */
    public function up(): void
    {
        if (! Schema::hasTable('engine_daily_snapshots')) {
            Schema::create('engine_daily_snapshots', function (Blueprint $table) {
                $table->foreignId('user_id')->constrained('users')->cascadeOnDelete();
                $table->date('day');
                $table->string('main_phase', 24);
                $table->string('subphase', 32);
                $table->string('confidence', 8);
                $table->string('data_quality', 16);
                $table->unsignedInteger('warnings_mask')->default(0);
                $table->smallInteger('days_late')->nullable();
                $table->string('mode', 16);
                $table->unsignedBigInteger('engine_rev')->default(0);
                $table->timestamp('created_at')->nullable();

                $table->primary(['user_id', 'day']);
                $table->index('day');
            });
        }

        if (! Schema::hasTable('engine_prediction_log')) {
            Schema::create('engine_prediction_log', function (Blueprint $table) {
                $table->id();
                $table->foreignId('user_id')->constrained('users')->cascadeOnDelete();
                $table->date('cycle_start');
                $table->enum('kind', ['first', 'last']);
                $table->date('predicted_next');
                $table->unsignedTinyInteger('basis_cycles');
                $table->string('ovulation_source', 16);
                $table->string('confidence', 8);
                $table->date('actual_start')->nullable();
                $table->smallInteger('error_days')->nullable();
                $table->timestamp('created_at')->nullable();

                $table->unique(['user_id', 'cycle_start', 'kind']);
            });
        }
    }

    /**
     * Reverse the migrations.
     */
    public function down(): void
    {
        Schema::dropIfExists('engine_prediction_log');
        Schema::dropIfExists('engine_daily_snapshots');
    }
};
