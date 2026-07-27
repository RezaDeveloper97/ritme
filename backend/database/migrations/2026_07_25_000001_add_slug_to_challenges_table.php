<?php

use Illuminate\Database\Migrations\Migration;
use Illuminate\Database\Schema\Blueprint;
use Illuminate\Support\Facades\Schema;

return new class extends Migration
{
    /**
     * A stable identifier for seeded challenges so the content seeder can be
     * re-run (to add new challenges) without duplicating existing rows.
     */
    public function up(): void
    {
        Schema::table('challenges', function (Blueprint $table) {
            $table->string('slug')->nullable()->unique()->after('id');
        });
    }

    public function down(): void
    {
        Schema::table('challenges', function (Blueprint $table) {
            $table->dropUnique(['slug']);
            $table->dropColumn('slug');
        });
    }
};
