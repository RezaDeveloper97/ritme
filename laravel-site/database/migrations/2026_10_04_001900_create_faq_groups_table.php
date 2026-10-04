<?php

declare(strict_types=1);

use Illuminate\Database\Migrations\Migration;
use Illuminate\Database\Schema\Blueprint;
use Illuminate\Support\Facades\Schema;

/*
 * FAQ groups (L3-09). `slug` is the stable key pages ask for (`home`, `plus`, `stage-cycle` …) and the in-page
 * anchor on /faq; `is_listed` puts the group on /faq (the contextual groups — home, plus, contact, stages — are not).
 */
return new class extends Migration
{
    public function up(): void
    {
        Schema::create('faq_groups', function (Blueprint $table): void {
            $table->id();
            $table->string('slug', 64)->unique();
            $table->string('title', 191);
            $table->boolean('is_listed')->default(false);
            $table->unsignedSmallInteger('sort_order')->default(0);
            $table->timestamps();

            $table->index(['is_listed', 'sort_order']);
        });
    }

    public function down(): void
    {
        Schema::dropIfExists('faq_groups');
    }
};
