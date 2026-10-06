<?php

use Illuminate\Database\Migrations\Migration;
use Illuminate\Database\Schema\Blueprint;
use Illuminate\Support\Facades\Schema;

return new class extends Migration
{
    /**
     * Run the migrations.
     *
     * Twin of backend-go/db/migrations/00042_health_share_links.sql
     * (docs/go-migration/migrations.md): the doctor report's 7-day share
     * links (bloom B-N6-04). The report snapshot is stored encrypted with a
     * key derived from the link token; only the token's SHA-256 is kept.
     * Used only by the Go internal/sharelinks package. No model or routes.
     */
    public function up(): void
    {
        Schema::create('health_share_links', function (Blueprint $table) {
            $table->id();
            $table->foreignId('user_id')->constrained()->cascadeOnDelete();
            $table->char('token_hash', 64)->unique();      // sha256 hex of the token
            $table->mediumText('payload')->nullable();     // v1:base64(nonce‖ciphertext‖tag); NULL once revoked/expired
            $table->json('sections');                      // included section keys
            $table->date('range_from');
            $table->date('range_to');
            $table->timestamp('expires_at')->nullable();
            $table->timestamp('revoked_at')->nullable();
            $table->unsignedInteger('view_count')->default(0);
            $table->timestamp('last_viewed_at')->nullable();
            $table->timestamps();

            $table->index(['user_id', 'created_at']);
            $table->index('expires_at');
        });
    }

    /**
     * Reverse the migrations.
     */
    public function down(): void
    {
        Schema::dropIfExists('health_share_links');
    }
};
