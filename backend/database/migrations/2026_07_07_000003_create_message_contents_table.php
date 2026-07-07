<?php

use Illuminate\Database\Migrations\Migration;
use Illuminate\Database\Schema\Blueprint;
use Illuminate\Support\Facades\Schema;

/**
 * Editable storage for every "smart message" text that used to be hardcoded in
 * the MessageSystem engines/layers/modules. A row is identified by
 * (group, item_key, locale); `payload` holds the content shape for that group
 * (e.g. short/long/action/dos/donts, or foods/avoid/tip). The engines resolve
 * content through {@see App\Services\MessageSystem\Support\MessageContentRepository}.
 */
return new class extends Migration
{
    public function up(): void
    {
        Schema::create('message_contents', function (Blueprint $table) {
            $table->id();
            $table->string('group')->index();       // e.g. cycle_base_non_ttc
            $table->string('item_key')->index();     // e.g. menstruation / pain / 1
            $table->string('locale', 5);             // fa / en
            $table->string('label')->nullable();     // human description for admin UI
            $table->json('payload');                 // the content array
            $table->boolean('is_active')->default(true);
            $table->boolean('is_approved')->default(true);
            $table->unsignedInteger('sort_order')->default(0);
            $table->timestamps();

            $table->unique(['group', 'item_key', 'locale']);
        });
    }

    public function down(): void
    {
        Schema::dropIfExists('message_contents');
    }
};
