<?php

use Illuminate\Database\Migrations\Migration;
use Illuminate\Database\Schema\Blueprint;
use Illuminate\Support\Facades\Schema;

return new class extends Migration
{
    /**
     * Run the migrations.
     *
     * Twin of backend-go/db/migrations/00013_cycle_settings.sql
     * (docs/go-migration/migrations.md): the reminder schedule on
     * notification_preferences and the per-user «خودکار از داده‌ها» flag,
     * written by the Go-only /api/v1/profile/cycle-settings endpoints
     * (B-N1-09). Schema only — no models or routes here.
     */
    public function up(): void
    {
        Schema::table('notification_preferences', function (Blueprint $table) {
            $table->json('schedule')->nullable()->after('categories'); // {"<category>": {days_before?, time}}
        });

        Schema::create('cycle_preferences', function (Blueprint $table) {
            $table->id();
            $table->foreignId('user_id')->unique()->constrained()->cascadeOnDelete();
            $table->boolean('lengths_auto')->default(true);
            $table->timestamps();
        });
    }

    /**
     * Reverse the migrations.
     */
    public function down(): void
    {
        Schema::dropIfExists('cycle_preferences');
        Schema::table('notification_preferences', function (Blueprint $table) {
            $table->dropColumn('schedule');
        });
    }
};
