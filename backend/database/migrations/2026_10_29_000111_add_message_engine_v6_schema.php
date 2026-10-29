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
     * Twin of backend-go/db/migrations/00111_message_engine_v6.sql (docs/go-migration/migrations.md, T-M10-02, D-160):
     * message engine schema v6 — message_rules rule_key / approved_version / topics / caps / variants / starts_at /
     * ends_at / hour_windows / goal_event / locked, message_events version + slot (variant widened to 16), the
     * message_event_daily rollup, the message_inbox alerts inbox and challenge_programs contraindications /
     * start_phase. Used only by the Go backend: schema only (the goose file also re-maps kinds, M-18), no model or
     * routes.
     */
    public function up(): void
    {
        if (! Schema::hasColumn('message_rules', 'rule_key')) {
            Schema::table('message_rules', function (Blueprint $table) {
                $table->string('rule_key', 64)->nullable();
                $table->unsignedInteger('approved_version')->nullable();
                $table->json('topics')->nullable();
                $table->json('caps')->nullable();         // [{count, period: day|week|cycle}]
                $table->json('variants')->nullable();     // [{key, weight}]
                $table->dateTime('starts_at')->nullable();
                $table->dateTime('ends_at')->nullable();
                $table->json('hour_windows')->nullable(); // [[from, to], …] user-local hours
                $table->string('goal_event', 48)->nullable();
                $table->boolean('locked')->default(false);

                $table->unique('rule_key');
            });
        }

        if (! Schema::hasColumn('message_events', 'slot')) {
            DB::statement("ALTER TABLE `message_events` MODIFY `variant` varchar(16) NOT NULL DEFAULT 'a'");
            Schema::table('message_events', function (Blueprint $table) {
                $table->unsignedInteger('version')->nullable();
                $table->string('slot', 32)->nullable();

                $table->index(['user_id', 'created_at']);
                $table->index(['created_at', 'rule_id']);
            });
        }

        if (! Schema::hasTable('message_event_daily')) {
            Schema::create('message_event_daily', function (Blueprint $table) {
                $table->date('day');
                $table->unsignedBigInteger('rule_id');
                $table->unsignedInteger('version');
                $table->string('variant', 16);
                $table->string('slot', 32);
                $table->string('event', 12);
                $table->unsignedInteger('count');

                $table->primary(['day', 'rule_id', 'version', 'variant', 'slot', 'event']);
                $table->index(['rule_id', 'day']);
            });
        }

        if (! Schema::hasTable('message_inbox')) {
            Schema::create('message_inbox', function (Blueprint $table) {
                $table->id();
                $table->foreignId('user_id')->constrained()->cascadeOnDelete();
                $table->unsignedBigInteger('rule_id');
                $table->unsignedInteger('version');
                $table->string('rule_key', 64)->nullable();
                $table->string('slot', 32)->default('alerts_inbox');
                $table->string('kind', 16);
                $table->string('level', 16);
                $table->string('state', 24);
                // 1 = open (one per user and message, M-16), NULL = history.
                $table->boolean('open_flag')->nullable()
                    ->storedAs("IF(`state` IN ('active','acknowledged','added_to_visit_note'), 1, NULL)");
                $table->string('evidence_key', 96)->nullable();
                $table->unsignedInteger('hit_count')->default(1);
                $table->json('evidence')->nullable();
                $table->timestamp('first_at')->nullable();
                $table->timestamp('last_at')->nullable();
                $table->timestamp('acked_at')->nullable();
                $table->timestamp('noted_at')->nullable();
                $table->timestamp('resolved_at')->nullable();
                $table->timestamp('archived_at')->nullable();
                $table->unsignedBigInteger('legacy_alert_id')->nullable();
                $table->timestamps();

                $table->unique(['user_id', 'rule_id', 'open_flag']);
                $table->index(['user_id', 'state', 'level', 'last_at']);
                $table->index(['state', 'last_at']);
            });
        }

        if (! Schema::hasColumn('challenge_programs', 'contraindications')) {
            Schema::table('challenge_programs', function (Blueprint $table) {
                $table->json('contraindications')->nullable();
                $table->string('start_phase', 24)->nullable();
            });
        }
    }

    /**
     * Reverse the migrations.
     */
    public function down(): void
    {
        if (Schema::hasColumn('challenge_programs', 'contraindications')) {
            Schema::table('challenge_programs', function (Blueprint $table) {
                $table->dropColumn(['start_phase', 'contraindications']);
            });
        }

        Schema::dropIfExists('message_inbox');
        Schema::dropIfExists('message_event_daily');

        if (Schema::hasColumn('message_events', 'slot')) {
            Schema::table('message_events', function (Blueprint $table) {
                $table->dropIndex(['created_at', 'rule_id']);
                $table->dropIndex(['user_id', 'created_at']);
                $table->dropColumn(['slot', 'version']);
            });
            DB::statement('UPDATE `message_events` SET `variant` = LEFT(`variant`, 1) WHERE CHAR_LENGTH(`variant`) > 1');
            DB::statement("ALTER TABLE `message_events` MODIFY `variant` varchar(1) NOT NULL DEFAULT 'a'");
        }

        if (Schema::hasColumn('message_rules', 'rule_key')) {
            Schema::table('message_rules', function (Blueprint $table) {
                $table->dropUnique(['rule_key']);
                $table->dropColumn([
                    'locked', 'goal_event', 'hour_windows', 'ends_at', 'starts_at', 'variants', 'caps', 'topics',
                    'approved_version', 'rule_key',
                ]);
            });
        }
    }
};
