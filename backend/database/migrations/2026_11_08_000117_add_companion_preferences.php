<?php

use Illuminate\Database\Migrations\Migration;
use Illuminate\Database\Schema\Blueprint;
use Illuminate\Support\Facades\Schema;

return new class extends Migration
{
    /**
     * Run the migrations.
     *
     * Twin of backend-go/db/migrations/00117_companion_preferences.sql («ریتمی همراه» v2): the owner's per-link
     * preferences on companions — goal, the four day-signal consents (share_pain / share_mood / share_energy /
     * share_sleep, off by default) and action_prefs (comma-separated codes). Used only by the Go backend: schema
     * only, no model or routes.
     */
    public function up(): void
    {
        if (! Schema::hasColumn('companions', 'goal')) {
            Schema::table('companions', function (Blueprint $table) {
                $table->string('goal', 16)->nullable();          // cycle|support|low_energy|quality_time|info|unknown
                $table->boolean('share_pain')->default(false);
                $table->boolean('share_mood')->default(false);
                $table->boolean('share_energy')->default(false);
                $table->boolean('share_sleep')->default(false);
                $table->string('action_prefs', 64)->nullable();  // food,gift,time,calm,help (ordered)
            });
        }
    }

    /**
     * Reverse the migrations.
     */
    public function down(): void
    {
        if (Schema::hasColumn('companions', 'goal')) {
            Schema::table('companions', function (Blueprint $table) {
                $table->dropColumn(['action_prefs', 'share_sleep', 'share_energy', 'share_mood', 'share_pain', 'goal']);
            });
        }
    }
};
