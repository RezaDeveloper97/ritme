<?php

use Illuminate\Database\Migrations\Migration;
use Illuminate\Database\Schema\Blueprint;
use Illuminate\Support\Facades\Schema;

return new class extends Migration
{
    /**
     * Twin of backend-go/db/migrations/00082_insurance.sql: insurance self-tracking (canvas
     * CB-INS-01) — insurers catalog, contracted centres, the user's policy, members, coverage
     * caps, claims with their timeline and record-document links, and health questionnaire
     * drafts. Used only by the Go internal/insurance package; nothing is sent to an insurer.
     */
    public function up(): void
    {
        Schema::create('insurance_insurers', function (Blueprint $table) {
            $table->id();
            $table->string('code', 40)->unique();
            $table->string('name', 120);
            $table->string('website', 255)->nullable();
            $table->string('phone', 32)->nullable();
            $table->boolean('is_active')->default(true);
            $table->integer('sort_order')->default(0);
            $table->timestamps();

            $table->index(['is_active', 'sort_order']);
        });

        Schema::create('insurance_centres', function (Blueprint $table) {
            $table->id();
            $table->foreignId('insurer_id')->nullable()->constrained('insurance_insurers')->nullOnDelete();
            $table->string('name', 160);
            $table->string('type', 16);
            $table->string('services', 160)->nullable();
            $table->string('city', 80)->nullable();
            $table->string('address', 300)->nullable();
            $table->string('phone', 32)->nullable();
            $table->decimal('lat', 10, 7)->nullable();
            $table->decimal('lng', 10, 7)->nullable();
            $table->boolean('is_direct')->default(false);
            $table->boolean('is_active')->default(true);
            $table->integer('sort_order')->default(0);
            $table->timestamps();

            $table->index(['is_active', 'type']);
        });

        Schema::create('insurance_policies', function (Blueprint $table) {
            $table->id();
            $table->foreignId('user_id')->unique()->constrained()->cascadeOnDelete();
            $table->foreignId('insurer_id')->nullable()->constrained('insurance_insurers')->nullOnDelete();
            $table->string('insurer_name', 120)->nullable();
            $table->string('plan_name', 120)->nullable();
            $table->string('number_last4', 4)->nullable();
            $table->date('valid_from')->nullable();
            $table->date('valid_to')->nullable();
            $table->unsignedSmallInteger('claim_deadline_days')->nullable();
            $table->string('iban_last4', 4)->nullable();
            $table->text('not_covered_note')->nullable();
            $table->timestamps();
        });

        Schema::create('insurance_policy_members', function (Blueprint $table) {
            $table->id();
            $table->foreignId('policy_id')->constrained('insurance_policies')->cascadeOnDelete();
            $table->string('kind', 8);
            $table->foreignId('child_id')->nullable()->constrained('children')->nullOnDelete();
            $table->string('name', 64);
            $table->integer('sort_order')->default(0);
            $table->timestamps();

            $table->index(['policy_id', 'sort_order']);
        });

        Schema::create('insurance_coverages', function (Blueprint $table) {
            $table->id();
            $table->foreignId('policy_id')->constrained('insurance_policies')->cascadeOnDelete();
            $table->string('category', 16);
            $table->string('status', 16)->default('covered');
            $table->unsignedBigInteger('cap_amount')->nullable();
            $table->unsignedTinyInteger('deductible_pct')->nullable();
            $table->unsignedSmallInteger('waiting_days')->nullable();
            $table->string('note', 160)->nullable();
            $table->integer('sort_order')->default(0);
            $table->timestamps();

            $table->unique(['policy_id', 'category']);
        });

        Schema::create('insurance_claims', function (Blueprint $table) {
            $table->id();
            $table->foreignId('user_id')->constrained()->cascadeOnDelete();
            $table->foreignId('policy_id')->constrained('insurance_policies')->cascadeOnDelete();
            $table->foreignId('member_id')->nullable()->constrained('insurance_policy_members')->nullOnDelete();
            $table->string('cost_type', 16);
            $table->string('title', 120)->nullable();
            $table->foreignId('centre_id')->nullable()->constrained('insurance_centres')->nullOnDelete();
            $table->string('centre_name', 160)->nullable();
            $table->date('service_date');
            $table->unsignedBigInteger('amount');
            $table->unsignedBigInteger('over_tariff')->default(0);
            $table->unsignedBigInteger('paid_amount')->nullable();
            $table->string('status', 16)->default('submitted');
            $table->date('deadline_on')->nullable();
            $table->text('notes_enc')->nullable();
            $table->timestamps();

            $table->index(['user_id', 'service_date']);
        });

        Schema::create('insurance_claim_events', function (Blueprint $table) {
            $table->id();
            $table->foreignId('claim_id')->constrained('insurance_claims')->cascadeOnDelete();
            $table->string('status', 16);
            $table->date('occurred_on');
            $table->date('deadline_on')->nullable();
            $table->unsignedBigInteger('amount')->nullable();
            $table->text('note_enc')->nullable();
            $table->timestamps();

            $table->index(['claim_id', 'occurred_on']);
        });

        Schema::create('insurance_claim_documents', function (Blueprint $table) {
            $table->id();
            $table->foreignId('claim_id')->constrained('insurance_claims')->cascadeOnDelete();
            $table->foreignId('document_id')->constrained('record_documents')->cascadeOnDelete();
            $table->string('label', 16)->nullable();
            $table->timestamps();

            $table->unique(['claim_id', 'document_id']);
        });

        Schema::create('insurance_questionnaires', function (Blueprint $table) {
            $table->id();
            $table->foreignId('user_id')->constrained()->cascadeOnDelete();
            $table->string('purpose', 16);
            $table->string('label', 80)->nullable();
            $table->string('status', 16)->default('draft');
            $table->json('answers');
            $table->text('explanation_enc')->nullable();
            $table->json('timeline')->nullable();
            $table->dateTime('confirmed_at')->nullable();
            $table->timestamps();

            $table->index(['user_id', 'created_at']);
        });
    }

    public function down(): void
    {
        Schema::dropIfExists('insurance_questionnaires');
        Schema::dropIfExists('insurance_claim_documents');
        Schema::dropIfExists('insurance_claim_events');
        Schema::dropIfExists('insurance_claims');
        Schema::dropIfExists('insurance_coverages');
        Schema::dropIfExists('insurance_policy_members');
        Schema::dropIfExists('insurance_policies');
        Schema::dropIfExists('insurance_centres');
        Schema::dropIfExists('insurance_insurers');
    }
};
