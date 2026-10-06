<?php

use Illuminate\Database\Migrations\Migration;
use Illuminate\Database\Schema\Blueprint;
use Illuminate\Support\Facades\Schema;

return new class extends Migration
{
    /**
     * Run the migrations.
     *
     * Twin of backend-go/db/migrations/00036_health_record.sql
     * (docs/go-migration/migrations.md): the user-owned parts of the
     * health record (bloom B-N6-03) — blood type and allergies, and the
     * pregnancies / births a user adds by hand. Used only by the Go
     * internal/healthrecord package. No model or routes here.
     */
    public function up(): void
    {
        Schema::create('health_records', function (Blueprint $table) {
            $table->id();
            $table->foreignId('user_id')->unique()->constrained()->cascadeOnDelete();
            $table->string('blood_type', 4)->nullable();   // A+|A-|B+|B-|AB+|AB-|O+|O-
            $table->json('allergies')->nullable();         // free-text list; [] = none
            $table->timestamps();
        });

        Schema::create('health_record_pregnancies', function (Blueprint $table) {
            $table->id();
            $table->foreignId('user_id')->constrained()->cascadeOnDelete();
            $table->string('outcome', 16);                 // vaginal|cesarean|ended
            $table->date('ended_on')->nullable();
            $table->unsignedTinyInteger('baby_count')->nullable();
            $table->timestamps();

            $table->index(['user_id', 'ended_on']);
        });
    }

    /**
     * Reverse the migrations.
     */
    public function down(): void
    {
        Schema::dropIfExists('health_record_pregnancies');
        Schema::dropIfExists('health_records');
    }
};
