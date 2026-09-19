<?php

use Illuminate\Database\Migrations\Migration;
use Illuminate\Database\Schema\Blueprint;
use Illuminate\Support\Facades\DB;
use Illuminate\Support\Facades\Schema;

/**
 * `cycle_calculations` was write-only storage: CalculateCycleDataJob upserted 366
 * precomputed days into it on every engine-input write, and nothing ever read them
 * back (every cycle endpoint computes live). On prod it held ~89k rows / ~238 MB,
 * about 99 % of the database (perf baseline §2.3). The job and the model are gone
 * (T-M1-13), so the table goes too.
 *
 * With the job gone, a recalculation completes synchronously
 * (UserProfile::markRecalculated), so no profile can be "processing" any more.
 * Profiles left on "processing" by a job that will now never run are settled to
 * "completed" here; otherwise clients would keep polling /cycle/status until
 * the user's next write.
 *
 * down() recreates the table with its original schema (2025_12_08_000001) but
 * EMPTY: the dropped rows are not restored. That loss is acceptable because the
 * data was never read, and every row can be recomputed from the user's inputs.
 * The status settle is not reverted either: "processing" would be a lie.
 */
return new class extends Migration
{
    public function up(): void
    {
        DB::table('user_profiles')
            ->where('calculation_status', 'processing')
            ->update([
                'calculation_status' => 'completed',
                'calculation_completed_at' => now(),
            ]);

        Schema::dropIfExists('cycle_calculations');
    }

    public function down(): void
    {
        if (Schema::hasTable('cycle_calculations')) {
            return;
        }

        Schema::create('cycle_calculations', function (Blueprint $table) {
            $table->id();
            $table->foreignId('user_id')->constrained()->onDelete('cascade');
            $table->date('calculation_date')->index();
            $table->integer('version')->default(1);
            $table->boolean('is_locked')->default(false);

            $table->integer('cycle_day')->nullable();
            $table->string('phase')->nullable();
            $table->string('subphase')->nullable();
            $table->integer('estimated_ovulation_day')->nullable();
            $table->integer('cycle_length_used')->nullable();

            $table->boolean('is_fertile_window')->default(false);
            $table->boolean('is_pms_window')->default(false);
            $table->boolean('is_period_tomorrow')->default(false);
            $table->boolean('is_luteal_spotting')->default(false);

            $table->decimal('cycle_score', 5, 4)->nullable();
            $table->decimal('age_factor', 5, 4)->nullable();
            $table->decimal('base_probability', 5, 4)->nullable();
            $table->decimal('symptom_score', 5, 4)->nullable();
            $table->decimal('final_probability', 5, 2)->nullable();

            $table->string('cycle_variability')->nullable();
            $table->integer('uncertainty_range')->nullable();

            $table->json('text_flags')->nullable();
            $table->json('daily_tips')->nullable();

            $table->json('source_profile_data')->nullable();
            $table->json('source_daily_log_data')->nullable();

            $table->timestamps();

            $table->unique(['user_id', 'calculation_date', 'version']);
        });
    }
};
