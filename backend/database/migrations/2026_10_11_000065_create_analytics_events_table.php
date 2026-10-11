<?php

use Illuminate\Database\Migrations\Migration;
use Illuminate\Database\Schema\Blueprint;
use Illuminate\Support\Facades\Schema;

return new class extends Migration
{
    /**
     * Twin of backend-go/db/migrations/00065_analytics_events.sql: first-party,
     * allow-listed, non-health product analytics events (bloom B-N9-08, D-89).
     * Used only by the Go internal/analyticsevents + internal/metrics packages.
     */
    public function up(): void
    {
        Schema::create('analytics_events', function (Blueprint $table) {
            $table->id();
            $table->foreignId('user_id')->constrained()->cascadeOnDelete();
            $table->string('name', 48);
            $table->json('props')->nullable();
            $table->date('occurred_on');
            $table->dateTime('occurred_at');
            $table->timestamp('created_at')->nullable();

            $table->index(['occurred_on', 'user_id']);
            $table->index(['name', 'occurred_on']);
            $table->index(['user_id', 'name', 'occurred_on']);
        });
    }

    public function down(): void
    {
        Schema::dropIfExists('analytics_events');
    }
};
