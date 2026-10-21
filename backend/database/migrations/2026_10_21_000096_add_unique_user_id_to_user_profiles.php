<?php

use Illuminate\Database\Migrations\Migration;
use Illuminate\Database\Schema\Blueprint;
use Illuminate\Support\Facades\DB;
use Illuminate\Support\Facades\Schema;

return new class extends Migration
{
    private const COLUMNS = '`id`, `user_id`, `birthday`, `weight`, `height`, `period_duration`, `cycle_duration`, '
        .'`last_period_start`, `user_goal`, `pregnancy_intention`, `chronic_conditions`, `subscription_type`, '
        .'`calculation_status`, `calculation_started_at`, `calculation_completed_at`, `calculation_version`, '
        .'`created_at`, `updated_at`';

    /** The nullable columns the kept row fills from the newest removed row that has a value. */
    private const FILLED = [
        'birthday', 'weight', 'height', 'period_duration', 'cycle_duration', 'last_period_start',
        'pregnancy_intention', 'chronic_conditions', 'calculation_started_at', 'calculation_completed_at',
        'created_at', 'updated_at',
    ];

    /**
     * Run the migrations.
     *
     * Twin of backend-go/db/migrations/00096_user_profiles_user_id_unique.sql
     * (docs/go-migration/migrations.md, T-M9-23, D-127): one profile per user.
     * On MariaDB/MySQL it runs the same statements as goose: every row except
     * the lowest id of a user is copied into user_profiles_dedupe_backup, the
     * kept row's NULL columns are filled from the newest removed row (highest
     * id) with a value, the removed rows are deleted, then
     * UNIQUE(user_id) replaces the non-unique FK index. Every step is safe to
     * run again. sqlite (local dev / phpunit only) gets the table and the
     * unique index without the dedupe: a duplicate there fails loudly.
     */
    public function up(): void
    {
        if (! Schema::hasTable('user_profiles_dedupe_backup')) {
            Schema::create('user_profiles_dedupe_backup', function (Blueprint $table) {
                $table->unsignedBigInteger('id')->primary();
                $table->unsignedBigInteger('user_id');
                $table->date('birthday')->nullable();
                $table->decimal('weight', 5, 2)->nullable();
                $table->unsignedSmallInteger('height')->nullable();
                $table->unsignedTinyInteger('period_duration')->nullable();
                $table->unsignedTinyInteger('cycle_duration')->nullable();
                $table->date('last_period_start')->nullable();
                $table->string('user_goal')->default('non_ttc');
                $table->string('pregnancy_intention')->nullable();
                $table->json('chronic_conditions')->nullable();
                $table->string('subscription_type')->default('free');
                $table->string('calculation_status')->default('pending');
                $table->timestamp('calculation_started_at')->nullable();
                $table->timestamp('calculation_completed_at')->nullable();
                $table->integer('calculation_version')->default(0);
                $table->timestamps();
                $table->timestamp('removed_at')->nullable();
                $table->index('user_id');
            });
        }

        if (! $this->mariadb()) {
            Schema::table('user_profiles', function (Blueprint $table) {
                $table->unique('user_id');
            });

            return;
        }

        $columns = self::COLUMNS;
        $selected = 'p.'.str_replace(', `', ', p.`', $columns);
        DB::statement("INSERT IGNORE INTO `user_profiles_dedupe_backup` ({$columns}, `removed_at`)
SELECT {$selected}, CONVERT_TZ(UTC_TIMESTAMP(), '+00:00', '+03:30')
FROM `user_profiles` p
JOIN (SELECT `user_id`, MIN(`id`) AS `kept_id` FROM `user_profiles` GROUP BY `user_id` HAVING COUNT(*) > 1) d
  ON d.`user_id` = p.`user_id` AND p.`id` <> d.`kept_id`
ORDER BY p.`id`");

        $sets = implode(",\n", array_map(fn (string $c) => "  k.`{$c}` = COALESCE(k.`{$c}`, (SELECT b.`{$c}` FROM `user_profiles_dedupe_backup` b
    WHERE b.`user_id` = k.`user_id` AND b.`id` > k.`id` AND b.`{$c}` IS NOT NULL ORDER BY b.`id` DESC LIMIT 1))", self::FILLED));
        DB::statement("UPDATE `user_profiles` k SET\n{$sets}
WHERE k.`id` NOT IN (SELECT `id` FROM `user_profiles_dedupe_backup`)
  AND k.`user_id` IN (SELECT `user_id` FROM `user_profiles_dedupe_backup`)");

        DB::statement('DELETE FROM `user_profiles` WHERE `id` IN (SELECT `id` FROM `user_profiles_dedupe_backup`)');
        DB::statement('ALTER TABLE `user_profiles` ADD UNIQUE KEY IF NOT EXISTS `user_profiles_user_id_unique` (`user_id`)');
        DB::statement('ALTER TABLE `user_profiles` DROP INDEX IF EXISTS `user_profiles_user_id_foreign`');
    }

    /**
     * Reverse the migrations: the non-unique FK index comes back, the unique
     * key goes, every backed-up row is restored with its original id (unless
     * the id exists again or the user was deleted), then the backup table is
     * dropped. The NULLs filled on the kept row stay filled.
     */
    public function down(): void
    {
        if ($this->mariadb()) {
            DB::statement('ALTER TABLE `user_profiles` ADD KEY IF NOT EXISTS `user_profiles_user_id_foreign` (`user_id`)');
            DB::statement('ALTER TABLE `user_profiles` DROP INDEX IF EXISTS `user_profiles_user_id_unique`');
        } else {
            Schema::table('user_profiles', function (Blueprint $table) {
                $table->dropUnique(['user_id']);
            });
        }

        if (Schema::hasTable('user_profiles_dedupe_backup')) {
            $columns = self::COLUMNS;
            $selected = 'b.'.str_replace(', `', ', b.`', $columns);
            DB::statement("INSERT INTO `user_profiles` ({$columns})
SELECT {$selected}
FROM `user_profiles_dedupe_backup` b
JOIN `users` u ON u.`id` = b.`user_id`
WHERE b.`id` NOT IN (SELECT `id` FROM `user_profiles`)
ORDER BY b.`id`");
        }

        Schema::dropIfExists('user_profiles_dedupe_backup');
    }

    private function mariadb(): bool
    {
        return in_array(DB::getDriverName(), ['mysql', 'mariadb'], true);
    }
};
