<?php

use Illuminate\Database\Migrations\Migration;
use Illuminate\Database\Schema\Blueprint;
use Illuminate\Support\Facades\Schema;

return new class extends Migration
{
    /**
     * Run the migrations.
     *
     * Pool of daily affirmations (تأکید روزانه). One is picked deterministically
     * per user/day, optionally filtered by cycle phase.
     */
    public function up(): void
    {
        Schema::create('affirmations', function (Blueprint $table) {
            $table->id();
            $table->json('text');                        // {fa, en}
            $table->string('cycle_phase')->nullable();   // null = any phase
            $table->boolean('is_active')->default(true);
            $table->integer('sort_order')->default(0);
            $table->timestamps();

            $table->index(['is_active', 'cycle_phase']);
        });
    }

    public function down(): void
    {
        Schema::dropIfExists('affirmations');
    }
};
