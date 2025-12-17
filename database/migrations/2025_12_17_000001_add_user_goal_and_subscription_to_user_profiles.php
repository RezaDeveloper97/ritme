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
            // User goal: ttc (Trying to Conceive) or non_ttc (not trying to conceive)
            $table->string('user_goal')->default('non_ttc')->after('last_period_start');

            // Subscription type: free or premium
            $table->string('subscription_type')->default('free')->after('user_goal');
        });
    }

    /**
     * Reverse the migrations.
     */
    public function down(): void
    {
        Schema::table('user_profiles', function (Blueprint $table) {
            $table->dropColumn(['user_goal', 'subscription_type']);
        });
    }
};
