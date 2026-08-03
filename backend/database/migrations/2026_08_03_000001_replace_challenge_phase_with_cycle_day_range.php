<?php

use Illuminate\Database\Migrations\Migration;
use Illuminate\Database\Schema\Blueprint;
use Illuminate\Support\Facades\DB;
use Illuminate\Support\Facades\Schema;

return new class extends Migration
{
    /**
     * Challenges are now targeted by *cycle day* (1..35) instead of cycle
     * phase: a phase covers a wide, user-dependent span, while admins author
     * challenges for concrete days ("روز ۱ تا ۳ قاعدگی").
     *
     * Both bounds are nullable and independent:
     *   from=null, to=null → every day (the default)
     *   from=6,    to=12   → days 6..12
     *   from=20,   to=null → day 20 onwards
     *   from=null, to=5    → up to day 5
     */
    private const MAX_CYCLE_DAY = 35;

    /**
     * Phase → day range for the backfill, using the canonical defaults
     * {@see \App\Services\HealthEngine\CyclePhaseMapper} draws for a 28-day
     * cycle with 5 bleeding days and ovulation on day 14.
     */
    private const PHASE_RANGES = [
        'menstruation' => [1, 5],
        'follicular' => [6, 13],
        'ovulation' => [14, 15],
        'luteal' => [16, self::MAX_CYCLE_DAY],
    ];

    public function up(): void
    {
        Schema::table('challenges', function (Blueprint $table) {
            $table->unsignedTinyInteger('cycle_day_from')->nullable()->after('description');
            $table->unsignedTinyInteger('cycle_day_to')->nullable()->after('cycle_day_from');
        });

        foreach (self::PHASE_RANGES as $phase => [$from, $to]) {
            DB::table('challenges')
                ->where('cycle_phase', $phase)
                ->update(['cycle_day_from' => $from, 'cycle_day_to' => $to]);
        }

        Schema::table('challenges', function (Blueprint $table) {
            $table->dropIndex(['is_active', 'cycle_phase']);
            $table->dropColumn('cycle_phase');
            $table->index(['is_active', 'cycle_day_from', 'cycle_day_to'], 'challenges_active_day_index');
        });
    }

    public function down(): void
    {
        Schema::table('challenges', function (Blueprint $table) {
            $table->dropIndex('challenges_active_day_index');
            $table->string('cycle_phase')->nullable()->after('description');
        });

        // Only ranges that exactly match a phase can be mapped back; anything
        // an admin authored in between is a day range with no phase equivalent
        // and stays null (= any phase).
        foreach (self::PHASE_RANGES as $phase => [$from, $to]) {
            DB::table('challenges')
                ->where('cycle_day_from', $from)
                ->where('cycle_day_to', $to)
                ->update(['cycle_phase' => $phase]);
        }

        Schema::table('challenges', function (Blueprint $table) {
            $table->dropColumn(['cycle_day_from', 'cycle_day_to']);
            $table->index(['is_active', 'cycle_phase']);
        });
    }
};
