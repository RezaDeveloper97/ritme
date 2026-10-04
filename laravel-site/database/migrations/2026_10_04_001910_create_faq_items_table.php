<?php

declare(strict_types=1);

use Illuminate\Database\Migrations\Migration;
use Illuminate\Database\Schema\Blueprint;
use Illuminate\Support\Facades\Schema;

/*
 * FAQ questions (L3-09). `answer` is sanitised rich text (FaqObserver, RichHtmlSanitizer allow-list); only published
 * items are rendered and emitted as FAQPage JSON-LD.
 */
return new class extends Migration
{
    public function up(): void
    {
        Schema::create('faq_items', function (Blueprint $table): void {
            $table->id();
            $table->foreignId('faq_group_id')->constrained('faq_groups')->cascadeOnDelete();
            $table->string('question', 255);
            $table->text('answer');
            $table->boolean('is_published')->default(true);
            $table->unsignedSmallInteger('sort_order')->default(0);
            $table->timestamps();

            $table->index(['faq_group_id', 'is_published', 'sort_order']);
        });
    }

    public function down(): void
    {
        Schema::dropIfExists('faq_items');
    }
};
