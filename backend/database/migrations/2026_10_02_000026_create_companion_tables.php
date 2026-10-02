<?php

use Illuminate\Database\Migrations\Migration;
use Illuminate\Database\Schema\Blueprint;
use Illuminate\Support\Facades\Schema;

return new class extends Migration
{
    /**
     * Run the migrations.
     *
     * Twin of backend-go/db/migrations/00026_companions.sql
     * (docs/go-migration/migrations.md): the companion «همدم» & family
     * tables (bloom B-N4-01) used only by the Go internal/companion package.
     * Schema only — no model or routes here. Codes are validated in Go.
     */
    public function up(): void
    {
        Schema::create('companions', function (Blueprint $table) {
            $table->id();
            $table->foreignId('owner_id')->constrained('users')->cascadeOnDelete();
            $table->foreignId('companion_user_id')->nullable()->constrained('users')->cascadeOnDelete();
            $table->string('type', 16);                          // partner|spouse
            $table->string('status', 16)->default('invited');   // invited|active|revoked
            $table->string('display_name', 100)->nullable();
            $table->timestamp('invited_at')->nullable();
            $table->timestamp('accepted_at')->nullable();
            $table->timestamp('revoked_at')->nullable();
            $table->string('revoked_by', 16)->nullable();       // owner|companion
            $table->timestamps();

            $table->index(['owner_id', 'status']);
            $table->index(['companion_user_id', 'status']);
        });

        Schema::create('companion_invites', function (Blueprint $table) {
            $table->id();
            $table->foreignId('companion_id')->constrained('companions')->cascadeOnDelete();
            $table->foreignId('owner_id')->constrained('users')->cascadeOnDelete();
            $table->string('phone', 11)->nullable();
            $table->char('code_hash', 64)->unique();             // HMAC-SHA256 hex of the 6-char code
            $table->unsignedTinyInteger('attempts')->default(0);
            $table->timestamp('expires_at')->nullable();
            $table->timestamp('used_at')->nullable();
            $table->foreignId('used_by_id')->nullable()->constrained('users')->nullOnDelete();
            $table->timestamp('revoked_at')->nullable();
            $table->timestamps();

            $table->index(['owner_id', 'created_at']);
        });

        Schema::create('companion_grants', function (Blueprint $table) {
            $table->id();
            $table->foreignId('companion_id')->constrained('companions')->cascadeOnDelete();
            $table->string('section', 32);                       // cycle|symptoms|meds|appointments|pregnancy
            $table->string('level', 8);                          // view|edit (no row = none)
            $table->timestamps();

            $table->unique(['companion_id', 'section']);
        });

        Schema::create('families', function (Blueprint $table) {
            $table->id();
            $table->foreignId('owner_id')->constrained('users')->cascadeOnDelete();
            $table->foreignId('spouse_user_id')->nullable()->constrained('users')->cascadeOnDelete();
            $table->foreignId('companion_id')->unique()->constrained('companions')->cascadeOnDelete();
            $table->timestamps();
        });

        Schema::create('family_children', function (Blueprint $table) {
            $table->id();
            $table->foreignId('family_id')->constrained('families')->cascadeOnDelete();
            $table->unsignedBigInteger('child_id');              // FK → children(id) added by bloom B-N5-02
            $table->timestamps();

            $table->unique(['family_id', 'child_id']);
            $table->index('child_id');
        });

        Schema::create('companion_audit_logs', function (Blueprint $table) {
            $table->id();
            $table->foreignId('owner_id')->constrained('users')->cascadeOnDelete();
            $table->foreignId('actor_id')->nullable()->constrained('users')->nullOnDelete();
            $table->foreignId('companion_id')->nullable()->constrained('companions')->nullOnDelete();
            $table->string('section', 32)->nullable();
            $table->string('action', 32);                        // read|write|invited|accepted|revoked|grants_changed
            $table->timestamp('created_at')->nullable();

            $table->index(['owner_id', 'created_at']);
        });
    }

    /**
     * Reverse the migrations.
     */
    public function down(): void
    {
        Schema::dropIfExists('companion_audit_logs');
        Schema::dropIfExists('family_children');
        Schema::dropIfExists('families');
        Schema::dropIfExists('companion_grants');
        Schema::dropIfExists('companion_invites');
        Schema::dropIfExists('companions');
    }
};
