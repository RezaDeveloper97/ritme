<?php

use Illuminate\Database\Migrations\Migration;
use Illuminate\Database\Schema\Blueprint;
use Illuminate\Support\Facades\Schema;

return new class extends Migration
{
    /**
     * Twin of backend-go/db/migrations/00075_partner_lab_requests.sql: the admin catalog of
     * orderable home-lab / screening services and the users' booking requests (bloom B-N10-04,
     * D-94). Used only by the Go internal/partners package.
     */
    public function up(): void
    {
        Schema::create('partner_lab_services', function (Blueprint $table) {
            $table->id();
            $table->string('code', 40)->unique();
            $table->string('kind', 24);
            $table->json('title');
            $table->json('description')->nullable();
            $table->string('partner_name', 120);
            $table->string('partner_code', 40)->default('fake');
            $table->boolean('is_active')->default(true);
            $table->integer('sort_order')->default(0);
            $table->timestamps();

            $table->index(['is_active', 'sort_order']);
        });

        Schema::create('partner_lab_requests', function (Blueprint $table) {
            $table->id();
            $table->string('reference', 16)->unique();
            $table->foreignId('user_id')->constrained()->cascadeOnDelete();
            $table->foreignId('service_id')->constrained('partner_lab_services');
            $table->string('status', 16)->default('requested');
            $table->string('partner_ref', 64)->nullable();
            $table->string('partner_status', 16)->nullable();
            $table->date('preferred_date');
            $table->string('preferred_window', 16);
            $table->boolean('consent_share')->default(false);
            $table->boolean('consent_import')->default(false);
            $table->longText('payload_enc')->nullable();
            $table->unsignedBigInteger('imported_lab_id')->nullable();
            $table->dateTime('closed_at')->nullable();
            $table->dateTime('purged_at')->nullable();
            $table->timestamps();

            $table->index(['user_id', 'created_at']);
            $table->index(['status', 'created_at']);
            $table->index('closed_at');
        });
    }

    public function down(): void
    {
        Schema::dropIfExists('partner_lab_requests');
        Schema::dropIfExists('partner_lab_services');
    }
};
