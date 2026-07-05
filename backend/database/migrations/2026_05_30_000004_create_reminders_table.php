<?php

use Illuminate\Database\Migrations\Migration;
use Illuminate\Database\Schema\Blueprint;
use Illuminate\Support\Facades\Schema;

return new class extends Migration
{
    /**
     * Run the migrations.
     *
     * Unified reminders table powering both "یادآور دکتر" (doctor) and
     * "یادآور دارو" (medication) home sections, distinguished by `type`.
     */
    public function up(): void
    {
        Schema::create('reminders', function (Blueprint $table) {
            $table->id();
            $table->foreignId('user_id')->constrained()->cascadeOnDelete();
            $table->string('type');                          // App\Enums\ReminderType
            $table->string('title');                         // doctor name / medication name
            $table->string('subtitle')->nullable();          // specialty / dosage
            $table->text('notes')->nullable();

            // Scheduling: a one-off datetime and/or a recurring time-of-day.
            $table->dateTime('scheduled_at')->nullable();    // one-off (e.g. appointment)
            $table->string('recurrence')->default('none');   // none, daily, weekly, monthly
            $table->time('recurrence_time')->nullable();     // time-of-day for recurring reminders
            $table->date('starts_on')->nullable();
            $table->date('ends_on')->nullable();

            $table->boolean('is_active')->default(true);
            $table->json('meta')->nullable();                // extra type-specific data (phone, location...)
            $table->timestamps();

            $table->index(['user_id', 'type', 'is_active']);
            $table->index(['user_id', 'scheduled_at']);
        });
    }

    public function down(): void
    {
        Schema::dropIfExists('reminders');
    }
};
