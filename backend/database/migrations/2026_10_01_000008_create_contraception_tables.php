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
     * Twin of backend-go/db/migrations/00017_contraception.sql
     * (docs/go-migration/migrations.md): the contraception tables written by
     * the Go-only /api/v1/contraception endpoints (CB-CONTRA-01) plus the same
     * `catalog_items` seed rows (`missed_pill_rules`, needs clinical review),
     * so `make schema-diff` row counts match. No model or routes here.
     * insertOrIgnore never overwrites an admin edit.
     */
    public function up(): void
    {
        Schema::create('contraception_methods', function (Blueprint $table) {
            $table->id();
            $table->foreignId('user_id')->unique()->constrained()->cascadeOnDelete();
            $table->string('method', 32);
            $table->string('pack_type', 8)->nullable();          // 21_7|28|24_4
            $table->date('pack_started_on')->nullable();
            $table->unsignedTinyInteger('packs_left')->nullable();
            $table->date('packs_counted_on')->nullable();
            $table->date('inserted_on')->nullable();
            $table->unsignedTinyInteger('iud_lifetime_years')->nullable();
            $table->boolean('followup_done')->default(false);
            $table->date('injected_on')->nullable();
            $table->date('replace_on')->nullable();
            $table->timestamps();
        });

        Schema::create('contraception_pill_logs', function (Blueprint $table) {
            $table->id();
            $table->foreignId('user_id')->constrained()->cascadeOnDelete();
            $table->date('log_date');
            $table->string('status', 8);                         // taken|missed
            $table->dateTime('logged_at');
            $table->timestamps();

            $table->unique(['user_id', 'log_date']);
        });

        Schema::create('contraception_reminders', function (Blueprint $table) {
            $table->id();
            $table->foreignId('user_id')->constrained()->cascadeOnDelete();
            $table->string('kind', 32);
            $table->foreignId('reminder_id')->constrained()->cascadeOnDelete();
            $table->timestamps();

            $table->unique(['user_id', 'kind']);
        });

        $now = now();
        $json = fn (?array $v) => $v === null ? null : json_encode($v, JSON_UNESCAPED_UNICODE | JSON_UNESCAPED_SLASHES);
        $row = fn (string $code, int $sort, array $title, array $body, array $meta) => [
            'group' => 'missed_pill_rules',
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
        $combinedNote = [
            'fa' => 'این راهنمای عمومی قرص ترکیبی است. در صورت شک با پزشک یا داروساز صحبت کن.',
            'en' => 'This is general guidance for the combined pill. If in doubt, talk to a doctor or pharmacist.',
        ];
        $continue = ['fa' => 'بقیه قرص‌ها را طبق معمول ادامه بده.', 'en' => 'Carry on with the rest of the pack as usual.'];

        DB::table('catalog_items')->insertOrIgnore([
            $row('combined_one', 1, [
                'fa' => '۱ قرص (کمتر از ۴۸ ساعت دیر)',
                'en' => '1 pill (less than 48 hours late)',
            ], $combinedNote, [
                'methods' => ['combined_pill'],
                'missed' => 1,
                'severity' => 'caution',
                'steps' => [
                    ['fa' => 'قرص جاافتاده را همین حالا بخور، حتی اگر یعنی امروز دو قرص بخوری.', 'en' => 'Take the missed pill now, even if it means taking two pills today.'],
                    $continue,
                ],
            ]),
            $row('combined_two_plus', 2, ['fa' => '۲ قرص یا بیشتر', 'en' => '2 or more pills'], $combinedNote, [
                'methods' => ['combined_pill'],
                'missed' => 2,
                'severity' => 'caution',
                'steps' => [
                    ['fa' => 'آخرین قرص جاافتاده را همین حالا بخور، حتی اگر یعنی امروز دو قرص بخوری.', 'en' => 'Take the last missed pill now, even if it means taking two pills today.'],
                    $continue,
                    ['fa' => 'تا ۷ روز پشت سر هم قرص نخورده‌ای، از کاندوم استفاده کن.', 'en' => 'Use condoms until you have taken 7 pills in a row.'],
                    ['fa' => 'اگر در ۷ روز آخر قرص‌های فعال هستی، بعد از تمام شدنشان روزهای استراحت را حذف کن و بسته بعد را مستقیم شروع کن.', 'en' => 'If you are in the last 7 active pills, skip the break after them and start the next pack straight away.'],
                ],
            ]),
            $row('week1_unprotected', 3, [
                'fa' => 'اگر در هفته اول بسته رابطه محافظت‌نشده داشتی',
                'en' => 'If you had unprotected sex in the first week of the pack',
            ], [
                'fa' => 'ممکن است به پیشگیری اضطراری نیاز داشته باشی. هر چه زودتر با پزشک یا داروساز مشورت کن.',
                'en' => 'You may need emergency contraception. Talk to a doctor or pharmacist as soon as possible.',
            ], ['methods' => ['combined_pill'], 'severity' => 'urgent', 'pack_week' => 1]),
            $row('progestin_note', 4, ['fa' => 'قرص تک‌هورمونی', 'en' => 'Progestogen-only pill'], [
                'fa' => 'برای قرص تک‌هورمونی قواعد فرق دارد و به نوع قرص بستگی دارد؛ در صورت شک با پزشک یا داروساز صحبت کن.',
                'en' => 'The rules differ for the progestogen-only pill and depend on the type of pill; if in doubt, talk to a doctor or pharmacist.',
            ], [
                'methods' => ['progestin_pill'],
                'severity' => 'caution',
                'steps' => [
                    ['fa' => 'قرص جاافتاده را به محض یادآوری بخور و قرص بعدی را سر ساعت همیشگی بخور.', 'en' => 'Take the missed pill as soon as you remember and the next one at the usual time.'],
                    ['fa' => 'اگر بیشتر از زمان مجاز نوع قرصت دیر کردی، تا ۲ روز از کاندوم استفاده کن.', 'en' => 'If you are later than your pill type allows, use condoms for the next 2 days.'],
                ],
            ]),
        ]);
    }

    public function down(): void
    {
        DB::table('catalog_items')->where('group', 'missed_pill_rules')
            ->whereIn('code', ['combined_one', 'combined_two_plus', 'week1_unprotected', 'progestin_note'])->delete();
        Schema::dropIfExists('contraception_reminders');
        Schema::dropIfExists('contraception_pill_logs');
        Schema::dropIfExists('contraception_methods');
    }
};
