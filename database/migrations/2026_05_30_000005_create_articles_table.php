<?php

use Illuminate\Database\Migrations\Migration;
use Illuminate\Database\Schema\Blueprint;
use Illuminate\Support\Facades\Schema;

return new class extends Migration
{
    /**
     * Run the migrations.
     *
     * Educational articles surfaced by "بر اساس سیکل فعلی شما" and tagged by
     * cycle phase. Bilingual fields follow the {fa, en} JSON convention.
     */
    public function up(): void
    {
        Schema::create('articles', function (Blueprint $table) {
            $table->id();
            $table->string('slug')->unique();
            $table->json('title');                       // {fa, en}
            $table->json('excerpt')->nullable();         // {fa, en}
            $table->json('body')->nullable();            // {fa, en}
            $table->string('cycle_phase')->nullable();   // null = general; else App\Enums\CyclePhase
            $table->string('category')->nullable();
            $table->unsignedSmallInteger('read_time_minutes')->nullable();
            $table->string('image_url')->nullable();
            $table->boolean('is_published')->default(true);
            $table->timestamp('published_at')->nullable();
            $table->integer('sort_order')->default(0);
            $table->timestamps();

            $table->index(['is_published', 'cycle_phase']);
        });
    }

    public function down(): void
    {
        Schema::dropIfExists('articles');
    }
};
