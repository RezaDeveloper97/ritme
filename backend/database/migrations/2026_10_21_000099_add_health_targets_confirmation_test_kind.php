<?php

use Illuminate\Database\Migrations\Migration;
use Illuminate\Database\Schema\Blueprint;
use Illuminate\Support\Facades\DB;
use Illuminate\Support\Facades\Schema;

return new class extends Migration
{
    /**
     * Twin of backend-go/db/migrations/00099_health_targets_confirmation_test_kind.sql (docs/go-migration/migrations.md,
     * T-M10-17): per-user diabetic glucose targets, the per-value lab confirmation (backfilled from the report's
     * verified_at), the cervical test kind of a checkup record and the informational flag of a checkup type. Used only
     * by the Go backend. Schema only — no model, routes or behaviour here.
     */
    public function up(): void
    {
        if (! Schema::hasTable('vital_targets')) {
            Schema::create('vital_targets', function (Blueprint $table) {
                $table->id();
                $table->foreignId('user_id')->constrained('users')->cascadeOnDelete();
                $table->string('context', 16);
                $table->unsignedSmallInteger('low_mgdl')->nullable();
                $table->unsignedSmallInteger('high_mgdl')->nullable();
                $table->string('set_by', 8);
                $table->string('note', 160)->nullable();
                $table->timestamps();

                $table->unique(['user_id', 'context']);
            });
        }

        if (! Schema::hasColumn('lab_markers', 'confirmed_at')) {
            Schema::table('lab_markers', function (Blueprint $table) {
                $table->timestamp('confirmed_at')->nullable();
            });
        }
        // Same rows as goose's UPDATE … JOIN, spelled with correlated subqueries so sqlite (phpunit) runs it too.
        DB::statement('UPDATE lab_markers
            SET confirmed_at = (SELECT r.verified_at FROM lab_reports r WHERE r.id = lab_markers.lab_id)
            WHERE confirmed_at IS NULL
              AND EXISTS (SELECT 1 FROM lab_reports r WHERE r.id = lab_markers.lab_id AND r.verified_at IS NOT NULL)');

        if (! Schema::hasColumn('checkup_records', 'test_kind')) {
            Schema::table('checkup_records', function (Blueprint $table) {
                $table->string('test_kind', 12)->nullable();
            });
        }

        if (! Schema::hasColumn('checkup_types', 'informational')) {
            Schema::table('checkup_types', function (Blueprint $table) {
                $table->boolean('informational')->default(false);
            });
        }
    }

    /**
     * Reverse the migrations.
     */
    public function down(): void
    {
        if (Schema::hasColumn('checkup_types', 'informational')) {
            Schema::table('checkup_types', function (Blueprint $table) {
                $table->dropColumn('informational');
            });
        }
        if (Schema::hasColumn('checkup_records', 'test_kind')) {
            Schema::table('checkup_records', function (Blueprint $table) {
                $table->dropColumn('test_kind');
            });
        }
        if (Schema::hasColumn('lab_markers', 'confirmed_at')) {
            Schema::table('lab_markers', function (Blueprint $table) {
                $table->dropColumn('confirmed_at');
            });
        }

        Schema::dropIfExists('vital_targets');
    }
};
