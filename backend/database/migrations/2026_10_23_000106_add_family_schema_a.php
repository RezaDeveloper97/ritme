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
     * Twin of backend-go/db/migrations/00106_family_schema_a.sql (docs/go-migration/migrations.md, T-M11-02, D-190):
     * family schema v6 part A — companions role / template / expires_at / urgent_sms / owner_confirmed_at and three
     * composite keys (the (companion_user_id, status, owner_id) key replaces its (companion_user_id, status) prefix),
     * companion_grants.mode_epoch, the family_owner_state table, and the activity columns of companion_audit_logs.
     * Used only by the Go backend: schema only (the goose file also backfills role / template), no model or routes.
     */
    public function up(): void
    {
        if (! Schema::hasColumn('companions', 'role')) {
            Schema::table('companions', function (Blueprint $table) {
                $table->string('role', 16)->nullable();          // partner|other_parent|carer|friend|teen_parent
                $table->string('template', 24)->nullable();      // cycle_buddy|meds_appointments|full_family|custom|legacy
                $table->timestamp('expires_at')->nullable();
                $table->boolean('urgent_sms')->default(false);
                $table->timestamp('owner_confirmed_at')->nullable();

                $table->index(['companion_user_id', 'status', 'owner_id']);
                $table->index(['owner_id', 'companion_user_id', 'status']);
                $table->index(['status', 'expires_at']);
            });
            Schema::table('companions', function (Blueprint $table) {
                $table->dropIndex(['companion_user_id', 'status']);
            });
        }

        if (! Schema::hasColumn('companion_grants', 'mode_epoch')) {
            Schema::table('companion_grants', function (Blueprint $table) {
                $table->unsignedInteger('mode_epoch')->nullable();
            });
        }

        if (! Schema::hasTable('family_owner_state')) {
            Schema::create('family_owner_state', function (Blueprint $table) {
                $table->unsignedBigInteger('owner_id')->primary();
                $table->string('mode', 16);
                $table->unsignedInteger('mode_epoch')->default(1);
                $table->boolean('review_pending')->default(false);
                $table->timestamp('updated_at')->nullable();

                $table->foreign('owner_id')->references('id')->on('users')->cascadeOnDelete();
            });
        }

        if (! Schema::hasColumn('companion_audit_logs', 'subject_type')) {
            Schema::table('companion_audit_logs', function (Blueprint $table) {
                $table->string('subject_type', 16)->nullable();
                $table->unsignedBigInteger('subject_id')->nullable();
                $table->json('diff_json')->nullable();
            });
            // Blueprint has no MEDIUMBLOB (binary() is BLOB): sealed before/after payloads may exceed 64 KB.
            DB::statement('ALTER TABLE `companion_audit_logs` ADD COLUMN `payload_sealed` mediumblob NULL AFTER `diff_json`');
            Schema::table('companion_audit_logs', function (Blueprint $table) {
                $table->unsignedTinyInteger('payload_key_id')->nullable();
                $table->timestamp('undo_deadline')->nullable();
                $table->timestamp('undone_at')->nullable();
                $table->unsignedBigInteger('undone_by')->nullable();
                $table->unsignedBigInteger('proposal_id')->nullable();

                $table->index(['subject_type', 'subject_id', 'created_at']);
                $table->index('undo_deadline');
                $table->index('created_at');
            });
        }
    }

    /**
     * Reverse the migrations.
     */
    public function down(): void
    {
        if (Schema::hasColumn('companion_audit_logs', 'subject_type')) {
            Schema::table('companion_audit_logs', function (Blueprint $table) {
                $table->dropIndex(['created_at']);
                $table->dropIndex(['undo_deadline']);
                $table->dropIndex(['subject_type', 'subject_id', 'created_at']);
                $table->dropColumn([
                    'proposal_id', 'undone_by', 'undone_at', 'undo_deadline', 'payload_key_id', 'payload_sealed',
                    'diff_json', 'subject_id', 'subject_type',
                ]);
            });
        }

        Schema::dropIfExists('family_owner_state');

        if (Schema::hasColumn('companion_grants', 'mode_epoch')) {
            Schema::table('companion_grants', function (Blueprint $table) {
                $table->dropColumn('mode_epoch');
            });
        }

        if (Schema::hasColumn('companions', 'role')) {
            Schema::table('companions', function (Blueprint $table) {
                $table->index(['companion_user_id', 'status']);
            });
            Schema::table('companions', function (Blueprint $table) {
                $table->dropIndex(['status', 'expires_at']);
                $table->dropIndex(['owner_id', 'companion_user_id', 'status']);
                $table->dropIndex(['companion_user_id', 'status', 'owner_id']);
                $table->dropColumn(['owner_confirmed_at', 'urgent_sms', 'expires_at', 'template', 'role']);
            });
        }
    }
};
