<?php

declare(strict_types=1);

use Illuminate\Database\Migrations\Migration;
use Illuminate\Database\Schema\Blueprint;
use Illuminate\Support\Facades\Schema;

/**
 * spatie/laravel-activitylog table (the package's three stubs — create, `event`, `batch_uuid` — folded into one).
 */
return new class extends Migration
{
    public function up(): void
    {
        Schema::connection($this->connection())->create($this->table(), function (Blueprint $table): void {
            $table->bigIncrements('id');
            $table->string('log_name')->nullable();
            $table->text('description');
            $table->nullableMorphs('subject', 'subject');
            $table->string('event')->nullable();
            $table->nullableMorphs('causer', 'causer');
            $table->json('properties')->nullable();
            $table->uuid('batch_uuid')->nullable();
            $table->timestamps();
            $table->index('log_name');
        });
    }

    public function down(): void
    {
        Schema::connection($this->connection())->dropIfExists($this->table());
    }

    private function connection(): ?string
    {
        $connection = config('activitylog.database_connection');

        return is_string($connection) && $connection !== '' ? $connection : null;
    }

    private function table(): string
    {
        $table = config('activitylog.table_name', 'activity_log');

        return is_string($table) && $table !== '' ? $table : 'activity_log';
    }
};
