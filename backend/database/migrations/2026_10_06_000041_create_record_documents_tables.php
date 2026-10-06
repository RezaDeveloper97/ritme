<?php

use Illuminate\Database\Migrations\Migration;
use Illuminate\Database\Schema\Blueprint;
use Illuminate\Support\Facades\Schema;

return new class extends Migration
{
    /**
     * Run the migrations.
     *
     * Twin of backend-go/db/migrations/00041_record_documents.sql
     * (docs/go-migration/migrations.md): record documents, extras and
     * timeline (canvas-build CB-REC-01) on top of the health record
     * (bloom B-N6-03) and the generic file storage (CB-CORE-05). Used only
     * by the Go internal/healthrecord package. No model or routes here.
     */
    public function up(): void
    {
        Schema::table('health_records', function (Blueprint $table) {
            $table->boolean('allergies_on_emergency_card')->default(true)->after('allergies');
            $table->json('surgeries')->nullable()->after('allergies_on_emergency_card');      // [{title, date}]
            $table->json('family_history')->nullable()->after('surgeries');                    // [{condition, relative}]
        });

        Schema::create('record_documents', function (Blueprint $table) {
            $table->id();
            $table->foreignId('user_id')->constrained()->cascadeOnDelete();
            $table->string('kind', 16);                   // imaging|visit|prescription|hospital|other
            $table->string('title', 120)->nullable();
            $table->date('document_date')->nullable();
            $table->date('ended_on')->nullable();         // hospital discharge
            $table->string('centre', 120)->nullable();
            $table->string('doctor', 120)->nullable();
            $table->text('note')->nullable();
            $table->json('extracted')->nullable();        // AI extraction (CB-REC-02)
            $table->string('review_state', 16)->default('manual'); // manual|pending|needs_review|confirmed|failed
            $table->timestamps();

            $table->index(['user_id', 'document_date']);
        });

        Schema::create('record_document_files', function (Blueprint $table) {
            $table->id();
            $table->foreignId('user_id')->constrained()->cascadeOnDelete();
            $table->foreignId('document_id')->constrained('record_documents')->cascadeOnDelete();
            $table->foreignId('file_id')->unique()->constrained('files')->cascadeOnDelete();
            $table->unsignedTinyInteger('position')->default(0);
            $table->timestamps();
        });

        Schema::create('record_document_links', function (Blueprint $table) {
            $table->id();
            $table->foreignId('user_id')->constrained()->cascadeOnDelete();
            $table->foreignId('document_id')->constrained('record_documents')->cascadeOnDelete();
            $table->string('target_type', 16);            // claim|pregnancy
            $table->unsignedBigInteger('target_id')->default(0);
            $table->string('state', 16);                  // attached|waiting|applied
            $table->timestamps();

            $table->unique(['document_id', 'target_type', 'target_id']);
            $table->index(['user_id', 'target_type', 'target_id']);
        });
    }

    /**
     * Reverse the migrations.
     */
    public function down(): void
    {
        Schema::dropIfExists('record_document_links');
        Schema::dropIfExists('record_document_files');
        Schema::dropIfExists('record_documents');
        Schema::table('health_records', function (Blueprint $table) {
            $table->dropColumn(['allergies_on_emergency_card', 'surgeries', 'family_history']);
        });
    }
};
