<?php

use Illuminate\Database\Migrations\Migration;
use Illuminate\Database\Schema\Blueprint;
use Illuminate\Support\Facades\Schema;

return new class extends Migration
{
    /**
     * Run the migrations.
     */
    public function up(): void
    {
        Schema::table('cycle_histories', function (Blueprint $table) {
            // Where this period record came from (spec §15). Existing rows are real
            // user logs, so default to that; the engine sets it explicitly onward.
            $table->string('source')->default('user_logged')->after('is_confirmed');
            // Non-fatal quality markers (incomplete_end_missing, outliers, …): kept in
            // history but honoured when building prediction medians (spec §8).
            $table->json('data_quality_flags')->nullable()->after('source');
        });
    }

    /**
     * Reverse the migrations.
     */
    public function down(): void
    {
        Schema::table('cycle_histories', function (Blueprint $table) {
            $table->dropColumn(['source', 'data_quality_flags']);
        });
    }
};
