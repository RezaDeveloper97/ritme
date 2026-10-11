<?php

use Illuminate\Database\Migrations\Migration;
use Illuminate\Database\Schema\Blueprint;
use Illuminate\Support\Facades\DB;
use Illuminate\Support\Facades\Schema;

return new class extends Migration
{
    /**
     * Twin of backend-go/db/migrations/00071_shop_catalog.sql: admin-managed shop
     * categories and link-out products with daily click counters (bloom B-N10-03,
     * D-91). Used only by the Go internal/shop package. Never joins health data.
     */
    public function up(): void
    {
        Schema::create('shop_categories', function (Blueprint $table) {
            $table->id();
            $table->string('slug', 64)->unique();
            $table->json('title');
            $table->string('icon', 32)->nullable();
            $table->integer('sort_order')->default(0);
            $table->boolean('is_active')->default(true);
            $table->timestamps();

            $table->index(['is_active', 'sort_order']);
        });

        Schema::create('shop_products', function (Blueprint $table) {
            $table->id();
            $table->foreignId('category_id')->constrained('shop_categories');
            $table->json('title');
            $table->string('image_path', 255)->nullable();
            $table->string('price_text', 80)->nullable();
            $table->string('badge', 60)->nullable();
            $table->string('external_url', 1000)->nullable();
            $table->string('partner_label', 120)->nullable();
            $table->integer('sort_order')->default(0);
            $table->boolean('is_active')->default(true);
            $table->timestamps();

            $table->index(['category_id', 'is_active', 'sort_order']);
        });

        Schema::create('shop_click_counts', function (Blueprint $table) {
            $table->foreignId('product_id')->constrained('shop_products')->cascadeOnDelete();
            $table->date('day');
            $table->unsignedInteger('clicks')->default(0);

            $table->primary(['product_id', 'day']);
        });

        $now = now();
        DB::table('shop_categories')->insertOrIgnore([
            ['slug' => 'layette', 'title' => '{"fa":"سیسمونی و نوزاد","en":"Layette & baby"}', 'icon' => 'bottle',
                'sort_order' => 1, 'is_active' => 1, 'created_at' => $now, 'updated_at' => $now],
            ['slug' => 'beauty', 'title' => '{"fa":"آرایشی و بهداشتی","en":"Beauty & hygiene"}', 'icon' => 'sparkle',
                'sort_order' => 2, 'is_active' => 1, 'created_at' => $now, 'updated_at' => $now],
        ]);
    }

    public function down(): void
    {
        Schema::dropIfExists('shop_click_counts');
        Schema::dropIfExists('shop_products');
        Schema::dropIfExists('shop_categories');
    }
};
