<?php

use Illuminate\Database\Migrations\Migration;
use Illuminate\Database\Schema\Blueprint;
use Illuminate\Support\Facades\Schema;

// Schema-only twin of backend-go/db/migrations/00122_reminder_sms.sql: the reminder occurrences already sent by SMS
// to Ritme Plus users. Written by the Go-only reminder SMS sender. No model here.
return new class extends Migration
{
    public function up(): void
    {
        Schema::create('reminder_sms', function (Blueprint $table) {
            $table->id();
            $table->foreignId('user_id')->constrained()->cascadeOnDelete();
            $table->foreignId('reminder_id')->constrained()->cascadeOnDelete();
            $table->string('occurrence', 32);
            $table->boolean('sent')->default(false);
            $table->dateTime('sent_at');

            $table->unique(['reminder_id', 'occurrence']);
            $table->index(['user_id', 'sent_at']);
        });
    }

    public function down(): void
    {
        Schema::dropIfExists('reminder_sms');
    }
};
