<?php

use Illuminate\Database\Migrations\Migration;
use Illuminate\Database\Schema\Blueprint;
use Illuminate\Support\Facades\Schema;

return new class extends Migration
{
    /** The child and baby-log tables (and the to-do pair) that get created_by / updated_by. */
    private const ACTOR_TABLES = [
        'children', 'child_measurements', 'child_vaccine_doses', 'child_milestone_checks',
        'baby_feeds', 'baby_sleeps', 'baby_diapers', 'todo_tasks', 'todo_items',
    ];

    /**
     * Run the migrations.
     *
     * Twin of backend-go/db/migrations/00108_family_schema_c.sql (docs/go-migration/migrations.md, T-M11-04, D-192):
     * family schema v6 part C — child_guardians, created_by / updated_by (FK users, set null) on the child, baby-log
     * and to-do tables, baby_med_doses (F-26), and todo_tasks.scope / assigned_to (household list, F-24, F-25).
     * children.owner_id keeps its cascading FK: this (frozen) app deletes accounts through it. Used only by the Go
     * backend: schema only (the goose file also backfills guardians), no model or routes, so no JSON changes here.
     */
    public function up(): void
    {
        if (! Schema::hasTable('child_guardians')) {
            Schema::create('child_guardians', function (Blueprint $table) {
                $table->id();
                $table->unsignedBigInteger('child_id');
                $table->unsignedBigInteger('user_id');
                $table->string('level', 8);                 // write|limited
                $table->json('sub_scopes')->nullable();     // limited: ["feeding","diaper","sleep"]
                $table->string('status', 10);               // invited|active|left|removed
                $table->unsignedBigInteger('invited_by')->nullable();
                $table->unsignedBigInteger('link_id')->nullable();
                $table->timestamps();

                $table->unique(['child_id', 'user_id']);
                $table->index(['user_id', 'status']);
                $table->foreign('child_id')->references('id')->on('children')->cascadeOnDelete();
                $table->foreign('user_id')->references('id')->on('users')->cascadeOnDelete();
                $table->foreign('invited_by')->references('id')->on('users')->nullOnDelete();
                $table->foreign('link_id')->references('id')->on('companions')->nullOnDelete();
            });
        }

        foreach (self::ACTOR_TABLES as $name) {
            if (! Schema::hasColumn($name, 'created_by')) {
                Schema::table($name, function (Blueprint $table) {
                    $table->unsignedBigInteger('created_by')->nullable();
                    $table->unsignedBigInteger('updated_by')->nullable();

                    $table->foreign('created_by')->references('id')->on('users')->nullOnDelete();
                    $table->foreign('updated_by')->references('id')->on('users')->nullOnDelete();
                });
            }
        }

        if (! Schema::hasTable('baby_med_doses')) {
            Schema::create('baby_med_doses', function (Blueprint $table) {
                $table->id();
                $table->unsignedBigInteger('child_id');
                $table->string('name', 80);
                $table->string('dose_text', 40)->nullable();
                $table->dateTime('given_at');
                $table->string('note', 200)->nullable();
                $table->unsignedBigInteger('created_by')->nullable();
                $table->unsignedBigInteger('updated_by')->nullable();
                $table->timestamps();

                $table->index(['child_id', 'given_at']);
                $table->foreign('child_id')->references('id')->on('children')->cascadeOnDelete();
                $table->foreign('created_by')->references('id')->on('users')->nullOnDelete();
                $table->foreign('updated_by')->references('id')->on('users')->nullOnDelete();
            });
        }

        if (! Schema::hasColumn('todo_tasks', 'scope')) {
            Schema::table('todo_tasks', function (Blueprint $table) {
                $table->string('scope', 10)->default('personal'); // personal|household
                $table->unsignedBigInteger('assigned_to')->nullable();

                $table->index(['user_id', 'scope', 'done_at']);
                $table->foreign('assigned_to')->references('id')->on('users')->nullOnDelete();
            });
        }
    }

    /**
     * Reverse the migrations.
     */
    public function down(): void
    {
        if (Schema::hasColumn('todo_tasks', 'scope')) {
            Schema::table('todo_tasks', function (Blueprint $table) {
                $table->dropForeign(['assigned_to']);
                // Dropping scope alone would shrink this key to a duplicate of (user_id, done_at).
                $table->dropIndex(['user_id', 'scope', 'done_at']);
                $table->dropColumn(['assigned_to', 'scope']); // the FK-backed single-column keys go with their columns
            });
        }

        Schema::dropIfExists('baby_med_doses');

        foreach (array_reverse(self::ACTOR_TABLES) as $name) {
            if (Schema::hasColumn($name, 'created_by')) {
                Schema::table($name, function (Blueprint $table) {
                    $table->dropForeign(['updated_by']);
                    $table->dropForeign(['created_by']);
                    $table->dropColumn(['updated_by', 'created_by']);
                });
            }
        }

        Schema::dropIfExists('child_guardians');
    }
};
