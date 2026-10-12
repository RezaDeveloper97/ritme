<?php

use Illuminate\Database\Migrations\Migration;
use Illuminate\Database\Schema\Blueprint;
use Illuminate\Support\Facades\Schema;

return new class extends Migration
{
    /**
     * Twin of backend-go/db/migrations/00090_share_include_losses.sql (user decision 2026-10-12, CB-REC-06 M2):
     * the per-share loss opt-in of a doctor report link / 24h doctor code (off by default). Used only by the Go
     * internal/sharelinks package. No model or routes.
     */
    public function up(): void
    {
        if (! Schema::hasColumn('health_share_links', 'include_losses')) {
            Schema::table('health_share_links', function (Blueprint $table) {
                $table->boolean('include_losses')->default(false)->after('sections');
            });
        }
    }

    public function down(): void
    {
        if (Schema::hasColumn('health_share_links', 'include_losses')) {
            Schema::table('health_share_links', function (Blueprint $table) {
                $table->dropColumn('include_losses');
            });
        }
    }
};
