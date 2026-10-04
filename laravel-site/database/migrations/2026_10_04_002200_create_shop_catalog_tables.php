<?php

declare(strict_types=1);

use Illuminate\Database\Migrations\Migration;
use Illuminate\Database\Schema\Blueprint;
use Illuminate\Support\Facades\Schema;

/*
 * Shop catalog tables (L6-01) in one migration — they are created and dropped together. Money columns are integer
 * RIALS (App\Support\Money\Money). Works on MySQL/MariaDB and SQLite (Persian slugs: 191-char unique strings).
 *
 * Shop brands (L6-01). Single-seller shop: Ritme sells everything, product cards show the brand (no seller model —
 * a `seller_id` can be added to shop_products later without touching brands).
 *
 * Shop categories (L6-01): a tree (`parent_id`; roots are the two departments «سیسمونی و نوزاد», «آرایشی و بهداشتی»).
 * `intro` is plain text for the listing page, `illustration` names a local SVG (resources/svg/illustrations) used as
 * the cover until an image is uploaded. Persian slugs allowed (191 chars for utf8mb4 unique indexes).
 *
 * Shop products (L6-01). MONEY IS STORED AS INTEGER RIALS (`price`, `compare_at_price`; App\Support\Money\Money,
 * shown in tomans = rial / 10). `specs` is a JSON list of {label, value}; `size_chart` is JSON {columns: [...],
 * rows: [[...], ...]}; `life_stages` a JSON list of LifeStage values. `stock_qty`/`stock_status` are the product's own
 * stock, or — when the product has variants — the sum over its active variants (maintained by the variant observer).
 * `rating_avg`/`rating_count` cache approved, non-demo reviews; `sales_count` is fed by orders (L6-05). `is_demo`
 * marks design sample products (excluded from ratings and sitemaps). Single seller: no seller column yet.
 *
 * Slug history of products (L6-01): previous slugs of published products, so old URLs 301 to the current one (L6-03).
 *
 * Product ↔ category (L6-01, many-to-many). The primary category (breadcrumbs, canonical listing) is
 * shop_products.primary_category_id and is always also present here. Written through SyncProductCategories.
 *
 * Ordered gallery of a product (L6-01), besides shop_products.cover_media_id. Written through SyncProductGallery.
 * Has its own `id` because FindMediaUsages reads `id` from every registered media column's table.
 *
 * Simple product variants (L6-01): one size and/or colour with its own sku, optional own price (integer rials; null =
 * the product price) and stock. `color_hex` is swatch DATA (#RRGGBB) rendered by the product page, not a CSS token.
 *
 * Product reviews (L6-01), moderated: pending|approved|rejected. `variant_label` is the bought option shown with the
 * review («سایز ۳-۶ ماه»). `is_verified_purchase` is set once orders exist (L6-05). `is_demo` marks seeded design
 * samples: shown labelled, never counted towards the rating. `ip_hash` is a salted hash, never the IP.
 *
 * Hand-picked «معمولاً با این می‌خرند» products (L6-01). FrequentlyBoughtWith lists these first and tops up with best
 * sellers of the same category; co-purchase data from orders (L6-05) can replace the top-up later.
 */
return new class extends Migration
{
    public function up(): void
    {
        Schema::create('shop_brands', function (Blueprint $table): void {
            $table->id();
            $table->string('name', 120);
            $table->string('slug', 191)->unique();
            $table->text('description')->nullable();
            $table->foreignId('logo_media_id')->nullable()->constrained('media')->nullOnDelete();
            $table->unsignedSmallInteger('sort_order')->default(0);
            $table->boolean('is_active')->default(true);
            $table->timestamps();
        });

        Schema::create('shop_categories', function (Blueprint $table): void {
            $table->id();
            $table->foreignId('parent_id')->nullable()->constrained('shop_categories')->nullOnDelete();
            $table->string('name', 120);
            $table->string('slug', 191)->unique();
            $table->text('intro')->nullable();
            $table->foreignId('cover_media_id')->nullable()->constrained('media')->nullOnDelete();
            $table->string('illustration', 60)->nullable();
            $table->unsignedSmallInteger('sort_order')->default(0);
            $table->boolean('is_active')->default(true);
            $table->timestamps();

            $table->index(['parent_id', 'sort_order']);
        });

        Schema::create('shop_products', function (Blueprint $table): void {
            $table->id();
            $table->string('title', 191);
            $table->string('slug', 191)->unique();
            $table->string('sku', 64)->nullable()->unique();
            $table->foreignId('brand_id')->nullable()->constrained('shop_brands')->nullOnDelete();
            $table->foreignId('primary_category_id')->nullable()->constrained('shop_categories')->nullOnDelete();

            $table->string('short_description', 500)->nullable();
            $table->longText('description')->nullable();
            $table->json('specs')->nullable();
            $table->json('size_chart')->nullable();
            $table->string('badge', 40)->nullable();

            $table->unsignedBigInteger('price');
            $table->unsignedBigInteger('compare_at_price')->nullable();
            $table->unsignedInteger('stock_qty')->default(0);
            $table->string('stock_status', 16)->default('out_of_stock');
            $table->unsignedInteger('weight_grams')->nullable();

            $table->foreignId('cover_media_id')->nullable()->constrained('media')->nullOnDelete();
            $table->string('illustration', 60)->nullable();
            $table->json('life_stages')->nullable();

            $table->decimal('rating_avg', 3, 2)->default(0);
            $table->unsignedInteger('rating_count')->default(0);
            $table->unsignedInteger('sales_count')->default(0);

            $table->boolean('is_published')->default(false);
            $table->boolean('is_featured')->default(false);
            $table->boolean('is_demo')->default(false);
            $table->unsignedInteger('sort_order')->default(0);
            $table->timestamps();

            $table->index(['is_published', 'sales_count']);
            $table->index(['is_published', 'price']);
            $table->index(['is_published', 'created_at']);
            $table->index(['is_published', 'rating_avg']);
        });

        Schema::create('shop_product_slugs', function (Blueprint $table): void {
            $table->id();
            $table->foreignId('product_id')->constrained('shop_products')->cascadeOnDelete();
            $table->string('slug', 191)->unique();
            $table->timestamp('created_at')->nullable();
        });

        Schema::create('shop_category_product', function (Blueprint $table): void {
            $table->id();
            $table->foreignId('product_id')->constrained('shop_products')->cascadeOnDelete();
            $table->foreignId('category_id')->constrained('shop_categories')->cascadeOnDelete();

            $table->unique(['product_id', 'category_id']);
            $table->index('category_id');
        });

        Schema::create('shop_product_media', function (Blueprint $table): void {
            $table->id();
            $table->foreignId('product_id')->constrained('shop_products')->cascadeOnDelete();
            $table->foreignId('media_id')->constrained('media')->cascadeOnDelete();
            $table->unsignedSmallInteger('sort_order')->default(0);

            $table->unique(['product_id', 'media_id']);
            $table->index('media_id');
        });

        Schema::create('shop_product_variants', function (Blueprint $table): void {
            $table->id();
            $table->foreignId('product_id')->constrained('shop_products')->cascadeOnDelete();
            $table->string('sku', 64)->nullable()->unique();
            $table->string('size', 40)->nullable();
            $table->string('color', 40)->nullable();
            $table->string('color_hex', 7)->nullable();
            $table->unsignedBigInteger('price')->nullable();
            $table->unsignedBigInteger('compare_at_price')->nullable();
            $table->unsignedInteger('stock_qty')->default(0);
            $table->unsignedSmallInteger('sort_order')->default(0);
            $table->boolean('is_active')->default(true);
            $table->timestamps();

            $table->unique(['product_id', 'size', 'color']);
            $table->index(['product_id', 'is_active', 'sort_order']);
        });

        Schema::create('shop_reviews', function (Blueprint $table): void {
            $table->id();
            $table->foreignId('product_id')->constrained('shop_products')->cascadeOnDelete();
            $table->string('author_name', 80);
            $table->unsignedTinyInteger('rating');
            $table->text('body');
            $table->string('variant_label', 80)->nullable();
            $table->string('status', 16)->default('pending');
            $table->boolean('is_verified_purchase')->default(false);
            $table->boolean('is_demo')->default(false);
            $table->string('ip_hash', 64)->nullable();
            $table->timestamp('approved_at')->nullable();
            $table->timestamps();

            $table->index(['product_id', 'status', 'approved_at']);
            $table->index(['status', 'created_at']);
        });

        Schema::create('shop_product_cross_sells', function (Blueprint $table): void {
            $table->id();
            $table->foreignId('product_id')->constrained('shop_products')->cascadeOnDelete();
            $table->foreignId('related_id')->constrained('shop_products')->cascadeOnDelete();
            $table->unsignedSmallInteger('sort_order')->default(0);

            $table->unique(['product_id', 'related_id']);
        });
    }

    public function down(): void
    {
        Schema::dropIfExists('shop_product_cross_sells');
        Schema::dropIfExists('shop_reviews');
        Schema::dropIfExists('shop_product_variants');
        Schema::dropIfExists('shop_product_media');
        Schema::dropIfExists('shop_category_product');
        Schema::dropIfExists('shop_product_slugs');
        Schema::dropIfExists('shop_products');
        Schema::dropIfExists('shop_categories');
        Schema::dropIfExists('shop_brands');
    }
};
