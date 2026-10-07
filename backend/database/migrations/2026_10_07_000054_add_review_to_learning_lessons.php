<?php

use Illuminate\Database\Migrations\Migration;
use Illuminate\Database\Schema\Blueprint;
use Illuminate\Support\Facades\Schema;

return new class extends Migration
{
    /**
     * Run the migrations.
     *
     * Twin of backend-go/db/migrations/00054_learning_review.sql
     * (docs/go-migration/migrations.md): admin moderation of courses
     * (bloom B-N8-08) — the content-review state of learning_lessons and the
     * learning_moderation_log audit trail. Used only by the Go
     * internal/admin/learning package. No model or routes.
     */
    public function up(): void
    {
        Schema::table('learning_lessons', function (Blueprint $table) {
            $table->string('review_status', 10)->default('pending')->after('published_at'); // pending | approved | flagged
            $table->dateTime('reviewed_at')->nullable()->after('review_status');
            $table->unsignedBigInteger('reviewed_by')->nullable()->after('reviewed_at');  // admins.id, no FK
            $table->string('review_note', 300)->nullable()->after('reviewed_by');

            $table->index('review_status');
        });

        Schema::create('learning_moderation_log', function (Blueprint $table) {
            $table->id();
            $table->unsignedBigInteger('admin_id')->nullable();       // no FK: outlives a deleted admin
            $table->string('action', 32);                             // instructor.approve | lesson.flag | …
            $table->string('target_type', 16);                        // instructor | lesson
            $table->unsignedBigInteger('target_id');
            $table->unsignedBigInteger('instructor_id')->nullable();  // no FK: kept after the instructor is deleted
            $table->string('note', 300)->nullable();
            $table->timestamps();

            $table->index(['target_type', 'target_id']);
            $table->index('instructor_id');
            $table->index('created_at');
        });
    }

    /**
     * Reverse the migrations.
     */
    public function down(): void
    {
        Schema::dropIfExists('learning_moderation_log');
        Schema::table('learning_lessons', function (Blueprint $table) {
            $table->dropIndex(['review_status']);
            $table->dropColumn(['review_note', 'reviewed_by', 'reviewed_at', 'review_status']);
        });
    }
};
