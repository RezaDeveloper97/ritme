<?php

use Illuminate\Database\Migrations\Migration;
use Illuminate\Database\Schema\Blueprint;
use Illuminate\Support\Facades\Schema;

return new class extends Migration
{
    /**
     * Run the migrations.
     *
     * Twin of backend-go/db/migrations/00032_ai_platform.sql
     * (docs/go-migration/migrations.md): the version of the accepted consent
     * text on `user_consents` and the per-call AI usage + cost log written by
     * the Go AI adapter platform (B-N6-05). Schema only — no models or routes.
     * The log never holds prompts, answers, files or other content.
     */
    public function up(): void
    {
        Schema::table('user_consents', function (Blueprint $table) {
            $table->unsignedSmallInteger('version')->nullable()->after('granted');
        });

        Schema::create('ai_usage_logs', function (Blueprint $table) {
            $table->id();
            $table->foreignId('user_id')->nullable();
            $table->string('feature', 32);
            $table->string('op', 32);
            $table->string('provider', 16);
            $table->string('model', 64);
            $table->unsignedInteger('input_tokens')->default(0);
            $table->unsignedInteger('output_tokens')->default(0);
            $table->unsignedInteger('audio_bytes')->default(0);
            $table->unsignedInteger('image_bytes')->default(0);
            $table->unsignedBigInteger('cost_micros')->default(0);
            $table->unsignedInteger('latency_ms')->default(0);
            $table->boolean('ok')->default(false);
            $table->timestamp('created_at')->nullable();

            $table->index('created_at');
            $table->index(['user_id', 'created_at']);
            $table->index(['feature', 'created_at']);
            $table->foreign('user_id')->references('id')->on('users')->nullOnDelete();
        });
    }

    /**
     * Reverse the migrations.
     */
    public function down(): void
    {
        Schema::dropIfExists('ai_usage_logs');
        Schema::table('user_consents', function (Blueprint $table) {
            $table->dropColumn('version');
        });
    }
};
