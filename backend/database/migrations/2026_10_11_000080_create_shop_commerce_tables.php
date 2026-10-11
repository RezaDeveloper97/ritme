<?php

use Illuminate\Database\Migrations\Migration;
use Illuminate\Database\Schema\Blueprint;
use Illuminate\Support\Facades\Schema;

return new class extends Migration
{
    /**
     * Twin of backend-go/db/migrations/00080_shop_commerce.sql: the internal shop with
     * pay-on-delivery orders (CB-SHOP-01, DECISIONS #13/#15) on top of bloom's link-out
     * catalog (00071). Extends shop_categories/shop_products additively and adds sellers,
     * variants, reviews, carts, addresses, delivery slots, orders/shipments/items/events,
     * returns, the layette checklist and the period-supply reminder opt-in.
     * Used only by the Go internal/shop package. Never joins health data. Money = toman.
     */
    public function up(): void
    {
        Schema::create('shop_sellers', function (Blueprint $table) {
            $table->id();
            $table->string('slug', 64)->unique();
            $table->json('name');
            $table->string('logo_path', 255)->nullable();
            $table->boolean('is_verified')->default(false);
            $table->unsignedTinyInteger('ship_min_days')->default(1);
            $table->unsignedTinyInteger('ship_max_days')->default(3);
            $table->boolean('discreet_packaging')->default(false);
            $table->unsignedBigInteger('shipping_fee')->default(0);
            $table->unsignedBigInteger('free_shipping_over')->nullable();
            $table->unsignedBigInteger('cod_max_total')->nullable();
            $table->unsignedTinyInteger('return_days')->default(7);
            $table->integer('sort_order')->default(0);
            $table->boolean('is_active')->default(true);
            $table->timestamps();

            $table->index(['is_active', 'sort_order']);
        });

        Schema::table('shop_categories', function (Blueprint $table) {
            $table->foreignId('parent_id')->nullable()->constrained('shop_categories')->nullOnDelete();
        });

        Schema::table('shop_products', function (Blueprint $table) {
            $table->foreignId('seller_id')->nullable()->constrained('shop_sellers')->nullOnDelete();
            $table->json('subtitle')->nullable();
            $table->json('description')->nullable();
            $table->json('images')->nullable();
            $table->json('attributes')->nullable();
            $table->string('irc_code', 40)->nullable();
            $table->date('expires_on')->nullable();
            $table->json('skin_types')->nullable();
            $table->json('badges')->nullable();
            $table->boolean('discreet_packaging')->default(false);
            $table->decimal('rating_avg', 3, 2)->nullable();
            $table->unsignedInteger('rating_count')->default(0);
            $table->unsignedInteger('purchase_count')->default(0);
        });

        Schema::create('shop_product_variants', function (Blueprint $table) {
            $table->id();
            $table->foreignId('product_id')->constrained('shop_products')->cascadeOnDelete();
            $table->string('sku', 64)->nullable()->unique();
            $table->json('size')->nullable();
            $table->json('color')->nullable();
            $table->json('volume')->nullable();
            $table->unsignedBigInteger('price');
            $table->unsignedBigInteger('compare_at_price')->nullable();
            $table->unsignedInteger('stock')->default(0);
            $table->integer('sort_order')->default(0);
            $table->boolean('is_active')->default(true);
            $table->timestamps();

            $table->index(['product_id', 'is_active', 'sort_order']);
        });

        Schema::create('shop_carts', function (Blueprint $table) {
            $table->id();
            $table->foreignId('user_id')->unique()->constrained()->cascadeOnDelete();
            $table->timestamps();
        });

        Schema::create('shop_cart_items', function (Blueprint $table) {
            $table->id();
            $table->foreignId('cart_id')->constrained('shop_carts')->cascadeOnDelete();
            $table->foreignId('variant_id')->constrained('shop_product_variants')->cascadeOnDelete();
            $table->unsignedSmallInteger('quantity')->default(1);
            $table->timestamps();

            $table->unique(['cart_id', 'variant_id']);
        });

        Schema::create('shop_addresses', function (Blueprint $table) {
            $table->id();
            $table->foreignId('user_id')->constrained()->cascadeOnDelete();
            $table->string('label', 40)->nullable();
            $table->string('recipient_name', 120);
            $table->string('recipient_phone', 20);
            $table->string('province', 60)->nullable();
            $table->string('city', 60);
            $table->string('address_line', 500);
            $table->string('postal_code', 10)->nullable();
            $table->boolean('is_default')->default(false);
            $table->timestamps();
        });

        Schema::create('shop_delivery_slots', function (Blueprint $table) {
            $table->id();
            $table->foreignId('seller_id')->constrained('shop_sellers')->cascadeOnDelete();
            $table->date('slot_date');
            $table->time('starts_at');
            $table->time('ends_at');
            $table->unsignedSmallInteger('capacity')->nullable();
            $table->unsignedSmallInteger('booked')->default(0);
            $table->boolean('is_active')->default(true);
            $table->timestamps();

            $table->unique(['seller_id', 'slot_date', 'starts_at']);
        });

        Schema::create('shop_orders', function (Blueprint $table) {
            $table->id();
            $table->string('code', 16)->unique();
            $table->foreignId('user_id')->constrained()->cascadeOnDelete();
            $table->foreignId('address_id')->nullable()->constrained('shop_addresses')->nullOnDelete();
            $table->string('recipient_name', 120);
            $table->string('recipient_phone', 20);
            $table->string('city', 60);
            $table->string('address_text', 600);
            $table->string('postal_code', 10)->nullable();
            $table->string('payment_method', 16)->default('cod');
            $table->string('payment_status', 16)->default('unpaid');
            $table->string('status', 20)->default('placed');
            $table->boolean('discreet')->default(false);
            $table->unsignedBigInteger('items_total');
            $table->unsignedBigInteger('shipping_total')->default(0);
            $table->unsignedBigInteger('discount_total')->default(0);
            $table->unsignedBigInteger('payable_total');
            $table->timestamp('placed_at')->nullable();
            $table->timestamp('cancelled_at')->nullable();
            $table->timestamps();

            $table->index(['user_id', 'created_at']);
            $table->index(['status', 'created_at']);
        });

        Schema::create('shop_order_shipments', function (Blueprint $table) {
            $table->id();
            $table->foreignId('order_id')->constrained('shop_orders')->cascadeOnDelete();
            $table->foreignId('seller_id')->constrained('shop_sellers');
            $table->foreignId('delivery_slot_id')->nullable()->constrained('shop_delivery_slots')->nullOnDelete();
            $table->date('slot_date')->nullable();
            $table->time('slot_starts_at')->nullable();
            $table->time('slot_ends_at')->nullable();
            $table->string('status', 20)->default('awaiting_seller');
            $table->unsignedBigInteger('items_total');
            $table->unsignedBigInteger('shipping_fee')->default(0);
            $table->string('tracking_code', 64)->nullable();
            $table->timestamp('delivered_at')->nullable();
            $table->timestamps();

            $table->index(['seller_id', 'status']);
        });

        Schema::create('shop_order_items', function (Blueprint $table) {
            $table->id();
            $table->foreignId('order_id')->constrained('shop_orders')->cascadeOnDelete();
            $table->foreignId('shipment_id')->constrained('shop_order_shipments')->cascadeOnDelete();
            $table->foreignId('product_id')->nullable()->constrained('shop_products')->nullOnDelete();
            $table->foreignId('variant_id')->nullable()->constrained('shop_product_variants')->nullOnDelete();
            $table->json('title');
            $table->json('variant_label')->nullable();
            $table->unsignedBigInteger('unit_price');
            $table->unsignedSmallInteger('quantity');
            $table->unsignedBigInteger('line_total');
            $table->timestamps();
        });

        Schema::create('shop_order_events', function (Blueprint $table) {
            $table->id();
            $table->foreignId('order_id')->constrained('shop_orders')->cascadeOnDelete();
            $table->foreignId('shipment_id')->nullable()->constrained('shop_order_shipments')->cascadeOnDelete();
            $table->string('status', 20);
            $table->string('note', 255)->nullable();
            $table->timestamp('created_at')->nullable();

            $table->index(['order_id', 'created_at']);
        });

        Schema::create('shop_returns', function (Blueprint $table) {
            $table->id();
            $table->foreignId('order_id')->constrained('shop_orders')->cascadeOnDelete();
            $table->foreignId('order_item_id')->constrained('shop_order_items')->cascadeOnDelete();
            $table->foreignId('user_id')->constrained()->cascadeOnDelete();
            $table->unsignedSmallInteger('quantity')->default(1);
            $table->string('reason', 32);
            $table->string('note', 1000)->nullable();
            $table->string('status', 16)->default('requested');
            $table->timestamps();

            $table->index(['status', 'created_at']);
        });

        Schema::create('shop_product_reviews', function (Blueprint $table) {
            $table->id();
            $table->foreignId('product_id')->constrained('shop_products')->cascadeOnDelete();
            $table->foreignId('user_id')->constrained()->cascadeOnDelete();
            $table->foreignId('order_item_id')->nullable()->unique()->constrained('shop_order_items')->nullOnDelete();
            $table->string('author_name', 60);
            $table->json('variant_label')->nullable();
            $table->unsignedTinyInteger('rating');
            $table->string('body', 2000)->nullable();
            $table->string('status', 16)->default('pending');
            $table->timestamps();

            $table->index(['product_id', 'status', 'created_at']);
        });

        Schema::create('shop_layette_items', function (Blueprint $table) {
            $table->id();
            $table->string('group_key', 32);
            $table->json('title');
            $table->json('note')->nullable();
            $table->unsignedSmallInteger('suggested_qty')->nullable();
            $table->boolean('is_essential')->default(false);
            $table->boolean('is_first_days')->default(false);
            $table->foreignId('category_id')->nullable()->constrained('shop_categories')->nullOnDelete();
            $table->integer('sort_order')->default(0);
            $table->boolean('is_active')->default(true);
            $table->timestamps();

            $table->index(['is_active', 'group_key', 'sort_order']);
        });

        Schema::create('shop_user_checklist_items', function (Blueprint $table) {
            $table->id();
            $table->foreignId('user_id')->constrained()->cascadeOnDelete();
            $table->foreignId('layette_item_id')->nullable()->constrained('shop_layette_items')->cascadeOnDelete();
            $table->string('custom_title', 120)->nullable();
            $table->string('group_key', 32);
            $table->string('state', 16)->default('todo');
            $table->unsignedSmallInteger('quantity')->nullable();
            $table->integer('sort_order')->default(0);
            $table->timestamps();

            $table->unique(['user_id', 'layette_item_id']);
        });

        Schema::create('shop_supply_reminders', function (Blueprint $table) {
            $table->foreignId('user_id')->primary()->constrained()->cascadeOnDelete();
            $table->boolean('is_enabled')->default(false);
            $table->unsignedTinyInteger('lead_days')->default(3);
            $table->foreignId('product_id')->nullable()->constrained('shop_products')->nullOnDelete();
            $table->timestamps();
        });
    }

    public function down(): void
    {
        Schema::dropIfExists('shop_supply_reminders');
        Schema::dropIfExists('shop_user_checklist_items');
        Schema::dropIfExists('shop_layette_items');
        Schema::dropIfExists('shop_product_reviews');
        Schema::dropIfExists('shop_returns');
        Schema::dropIfExists('shop_order_events');
        Schema::dropIfExists('shop_order_items');
        Schema::dropIfExists('shop_order_shipments');
        Schema::dropIfExists('shop_orders');
        Schema::dropIfExists('shop_delivery_slots');
        Schema::dropIfExists('shop_addresses');
        Schema::dropIfExists('shop_cart_items');
        Schema::dropIfExists('shop_carts');
        Schema::dropIfExists('shop_product_variants');
        Schema::table('shop_products', function (Blueprint $table) {
            $table->dropConstrainedForeignId('seller_id');
            $table->dropColumn(['subtitle', 'description', 'images', 'attributes', 'irc_code', 'expires_on',
                'skin_types', 'badges', 'discreet_packaging', 'rating_avg', 'rating_count', 'purchase_count']);
        });
        Schema::table('shop_categories', function (Blueprint $table) {
            $table->dropConstrainedForeignId('parent_id');
        });
        Schema::dropIfExists('shop_sellers');
    }
};
