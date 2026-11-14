<?php

use Illuminate\Database\Migrations\Migration;
use Illuminate\Database\Schema\Blueprint;
use Illuminate\Support\Facades\DB;
use Illuminate\Support\Facades\Schema;

// Schema-only twin of backend-go/db/migrations/00123_user_notifications_pushed_at.sql: when the Go Web Push sender
// handled an inbox row (NULL = not yet). Rows that exist now are marked handled. No model change here.
return new class extends Migration
{
    public function up(): void
    {
        if (! Schema::hasColumn('user_notifications', 'pushed_at')) {
            Schema::table('user_notifications', function (Blueprint $table) {
                $table->dateTime('pushed_at')->nullable();
                $table->index(['pushed_at', 'id']);
            });
        }
        DB::statement('UPDATE `user_notifications` SET `pushed_at` = COALESCE(`created_at`, NOW()) WHERE `pushed_at` IS NULL');
    }

    public function down(): void
    {
        if (Schema::hasColumn('user_notifications', 'pushed_at')) {
            Schema::table('user_notifications', function (Blueprint $table) {
                $table->dropIndex(['pushed_at', 'id']);
                $table->dropColumn('pushed_at');
            });
        }
    }
};
