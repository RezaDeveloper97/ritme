<?php

use Illuminate\Database\Migrations\Migration;
use Illuminate\Support\Facades\DB;

// Twin of backend-go/db/migrations/00129_user_fk_cascades.sql (T-M12-13): user columns that had no foreign key get
// ON DELETE CASCADE so a deleted account leaves nothing behind. Orphan rows (user_id naming no users row = leftovers
// of already-deleted accounts) are deleted first. FKs are added with foreign_key_checks off so InnoDB adds them
// in place (no table copy on message_events). Down drops the keys only.
return new class extends Migration
{
    private const TABLES = [
        'message_events' => false,
        'admin_health_access_requests' => true,  // + user_id index
        'lab_jobs' => true,                      // + user_id index
        'oauth_auth_codes' => false,
        'oauth_device_codes' => false,
        'user_profiles_dedupe_backup' => false,
    ];

    public function up(): void
    {
        if (! $this->mariadb()) {
            return; // sqlite (tests) builds these tables from the schema without the legacy gaps
        }
        foreach (array_keys(self::TABLES) as $t) {
            $nullOk = $t === 'oauth_device_codes' ? 'x.`user_id` IS NOT NULL AND ' : '';
            DB::statement("DELETE x FROM `{$t}` x LEFT JOIN `users` u ON u.`id` = x.`user_id` WHERE {$nullOk}u.`id` IS NULL");
        }
        DB::statement('SET SESSION foreign_key_checks = 0');
        try {
            foreach (self::TABLES as $t => $index) {
                $idx = $index ? "ADD INDEX IF NOT EXISTS `{$t}_user_id_index` (`user_id`), " : '';
                DB::statement("ALTER TABLE `{$t}` {$idx}ADD CONSTRAINT `{$t}_user_id_foreign` FOREIGN KEY IF NOT EXISTS (`user_id`) REFERENCES `users` (`id`) ON DELETE CASCADE, ALGORITHM=INPLACE, LOCK=NONE");
            }
        } finally {
            DB::statement('SET SESSION foreign_key_checks = 1');
        }
    }

    public function down(): void
    {
        if (! $this->mariadb()) {
            return;
        }
        foreach (self::TABLES as $t => $index) {
            DB::statement("ALTER TABLE `{$t}` DROP FOREIGN KEY IF EXISTS `{$t}_user_id_foreign`");
            if ($index) {
                DB::statement("ALTER TABLE `{$t}` DROP INDEX IF EXISTS `{$t}_user_id_index`");
            }
        }
    }

    private function mariadb(): bool
    {
        return in_array(DB::getDriverName(), ['mysql', 'mariadb'], true);
    }
};
