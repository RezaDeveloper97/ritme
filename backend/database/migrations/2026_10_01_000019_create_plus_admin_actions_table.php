<?php

use Illuminate\Database\Migrations\Migration;
use Illuminate\Database\Schema\Blueprint;
use Illuminate\Support\Facades\Schema;

return new class extends Migration
{
    /**
     * Run the migrations.
     *
     * Twin of backend-go/db/migrations/00019_plus_admin_actions.sql
     * (docs/go-migration/migrations.md): the Ritme Plus admin ledger of the
     * Go-only admin «اشتراک‌ها و پرداخت» module (B-N2-09) — plan, discount
     * and settings changes, refunds and subscription extensions with the
     * admin's note. No foreign keys: the money trail outlives deleted admin
     * accounts. Schema only — no models or routes here.
     */
    public function up(): void
    {
        Schema::create('plus_admin_actions', function (Blueprint $table) {
            $table->id();
            $table->unsignedBigInteger('admin_id')->nullable();
            $table->string('action', 48);
            $table->string('target_type', 32);
            $table->unsignedBigInteger('target_id')->default(0);
            $table->unsignedBigInteger('user_id')->nullable();
            $table->unsignedBigInteger('amount_rials')->nullable();
            $table->unsignedInteger('days')->nullable();
            $table->string('gateway', 32)->nullable();
            $table->string('gateway_ref', 191)->nullable();
            $table->string('note', 500)->nullable();
            $table->json('details')->nullable();
            $table->timestamps();

            $table->index(['target_type', 'target_id']);
            $table->index('user_id');
            $table->index('created_at');
        });
    }

    /**
     * Reverse the migrations.
     */
    public function down(): void
    {
        Schema::dropIfExists('plus_admin_actions');
    }
};
