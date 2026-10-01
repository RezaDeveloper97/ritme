<?php

use Illuminate\Database\Migrations\Migration;
use Illuminate\Database\Schema\Blueprint;
use Illuminate\Support\Facades\DB;
use Illuminate\Support\Facades\Schema;

return new class extends Migration
{
    /**
     * Run the migrations.
     *
     * Twin of backend-go/db/migrations/00018_plus_settings.sql
     * (docs/go-migration/migrations.md): admin-editable Ritme Plus settings
     * as key/value rows (B-N2-06; edited by the admin module of B-N2-09).
     * Seeds trial_offer_percent = 50 (the trial banner's offer), so
     * `make schema-diff` row counts match. Schema only — no models or
     * routes here. insertOrIgnore never overwrites an admin edit.
     */
    public function up(): void
    {
        Schema::create('plus_settings', function (Blueprint $table) {
            $table->id();
            $table->string('key', 64)->unique();
            $table->string('value', 255);
            $table->timestamps();
        });

        $now = now();
        DB::table('plus_settings')->insertOrIgnore([
            ['key' => 'trial_offer_percent', 'value' => '50', 'created_at' => $now, 'updated_at' => $now],
        ]);
    }

    /**
     * Reverse the migrations.
     */
    public function down(): void
    {
        Schema::dropIfExists('plus_settings');
    }
};
