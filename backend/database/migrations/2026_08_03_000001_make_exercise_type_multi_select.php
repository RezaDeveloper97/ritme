<?php

use Illuminate\Database\Migrations\Migration;
use Illuminate\Database\Schema\Blueprint;
use Illuminate\Support\Facades\DB;
use Illuminate\Support\Facades\Schema;

return new class extends Migration
{
    /**
     * A workout day is rarely one activity — the user can now record several
     * (walk + gym). Existing single values are wrapped into a one-item array
     * *before* the column changes type so no row is left holding non-JSON.
     */
    public function up(): void
    {
        DB::table('daily_health_logs')
            ->whereNotNull('exercise_type')
            ->orderBy('id')
            ->chunkById(500, function ($rows) {
                foreach ($rows as $row) {
                    DB::table('daily_health_logs')
                        ->where('id', $row->id)
                        ->update(['exercise_type' => json_encode([$row->exercise_type])]);
                }
            });

        Schema::table('daily_health_logs', function (Blueprint $table) {
            $table->json('exercise_type')->nullable()->change();
        });
    }

    public function down(): void
    {
        Schema::table('daily_health_logs', function (Blueprint $table) {
            $table->string('exercise_type')->nullable()->change();
        });

        // Collapse back to the first recorded activity.
        DB::table('daily_health_logs')
            ->whereNotNull('exercise_type')
            ->orderBy('id')
            ->chunkById(500, function ($rows) {
                foreach ($rows as $row) {
                    $decoded = json_decode((string) $row->exercise_type, true);
                    DB::table('daily_health_logs')
                        ->where('id', $row->id)
                        ->update(['exercise_type' => is_array($decoded) ? ($decoded[0] ?? null) : $row->exercise_type]);
                }
            });
    }
};
