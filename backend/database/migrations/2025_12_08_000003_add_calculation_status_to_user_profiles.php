<?php

use Illuminate\Database\Migrations\Migration;
use Illuminate\Database\Schema\Blueprint;
use Illuminate\Support\Facades\Schema;

return new class extends Migration
{
    /**
     * Run the migrations.
     */
    public function up(): void
    {
        Schema::table('user_profiles', function (Blueprint $table) {
            $table->string('calculation_status')->default('pending')->after('last_period_start');
            $table->timestamp('calculation_started_at')->nullable()->after('calculation_status');
            $table->timestamp('calculation_completed_at')->nullable()->after('calculation_started_at');
            $table->integer('calculation_version')->default(0)->after('calculation_completed_at');
        });
    }

    /**
     * Reverse the migrations.
     */
    public function down(): void
    {
        Schema::table('user_profiles', function (Blueprint $table) {
            $table->dropColumn([
                'calculation_status',
                'calculation_started_at',
                'calculation_completed_at',
                'calculation_version',
            ]);
        });
    }
};
