<?php

use Illuminate\Database\Migrations\Migration;
use Illuminate\Database\Schema\Blueprint;
use Illuminate\Support\Facades\DB;
use Illuminate\Support\Facades\Schema;

return new class extends Migration
{
    /**
     * Run the migrations.
     *
     * Twin of backend-go/db/migrations/00101_consent_events.sql (docs/go-migration/migrations.md, T-M10-23, D-174):
     * the append-only consent ledger read and written by the Go internal/consent package (user_consents stays the
     * current-state projection). Schema, the two append-only triggers and the same backfill as goose; no model or
     * routes here (the Laravel API ignores signup consents).
     */
    public function up(): void
    {
        Schema::create('consent_events', function (Blueprint $table) {
            $table->id();
            $table->foreignId('user_id')->constrained('users')->cascadeOnDelete();
            $table->string('consent', 64);
            $table->unsignedInteger('version');
            $table->string('action', 8);
            $table->json('scope')->nullable();
            $table->string('source', 24);
            $table->timestamp('created_at')->nullable();

            $table->index(['user_id', 'consent', 'id']);
        });

        // A foreign-key cascade (user deletion) does not fire triggers, so account deletion still removes the rows.
        DB::unprepared("CREATE TRIGGER IF NOT EXISTS `consent_events_no_update` BEFORE UPDATE ON `consent_events` FOR EACH ROW\n  SIGNAL SQLSTATE '45000' SET MESSAGE_TEXT = 'consent_events is append-only'");
        DB::unprepared("CREATE TRIGGER IF NOT EXISTS `consent_events_no_delete` BEFORE DELETE ON `consent_events` FOR EACH ROW\n  SIGNAL SQLSTATE '45000' SET MESSAGE_TEXT = 'consent_events is append-only'");

        DB::statement(<<<'SQL'
INSERT INTO `consent_events` (`user_id`, `consent`, `version`, `action`, `scope`, `source`, `created_at`)
SELECT b.user_id, b.consent, b.version, b.action, NULL, 'backfill', b.at
FROM (
  SELECT uc.user_id, uc.consent, COALESCE(uc.version, 0) AS version, 'grant' AS action,
         COALESCE(uc.granted_at, uc.created_at) AS at, 0 AS seq, uc.id
  FROM `user_consents` uc
  WHERE uc.granted = 1 OR uc.granted_at IS NOT NULL
  UNION ALL
  SELECT uc.user_id, uc.consent, COALESCE(uc.version, 0), 'revoke', uc.revoked_at, 1, uc.id
  FROM `user_consents` uc
  WHERE uc.revoked_at IS NOT NULL
) b
WHERE NOT EXISTS (SELECT 1 FROM `consent_events` e WHERE e.user_id = b.user_id AND e.consent = b.consent)
ORDER BY b.at, b.id, b.seq
SQL);
    }

    /**
     * Reverse the migrations.
     */
    public function down(): void
    {
        DB::unprepared('DROP TRIGGER IF EXISTS `consent_events_no_delete`');
        DB::unprepared('DROP TRIGGER IF EXISTS `consent_events_no_update`');
        Schema::dropIfExists('consent_events');
    }
};
