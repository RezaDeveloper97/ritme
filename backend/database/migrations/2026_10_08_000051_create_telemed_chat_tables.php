<?php

use Illuminate\Database\Migrations\Migration;
use Illuminate\Database\Schema\Blueprint;
use Illuminate\Support\Facades\Schema;

return new class extends Migration
{
    /**
     * Run the migrations.
     *
     * Twin of backend-go/db/migrations/00051_telemed_chat.sql
     * (docs/go-migration/migrations.md): doctor chat (body encrypted by Go),
     * e-prescription lines and their care reminder link (bloom B-N7-04),
     * used only by the Go internal/telemed package. No model or routes here.
     */
    public function up(): void
    {
        Schema::create('telemed_messages', function (Blueprint $table) {
            $table->id();
            $table->foreignId('booking_id')->constrained('telemed_bookings')->cascadeOnDelete();
            $table->string('sender', 8);                         // user|doctor
            $table->foreignId('admin_id')->nullable()->constrained('admins')->nullOnDelete();
            $table->text('body');                                // AES-256-GCM ciphertext (PRIVATE_NOTE_KEY)
            $table->foreignId('file_id')->nullable()->constrained('files')->nullOnDelete();
            $table->dateTime('read_at')->nullable();
            $table->timestamps();

            $table->index(['booking_id', 'id']);
        });

        Schema::create('telemed_prescription_items', function (Blueprint $table) {
            $table->id();
            $table->foreignId('booking_id')->constrained('telemed_bookings')->cascadeOnDelete();
            $table->string('drug', 120);
            $table->string('dose', 50)->nullable();
            $table->string('form', 16);                          // care medication form
            $table->string('times', 32);                         // "08:00,20:00"
            $table->unsignedSmallInteger('duration_days')->nullable();
            $table->string('instructions', 500)->nullable();
            $table->foreignId('reminder_id')->nullable()->constrained('reminders')->nullOnDelete();
            $table->dateTime('added_at')->nullable();
            $table->timestamps();
        });
    }

    /**
     * Reverse the migrations.
     */
    public function down(): void
    {
        Schema::dropIfExists('telemed_prescription_items');
        Schema::dropIfExists('telemed_messages');
    }
};
