<?php

use Illuminate\Database\Migrations\Migration;
use Illuminate\Database\Schema\Blueprint;
use Illuminate\Support\Facades\DB;
use Illuminate\Support\Facades\Schema;

// Twin of backend-go/db/migrations/00124_plus_store_billing.sql: Plus through Café Bazaar / Myket in-app billing
// (store SKU map + claimed purchase tokens) and minute-long developer test plans. Schema + the same seed rows.
return new class extends Migration
{
    public function up(): void
    {
        if (! Schema::hasColumn('plus_plans', 'duration_minutes')) {
            Schema::table('plus_plans', function (Blueprint $table) {
                $table->unsignedInteger('duration_minutes')->nullable()->after('duration_months');
            });
        }
        if (! Schema::hasColumn('plus_invoices', 'duration_minutes')) {
            Schema::table('plus_invoices', function (Blueprint $table) {
                $table->unsignedInteger('duration_minutes')->nullable()->after('duration_months');
            });
        }

        if (! Schema::hasTable('plus_store_products')) {
            Schema::create('plus_store_products', function (Blueprint $table) {
                $table->id();
                $table->string('store', 16);
                $table->string('sku', 100);
                $table->string('kind', 16);
                $table->foreignId('plan_id')->nullable()->constrained('plus_plans')->cascadeOnDelete();
                $table->timestamps();
                $table->unique(['store', 'sku']);
            });
        }

        if (! Schema::hasTable('plus_store_purchases')) {
            Schema::create('plus_store_purchases', function (Blueprint $table) {
                $table->id();
                $table->foreignId('user_id')->constrained()->cascadeOnDelete();
                $table->string('store', 16);
                $table->string('sku', 100);
                $table->string('purchase_token', 191);
                $table->string('order_id', 191)->nullable();
                $table->dateTime('purchase_time');
                $table->timestamps();
                $table->unique(['store', 'purchase_token']);
            });
        }

        $now = DB::raw("CONVERT_TZ(UTC_TIMESTAMP(), '+00:00', '+03:30')");
        DB::table('plus_plans')->insertOrIgnore([
            'code' => 'mtest', 'title' => '{"fa":"تست ۵ دقیقه‌ای","en":"5-minute test"}', 'badge' => null,
            'duration_months' => 1, 'duration_minutes' => 5, 'price_rials' => 20000, 'monthly_display_rials' => null,
            'is_highlighted' => 0, 'is_active' => 1, 'sort_order' => 4, 'created_at' => $now, 'updated_at' => $now,
        ]);
        DB::table('plus_store_products')->insertOrIgnore([
            'store' => 'bazaar', 'sku' => 'trial_subscription', 'kind' => 'trial', 'plan_id' => null,
            'created_at' => $now, 'updated_at' => $now,
        ]);
        $planId = DB::table('plus_plans')->where('code', 'mtest')->value('id');
        if ($planId) {
            DB::table('plus_store_products')->insertOrIgnore([
                'store' => 'bazaar', 'sku' => 'mtest', 'kind' => 'plan', 'plan_id' => $planId,
                'created_at' => $now, 'updated_at' => $now,
            ]);
        }
    }

    public function down(): void
    {
        Schema::dropIfExists('plus_store_purchases');
        Schema::dropIfExists('plus_store_products');
        DB::statement('DELETE FROM `plus_plans` WHERE `code` = \'mtest\' AND NOT EXISTS (SELECT 1 FROM `plus_invoices` WHERE `plan_id` = `plus_plans`.`id`)');
        foreach (['plus_invoices', 'plus_plans'] as $t) {
            if (Schema::hasColumn($t, 'duration_minutes')) {
                Schema::table($t, fn (Blueprint $table) => $table->dropColumn('duration_minutes'));
            }
        }
    }
};
