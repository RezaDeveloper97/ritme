<?php

declare(strict_types=1);

use Illuminate\Database\Migrations\Migration;
use Illuminate\Database\Schema\Blueprint;
use Illuminate\Support\Facades\Schema;

/*
 * Magazine categories (L4-01), tree-light: one optional parent. `label` is the short stage label on article cards
 * («چرخه»), `life_stage` links a category to a stage page.
 */
return new class extends Migration
{
    public function up(): void
    {
        Schema::create('blog_categories', function (Blueprint $table): void {
            $table->id();
            $table->foreignId('parent_id')->nullable()->constrained('blog_categories')->nullOnDelete();
            $table->string('name', 191);
            $table->string('label', 64)->nullable();
            $table->string('slug', 191)->unique();
            $table->text('description')->nullable();
            $table->string('life_stage', 16)->nullable();
            $table->unsignedSmallInteger('sort_order')->default(0);
            $table->timestamps();

            $table->index(['parent_id', 'sort_order']);
        });
    }

    public function down(): void
    {
        Schema::dropIfExists('blog_categories');
    }
};
