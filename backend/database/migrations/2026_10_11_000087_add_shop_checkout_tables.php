<?php

use Illuminate\Database\Migrations\Migration;
use Illuminate\Database\Schema\Blueprint;
use Illuminate\Support\Facades\Schema;

return new class extends Migration
{
    /**
     * Twin of backend-go/db/migrations/00087_shop_checkout.sql (CB-SHOP-03): admin discount codes, the code applied to
     * a cart / used by an order, and the per-user checkout idempotency key. Used only by the Go shop order package.
     */
    public function up(): void
    {
        if (! Schema::hasTable('shop_discount_codes')) {
            Schema::create('shop_discount_codes', function (Blueprint $table) {
                $table->id();
                $table->string('code', 32)->unique();
                $table->string('kind', 10)->default('percent');
                $table->unsignedBigInteger('value');
                $table->unsignedBigInteger('max_discount')->nullable();
                $table->unsignedBigInteger('min_items_total')->default(0);
                $table->unsignedInteger('max_uses')->nullable();
                $table->unsignedInteger('used_count')->default(0);
                $table->unsignedSmallInteger('per_user_limit')->nullable()->default(1);
                $table->timestamp('starts_at')->nullable();
                $table->timestamp('ends_at')->nullable();
                $table->boolean('is_active')->default(true);
                $table->timestamps();
            });
        }

        if (! Schema::hasColumn('shop_carts', 'discount_code_id')) {
            Schema::table('shop_carts', function (Blueprint $table) {
                $table->foreignId('discount_code_id')->nullable()->constrained('shop_discount_codes')->nullOnDelete();
            });
        }

        if (! Schema::hasColumn('shop_orders', 'idempotency_key')) {
            Schema::table('shop_orders', function (Blueprint $table) {
                $table->foreignId('discount_code_id')->nullable()->constrained('shop_discount_codes')->nullOnDelete();
                $table->string('idempotency_key', 64)->nullable();

                $table->unique(['user_id', 'idempotency_key']);
            });
        }
    }

    public function down(): void
    {
        if (Schema::hasColumn('shop_orders', 'idempotency_key')) {
            Schema::table('shop_orders', function (Blueprint $table) {
                $table->dropForeign(['discount_code_id']);
                $table->dropUnique(['user_id', 'idempotency_key']);
                $table->dropColumn(['idempotency_key', 'discount_code_id']);
            });
        }
        if (Schema::hasColumn('shop_carts', 'discount_code_id')) {
            Schema::table('shop_carts', function (Blueprint $table) {
                $table->dropForeign(['discount_code_id']);
                $table->dropColumn('discount_code_id');
            });
        }
        Schema::dropIfExists('shop_discount_codes');
    }
};
