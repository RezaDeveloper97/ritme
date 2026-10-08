<?php

use Illuminate\Database\Migrations\Migration;
use Illuminate\Database\Schema\Blueprint;
use Illuminate\Support\Facades\Schema;

return new class extends Migration
{
    /**
     * Run the migrations.
     *
     * Twin of backend-go/db/migrations/00055_learning_security.sql
     * (docs/go-migration/migrations.md): courses security fixes (bloom
     * B-N8-01b) — learning_grants.seen_at (instructor sees student details
     * only after the student opened a course) and the STOP suppression list
     * of the course-unlocked SMS. Used only by the Go internal/learning
     * package. No model or routes.
     */
    public function up(): void
    {
        Schema::table('learning_grants', function (Blueprint $table) {
            $table->dateTime('seen_at')->nullable()->after('revoked_at');

            $table->index(['instructor_id', 'created_at']);
        });

        Schema::create('learning_sms_suppressions', function (Blueprint $table) {
            $table->id();
            $table->string('phone', 11)->unique();
            $table->string('source', 16)->default('support'); // support | sms
            $table->timestamps();
        });
    }

    /**
     * Reverse the migrations.
     */
    public function down(): void
    {
        Schema::dropIfExists('learning_sms_suppressions');
        Schema::table('learning_grants', function (Blueprint $table) {
            $table->dropIndex(['instructor_id', 'created_at']);
            $table->dropColumn('seen_at');
        });
    }
};
