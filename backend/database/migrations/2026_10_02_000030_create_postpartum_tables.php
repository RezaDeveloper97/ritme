<?php

use Illuminate\Database\Migrations\Migration;
use Illuminate\Database\Schema\Blueprint;
use Illuminate\Support\Facades\Schema;

return new class extends Migration
{
    /**
     * Run the migrations.
     *
     * Twin of backend-go/db/migrations/00030_postpartum.sql
     * (docs/go-migration/migrations.md): the postpartum mode tables (bloom
     * B-N5-01) used only by the Go internal/postpartum package. Schema only —
     * no model or routes here. Codes are validated in Go.
     */
    public function up(): void
    {
        Schema::create('postpartum_profiles', function (Blueprint $table) {
            $table->id();
            $table->foreignId('user_id')->unique()->constrained('users')->cascadeOnDelete();
            $table->date('birth_date');
            $table->string('delivery_type', 16)->nullable();     // vaginal|cesarean
            $table->unsignedTinyInteger('baby_count')->default(1);
            $table->string('source', 16)->default('direct');     // pregnancy|direct
            $table->foreignId('pregnancy_profile_id')->nullable()->constrained('pregnancy_profiles')->nullOnDelete();
            $table->timestamp('pregnancy_closed_at')->nullable();
            $table->timestamps();
        });

        Schema::create('epds_checks', function (Blueprint $table) {
            $table->id();
            $table->foreignId('user_id')->constrained('users')->cascadeOnDelete();
            $table->string('kind', 8);                           // short|full
            $table->date('taken_on');
            $table->json('answers');                             // {item code: score 0–3}
            $table->unsignedTinyInteger('total');
            $table->unsignedTinyInteger('self_harm')->nullable();
            $table->boolean('urgent')->default(false);
            $table->timestamps();

            $table->unique(['user_id', 'kind', 'taken_on']);
            $table->index(['user_id', 'taken_on']);
        });
    }

    /**
     * Reverse the migrations.
     */
    public function down(): void
    {
        Schema::dropIfExists('epds_checks');
        Schema::dropIfExists('postpartum_profiles');
    }
};
