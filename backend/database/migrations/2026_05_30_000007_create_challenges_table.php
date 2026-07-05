<?php

use Illuminate\Database\Migrations\Migration;
use Illuminate\Database\Schema\Blueprint;
use Illuminate\Support\Facades\Schema;

return new class extends Migration
{
    /**
     * Run the migrations.
     *
     * Pool of daily challenges (چالش امروز). One is picked per user/day and
     * completion is tracked in user_challenge_completions.
     */
    public function up(): void
    {
        Schema::create('challenges', function (Blueprint $table) {
            $table->id();
            $table->json('title');                       // {fa, en}
            $table->json('description')->nullable();     // {fa, en}
            $table->string('cycle_phase')->nullable();   // null = any phase
            $table->string('category')->nullable();
            $table->string('difficulty')->nullable();    // easy, medium, hard
            $table->boolean('is_active')->default(true);
            $table->integer('sort_order')->default(0);
            $table->timestamps();

            $table->index(['is_active', 'cycle_phase']);
        });
    }

    public function down(): void
    {
        Schema::dropIfExists('challenges');
    }
};
