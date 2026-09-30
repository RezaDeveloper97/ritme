<?php

use Illuminate\Database\Migrations\Migration;
use Illuminate\Database\Schema\Blueprint;
use Illuminate\Support\Facades\Schema;

return new class extends Migration
{
    /**
     * Run the migrations.
     *
     * Schema-only twin of backend-go/db/migrations/00009_catalog_items.sql
     * (docs/go-migration/migrations.md): the admin-editable content catalog
     * (group / code / translatable title+body / audiences / meta) served by the
     * Go-only /api/v1/catalog/{group} and /api/admin/v1/catalog/* endpoints
     * (CB-CORE-03, docs/canvas-build/catalog.md). No model here.
     */
    public function up(): void
    {
        Schema::create('catalog_items', function (Blueprint $table) {
            $table->id();
            $table->string('group', 64);
            $table->string('code', 64);
            $table->integer('sort_order')->default(0);
            $table->boolean('is_active')->default(true);
            $table->json('audiences')->nullable();   // mode/audience codes; null = everyone
            $table->json('title');                   // {"fa": …, "en": …}
            $table->json('body')->nullable();
            $table->json('meta')->nullable();        // free-form per group
            $table->boolean('needs_review')->default(true);
            $table->timestamps();

            $table->unique(['group', 'code']);
            $table->index(['group', 'is_active', 'sort_order']);
        });
    }

    public function down(): void
    {
        Schema::dropIfExists('catalog_items');
    }
};
