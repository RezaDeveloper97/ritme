<?php

use Illuminate\Database\Migrations\Migration;
use Illuminate\Database\Schema\Blueprint;
use Illuminate\Support\Facades\Schema;

return new class extends Migration
{
    /**
     * Twin of backend-go/db/migrations/00093_insurance_policy_history.sql (user decisions 2026-10-12): a user keeps
     * several insurance policies over time (renewal creates a new one, the previous becomes read-only), so the
     * one-per-user unique key is replaced by a (user_id, valid_from) index; questionnaire evidence copied from the
     * health record is sealed in `evidence_enc` (PRIVATE_NOTE_KEY). Used only by the Go internal/insurance package.
     */
    public function up(): void
    {
        Schema::table('insurance_policies', function (Blueprint $table) {
            $table->index(['user_id', 'valid_from']);
            $table->dropUnique('insurance_policies_user_id_unique');
        });

        if (! Schema::hasColumn('insurance_questionnaires', 'evidence_enc')) {
            Schema::table('insurance_questionnaires', function (Blueprint $table) {
                $table->text('evidence_enc')->nullable()->after('answers');
            });
        }
    }

    public function down(): void
    {
        if (Schema::hasColumn('insurance_questionnaires', 'evidence_enc')) {
            Schema::table('insurance_questionnaires', function (Blueprint $table) {
                $table->dropColumn('evidence_enc');
            });
        }

        // Fails while a user has more than one policy (the history must be removed first).
        Schema::table('insurance_policies', function (Blueprint $table) {
            $table->unique('user_id');
            $table->dropIndex(['user_id', 'valid_from']);
        });
    }
};
