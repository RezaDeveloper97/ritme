<?php

declare(strict_types=1);

use Illuminate\Database\Migrations\Migration;
use Illuminate\Database\Schema\Blueprint;
use Illuminate\Support\Facades\Schema;

/*
 * Contact form messages (L3-10) — the admin inbox. Minimal data on purpose: name, one reply channel (email OR
 * phone), topic and the message; no IP, user agent or referrer (rate limiting keeps its own short-lived cache keys).
 * status: unread → read → archived (read_at = first time an admin opened it, kept when archived).
 */
return new class extends Migration
{
    public function up(): void
    {
        Schema::create('contact_messages', function (Blueprint $table): void {
            $table->id();
            $table->string('topic', 32);
            $table->string('name', 100);
            $table->string('email', 191)->nullable();
            $table->string('phone', 20)->nullable();
            $table->text('message');
            $table->string('status', 16)->default('unread');
            $table->timestamp('read_at')->nullable();
            $table->timestamps();

            $table->index(['status', 'created_at']);
        });
    }

    public function down(): void
    {
        Schema::dropIfExists('contact_messages');
    }
};
