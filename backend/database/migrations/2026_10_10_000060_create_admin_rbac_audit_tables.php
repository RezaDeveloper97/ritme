<?php

use Illuminate\Database\Migrations\Migration;
use Illuminate\Database\Schema\Blueprint;
use Illuminate\Support\Facades\Schema;

return new class extends Migration
{
    /**
     * Run the migrations.
     *
     * Twin of backend-go/db/migrations/00060_admin_rbac_audit.sql
     * (docs/go-migration/migrations.md): admin roles & permissions, the DB
     * audit log and temporary health-data access (bloom B-N9-02, D-84). Used
     * only by the Go internal/admin packages. No model or routes.
     */
    public function up(): void
    {
        Schema::create('admin_roles', function (Blueprint $table) {
            $table->id();
            $table->string('role', 32)->unique();
            $table->text('permissions');
            $table->unsignedBigInteger('updated_by')->nullable();
            $table->timestamps();
        });

        Schema::create('admin_audit_logs', function (Blueprint $table) {
            $table->id();
            $table->unsignedBigInteger('admin_id')->nullable();
            $table->string('action', 64);
            $table->string('target_type', 64);
            $table->unsignedBigInteger('target_id')->default(0);
            $table->string('ip', 45)->default('');
            $table->text('detail')->nullable();
            $table->timestamp('created_at')->nullable();

            $table->index('created_at');
            $table->index(['admin_id', 'created_at']);
            $table->index(['target_type', 'target_id']);
            $table->index('action');
        });

        Schema::create('admin_health_access_requests', function (Blueprint $table) {
            $table->id();
            $table->unsignedBigInteger('admin_id');
            $table->unsignedBigInteger('user_id');
            $table->string('reason', 500);
            $table->string('ticket', 64);
            $table->string('status', 16)->default('pending'); // pending | approved | rejected | revoked
            $table->unsignedBigInteger('decided_by')->nullable();
            $table->timestamp('decided_at')->nullable();
            $table->timestamp('expires_at')->nullable();
            $table->timestamps();

            $table->index(['admin_id', 'user_id', 'status']);
            $table->index(['status', 'created_at']);
        });
    }

    /**
     * Reverse the migrations.
     */
    public function down(): void
    {
        Schema::dropIfExists('admin_health_access_requests');
        Schema::dropIfExists('admin_audit_logs');
        Schema::dropIfExists('admin_roles');
    }
};
