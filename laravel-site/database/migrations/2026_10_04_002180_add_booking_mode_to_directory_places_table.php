<?php

declare(strict_types=1);

use Illuminate\Database\Migrations\Migration;
use Illuminate\Database\Schema\Blueprint;
use Illuminate\Support\Facades\Schema;

/**
 * How mothers reach a place (L5-06, BookingMode): `online` shows the booking request form on the place page, `phone`
 * hides it and shows the place's number instead.
 */
return new class extends Migration
{
    public function up(): void
    {
        Schema::table('directory_places', function (Blueprint $table): void {
            $table->string('booking_mode', 16)->default('online')->after('cancellation_policy');
        });
    }

    public function down(): void
    {
        Schema::table('directory_places', function (Blueprint $table): void {
            $table->dropColumn('booking_mode');
        });
    }
};
