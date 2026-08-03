<?php

use Illuminate\Database\Migrations\Migration;
use Illuminate\Database\Schema\Blueprint;
use Illuminate\Support\Facades\Schema;

return new class extends Migration
{
    /**
     * «چالش امروز» is now just "a task an admin defined, which the user can
     * tick off". Difficulty only ever existed to feed the streak-based
     * unlock ladder, and that ladder is gone — so the column goes with it.
     */
    public function up(): void
    {
        Schema::table('challenges', function (Blueprint $table) {
            $table->dropColumn('difficulty');
        });
    }

    public function down(): void
    {
        Schema::table('challenges', function (Blueprint $table) {
            $table->string('difficulty')->nullable()->after('category');
        });
    }
};
