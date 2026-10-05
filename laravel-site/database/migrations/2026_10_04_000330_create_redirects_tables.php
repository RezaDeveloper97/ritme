<?php

declare(strict_types=1);

use Illuminate\Database\Migrations\Migration;
use Illuminate\Database\Schema\Blueprint;
use Illuminate\Support\Facades\Schema;

/*
 * Redirect manager + 404 monitor (L7-03). Paths are stored decoded (Persian readable) and looked up by a sha1 of the
 * lower-cased normalised path (`from_hash` / `path_hash`), so long UTF-8 paths never need a long unique index (MySQL).
 * `to_hash` is the same hash of a local target path: it lets a new redirect re-point every row that pointed at its
 * source (chain collapse A→B→C ⇒ A→C). Hit counters are written in batches by the scheduler, never per request.
 * The 404 log holds no personal data: path, a query-less referer, the agent class (bot / human) and counts.
 */
return new class extends Migration
{
    public function up(): void
    {
        Schema::create('redirects', function (Blueprint $table): void {
            $table->id();
            $table->string('from_path', 700);
            $table->char('from_hash', 40);
            $table->string('to_url', 2048)->nullable();
            $table->char('to_hash', 40)->nullable()->index();
            $table->unsignedSmallInteger('code')->default(301);
            $table->boolean('is_regex')->default(false);
            $table->boolean('is_auto')->default(false);
            $table->unsignedInteger('hits')->default(0);
            $table->timestamp('last_hit_at')->nullable();
            $table->string('note', 255)->nullable();
            $table->timestamps();

            $table->unique(['from_hash', 'is_regex']);
        });

        Schema::create('not_found_logs', function (Blueprint $table): void {
            $table->id();
            $table->string('path', 700);
            $table->char('path_hash', 40)->unique();
            $table->string('referer', 500)->nullable();
            $table->string('agent', 8)->default('human');
            $table->unsignedInteger('hits')->default(0);
            $table->timestamp('first_seen_at')->nullable();
            $table->timestamp('last_seen_at')->nullable()->index();
        });
    }

    public function down(): void
    {
        Schema::dropIfExists('not_found_logs');
        Schema::dropIfExists('redirects');
    }
};
