<?php

use Illuminate\Database\Migrations\Migration;
use Illuminate\Database\Schema\Blueprint;
use Illuminate\Support\Facades\Schema;

return new class extends Migration
{
    /**
     * Run the migrations.
     *
     * Twin of backend-go/db/migrations/00063_clinical_config.sql
     * (docs/go-migration/migrations.md): versioned clinical thresholds with
     * two medical approvals (bloom B-N9-11, D-88). Used only by the Go
     * internal/clinicalconfig + internal/admin/clinicalcfg packages. No model
     * or routes.
     */
    public function up(): void
    {
        Schema::create('clinical_config_versions', function (Blueprint $table) {
            $table->id();
            $table->string('config_key', 64);
            $table->unsignedInteger('version');
            $table->text('value');
            $table->string('source', 255)->default('');
            $table->string('note', 500)->default('');
            $table->date('effective_from');
            $table->string('status', 16)->default('pending');
            $table->unsignedBigInteger('proposed_by')->nullable();
            $table->unsignedBigInteger('decided_by')->nullable();
            $table->timestamp('decided_at')->nullable();
            $table->timestamps();

            $table->unique(['config_key', 'version']);
            $table->index(['status', 'config_key']);
        });

        Schema::create('clinical_config_approvals', function (Blueprint $table) {
            $table->id();
            $table->unsignedBigInteger('version_id');
            $table->unsignedBigInteger('admin_id');
            $table->timestamp('created_at')->nullable();

            $table->unique(['version_id', 'admin_id']);
        });
    }

    /**
     * Reverse the migrations.
     */
    public function down(): void
    {
        Schema::dropIfExists('clinical_config_approvals');
        Schema::dropIfExists('clinical_config_versions');
    }
};
