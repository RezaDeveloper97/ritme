<?php

declare(strict_types=1);

use Illuminate\Database\Migrations\Migration;
use Illuminate\Database\Schema\Blueprint;
use Illuminate\Support\Facades\Schema;

/*
 * Shop orders (L6-05): cash on delivery only (PaymentGateway contract), single seller.
 *
 * shop_orders — `code` is the unguessable public code of /shop/order/{code}; `idempotency_key` (sha256 of the
 * checkout form token) makes a double submit return the first order instead of a second one. Minimal personal data:
 * recipient name, mobile, province (key) + city, address, postal code and an optional note — no email, IP or user
 * agent. Delivery: a preferred day + time window (a preference the shop confirms by phone) and the «بسته‌بندی ساده»
 * (discreet packaging) flag. Money columns are integer RIALS; `shipping_fee` null = not known at checkout (agreed on
 * the confirmation call). Status: pending|confirmed|shipped|delivered|cancelled (admin: L6-06); payment_status
 * unpaid|paid (COD → unpaid until delivery).
 *
 * shop_order_items — snapshots of what was bought (title, variant label, unit price) so later catalog edits never
 * change an order; product / variant ids are kept (nullable) for stock returns and sales counts.
 */
return new class extends Migration
{
    public function up(): void
    {
        Schema::create('shop_orders', function (Blueprint $table): void {
            $table->id();
            $table->string('code', 20)->unique();
            $table->string('idempotency_key', 64)->unique();
            $table->string('status', 16)->default('pending');

            $table->string('payment_method', 32);
            $table->string('payment_status', 16)->default('unpaid');

            $table->unsignedBigInteger('subtotal');
            $table->unsignedBigInteger('shipping_fee')->nullable();
            $table->unsignedBigInteger('total');
            $table->unsignedSmallInteger('items_count');

            $table->string('recipient_name', 100);
            $table->string('mobile', 20);
            $table->string('province', 40);
            $table->string('city', 60);
            $table->string('address', 500);
            $table->string('postal_code', 10)->nullable();
            $table->string('note', 500)->nullable();

            $table->date('delivery_date');
            $table->string('delivery_window', 16);
            $table->boolean('discreet_packaging')->default(true);
            $table->boolean('is_demo')->default(false);

            $table->timestamps();

            $table->index(['status', 'created_at']);
            $table->index('mobile');
        });

        Schema::create('shop_order_items', function (Blueprint $table): void {
            $table->id();
            $table->foreignId('order_id')->constrained('shop_orders')->cascadeOnDelete();
            $table->foreignId('product_id')->nullable()->constrained('shop_products')->nullOnDelete();
            $table->foreignId('variant_id')->nullable()->constrained('shop_product_variants')->nullOnDelete();
            $table->string('title', 191);
            $table->string('variant_label', 120)->nullable();
            $table->unsignedBigInteger('unit_price');
            $table->unsignedSmallInteger('quantity');
            $table->unsignedBigInteger('line_total');
            $table->unsignedBigInteger('cover_media_id')->nullable(); // no FK: a deleted image only loses the thumbnail
            $table->string('illustration', 60)->nullable();
            $table->boolean('stock_tracked')->default(true);
            $table->timestamps();

            $table->index(['product_id', 'order_id']);
        });
    }

    public function down(): void
    {
        Schema::dropIfExists('shop_order_items');
        Schema::dropIfExists('shop_orders');
    }
};
