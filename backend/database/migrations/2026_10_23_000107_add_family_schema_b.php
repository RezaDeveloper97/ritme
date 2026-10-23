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
     * Twin of backend-go/db/migrations/00107_family_schema_b.sql (docs/go-migration/migrations.md, T-M11-03, D-191):
     * family schema v6 part B — family_proposals (one pending row per dedupe key via the STORED pending_key),
     * family_share_requests, family_nudges, family_member_alerts, family_member_notices, family_attendance, and
     * user_notifications.link_id (FK companions, cascade). Used only by the Go backend: schema only, no model or routes.
     */
    public function up(): void
    {
        if (! Schema::hasTable('family_proposals')) {
            Schema::create('family_proposals', function (Blueprint $table) {
                $table->id();
                $table->string('subject_type', 8);                 // owner|child
                $table->unsignedBigInteger('subject_id');
                $table->unsignedBigInteger('link_id')->nullable();
                $table->unsignedBigInteger('actor_id');
                $table->unsignedBigInteger('approver_id');
                $table->string('kind', 24);                        // create_med|create_appointment|update_appointment|child_delete|guardian_remove
                $table->string('domain', 24);
            });
            // Blueprint has no MEDIUMBLOB (binary() is BLOB): sealed proposed records may exceed 64 KB.
            DB::statement('ALTER TABLE `family_proposals` ADD COLUMN `payload_sealed` mediumblob NULL');
            Schema::table('family_proposals', function (Blueprint $table) {
                $table->unsignedTinyInteger('payload_key_id')->nullable();
                $table->string('status', 12)->default('pending'); // pending|approved|rejected|expired|cancelled
                $table->string('dedupe_key', 80)->nullable();
                $table->string('pending_key', 80)->storedAs("IF(`status` = 'pending', `dedupe_key`, NULL)");
                $table->timestamp('expires_at');
                $table->unsignedBigInteger('decided_by')->nullable();
                $table->timestamp('decided_at')->nullable();
                $table->string('applied_ref', 64)->nullable();
                $table->timestamps();

                $table->unique('pending_key');
                $table->index(['approver_id', 'status', 'created_at']);
                $table->index(['actor_id', 'status', 'created_at']);
                $table->index(['status', 'expires_at']);
                $table->foreign('link_id')->references('id')->on('companions')->cascadeOnDelete();
                $table->foreign('actor_id')->references('id')->on('users')->cascadeOnDelete();
                $table->foreign('approver_id')->references('id')->on('users')->cascadeOnDelete();
            });
        }

        if (! Schema::hasTable('family_share_requests')) {
            Schema::create('family_share_requests', function (Blueprint $table) {
                $table->id();
                $table->unsignedBigInteger('from_user_id');
                $table->unsignedBigInteger('to_user_id');
                $table->unsignedBigInteger('link_id')->nullable();
                $table->json('domains');
                $table->string('status', 10)->default('pending'); // pending|accepted|declined|expired
                $table->timestamp('expires_at');
                $table->timestamp('decided_at')->nullable();
                $table->timestamp('created_at')->nullable();

                $table->index(['to_user_id', 'status', 'created_at']);
                $table->index(['from_user_id', 'to_user_id', 'created_at']);
                $table->foreign('from_user_id')->references('id')->on('users')->cascadeOnDelete();
                $table->foreign('to_user_id')->references('id')->on('users')->cascadeOnDelete();
                $table->foreign('link_id')->references('id')->on('companions')->cascadeOnDelete();
            });
        }

        if (! Schema::hasTable('family_nudges')) {
            Schema::create('family_nudges', function (Blueprint $table) {
                $table->id();
                $table->unsignedBigInteger('owner_id');
                $table->unsignedBigInteger('member_id');
                $table->unsignedBigInteger('link_id');
                $table->string('target_type', 12);                 // dose|appointment
                $table->unsignedBigInteger('target_id');
                $table->date('local_date');
                $table->string('slot', 5)->default('');           // dose HH:MM; '' for an appointment
                $table->timestamp('sent_at');

                $table->unique(['owner_id', 'target_type', 'target_id', 'local_date', 'slot'], 'family_nudges_target_slot_unique');
                $table->index(['member_id', 'owner_id', 'sent_at']);
                $table->index(['owner_id', 'sent_at']);
                $table->foreign('owner_id')->references('id')->on('users')->cascadeOnDelete();
                $table->foreign('member_id')->references('id')->on('users')->cascadeOnDelete();
                $table->foreign('link_id')->references('id')->on('companions')->cascadeOnDelete();
            });
        }

        if (! Schema::hasTable('family_member_alerts')) {
            Schema::create('family_member_alerts', function (Blueprint $table) {
                $table->id();
                $table->unsignedBigInteger('member_id');
                $table->unsignedBigInteger('owner_id');
                $table->unsignedBigInteger('link_id');
                $table->unsignedBigInteger('source_alert_id');
                $table->unsignedInteger('source_hit');
                $table->string('level', 12);
                $table->timestamp('created_at')->nullable();
                $table->timestamp('seen_at')->nullable();
                $table->timestamp('sms_sent_at')->nullable();

                $table->unique(['member_id', 'source_alert_id', 'source_hit']);
                $table->index(['member_id', 'created_at']);
                $table->index('link_id');
                $table->foreign('member_id')->references('id')->on('users')->cascadeOnDelete();
                $table->foreign('owner_id')->references('id')->on('users')->cascadeOnDelete();
                $table->foreign('link_id')->references('id')->on('companions')->cascadeOnDelete();
            });
        }

        if (! Schema::hasTable('family_member_notices')) {
            Schema::create('family_member_notices', function (Blueprint $table) {
                $table->id();
                $table->unsignedBigInteger('member_id');
                $table->unsignedBigInteger('owner_id');
                $table->unsignedBigInteger('link_id');
                $table->string('kind', 24);
                $table->string('dedupe_key', 80);
                $table->timestamp('due_at');
                $table->timestamp('sent_at')->nullable();
                $table->string('deferred_reason', 24)->nullable();
                $table->unsignedBigInteger('notification_id')->nullable();
                $table->timestamp('created_at')->nullable();

                $table->unique(['member_id', 'dedupe_key']);
                $table->index(['sent_at', 'due_at']);
                $table->index('link_id');
                $table->foreign('member_id')->references('id')->on('users')->cascadeOnDelete();
                $table->foreign('owner_id')->references('id')->on('users')->cascadeOnDelete();
                $table->foreign('link_id')->references('id')->on('companions')->cascadeOnDelete();
            });
        }

        if (! Schema::hasTable('family_attendance')) {
            Schema::create('family_attendance', function (Blueprint $table) {
                $table->id();
                $table->unsignedBigInteger('reminder_id');
                $table->unsignedBigInteger('member_id');
                $table->timestamp('created_at')->nullable();

                $table->unique(['reminder_id', 'member_id']);
                $table->foreign('reminder_id')->references('id')->on('reminders')->cascadeOnDelete();
                $table->foreign('member_id')->references('id')->on('users')->cascadeOnDelete();
            });
        }

        if (! Schema::hasColumn('user_notifications', 'link_id')) {
            Schema::table('user_notifications', function (Blueprint $table) {
                $table->unsignedBigInteger('link_id')->nullable();

                $table->index('link_id');
                $table->foreign('link_id')->references('id')->on('companions')->cascadeOnDelete();
            });
        }
    }

    /**
     * Reverse the migrations.
     */
    public function down(): void
    {
        if (Schema::hasColumn('user_notifications', 'link_id')) {
            Schema::table('user_notifications', function (Blueprint $table) {
                $table->dropForeign(['link_id']);
                $table->dropIndex(['link_id']);
                $table->dropColumn('link_id');
            });
        }

        Schema::dropIfExists('family_attendance');
        Schema::dropIfExists('family_member_notices');
        Schema::dropIfExists('family_member_alerts');
        Schema::dropIfExists('family_nudges');
        Schema::dropIfExists('family_share_requests');
        Schema::dropIfExists('family_proposals');
    }
};
