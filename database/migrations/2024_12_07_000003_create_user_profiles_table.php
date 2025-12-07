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
        Schema::create('user_profiles', function (Blueprint $table) {
            $table->id();
            $table->foreignId('user_id')->constrained()->onDelete('cascade');
            $table->date('birthday')->nullable();
            $table->decimal('weight', 5, 2)->nullable(); // in kg
            $table->unsignedSmallInteger('height')->nullable(); // in cm
            $table->unsignedTinyInteger('period_duration')->nullable(); // days
            $table->unsignedTinyInteger('cycle_duration')->nullable(); // days
            $table->date('last_period_start')->nullable();
            $table->timestamps();
        });
    }

    /**
     * Reverse the migrations.
     */
    public function down(): void
    {
        Schema::dropIfExists('user_profiles');
    }
};
