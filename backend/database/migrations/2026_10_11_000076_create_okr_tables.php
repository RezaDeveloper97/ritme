<?php

use Illuminate\Database\Migrations\Migration;
use Illuminate\Database\Schema\Blueprint;
use Illuminate\Support\Facades\Schema;

return new class extends Migration
{
    /**
     * Twin of backend-go/db/migrations/00076_okr.sql: OKR objectives and key results, a key result
     * optionally bound to a live aggregate metric (bloom B-N9-09, D-95). Used only by the Go
     * internal/okr package. No model or routes.
     */
    public function up(): void
    {
        Schema::create('okr_objectives', function (Blueprint $table) {
            $table->id();
            $table->string('title', 255);
            $table->string('owner', 120)->nullable();
            $table->string('period', 40);
            $table->date('starts_on');
            $table->date('ends_on');
            $table->integer('sort_order')->default(0);
            $table->timestamps();

            $table->index(['starts_on', 'ends_on']);
        });

        Schema::create('okr_key_results', function (Blueprint $table) {
            $table->id();
            $table->foreignId('objective_id')->constrained('okr_objectives')->cascadeOnDelete();
            $table->string('title', 255);
            $table->string('metric_key', 40)->nullable();
            $table->string('unit', 16)->default('count');
            $table->decimal('baseline', 16, 2)->default(0);
            $table->decimal('target', 16, 2);
            $table->decimal('manual_value', 16, 2)->nullable();
            $table->string('owner', 120)->nullable();
            $table->integer('sort_order')->default(0);
            $table->timestamps();

            $table->index(['objective_id', 'sort_order']);
        });
    }

    public function down(): void
    {
        Schema::dropIfExists('okr_key_results');
        Schema::dropIfExists('okr_objectives');
    }
};
