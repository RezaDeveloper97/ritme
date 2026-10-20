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
     * Twin of backend-go/db/migrations/00094_app_modules.sql
     * (docs/go-migration/migrations.md): admin kill-switches for whole app
     * modules. Seeds every module enabled, so `make schema-diff` row counts
     * match. Schema only — no models or routes here. insertOrIgnore never
     * overwrites an admin edit.
     */
    public function up(): void
    {
        Schema::create('app_modules', function (Blueprint $table) {
            $table->id();
            $table->string('key', 64)->unique();
            $table->boolean('enabled')->default(true);
            $table->timestamps();
        });

        $now = now();
        $rows = [];
        foreach ([
            'companion', 'ttc', 'ivf', 'menopause', 'teen', 'postpartum', 'children', 'contraception',
            'programs', 'mother_child', 'learning', 'my_courses', 'shop', 'insurance', 'record', 'devices',
            'doctors', 'todo', 'reminders', 'home_articles',
        ] as $key) {
            $rows[] = ['key' => $key, 'enabled' => true, 'created_at' => $now, 'updated_at' => $now];
        }
        DB::table('app_modules')->insertOrIgnore($rows);
    }

    /**
     * Reverse the migrations.
     */
    public function down(): void
    {
        Schema::dropIfExists('app_modules');
    }
};
