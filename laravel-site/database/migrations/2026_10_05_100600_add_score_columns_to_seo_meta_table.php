<?php

declare(strict_types=1);

use Illuminate\Database\Migrations\Migration;
use Illuminate\Database\Schema\Blueprint;
use Illuminate\Support\Facades\Schema;

/*
 * L7-06: the content analyser's last score (0–100, null = not checked yet) and the cornerstone flag, so admin lists can
 * show an SEO score column and a «نیاز به کار» filter without re-analysing every page on each render.
 */
return new class extends Migration
{
    public function up(): void
    {
        Schema::table('seo_meta', function (Blueprint $table): void {
            $table->unsignedTinyInteger('score')->nullable()->after('focus_keyword');
            $table->timestamp('score_checked_at')->nullable()->after('score');
            $table->boolean('cornerstone')->default(false)->after('score_checked_at');
            $table->index('score');
        });
    }

    public function down(): void
    {
        Schema::table('seo_meta', function (Blueprint $table): void {
            $table->dropIndex(['score']);
            $table->dropColumn(['score', 'score_checked_at', 'cornerstone']);
        });
    }
};
