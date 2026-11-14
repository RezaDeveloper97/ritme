<?php

use Illuminate\Database\Migrations\Migration;
use Illuminate\Database\Schema\Blueprint;
use Illuminate\Support\Facades\Schema;

// Schema-only twin of backend-go/db/migrations/00121_push_subscriptions.sql: Web Push subscriptions and the
// medication doses already pushed. Written by the Go-only /api/v1/push and care reminder sender. No model here.
return new class extends Migration
{
    public function up(): void
    {
        Schema::create('push_subscriptions', function (Blueprint $table) {
            $table->id();
            $table->foreignId('user_id')->constrained()->cascadeOnDelete();
            $table->text('endpoint');
            $table->char('endpoint_hash', 64)->unique();     // sha256 hex of the endpoint
            $table->string('p256dh');
            $table->string('auth');
            $table->string('locale', 5)->default('fa');
            $table->timestamps();
        });

        Schema::create('reminder_pushes', function (Blueprint $table) {
            $table->id();
            $table->foreignId('user_id')->constrained()->cascadeOnDelete();
            $table->foreignId('reminder_id')->constrained()->cascadeOnDelete();
            $table->date('intake_date');
            $table->char('slot', 5);
            $table->dateTime('sent_at');

            $table->unique(['reminder_id', 'intake_date', 'slot']);
        });
    }

    public function down(): void
    {
        Schema::dropIfExists('reminder_pushes');
        Schema::dropIfExists('push_subscriptions');
    }
};
