<?php

use Illuminate\Database\Migrations\Migration;
use Illuminate\Database\Schema\Blueprint;
use Illuminate\Support\Facades\Schema;

return new class extends Migration
{
    /**
     * Run the migrations.
     *
     * Twin of backend-go/db/migrations/00011_notification_preferences.sql
     * (docs/go-migration/migrations.md): per-user notification settings
     * written by the Go-only /api/v1/profile/notification-settings endpoints
     * (B-N1-11). Schema only — no model or routes here.
     */
    public function up(): void
    {
        Schema::create('notification_preferences', function (Blueprint $table) {
            $table->id();
            $table->foreignId('user_id')->unique()->constrained()->cascadeOnDelete();
            $table->json('categories')->nullable();              // {"<category>": bool}, explicit choices only
            $table->boolean('quiet_hours_enabled')->default(true);
            $table->time('quiet_start')->default('23:00:00');
            $table->time('quiet_end')->default('08:00:00');
            $table->boolean('neutral_copy')->default(true);
            $table->timestamps();
        });
    }

    /**
     * Reverse the migrations.
     */
    public function down(): void
    {
        Schema::dropIfExists('notification_preferences');
    }
};
