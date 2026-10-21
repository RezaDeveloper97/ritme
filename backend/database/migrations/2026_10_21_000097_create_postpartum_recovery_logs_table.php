<?php

use Illuminate\Database\Migrations\Migration;
use Illuminate\Database\Schema\Blueprint;
use Illuminate\Support\Facades\Schema;

return new class extends Migration
{
    /**
     * Run the migrations.
     *
     * Twin of backend-go/db/migrations/00097_postpartum_recovery_logs.sql
     * (docs/go-migration/migrations.md): the postpartum red flags and the
     * postpartum visit log per day (T-M9-06, D-135), used only by the Go
     * internal/postpartum package. Schema only — no model or routes here.
     */
    public function up(): void
    {
        Schema::create('postpartum_recovery_logs', function (Blueprint $table) {
            $table->id();
            $table->foreignId('user_id')->constrained('users')->cascadeOnDelete();
            $table->date('log_date');
            $table->decimal('temperature_c', 3, 1)->nullable();
            $table->boolean('pad_soaked_within_hour')->nullable();
            $table->boolean('severe_pain')->nullable();
            $table->boolean('breathless')->nullable();
            $table->boolean('postpartum_visit')->nullable();
            $table->timestamps();

            $table->unique(['user_id', 'log_date']);
        });
    }

    /**
     * Reverse the migrations.
     */
    public function down(): void
    {
        Schema::dropIfExists('postpartum_recovery_logs');
    }
};
