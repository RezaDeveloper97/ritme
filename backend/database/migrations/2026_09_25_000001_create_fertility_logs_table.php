<?php

use Illuminate\Database\Migrations\Migration;
use Illuminate\Database\Schema\Blueprint;
use Illuminate\Support\Facades\Schema;

return new class extends Migration
{
    /**
     * Run the migrations.
     *
     * Schema-only twin of backend-go/db/migrations/00004_fertility_logs.sql
     * (docs/go-migration/migrations.md): per-day LH test, cervical mucus and
     * BBT time, written by the Go-only /api/v1/fertility endpoints (T-M5-01).
     * No model here.
     */
    public function up(): void
    {
        Schema::create('fertility_logs', function (Blueprint $table) {
            $table->id();
            $table->foreignId('user_id')->constrained()->cascadeOnDelete();
            $table->date('log_date');
            $table->string('lh_test', 16)->nullable();         // negative|faint|positive
            $table->string('cervical_mucus', 16)->nullable();  // dry|sticky|creamy|egg_white
            $table->time('bbt_time')->nullable();
            $table->timestamps();

            $table->unique(['user_id', 'log_date']);
        });
    }

    public function down(): void
    {
        Schema::dropIfExists('fertility_logs');
    }
};
