<?php

use Illuminate\Database\Migrations\Migration;
use Illuminate\Database\Schema\Blueprint;
use Illuminate\Support\Facades\Schema;

/**
 * Editable storage for the privacy-policy screen ("حریم خصوصی") that used to be
 * hardcoded in the frontend's `profileInfo.privacy.sections` messages.
 *
 * One row = one card ("box") on that screen: a heading and a body, both
 * bilingual. Admins add, reorder, disable and delete boxes freely; the app
 * renders whatever is active, in `sort_order`.
 */
return new class extends Migration
{
    public function up(): void
    {
        Schema::create('privacy_sections', function (Blueprint $table) {
            $table->id();
            // Stable key for the rows shipped by PrivacySectionSeeder, so
            // re-seeding updates them instead of duplicating. Admin-created
            // rows leave it null.
            $table->string('key')->nullable()->unique();
            $table->json('heading');  // {fa, en}
            $table->json('body');     // {fa, en}
            $table->boolean('is_active')->default(true);
            $table->integer('sort_order')->default(0);
            $table->timestamps();

            $table->index(['is_active', 'sort_order']);
        });
    }

    public function down(): void
    {
        Schema::dropIfExists('privacy_sections');
    }
};
