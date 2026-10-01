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
     * Twin of backend-go/db/migrations/00023_condition_programs.sql
     * (docs/go-migration/migrations.md): the condition-program tables written
     * by the Go-only /api/v1/conditions endpoints (CB-COND-01) plus the same
     * `catalog_items` seed rows (condition_programs, pain_types,
     * pain_associated, pmdd_items, condition_alerts — needs clinical review),
     * so `make schema-diff` row counts match. No model or routes here.
     * insertOrIgnore never overwrites an admin edit.
     */
    public function up(): void
    {
        Schema::create('condition_enrolments', function (Blueprint $table) {
            $table->id();
            $table->foreignId('user_id')->constrained()->cascadeOnDelete();
            $table->string('program', 32);                       // endo|pmdd|heavy_bleeding|pcos
            $table->date('enrolled_on');
            $table->timestamps();

            $table->unique(['user_id', 'program']);
        });

        Schema::create('condition_pain_entries', function (Blueprint $table) {
            $table->id();
            $table->foreignId('user_id')->constrained()->cascadeOnDelete();
            $table->date('entry_date');
            $table->json('pain_types')->nullable();              // catalog pain_types codes
            $table->json('associated')->nullable();              // catalog pain_associated codes without a log slot
            $table->boolean('missed_activity')->nullable();
            $table->string('analgesic', 100)->nullable();
            $table->string('analgesic_time', 5)->nullable();     // HH:MM
            $table->string('analgesic_effect', 16)->nullable();  // no|a_little|helped
            $table->timestamps();

            $table->unique(['user_id', 'entry_date']);
        });

        Schema::create('pmdd_entries', function (Blueprint $table) {
            $table->id();
            $table->foreignId('user_id')->constrained()->cascadeOnDelete();
            $table->date('entry_date');
            $table->json('scores');                              // {pmdd_items code: 1–6}
            $table->timestamps();

            $table->unique(['user_id', 'entry_date']);
        });

        Schema::create('pbac_entries', function (Blueprint $table) {
            $table->id();
            $table->foreignId('user_id')->constrained()->cascadeOnDelete();
            $table->date('entry_date');
            $table->unsignedTinyInteger('light_count')->default(0);
            $table->unsignedTinyInteger('medium_count')->default(0);
            $table->unsignedTinyInteger('heavy_count')->default(0);
            $table->boolean('flooding')->default(false);
            $table->timestamps();

            $table->unique(['user_id', 'entry_date']);
        });

        $now = now();
        $json = fn (?array $v) => $v === null ? null : json_encode($v, JSON_UNESCAPED_UNICODE | JSON_UNESCAPED_SLASHES);
        $row = fn (string $group, string $code, int $sort, array $title, ?array $body, ?array $meta) => [
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
            $row('condition_programs', 'endo', 1, ['fa' => 'اندومتریوز', 'en' => 'Endometriosis'], ['fa' => 'دفترچه درد با محل و شدت', 'en' => 'A pain diary with location and intensity'], ['logs' => ['fa' => 'محل درد، شدت، نوع درد، اثر دارو', 'en' => 'pain location, intensity, type of pain, painkiller effect']]),
            $row('condition_programs', 'pmdd', 2, ['fa' => 'اختلال شدید پیش از قاعدگی (PMDD)', 'en' => 'Premenstrual dysphoric disorder (PMDD)'], ['fa' => 'پرسشنامه روزانه خلق برای ۲ سیکل', 'en' => 'A daily mood questionnaire for 2 cycles'], ['logs' => ['fa' => 'خلق، اضطراب، تحریک‌پذیری، تمرکز', 'en' => 'mood, anxiety, irritability, concentration']]),
            $row('condition_programs', 'heavy_bleeding', 3, ['fa' => 'خونریزی شدید', 'en' => 'Heavy bleeding'], ['fa' => 'جدول امتیاز خونریزی و خطر کم‌خونی', 'en' => 'A bleeding score chart and anaemia risk'], ['logs' => ['fa' => 'تعداد و میزان خیس شدن نوار، لخته', 'en' => 'number of pads and how soaked they are, clots']]),
            $row('condition_programs', 'pcos', 4, ['fa' => 'سندرم تخمدان پلی‌کیستیک', 'en' => 'Polycystic ovary syndrome (PCOS)'], ['fa' => 'نظم سیکل، پوست و مو، وزن و قند', 'en' => 'Cycle regularity, skin and hair, weight and blood sugar'], ['logs' => ['fa' => 'فاصله پریودها، آکنه، موهای زائد، وزن', 'en' => 'time between periods, acne, unwanted hair, weight']]),
            $row('pain_types', 'cramping', 1, ['fa' => 'گرفتگی', 'en' => 'Cramping'], null, null),
            $row('pain_types', 'stabbing', 2, ['fa' => 'تیر کشنده', 'en' => 'Stabbing'], null, null),
            $row('pain_types', 'burning', 3, ['fa' => 'سوزشی', 'en' => 'Burning'], null, null),
            $row('pain_types', 'dull_heavy', 4, ['fa' => 'مبهم و سنگین', 'en' => 'Dull and heavy'], null, null),
            $row('pain_associated', 'dyspareunia', 1, ['fa' => 'درد در رابطه', 'en' => 'Pain during sex'], null, ['log' => 'sex.symptoms.pain_during_intercourse']),
            $row('pain_associated', 'dyschezia', 2, ['fa' => 'درد هنگام اجابت مزاج', 'en' => 'Pain with bowel movements'], null, null),
            $row('pain_associated', 'dysuria', 3, ['fa' => 'درد هنگام ادرار', 'en' => 'Pain when urinating'], null, ['log' => 'urogenital.symptoms.urination_burning']),
            $row('pain_associated', 'bloating', 4, ['fa' => 'نفخ', 'en' => 'Bloating'], null, ['log' => 'symptoms.digestive.bloating']),
            $row('pain_associated', 'nausea', 5, ['fa' => 'تهوع', 'en' => 'Nausea'], null, ['log' => 'symptoms.digestive.nausea']),
            $row('pmdd_items', 'sadness', 1, ['fa' => 'غمگینی یا ناامیدی', 'en' => 'Sadness or hopelessness'], null, null),
            $row('pmdd_items', 'anxiety', 2, ['fa' => 'اضطراب یا تنش', 'en' => 'Anxiety or tension'], null, null),
            $row('pmdd_items', 'mood_swings', 3, ['fa' => 'نوسان خلق و زودرنجی', 'en' => 'Mood swings or feeling easily hurt'], null, null),
            $row('pmdd_items', 'anger', 4, ['fa' => 'عصبانیت یا درگیری با دیگران', 'en' => 'Anger or conflicts with others'], null, null),
            $row('pmdd_items', 'loss_of_interest', 5, ['fa' => 'بی‌علاقگی به کارهای معمول', 'en' => 'Less interest in usual activities'], null, null),
            $row('pmdd_items', 'concentration', 6, ['fa' => 'سختی در تمرکز', 'en' => 'Difficulty concentrating'], null, null),
            $row('condition_alerts', 'not_a_diagnosis', 1, ['fa' => 'این برنامه‌ها تشخیص نمی‌دهند', 'en' => 'These programs do not diagnose'], ['fa' => 'کمک می‌کنند با داده دقیق نزد پزشک بروی.', 'en' => 'They help you see your doctor with accurate data.'], ['severity' => 'info']),
            $row('condition_alerts', 'pain_scale', 2, ['fa' => 'شدت درد از ۰ تا ۱۰', 'en' => 'Pain from 0 to 10'], ['fa' => '۰ یعنی بدون درد و ۱۰ بدترین درد ممکن. درد شدید یا ناگهانی، تب یا غش را همان روز به پزشک بگو.', 'en' => '0 means no pain and 10 the worst pain possible. Tell a doctor the same day about severe or sudden pain, fever or fainting.'], ['severity' => 'info']),
            $row('condition_alerts', 'pbac_scoring', 3, ['fa' => 'امتیاز خونریزی چطور حساب می‌شود؟', 'en' => 'How is the bleeding score counted?'], ['fa' => 'هر نوار کمی خیس ۱ امتیاز، نیمه خیس ۵ و کاملاً خیس ۲۰ امتیاز دارد؛ لخته کوچک ۱، لخته بزرگ ۵ و نشت از نوار به لباس ۵ امتیاز. جمع یک پریود بالای ۱۰۰ معمولاً یعنی خونریزی شدید.', 'en' => 'Each lightly soaked pad scores 1, half soaked 5 and fully soaked 20; a small clot 1, a large clot 5 and leaking through to clothes 5. A period total over 100 usually means heavy bleeding.'], ['severity' => 'info']),
            $row('condition_alerts', 'pbac_over_100', 4, ['fa' => 'امتیاز این پریود از ۱۰۰ بیشتر شده', 'en' => 'This period\'s score is over 100'], ['fa' => 'معمولاً یعنی خونریزی شدید است. به پزشک بگو و درباره آزمایش کم‌خونی (هموگلوبین و فریتین) بپرس.', 'en' => 'This usually means heavy bleeding. Tell your doctor and ask about an anaemia test (haemoglobin and ferritin).'], ['severity' => 'caution']),
            $row('condition_alerts', 'pmdd_needs_two_cycles', 5, ['fa' => 'برای جمع‌بندی، ۲ سیکل ثبت کامل لازم است', 'en' => '2 fully logged cycles are needed for a summary'], ['fa' => 'یک سیکل کامل یعنی دست‌کم ۷ روز از ۱۰ روز آخر سیکل و ۴ روز از روزهای ۴ تا ۱۰ سیکل را ثبت کرده باشی.', 'en' => 'A cycle counts as fully logged when you rated at least 7 of its last 10 days and 4 of its days 4 to 10.'], ['severity' => 'info']),
            $row('condition_alerts', 'pmdd_pattern_luteal', 6, ['fa' => 'الگوی تو', 'en' => 'Your pattern'], ['fa' => 'علائم در ۱۰ روز آخر سیکل بالا می‌رود و با شروع پریود کم می‌شود.', 'en' => 'Symptoms rise in the last 10 days of the cycle and ease when your period starts.'], ['severity' => 'info']),
            $row('condition_alerts', 'pmdd_pattern_unclear', 7, ['fa' => 'الگوی تو', 'en' => 'Your pattern'], ['fa' => 'هنوز الگوی روشنی بین ۱۰ روز آخر سیکل و هفته بعد از پریود دیده نمی‌شود. ثبت روزانه را ادامه بده.', 'en' => 'There is no clear difference yet between the last 10 days of the cycle and the week after your period. Keep rating every day.'], ['severity' => 'info']),
            $row('condition_alerts', 'pmdd_not_enough_data', 8, ['fa' => 'هنوز داده کافی نیست', 'en' => 'Not enough data yet'], ['fa' => 'برای دیدن الگو، هم در هفته بعد از پریود و هم در ۱۰ روز آخر سیکل هر روز ثبت کن.', 'en' => 'To see a pattern, rate every day in the week after your period and in the last 10 days of the cycle.'], ['severity' => 'info']),
            $row('condition_alerts', 'pmdd_crisis', 9, ['fa' => 'اگر به آسیب زدن به خودت فکر می‌کنی', 'en' => 'If you are thinking about hurting yourself'], ['fa' => 'همین حالا با صدای مشاور ۱۴۸۰ یا اورژانس اجتماعی ۱۲۳ تماس بگیر.', 'en' => 'Call the 1480 counselling line or the 123 social emergency line right now.'], ['severity' => 'urgent', 'hotlines' => [['number' => '1480', 'label' => ['fa' => 'صدای مشاور', 'en' => 'Counselling line']], ['number' => '123', 'label' => ['fa' => 'اورژانس اجتماعی', 'en' => 'Social emergency']]]]),
        ]);
    }

    public function down(): void
    {
        $seeded = [
            'condition_programs' => ['endo', 'pmdd', 'heavy_bleeding', 'pcos'],
            'pain_types' => ['cramping', 'stabbing', 'burning', 'dull_heavy'],
            'pain_associated' => ['dyspareunia', 'dyschezia', 'dysuria', 'bloating', 'nausea'],
            'pmdd_items' => ['sadness', 'anxiety', 'mood_swings', 'anger', 'loss_of_interest', 'concentration'],
            'condition_alerts' => ['not_a_diagnosis', 'pain_scale', 'pbac_scoring', 'pbac_over_100', 'pmdd_needs_two_cycles',
                'pmdd_pattern_luteal', 'pmdd_pattern_unclear', 'pmdd_not_enough_data', 'pmdd_crisis'],
        ];
        foreach ($seeded as $group => $codes) {
            DB::table('catalog_items')->where('group', $group)->whereIn('code', $codes)->delete();
        }
        Schema::dropIfExists('pbac_entries');
        Schema::dropIfExists('pmdd_entries');
        Schema::dropIfExists('condition_pain_entries');
        Schema::dropIfExists('condition_enrolments');
    }
};
