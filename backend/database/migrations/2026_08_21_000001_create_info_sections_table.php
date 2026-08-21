<?php

use Illuminate\Database\Migrations\Migration;
use Illuminate\Database\Schema\Blueprint;
use Illuminate\Support\Facades\DB;
use Illuminate\Support\Facades\Schema;

/**
 * Generalises `privacy_sections` into `info_sections`.
 *
 * The privacy screen was the first admin-managed text screen; راهنما/پشتیبانی,
 * قوانین and درباره‌ما have exactly the same shape (a list of heading+body
 * cards), so they share one table keyed by `group` rather than growing a
 * near-identical table each time.
 *
 * New here: an optional per-box call-to-action (`link_label` + `link_url`) so a
 * support box can carry a real "ایمیل بزن" / تلگرام button instead of an
 * address buried in prose.
 *
 * A fresh table plus a copy is used rather than a rename: the unique key moves
 * from `key` to `(group, key)`, and rebuilding indexes in place behaves
 * differently on SQLite (local dev) and MariaDB (prod).
 */
return new class extends Migration
{
    public function up(): void
    {
        Schema::create('info_sections', function (Blueprint $table) {
            $table->id();
            // Which screen this box belongs to: privacy | terms | about | help.
            $table->string('group', 32)->default('privacy');
            // Stable key for seeded rows so re-seeding updates instead of
            // duplicating. Admin-created rows leave it null.
            $table->string('key')->nullable();
            $table->json('heading');           // {fa, en}
            $table->json('body');              // {fa, en}
            $table->json('link_label')->nullable(); // {fa, en} — button caption
            $table->string('link_url')->nullable(); // https:, mailto:, tel:
            $table->boolean('is_active')->default(true);
            $table->integer('sort_order')->default(0);
            $table->timestamps();

            $table->unique(['group', 'key']);
            $table->index(['group', 'is_active', 'sort_order']);
        });

        if (Schema::hasTable('privacy_sections')) {
            // Carry the live policy across untouched — prod has admin-edited
            // copy in here that must survive the move.
            foreach (DB::table('privacy_sections')->orderBy('id')->get() as $row) {
                DB::table('info_sections')->insert([
                    'group' => 'privacy',
                    'key' => $row->key,
                    'heading' => $row->heading,
                    'body' => $row->body,
                    'is_active' => $row->is_active,
                    'sort_order' => $row->sort_order,
                    'created_at' => $row->created_at,
                    'updated_at' => $row->updated_at,
                ]);
            }

            Schema::drop('privacy_sections');
        }
    }

    public function down(): void
    {
        Schema::create('privacy_sections', function (Blueprint $table) {
            $table->id();
            $table->string('key')->nullable()->unique();
            $table->json('heading');
            $table->json('body');
            $table->boolean('is_active')->default(true);
            $table->integer('sort_order')->default(0);
            $table->timestamps();

            $table->index(['is_active', 'sort_order']);
        });

        // Only the privacy group has somewhere to go back to; the other groups
        // were never in the old table.
        foreach (DB::table('info_sections')->where('group', 'privacy')->orderBy('id')->get() as $row) {
            DB::table('privacy_sections')->insert([
                'key' => $row->key,
                'heading' => $row->heading,
                'body' => $row->body,
                'is_active' => $row->is_active,
                'sort_order' => $row->sort_order,
                'created_at' => $row->created_at,
                'updated_at' => $row->updated_at,
            ]);
        }

        Schema::dropIfExists('info_sections');
    }
};
