<?php

use Illuminate\Database\Migrations\Migration;
use Illuminate\Database\Schema\Blueprint;
use Illuminate\Support\Facades\Schema;

return new class extends Migration
{
    /**
     * Twin of backend-go/db/migrations/00064_metric_snapshots.sql: one aggregate
     * JSON snapshot per day for the admin overview (bloom B-N9-03, D-85). Used
     * only by the Go internal/metrics package. No model or routes.
     */
    public function up(): void
    {
        Schema::create('metric_snapshots', function (Blueprint $table) {
            $table->id();
            $table->date('day')->unique();
            $table->json('payload');
            $table->dateTime('computed_at');
            $table->timestamps();
        });
    }

    public function down(): void
    {
        Schema::dropIfExists('metric_snapshots');
    }
};
