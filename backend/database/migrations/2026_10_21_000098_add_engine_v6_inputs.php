<?php

use Illuminate\Database\Migrations\Migration;
use Illuminate\Database\Schema\Blueprint;
use Illuminate\Support\Facades\Schema;

return new class extends Migration
{
    /**
     * Twin of backend-go/db/migrations/00098_engine_v6_inputs.sql (docs/go-migration/migrations.md, T-M9-24): the
     * engine v6 inputs (declared onboarding LMP, user time zone, skipped cycles, breastfeeding flag) and the per-user
     * revision. Used only by the Go backend. Schema only — no model, routes or data here.
     */
    public function up(): void
    {
        if (! Schema::hasColumn('user_profiles', 'onboarding_last_period_start')) {
            Schema::table('user_profiles', function (Blueprint $table) {
                $table->date('onboarding_last_period_start')->nullable();
            });
        }
        if (! Schema::hasColumn('user_profiles', 'timezone')) {
            Schema::table('user_profiles', function (Blueprint $table) {
                $table->string('timezone', 64)->nullable();
            });
        }

        if (! Schema::hasColumn('postpartum_profiles', 'breastfeeding')) {
            Schema::table('postpartum_profiles', function (Blueprint $table) {
                $table->boolean('breastfeeding')->nullable();
            });
        }

        if (! Schema::hasTable('cycle_skips')) {
            Schema::create('cycle_skips', function (Blueprint $table) {
                $table->id();
                $table->foreignId('user_id')->constrained('users')->cascadeOnDelete();
                $table->date('expected_start');
                $table->timestamps();

                $table->unique(['user_id', 'expected_start']);
            });
        }

        if (! Schema::hasTable('user_revisions')) {
            Schema::create('user_revisions', function (Blueprint $table) {
                $table->unsignedBigInteger('user_id')->primary();
                $table->unsignedBigInteger('engine_rev')->default(0);
                $table->unsignedBigInteger('analysis_rev')->default(0);
                $table->timestamp('updated_at')->nullable();

                $table->foreign('user_id')->references('id')->on('users')->cascadeOnDelete();
            });
        }
    }

    /**
     * Reverse the migrations.
     */
    public function down(): void
    {
        Schema::dropIfExists('user_revisions');
        Schema::dropIfExists('cycle_skips');

        if (Schema::hasColumn('postpartum_profiles', 'breastfeeding')) {
            Schema::table('postpartum_profiles', function (Blueprint $table) {
                $table->dropColumn('breastfeeding');
            });
        }
        if (Schema::hasColumn('user_profiles', 'timezone')) {
            Schema::table('user_profiles', function (Blueprint $table) {
                $table->dropColumn('timezone');
            });
        }
        if (Schema::hasColumn('user_profiles', 'onboarding_last_period_start')) {
            Schema::table('user_profiles', function (Blueprint $table) {
                $table->dropColumn('onboarding_last_period_start');
            });
        }
    }
};
