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
     * Twin of backend-go/db/migrations/00105_admin_metrics_audit_chain.sql (docs/go-migration/migrations.md, T-M10-25):
     * metric alert rules and alerts, per-day user activity, the admin user-directory projection, saved segments,
     * users.ref_code, the audit hash chain (two columns + a one-row head) and persisted safety self-test runs, plus
     * append-only triggers on admin_audit_logs and clinical_config_approvals and three KPI indexes. Used only by the Go
     * backend: schema and seed rows only, no model or routes here (the Laravel admin never updates or deletes either
     * table).
     */
    public function up(): void
    {
        if (! Schema::hasTable('user_activity_days')) {
            Schema::create('user_activity_days', function (Blueprint $table) {
                $table->foreignId('user_id')->constrained('users')->cascadeOnDelete();
                $table->date('day');
                $table->boolean('session')->default(false);
                $table->boolean('logged')->default(false);

                $table->primary(['day', 'user_id']);
                $table->index(['user_id', 'day']);
            });
        }

        if (! Schema::hasTable('metric_alert_rules')) {
            Schema::create('metric_alert_rules', function (Blueprint $table) {
                $table->id();
                $table->string('metric_key', 64)->unique();
                $table->string('comparator', 16);
                $table->decimal('threshold', 12, 4);
                $table->unsignedSmallInteger('window_days');
                $table->unsignedInteger('min_volume')->nullable();
                $table->boolean('enabled')->default(true);
                $table->timestamps();
            });
        }

        if (! Schema::hasTable('metric_alerts')) {
            Schema::create('metric_alerts', function (Blueprint $table) {
                $table->id();
                $table->foreignId('rule_id')->constrained('metric_alert_rules')->cascadeOnDelete();
                $table->string('subject', 64)->default('');
                $table->date('day');
                $table->decimal('value', 12, 4);
                $table->decimal('threshold', 12, 4);
                $table->string('state', 12)->default('open');
                // 1 while open, NULL once resolved: one open alert per (rule, subject), any number of resolved ones.
                $table->boolean('open_flag')->storedAs("IF(`state` = 'open', 1, NULL)");
                $table->timestamp('opened_at')->nullable();
                $table->timestamp('resolved_at')->nullable();

                $table->unique(['rule_id', 'subject', 'open_flag']);
                $table->index(['state', 'opened_at']);
            });
        }

        if (! Schema::hasTable('admin_user_activity')) {
            Schema::create('admin_user_activity', function (Blueprint $table) {
                $table->unsignedBigInteger('user_id')->primary();
                $table->timestamp('last_activity_at')->nullable();
                $table->unsignedTinyInteger('log_days_30')->default(0);
                $table->unsignedTinyInteger('log_days_recent14')->default(0);
                $table->unsignedTinyInteger('log_days_prior14')->default(0);
                $table->string('churn', 8);
                $table->string('mode', 16);
                $table->string('plan', 16);
                $table->string('data_quality', 16)->nullable();
                $table->string('main_warning', 48)->nullable();
                $table->date('updated_on');

                $table->index('churn');
                $table->index('plan');
                $table->index('mode');
                $table->index('data_quality');
                $table->index('last_activity_at');
                $table->foreign('user_id')->references('id')->on('users')->cascadeOnDelete();
            });
        }

        if (! Schema::hasTable('admin_segments')) {
            Schema::create('admin_segments', function (Blueprint $table) {
                $table->id();
                $table->string('name', 80)->unique();
                $table->json('filters');
                $table->unsignedBigInteger('created_by')->nullable();
                $table->timestamps();
            });
        }

        if (! Schema::hasColumn('users', 'ref_code')) {
            Schema::table('users', function (Blueprint $table) {
                $table->string('ref_code', 12)->nullable()->unique();
                $table->index('created_at');
            });
        }

        if (! Schema::hasColumn('admin_audit_logs', 'prev_hash')) {
            Schema::table('admin_audit_logs', function (Blueprint $table) {
                $table->char('prev_hash', 64)->nullable();
                $table->char('row_hash', 64)->nullable();
            });
        }

        if (! Schema::hasTable('admin_audit_chain_head')) {
            Schema::create('admin_audit_chain_head', function (Blueprint $table) {
                $table->unsignedTinyInteger('id')->primary();
                $table->unsignedBigInteger('last_id')->nullable();
                $table->char('last_hash', 64)->nullable();
                $table->unsignedBigInteger('genesis_last_id')->nullable();
                $table->char('genesis_hash', 64)->nullable();
            });
            DB::statement('ALTER TABLE `admin_audit_chain_head` ADD CONSTRAINT `admin_audit_chain_head_singleton` CHECK (`id` = 1)');
        }
        DB::table('admin_audit_chain_head')->insertOrIgnore(['id' => 1]);

        if (! Schema::hasTable('safety_scenario_runs')) {
            Schema::create('safety_scenario_runs', function (Blueprint $table) {
                $table->id();
                $table->char('run_id', 26);
                $table->string('scenario', 64);
                $table->boolean('passed');
                $table->json('detail')->nullable();
                $table->unsignedInteger('duration_ms');
                $table->timestamp('created_at')->nullable();

                $table->index('created_at');
                $table->index(['scenario', 'created_at']);
                $table->index('run_id');
            });
        }

        DB::statement('CREATE INDEX IF NOT EXISTS `daily_health_logs_log_date_user_id_index` ON `daily_health_logs` (`log_date`,`user_id`)');
        DB::statement('CREATE INDEX IF NOT EXISTS `cycle_histories_period_start_date_index` ON `cycle_histories` (`period_start_date`)');

        // SPEC §5.1 thresholds (+ the T-M10-24 self-test); insertOrIgnore keeps a rule an admin already edited.
        $now = now();
        $rule = fn (string $key, string $comparator, string $threshold, int $window, ?int $minVolume) => [
            'metric_key' => $key, 'comparator' => $comparator, 'threshold' => $threshold, 'window_days' => $window,
            'min_volume' => $minVolume, 'enabled' => true, 'created_at' => $now, 'updated_at' => $now,
        ];
        DB::table('metric_alert_rules')->insertOrIgnore([
            $rule('ocr_correction_rate', 'gt', '0.1200', 7, null),
            $rule('message_ctr', 'drop_vs_mean', '0.3000', 7, 200),
            $rule('rui_users', 'rising_days', '7.0000', 7, null),
            $rule('engine_mae', 'gt', '2.5000', 30, null),
            $rule('safety_selftest_failed', 'gt', '0.0000', 1, null),
        ]);

        // Append-only in the database. FK actions do not fire triggers, so neither table may gain a foreign key whose
        // action updates or deletes rows.
        foreach (['admin_audit_logs', 'clinical_config_approvals'] as $t) {
            DB::unprepared("CREATE TRIGGER IF NOT EXISTS `{$t}_no_update` BEFORE UPDATE ON `{$t}` FOR EACH ROW\n  SIGNAL SQLSTATE '45000' SET MESSAGE_TEXT = '{$t} is append-only'");
            DB::unprepared("CREATE TRIGGER IF NOT EXISTS `{$t}_no_delete` BEFORE DELETE ON `{$t}` FOR EACH ROW\n  SIGNAL SQLSTATE '45000' SET MESSAGE_TEXT = '{$t} is append-only'");
        }
    }

    /**
     * Reverse the migrations.
     */
    public function down(): void
    {
        foreach (['clinical_config_approvals', 'admin_audit_logs'] as $t) {
            DB::unprepared("DROP TRIGGER IF EXISTS `{$t}_no_delete`");
            DB::unprepared("DROP TRIGGER IF EXISTS `{$t}_no_update`");
        }

        DB::statement('DROP INDEX IF EXISTS `cycle_histories_period_start_date_index` ON `cycle_histories`');
        DB::statement('DROP INDEX IF EXISTS `daily_health_logs_log_date_user_id_index` ON `daily_health_logs`');

        Schema::dropIfExists('safety_scenario_runs');
        Schema::dropIfExists('admin_audit_chain_head');

        if (Schema::hasColumn('admin_audit_logs', 'prev_hash')) {
            Schema::table('admin_audit_logs', function (Blueprint $table) {
                $table->dropColumn(['row_hash', 'prev_hash']);
            });
        }

        if (Schema::hasColumn('users', 'ref_code')) {
            Schema::table('users', function (Blueprint $table) {
                $table->dropIndex(['created_at']);
                $table->dropUnique(['ref_code']);
                $table->dropColumn('ref_code');
            });
        }

        Schema::dropIfExists('admin_segments');
        Schema::dropIfExists('admin_user_activity');
        Schema::dropIfExists('metric_alerts');
        Schema::dropIfExists('metric_alert_rules');
        Schema::dropIfExists('user_activity_days');
    }
};
