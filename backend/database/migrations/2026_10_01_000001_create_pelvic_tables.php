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
     * Twin of backend-go/db/migrations/00010_pelvic_floor.sql
     * (docs/go-migration/migrations.md): the pelvic floor program tables
     * written by the Go-only /api/v1/pelvic endpoints (CB-PELV-01) plus the
     * same `catalog_items` seed rows (`pelvic_levels`, `pelvic_alerts`), so
     * `make schema-diff` row counts match. No model or routes here.
     * insertOrIgnore never overwrites an admin edit.
     */
    public function up(): void
    {
        Schema::create('pelvic_programs', function (Blueprint $table) {
            $table->id();
            $table->foreignId('user_id')->unique()->constrained()->cascadeOnDelete();
            $table->date('started_on');
            $table->timestamps();
        });

        Schema::create('pelvic_sessions', function (Blueprint $table) {
            $table->id();
            $table->foreignId('user_id')->constrained()->cascadeOnDelete();
            $table->date('session_date');
            $table->unsignedSmallInteger('sessions_count')->default(0);
            $table->unsignedSmallInteger('sets_completed')->default(0);
            $table->unsignedInteger('duration_sec')->default(0);
            $table->string('level_code', 64)->nullable();
            $table->timestamps();

            $table->unique(['user_id', 'session_date']);
        });

        Schema::create('pelvic_bladder_logs', function (Blueprint $table) {
            $table->id();
            $table->foreignId('user_id')->constrained()->cascadeOnDelete();
            $table->date('log_date');
            $table->string('leak', 16)->nullable();              // none|cough|urgency|unexplained
            $table->unsignedTinyInteger('night_voids')->nullable();
            $table->json('uti_symptoms')->nullable();            // [burning|frequency|cloudy_odor]
            $table->timestamps();

            $table->unique(['user_id', 'log_date']);
        });

        $now = now();
        $cue = [
            'fa' => 'عضلاتی را منقبض کن که با آن جلوی ادرار را می‌گیری. شکم، باسن و ران‌ها شل بمانند و نفست را حبس نکن.',
            'en' => "Squeeze the muscles you use to stop the flow of urine. Keep your belly, buttocks and thighs relaxed and don't hold your breath.",
        ];
        $json = fn (?array $v) => $v === null ? null : json_encode($v, JSON_UNESCAPED_UNICODE | JSON_UNESCAPED_SLASHES);
        $row = fn (string $group, string $code, int $sort, array $title, ?array $body, array $meta) => [
            'group' => $group,
            'code' => $code,
            'sort_order' => $sort,
            'is_active' => 1,
            'audiences' => null,
            'title' => $json($title),
            'body' => $json($body),
            'meta' => $json($meta),
            'needs_review' => 1,
            'created_at' => $now,
            'updated_at' => $now,
        ];

        DB::table('catalog_items')->insertOrIgnore([
            $row('pelvic_levels', 'level_1', 1, ['fa' => 'سطح ۱', 'en' => 'Level 1'], $cue, ['week_from' => 1, 'hold_sec' => 3, 'rest_sec' => 3, 'reps' => 10, 'sets' => 3]),
            $row('pelvic_levels', 'level_2', 2, ['fa' => 'سطح ۲', 'en' => 'Level 2'], $cue, ['week_from' => 3, 'hold_sec' => 5, 'rest_sec' => 5, 'reps' => 10, 'sets' => 3]),
            $row('pelvic_levels', 'level_3', 3, ['fa' => 'سطح ۳', 'en' => 'Level 3'], $cue, ['week_from' => 5, 'hold_sec' => 8, 'rest_sec' => 8, 'reps' => 10, 'sets' => 3]),
            $row('pelvic_levels', 'level_4', 4, ['fa' => 'سطح ۴', 'en' => 'Level 4'], $cue, ['week_from' => 7, 'hold_sec' => 10, 'rest_sec' => 10, 'reps' => 10, 'sets' => 3]),
            $row('pelvic_alerts', 'uti_warning', 1, [
                'fa' => 'اگر تب، لرز یا درد پهلو داری، زودتر به پزشک مراجعه کن.',
                'en' => 'If you have a fever, chills or pain in your side, see a doctor soon.',
            ], null, ['severity' => 'urgent']),
            $row('pelvic_alerts', 'program_suitability', 2, [
                'fa' => 'مناسب بعد از زایمان، یائسگی و هر وقت نشت ادرار داری. اگر درد لگن یا سنگینی داری، اول با پزشک یا فیزیوتراپ لگن مشورت کن.',
                'en' => 'Suited to after childbirth, menopause and any time you leak urine. If you have pelvic pain or heaviness, talk to a doctor or pelvic physiotherapist first.',
            ], null, ['severity' => 'info']),
        ]);
    }

    public function down(): void
    {
        DB::table('catalog_items')->where('group', 'pelvic_levels')
            ->whereIn('code', ['level_1', 'level_2', 'level_3', 'level_4'])->delete();
        DB::table('catalog_items')->where('group', 'pelvic_alerts')
            ->whereIn('code', ['uti_warning', 'program_suitability'])->delete();
        Schema::dropIfExists('pelvic_bladder_logs');
        Schema::dropIfExists('pelvic_sessions');
        Schema::dropIfExists('pelvic_programs');
    }
};
