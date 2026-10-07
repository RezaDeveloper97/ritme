<?php

use Illuminate\Database\Migrations\Migration;
use Illuminate\Database\Schema\Blueprint;
use Illuminate\Support\Facades\Schema;

return new class extends Migration
{
    /**
     * Run the migrations.
     *
     * Twin of backend-go/db/migrations/00046_learning.sql
     * (docs/go-migration/migrations.md): courses, instructors, groups and
     * phone-based access grants (bloom B-N8-01), used only by the Go
     * internal/learning package. No model or routes here.
     */
    public function up(): void
    {
        Schema::create('learning_instructors', function (Blueprint $table) {
            $table->id();
            $table->foreignId('user_id')->unique()->constrained()->cascadeOnDelete();
            $table->string('display_name', 80);
            $table->string('title', 60)->nullable();
            $table->string('bio', 500)->nullable();
            $table->string('status', 10)->default('pending');  // pending|approved|revoked
            $table->dateTime('approved_at')->nullable();
            $table->unsignedBigInteger('approved_by')->nullable();
            $table->dateTime('revoked_at')->nullable();
            $table->timestamps();

            $table->index('status');
        });

        Schema::create('learning_courses', function (Blueprint $table) {
            $table->id();
            $table->foreignId('instructor_id')->constrained('learning_instructors')->cascadeOnDelete();
            $table->string('kind', 12)->default('course');      // course|standalone
            $table->string('title', 150);
            $table->text('description')->nullable();
            $table->unsignedBigInteger('cover_media_id')->nullable();
            $table->string('status', 10)->default('draft');     // draft|published
            $table->dateTime('published_at')->nullable();
            $table->unsignedSmallInteger('sort_order')->default(0);
            $table->timestamps();

            $table->index(['instructor_id', 'status']);
        });

        Schema::create('learning_chapters', function (Blueprint $table) {
            $table->id();
            $table->foreignId('course_id')->constrained('learning_courses')->cascadeOnDelete();
            $table->string('title', 150);
            $table->unsignedSmallInteger('sort_order')->default(0);
            $table->dateTime('unlock_at')->nullable();
            $table->timestamps();

            $table->index(['course_id', 'sort_order']);
        });

        Schema::create('learning_lessons', function (Blueprint $table) {
            $table->id();
            $table->foreignId('course_id')->constrained('learning_courses')->cascadeOnDelete();
            $table->foreignId('chapter_id')->nullable()->constrained('learning_chapters')->nullOnDelete();
            $table->string('kind', 8);                          // video|audio|pdf
            $table->string('title', 150);
            $table->text('description')->nullable();
            $table->unsignedInteger('duration_seconds')->nullable();
            $table->unsignedSmallInteger('page_count')->nullable();
            $table->unsignedBigInteger('size_bytes')->nullable();
            $table->unsignedBigInteger('media_id')->nullable();
            $table->string('media_status', 12)->default('none'); // none|uploading|processing|ready|failed
            $table->string('status', 10)->default('draft');      // draft|published
            $table->dateTime('published_at')->nullable();
            $table->unsignedSmallInteger('sort_order')->default(0);
            $table->timestamps();

            $table->index(['course_id', 'sort_order']);
            $table->index('chapter_id');
        });

        Schema::create('learning_groups', function (Blueprint $table) {
            $table->id();
            $table->foreignId('instructor_id')->constrained('learning_instructors')->cascadeOnDelete();
            $table->string('name', 100);
            $table->timestamps();

            $table->index('instructor_id');
        });

        Schema::create('learning_group_courses', function (Blueprint $table) {
            $table->id();
            $table->foreignId('group_id')->constrained('learning_groups')->cascadeOnDelete();
            $table->foreignId('course_id')->constrained('learning_courses')->cascadeOnDelete();
            $table->timestamps();

            $table->unique(['group_id', 'course_id']);
            $table->index('course_id');
        });

        Schema::create('learning_grants', function (Blueprint $table) {
            $table->id();
            $table->foreignId('instructor_id')->constrained('learning_instructors')->cascadeOnDelete();
            $table->string('phone', 11);                        // normalised 09xxxxxxxxx
            $table->foreignId('user_id')->nullable()->constrained()->cascadeOnDelete();
            $table->foreignId('group_id')->nullable()->constrained('learning_groups')->cascadeOnDelete();
            $table->foreignId('course_id')->nullable()->constrained('learning_courses')->cascadeOnDelete();
            $table->string('duration', 10);                     // unlimited|days|until
            $table->unsignedSmallInteger('duration_days')->nullable();
            $table->date('until_date')->nullable();
            $table->string('status', 10)->default('pending');   // pending|active|revoked
            $table->dateTime('activated_at')->nullable();
            $table->dateTime('expires_at')->nullable();
            $table->dateTime('revoked_at')->nullable();
            $table->timestamps();

            $table->index(['phone', 'status']);
            $table->index(['user_id', 'status']);
            $table->index(['instructor_id', 'status']);
            $table->index('group_id');
            $table->index('course_id');
        });

        Schema::create('learning_progress', function (Blueprint $table) {
            $table->id();
            $table->foreignId('user_id')->constrained()->cascadeOnDelete();
            $table->foreignId('lesson_id')->constrained('learning_lessons')->cascadeOnDelete();
            $table->foreignId('course_id')->constrained('learning_courses')->cascadeOnDelete();
            $table->unsignedInteger('position_seconds')->default(0);
            $table->unsignedTinyInteger('percent')->default(0);
            $table->dateTime('completed_at')->nullable();
            $table->dateTime('last_seen_at')->nullable();
            $table->timestamps();

            $table->unique(['user_id', 'lesson_id']);
            $table->index(['user_id', 'last_seen_at']);
            $table->index('lesson_id');
            $table->index('course_id');
        });

        Schema::create('learning_sms_outbox', function (Blueprint $table) {
            $table->id();
            $table->foreignId('grant_id')->constrained('learning_grants')->cascadeOnDelete();
            $table->string('status', 10)->default('pending');   // pending|sent|skipped|failed
            $table->string('reason', 20)->nullable();
            $table->dateTime('due_at');
            $table->unsignedTinyInteger('attempts')->default(0);
            $table->dateTime('lease_until')->nullable();
            $table->dateTime('sent_at')->nullable();
            $table->timestamps();

            $table->index(['status', 'due_at']);
            $table->index('grant_id');
        });
    }

    /**
     * Reverse the migrations.
     */
    public function down(): void
    {
        Schema::dropIfExists('learning_sms_outbox');
        Schema::dropIfExists('learning_progress');
        Schema::dropIfExists('learning_grants');
        Schema::dropIfExists('learning_group_courses');
        Schema::dropIfExists('learning_groups');
        Schema::dropIfExists('learning_lessons');
        Schema::dropIfExists('learning_chapters');
        Schema::dropIfExists('learning_courses');
        Schema::dropIfExists('learning_instructors');
    }
};
