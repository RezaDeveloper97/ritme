<?php

use Illuminate\Database\Migrations\Migration;
use Illuminate\Database\Schema\Blueprint;
use Illuminate\Support\Facades\Schema;

return new class extends Migration
{
    /**
     * Run the migrations.
     *
     * Schema-only twin of backend-go/db/migrations/00002_reminder_intakes.sql
     * (docs/go-migration/migrations.md): one row per medication dose taken,
     * written by the Go-only /api/v1/care endpoints (T-M3-01). No model here.
     */
    public function up(): void
    {
        Schema::create('reminder_intakes', function (Blueprint $table) {
            $table->id();
            $table->foreignId('user_id')->constrained()->cascadeOnDelete();
            $table->foreignId('reminder_id')->constrained()->cascadeOnDelete();
            $table->date('intake_date');
            $table->char('slot', 5);                         // 'HH:MM', one of the medication's times
            $table->dateTime('taken_at');
            $table->timestamps();

            $table->unique(['reminder_id', 'intake_date', 'slot']);
            $table->index(['user_id', 'intake_date']);
        });
    }

    public function down(): void
    {
        Schema::dropIfExists('reminder_intakes');
    }
};
