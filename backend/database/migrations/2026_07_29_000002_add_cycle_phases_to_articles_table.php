<?php

use Illuminate\Database\Migrations\Migration;
use Illuminate\Database\Schema\Blueprint;
use Illuminate\Support\Facades\DB;
use Illuminate\Support\Facades\Schema;

return new class extends Migration
{
    /**
     * An article can now be tagged with SEVERAL cycle phases, so the single
     * `cycle_phase` string becomes a `cycle_phases` JSON array. Existing rows
     * carry their one phase over as a single-element array; `null` keeps its
     * meaning of "general — show in every phase".
     */
    public function up(): void
    {
        Schema::table('articles', function (Blueprint $table) {
            $table->json('cycle_phases')->nullable()->after('cycle_phase');
        });

        // Encoded in PHP so the JSON is byte-identical on MySQL and SQLite.
        $this->eachRow('cycle_phase', function (int $id, $phase) {
            DB::table('articles')->where('id', $id)->update([
                'cycle_phases' => $phase === null ? null : json_encode([$phase]),
            ]);
        });

        Schema::table('articles', function (Blueprint $table) {
            // The composite index covered the column being dropped.
            $table->dropIndex(['is_published', 'cycle_phase']);
            $table->dropColumn('cycle_phase');
            $table->index('is_published');
        });
    }

    public function down(): void
    {
        Schema::table('articles', function (Blueprint $table) {
            $table->string('cycle_phase')->nullable()->after('category');
        });

        // Only the first phase survives the round trip — the column holds one.
        $this->eachRow('cycle_phases', function (int $id, $phases) {
            $decoded = json_decode((string) $phases, true);

            DB::table('articles')->where('id', $id)->update([
                'cycle_phase' => is_array($decoded) ? ($decoded[0] ?? null) : null,
            ]);
        });

        Schema::table('articles', function (Blueprint $table) {
            $table->dropIndex(['is_published']);
            $table->dropColumn('cycle_phases');
            $table->index(['is_published', 'cycle_phase']);
        });
    }

    private function eachRow(string $column, callable $handle): void
    {
        DB::table('articles')->select('id', $column)->orderBy('id')->chunk(200, function ($rows) use ($column, $handle) {
            foreach ($rows as $row) {
                $handle($row->id, $row->{$column});
            }
        });
    }
};
