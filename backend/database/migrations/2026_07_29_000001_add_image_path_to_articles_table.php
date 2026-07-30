<?php

use Illuminate\Database\Migrations\Migration;
use Illuminate\Database\Schema\Blueprint;
use Illuminate\Support\Facades\Schema;

return new class extends Migration
{
    /**
     * Cover images can now be uploaded from the admin panel instead of being
     * pasted as a URL. The uploaded file lives on the "public" disk and only
     * its relative path is stored; Article::image_url resolves it at read time
     * so the served URL follows APP_URL instead of being frozen in the row.
     * The existing image_url column stays as the manual/external fallback.
     */
    public function up(): void
    {
        Schema::table('articles', function (Blueprint $table) {
            $table->string('image_path')->nullable()->after('image_url');
        });
    }

    public function down(): void
    {
        Schema::table('articles', function (Blueprint $table) {
            $table->dropColumn('image_path');
        });
    }
};
