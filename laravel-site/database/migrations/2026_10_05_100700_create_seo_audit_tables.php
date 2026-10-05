<?php

declare(strict_types=1);

use Illuminate\Database\Migrations\Migration;
use Illuminate\Database\Schema\Blueprint;
use Illuminate\Support\Facades\Schema;

/*
 * SEO audit results (L7-05): one row per run (status, bounds, totals, health score), the pages it audited (status,
 * timing, weight, analyser score, inbound links, content hash for incremental runs) and their findings (severity, code,
 * Persian message, admin fix link). Paths are stored decoded; `path_hash` (sha1) keeps the indexes short on MySQL.
 * Old runs are pruned by the store action (children cascade).
 */
return new class extends Migration
{
    public function up(): void
    {
        Schema::create('seo_audit_runs', function (Blueprint $table): void {
            $table->id();
            $table->string('status', 16)->default('queued')->index();
            $table->string('trigger', 16)->default('cli');
            $table->unsignedBigInteger('triggered_by')->nullable(); // admin user id, no FK (users may be deleted)
            $table->unsignedInteger('max_pages')->default(0);
            $table->unsignedInteger('time_limit')->default(0);
            $table->unsignedInteger('pages_count')->default(0);
            $table->unsignedInteger('errors_count')->default(0);
            $table->unsignedInteger('warnings_count')->default(0);
            $table->unsignedInteger('notices_count')->default(0);
            $table->unsignedTinyInteger('score')->nullable();
            $table->boolean('truncated')->default(false);
            $table->string('truncated_by', 16)->nullable();
            $table->unsignedInteger('duration_ms')->nullable();
            $table->string('message', 1000)->nullable();
            $table->timestamp('started_at')->nullable();
            $table->timestamp('finished_at')->nullable()->index();
            $table->timestamps();
        });

        Schema::create('seo_audit_pages', function (Blueprint $table): void {
            $table->id();
            $table->foreignId('run_id')->constrained('seo_audit_runs')->cascadeOnDelete();
            $table->string('path', 2048);
            $table->char('path_hash', 40);
            $table->string('title', 500)->nullable();
            $table->unsignedSmallInteger('status')->default(0);
            $table->string('source', 16)->default('route');
            $table->string('route_name', 191)->nullable();
            $table->string('content_type', 16)->nullable();
            $table->boolean('indexable')->default(false);
            $table->boolean('in_sitemap')->default(false);
            $table->unsignedInteger('response_ms')->default(0);
            $table->unsignedInteger('html_bytes')->default(0);
            $table->unsignedSmallInteger('requests_count')->default(0);
            $table->unsignedInteger('inbound_links')->default(0);
            $table->unsignedTinyInteger('analysis_score')->nullable();
            $table->unsignedTinyInteger('score')->default(100);
            $table->unsignedSmallInteger('errors_count')->default(0);
            $table->unsignedSmallInteger('warnings_count')->default(0);
            $table->unsignedSmallInteger('notices_count')->default(0);
            $table->char('content_hash', 40)->nullable();
            $table->string('edit_url', 2048)->nullable();

            $table->unique(['run_id', 'path_hash']);
            $table->index(['run_id', 'analysis_score']);
        });

        Schema::create('seo_audit_issues', function (Blueprint $table): void {
            $table->id();
            $table->foreignId('run_id')->constrained('seo_audit_runs')->cascadeOnDelete();
            $table->foreignId('page_id')->nullable()->constrained('seo_audit_pages')->cascadeOnDelete();
            $table->string('path', 2048);
            $table->string('severity', 16);
            $table->string('code', 64);
            $table->string('message', 1000);
            $table->string('fix_url', 2048)->nullable();

            $table->index(['run_id', 'severity']);
            $table->index(['run_id', 'code']);
        });
    }

    public function down(): void
    {
        Schema::dropIfExists('seo_audit_issues');
        Schema::dropIfExists('seo_audit_pages');
        Schema::dropIfExists('seo_audit_runs');
    }
};
