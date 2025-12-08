<?php

use Illuminate\Database\Migrations\Migration;
use Illuminate\Database\Schema\Blueprint;
use Illuminate\Support\Facades\Schema;

return new class extends Migration
{
    /**
     * Run the migrations.
     */
    public function up(): void
    {
        Schema::create('cycle_histories', function (Blueprint $table) {
            $table->id();
            $table->foreignId('user_id')->constrained()->onDelete('cascade');
            $table->date('period_start_date');
            $table->date('period_end_date')->nullable();
            $table->integer('cycle_length')->nullable();
            $table->integer('bleeding_length')->nullable();
            $table->boolean('is_confirmed')->default(false);
            $table->timestamps();

            $table->unique(['user_id', 'period_start_date']);
            $table->index(['user_id', 'period_start_date']);
        });
    }

    /**
     * Reverse the migrations.
     */
    public function down(): void
    {
        Schema::dropIfExists('cycle_histories');
    }
};
