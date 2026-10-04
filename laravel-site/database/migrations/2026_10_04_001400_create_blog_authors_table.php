<?php

declare(strict_types=1);

use Illuminate\Database\Migrations\Migration;
use Illuminate\Database\Schema\Blueprint;
use Illuminate\Support\Facades\Schema;

/*
 * Magazine authors and medical reviewers (L4-01, E-E-A-T). `same_as` = profile URLs for JSON-LD sameAs;
 * `credentials` is the qualification shown to readers («متخصص زنان و زایمان»). Slugs may be Persian.
 */
return new class extends Migration
{
    public function up(): void
    {
        Schema::create('blog_authors', function (Blueprint $table): void {
            $table->id();
            $table->string('name', 191);
            $table->string('slug', 191)->unique();
            $table->string('job_title', 191)->nullable();
            $table->string('credentials', 255)->nullable();
            $table->text('bio')->nullable();
            $table->foreignId('avatar_media_id')->nullable()->constrained('media')->nullOnDelete();
            $table->json('same_as')->nullable();
            $table->boolean('is_medical_reviewer')->default(false);
            $table->timestamps();

            $table->index('is_medical_reviewer');
        });
    }

    public function down(): void
    {
        Schema::dropIfExists('blog_authors');
    }
};
