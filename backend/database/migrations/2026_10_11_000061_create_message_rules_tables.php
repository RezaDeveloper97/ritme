<?php

use Illuminate\Database\Migrations\Migration;
use Illuminate\Database\Schema\Blueprint;
use Illuminate\Support\Facades\Schema;

return new class extends Migration
{
    /**
     * Run the migrations.
     *
     * Twin of backend-go/db/migrations/00061_messages_v2.sql: rule-based smart
     * messages with levels, A/B variants, medical approval, versions and
     * events (bloom B-N9-05, D-86). Used only by the Go packages.
     */
    public function up(): void
    {
        Schema::create('message_rules', function (Blueprint $table) {
            $table->id();
            $table->string('title', 160);
            $table->string('level', 16)->default('info');
            $table->string('kind', 16)->default('product');
            $table->unsignedSmallInteger('priority')->default(1);
            $table->string('placement', 32)->default('smart_tip');
            $table->text('content');
            $table->boolean('variants_enabled')->default(false);
            $table->unsignedTinyInteger('ab_split')->default(50);
            $table->text('conditions');
            $table->text('exceptions');
            $table->unsignedSmallInteger('cap_count')->nullable();
            $table->string('cap_period', 8)->nullable();
            $table->unsignedSmallInteger('cooldown_days')->default(0);
            $table->unsignedTinyInteger('hours_from')->nullable();
            $table->unsignedTinyInteger('hours_to')->nullable();
            $table->date('starts_on')->nullable();
            $table->date('ends_on')->nullable();
            $table->unsignedTinyInteger('audience_percent')->default(100);
            $table->string('goal', 120)->nullable();
            $table->string('status', 16)->default('draft');
            $table->boolean('is_active')->default(false);
            $table->unsignedInteger('version')->default(1);
            $table->unsignedBigInteger('approved_by')->nullable();
            $table->timestamp('approved_at')->nullable();
            $table->unsignedBigInteger('created_by')->nullable();
            $table->timestamps();

            $table->index(['placement', 'is_active', 'priority']);
            $table->index('status');
        });

        Schema::create('message_rule_versions', function (Blueprint $table) {
            $table->id();
            $table->unsignedBigInteger('rule_id');
            $table->unsignedInteger('version');
            $table->text('snapshot');
            $table->unsignedBigInteger('changed_by')->nullable();
            $table->timestamp('created_at')->nullable();

            $table->unique(['rule_id', 'version']);
        });

        Schema::create('message_events', function (Blueprint $table) {
            $table->id();
            $table->unsignedBigInteger('rule_id');
            $table->unsignedBigInteger('user_id');
            $table->string('variant', 1)->default('a');
            $table->string('event', 12);
            $table->timestamp('created_at')->nullable();

            $table->index(['rule_id', 'event', 'created_at']);
            $table->index(['user_id', 'rule_id', 'created_at']);
        });
    }

    /**
     * Reverse the migrations.
     */
    public function down(): void
    {
        Schema::dropIfExists('message_events');
        Schema::dropIfExists('message_rule_versions');
        Schema::dropIfExists('message_rules');
    }
};
