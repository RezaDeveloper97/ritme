<?php

use Illuminate\Database\Migrations\Migration;
use Illuminate\Database\Schema\Blueprint;
use Illuminate\Support\Facades\Schema;

/**
 * cycle_histories carried both a UNIQUE and a plain INDEX on the same
 * (user_id, period_start_date). The unique index serves every lookup the plain one
 * did (and backs the user_id foreign key, being user_id-leading), so the plain one
 * only cost write time and space (perf baseline §2.2).
 */
return new class extends Migration
{
    private const INDEX = 'cycle_histories_user_id_period_start_date_index';

    public function up(): void
    {
        if (! Schema::hasIndex('cycle_histories', self::INDEX)) {
            return;
        }

        Schema::table('cycle_histories', function (Blueprint $table) {
            $table->dropIndex(self::INDEX);
        });
    }

    public function down(): void
    {
        if (Schema::hasIndex('cycle_histories', self::INDEX)) {
            return;
        }

        Schema::table('cycle_histories', function (Blueprint $table) {
            $table->index(['user_id', 'period_start_date'], self::INDEX);
        });
    }
};
