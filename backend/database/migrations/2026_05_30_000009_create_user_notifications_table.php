<?php

use Illuminate\Database\Migrations\Migration;
use Illuminate\Database\Schema\Blueprint;
use Illuminate\Support\Facades\Schema;

return new class extends Migration
{
    /**
     * Run the migrations.
     *
     * In-app notifications feeding the header bell badge (unread count) & list.
     * Named user_notifications to avoid clashing with Laravel's default
     * polymorphic `notifications` table.
     */
    public function up(): void
    {
        Schema::create('user_notifications', function (Blueprint $table) {
            $table->id();
            $table->foreignId('user_id')->constrained()->cascadeOnDelete();
            $table->string('type');                      // App\Enums\NotificationType
            $table->json('title');                       // {fa, en}
            $table->json('body')->nullable();            // {fa, en}
            $table->string('action_url')->nullable();
            $table->json('data')->nullable();            // arbitrary payload
            $table->timestamp('read_at')->nullable();
            $table->timestamps();

            $table->index(['user_id', 'read_at']);
        });
    }

    public function down(): void
    {
        Schema::dropIfExists('user_notifications');
    }
};
