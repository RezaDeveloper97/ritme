<?php

use Illuminate\Database\Migrations\Migration;
use Illuminate\Database\Schema\Blueprint;
use Illuminate\Support\Facades\Schema;

return new class extends Migration
{
    /**
     * Twin of backend-go/db/migrations/00091_shop_pii_enc.sql (user decision 2026-10-12): the sealed recipient /
     * address blob of shop_addresses and of the shop_orders snapshot (PRIVATE_NOTE_KEY, written and read by the Go
     * internal/shop/order package only). No model or routes.
     */
    public function up(): void
    {
        foreach (['shop_addresses', 'shop_orders'] as $name) {
            if (! Schema::hasColumn($name, 'pii_enc')) {
                Schema::table($name, function (Blueprint $table) {
                    $table->longText('pii_enc')->nullable()->after('postal_code');
                });
            }
        }
    }

    public function down(): void
    {
        foreach (['shop_orders', 'shop_addresses'] as $name) {
            if (Schema::hasColumn($name, 'pii_enc')) {
                Schema::table($name, function (Blueprint $table) {
                    $table->dropColumn('pii_enc');
                });
            }
        }
    }
};
