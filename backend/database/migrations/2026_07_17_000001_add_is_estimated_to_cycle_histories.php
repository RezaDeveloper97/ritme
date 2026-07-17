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
            // Whether this record is an estimate (e.g. the onboarding seed) rather than a
            // real observed period. Separate from is_confirmed per spec §13: an estimate is
            // neither user-logged nor user-confirmed and never feeds prediction medians.
            $table->boolean('is_estimated')->default(false)->after('is_confirmed');
        });
    }

    /**
     * Reverse the migrations.
     */
    public function down(): void
    {
        Schema::table('cycle_histories', function (Blueprint $table) {
            $table->dropColumn('is_estimated');
        });
    }
};
