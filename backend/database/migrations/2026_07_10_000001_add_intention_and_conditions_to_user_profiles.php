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
            // Pregnancy intention captured at onboarding: avoiding | pregnant | trying | unsure
            $table->string('pregnancy_intention')->nullable()->after('user_goal');

            // Optional self-reported chronic conditions (JSON array of ChronicCondition values)
            $table->json('chronic_conditions')->nullable()->after('pregnancy_intention');
        });
    }

    /**
     * Reverse the migrations.
     */
    public function down(): void
    {
        Schema::table('user_profiles', function (Blueprint $table) {
            $table->dropColumn(['pregnancy_intention', 'chronic_conditions']);
        });
    }
};
