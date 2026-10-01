<?php

use Illuminate\Database\Migrations\Migration;
use Illuminate\Database\Schema\Blueprint;
use Illuminate\Support\Facades\Schema;

return new class extends Migration
{
    /**
     * Run the migrations.
     *
     * Twin of backend-go/db/migrations/00021_health_log_preferences.sql
     * (docs/go-migration/migrations.md): the log preferences of the
     * Go-only /api/v1/logs endpoints (B-N3-02) — per user and life-stage
     * mode the category order, hidden categories and pinned quick tiles
     * (JSON lists, NULL = default), and the user's custom log items
     * (soft-deleted so logged days keep their label). Schema only — no
     * models or routes here.
     */
    public function up(): void
    {
        Schema::create('health_log_custom_items', function (Blueprint $table) {
            $table->id();
            $table->foreignId('user_id')->constrained()->cascadeOnDelete();
            $table->string('category', 32);
            $table->string('param', 32);
            $table->string('label', 40);
            $table->timestamps();
            $table->softDeletes();

            $table->index(['user_id', 'deleted_at']);
        });

        Schema::create('health_log_preferences', function (Blueprint $table) {
            $table->id();
            $table->foreignId('user_id')->constrained()->cascadeOnDelete();
            $table->string('mode', 16);
            $table->json('category_order')->nullable();
            $table->json('hidden')->nullable();
            $table->json('pinned')->nullable();
            $table->timestamps();

            $table->unique(['user_id', 'mode']);
        });
    }

    /**
     * Reverse the migrations.
     */
    public function down(): void
    {
        Schema::dropIfExists('health_log_preferences');
        Schema::dropIfExists('health_log_custom_items');
    }
};
