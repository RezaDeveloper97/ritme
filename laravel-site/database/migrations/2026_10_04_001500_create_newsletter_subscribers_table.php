<?php

declare(strict_types=1);

use Illuminate\Database\Migrations\Migration;
use Illuminate\Database\Schema\Blueprint;
use Illuminate\Support\Facades\Schema;

/*
 * Newsletter subscribers (L4-02), double opt-in. Minimal data on purpose: no name, IP or user agent.
 * pending = confirmed_at null; active = confirmed and not unsubscribed. `token` (random, rotated on every new
 * sign-up) authorises the confirm and unsubscribe links.
 */
return new class extends Migration
{
    public function up(): void
    {
        Schema::create('newsletter_subscribers', function (Blueprint $table): void {
            $table->id();
            $table->string('email', 191)->unique();
            $table->string('source', 100)->nullable();       // page the form was sent from (blog, blog/category/cycle …)
            $table->string('token', 64)->unique();
            $table->timestamp('consent_at');                 // form submitted with the consent wording shown
            $table->timestamp('confirmation_sent_at')->nullable();
            $table->timestamp('confirmed_at')->nullable()->index();
            $table->timestamp('unsubscribed_at')->nullable();
            $table->timestamps();
        });
    }

    public function down(): void
    {
        Schema::dropIfExists('newsletter_subscribers');
    }
};
