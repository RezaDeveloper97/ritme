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
        Schema::create('pregnancy_fetal_movements', function (Blueprint $table) {
            $table->id();
            $table->foreignId('user_id')->constrained()->onDelete('cascade');
            $table->date('log_date');
            $table->integer('pregnancy_week');

            // Movement status
            $table->string('movement_status'); // not_felt_yet, felt, normal, reduced, increased, none

            // Movement count (optional - for kick counting)
            $table->integer('movement_count')->nullable();

            // Time tracking
            $table->time('first_movement_time')->nullable();
            $table->time('last_movement_time')->nullable();

            // Notes
            $table->text('notes')->nullable();

            $table->timestamps();

            // One log per user per date
            $table->unique(['user_id', 'log_date']);
        });
    }

    /**
     * Reverse the migrations.
     */
    public function down(): void
    {
        Schema::dropIfExists('pregnancy_fetal_movements');
    }
};
