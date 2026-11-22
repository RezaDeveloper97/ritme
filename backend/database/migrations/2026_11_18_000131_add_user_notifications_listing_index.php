<?php

use Illuminate\Database\Migrations\Migration;
use Illuminate\Support\Facades\DB;
use Illuminate\Support\Facades\Schema;

// Schema-only twin of backend-go/db/migrations/00131_user_notifications_listing_index.sql (T-M12-32): the inbox list
// (newest first, ties by id) reads through a (user_id, created_at DESC, id) key instead of a filesort. The schema
// builder cannot express a DESC key part, so the index is plain SQL (MySQL 8, MariaDB and SQLite all accept it).
return new class extends Migration
{
    private const INDEX = 'user_notifications_user_id_created_at_id_index';

    public function up(): void
    {
        if (! Schema::hasIndex('user_notifications', self::INDEX)) {
            DB::statement('CREATE INDEX '.self::INDEX.' ON user_notifications (user_id, created_at DESC, id)');
        }
    }

    public function down(): void
    {
        if (Schema::hasIndex('user_notifications', self::INDEX)) {
            Schema::table('user_notifications', function ($table) {
                $table->dropIndex(self::INDEX);
            });
        }
    }
};
