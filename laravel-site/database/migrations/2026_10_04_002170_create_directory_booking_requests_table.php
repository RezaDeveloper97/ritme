<?php

declare(strict_types=1);

use Illuminate\Database\Migrations\Migration;
use Illuminate\Database\Schema\Blueprint;
use Illuminate\Support\Facades\Schema;

/*
 * Booking REQUESTS from the place page (L5-04): a parent asks for a service on a preferred day + time window and the
 * place confirms by phone — no slots, no payment. Minimal personal data: the parent's name and mobile, the child's age
 * in months and an optional note; no IP or user agent. `code` is the unguessable public code of /directory/booked/{code}.
 * Place / service names and the announced price are snapshots (the place may change or disappear later).
 * Status: new|confirmed|cancelled|done (admin: L5-06).
 */
return new class extends Migration
{
    public function up(): void
    {
        Schema::create('directory_booking_requests', function (Blueprint $table): void {
            $table->id();
            $table->string('code', 20)->unique();
            $table->string('status', 16)->default('new');

            $table->foreignId('place_id')->nullable()->constrained('directory_places')->nullOnDelete();
            $table->string('place_name', 191);
            $table->foreignId('service_id')->nullable()->constrained('directory_place_services')->nullOnDelete();
            $table->string('service_name', 191)->nullable();
            $table->unsignedInteger('service_price')->nullable();
            $table->string('service_price_unit', 60)->nullable();

            $table->date('preferred_date');
            $table->string('time_window', 16);

            $table->string('parent_name', 100);
            $table->string('mobile', 20);
            $table->unsignedSmallInteger('child_age_months')->nullable();
            $table->string('note', 500)->nullable();
            $table->timestamps();

            $table->index(['status', 'created_at']);
            $table->index(['place_id', 'created_at']);
        });
    }

    public function down(): void
    {
        Schema::dropIfExists('directory_booking_requests');
    }
};
