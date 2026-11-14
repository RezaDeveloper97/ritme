<?php

use Illuminate\Database\Migrations\Migration;
use Illuminate\Database\Schema\Blueprint;
use Illuminate\Support\Facades\Schema;

// Mirror of backend-go/db/migrations/00120_drop_pregnancy_week_bookmark.sql: the week-page bookmark is removed.
return new class extends Migration
{
    public function up(): void
    {
        Schema::table('pregnancy_week_user_state', function (Blueprint $table) {
            $table->dropColumn('bookmarked');
        });
    }

    public function down(): void
    {
        Schema::table('pregnancy_week_user_state', function (Blueprint $table) {
            $table->boolean('bookmarked')->default(false)->after('week');
        });
    }
};
