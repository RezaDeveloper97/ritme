<?php

use Illuminate\Database\Migrations\Migration;
use Illuminate\Support\Facades\DB;

// Schema-only twin of backend-go/db/migrations/00126_companion_goals_multi.sql: companions.goal now holds the goals
// the owner picked for a link, comma-separated (Go only; the Laravel API never serves it). No model change here.
return new class extends Migration
{
    public function up(): void
    {
        if (! $this->mariadb()) {
            return; // sqlite (tests) does not enforce varchar lengths
        }
        DB::statement('ALTER TABLE `companions` MODIFY `goal` varchar(64) DEFAULT NULL');
    }

    public function down(): void
    {
        if (! $this->mariadb()) {
            return;
        }
        DB::statement("UPDATE `companions` SET `goal` = SUBSTRING_INDEX(`goal`, ',', 1) WHERE `goal` LIKE '%,%'");
        DB::statement('ALTER TABLE `companions` MODIFY `goal` varchar(16) DEFAULT NULL');
    }

    private function mariadb(): bool
    {
        return in_array(DB::getDriverName(), ['mysql', 'mariadb'], true);
    }
};
