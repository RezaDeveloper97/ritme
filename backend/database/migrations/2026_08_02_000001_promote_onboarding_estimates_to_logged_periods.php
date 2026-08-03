<?php

use Illuminate\Database\Migrations\Migration;
use Illuminate\Support\Facades\DB;

/**
 * The period declared during onboarding is now stored as a real logged period so the
 * calendar paints those days like any user-logged period. Bring already-onboarded
 * users along: promote their onboarding estimate to a confirmed record.
 *
 * Only users with no confirmed period at all are touched — anyone who has logged a
 * real period owns their history, and a leftover estimate there stays untouched.
 * A declared end that is still in the future becomes an open (ongoing) period, the
 * same shape the sync in ProfileController now writes.
 */
return new class extends Migration
{
    public function up(): void
    {
        $today = now()->startOfDay()->toDateString();

        $usersWithRealLogs = DB::table('cycle_histories')
            ->where('is_confirmed', true)
            ->distinct()
            ->pluck('user_id');

        DB::table('cycle_histories')
            ->where('is_estimated', true)
            ->whereNotIn('user_id', $usersWithRealLogs)
            ->orderBy('id')
            ->each(function ($row) use ($today) {
                $ongoing = $row->period_end_date !== null
                    && substr((string) $row->period_end_date, 0, 10) > $today;

                DB::table('cycle_histories')->where('id', $row->id)->update([
                    'is_confirmed' => true,
                    'is_estimated' => false,
                    'source' => 'user_profile_confirmed',
                    'period_end_date' => $ongoing ? null : $row->period_end_date,
                    'bleeding_length' => $ongoing ? null : $row->bleeding_length,
                ]);
            });
    }

    public function down(): void
    {
        DB::table('cycle_histories')
            ->where('source', 'user_profile_confirmed')
            ->update([
                'is_confirmed' => false,
                'is_estimated' => true,
                'source' => 'onboarding_estimate',
            ]);
    }
};
