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
     * Twin of backend-go/db/migrations/00027_ivf.sql
     * (docs/go-migration/migrations.md): the IVF tables written by the
     * Go-only /api/v1/ivf endpoints (CB-IVF-01) plus the same `catalog_items`
     * seed rows (ivf_* groups, needs clinical review), so `make schema-diff`
     * row counts match. No model or routes here. insertOrIgnore never
     * overwrites an admin edit.
     */
    public function up(): void
    {
        Schema::create('ivf_cycles', function (Blueprint $table) {
            $table->id();
            $table->foreignId('user_id')->constrained()->cascadeOnDelete();
            $table->unsignedBigInteger('active_user_id')->nullable()->unique(); // = user_id while open
            $table->unsignedTinyInteger('number')->default(1);
            $table->string('protocol', 32)->nullable();
            $table->string('stage', 16)->default('prep');         // prep|stim|retrieval|transfer|tww|test
            $table->date('started_on');
            $table->date('stim_started_on')->nullable();
            $table->dateTime('retrieval_at')->nullable();
            $table->dateTime('transfer_at')->nullable();
            $table->date('beta_on')->nullable();
            $table->dateTime('next_scan_at')->nullable();
            $table->boolean('notify_companion')->default(false);
            $table->string('outcome', 16)->nullable();             // positive|negative|cancelled
            $table->date('outcome_on')->nullable();
            $table->timestamp('closed_at')->nullable();
            $table->timestamps();
        });

        Schema::create('ivf_meds', function (Blueprint $table) {
            $table->id();
            $table->foreignId('user_id')->constrained()->cascadeOnDelete();
            $table->foreignId('cycle_id')->constrained('ivf_cycles')->cascadeOnDelete();
            $table->foreignId('reminder_id')->unique()->constrained()->cascadeOnDelete();
            $table->string('role', 16);
            $table->string('route', 16);
            $table->dateTime('trigger_at')->nullable();
            $table->unsignedSmallInteger('stock_units')->nullable();
            $table->string('stock_unit', 24)->nullable();
            $table->unsignedSmallInteger('doses_per_unit')->default(1);
            $table->dateTime('stock_counted_at')->nullable();
            $table->timestamps();
        });

        Schema::create('ivf_dose_logs', function (Blueprint $table) {
            $table->id();
            $table->foreignId('user_id')->constrained()->cascadeOnDelete();
            $table->foreignId('ivf_med_id')->constrained('ivf_meds')->cascadeOnDelete();
            $table->date('dose_date');
            $table->char('slot', 5);
            $table->string('site', 32)->nullable();
            $table->timestamps();

            $table->unique(['ivf_med_id', 'dose_date', 'slot']);
        });

        Schema::create('ivf_scans', function (Blueprint $table) {
            $table->id();
            $table->foreignId('user_id')->constrained()->cascadeOnDelete();
            $table->foreignId('cycle_id')->constrained('ivf_cycles')->cascadeOnDelete();
            $table->date('scan_date');
            foreach (['right', 'left'] as $side) {
                foreach (['lt_10', '10_14', '15_17', '18_plus'] as $bin) {
                    $table->unsignedTinyInteger("{$side}_{$bin}")->default(0);
                }
            }
            $table->decimal('endometrium_mm', 4, 1)->nullable();
            $table->decimal('e2', 8, 2)->nullable();
            $table->string('e2_unit', 8)->nullable();
            $table->string('notes', 500)->nullable();
            $table->timestamps();

            $table->unique(['cycle_id', 'scan_date']);
        });

        Schema::create('ivf_tww_logs', function (Blueprint $table) {
            $table->id();
            $table->foreignId('user_id')->constrained()->cascadeOnDelete();
            $table->foreignId('cycle_id')->constrained('ivf_cycles')->cascadeOnDelete();
            $table->date('log_date');
            $table->string('mood', 16);
            $table->timestamps();

            $table->unique(['cycle_id', 'log_date']);
        });

        Schema::create('ivf_reminders', function (Blueprint $table) {
            $table->id();
            $table->foreignId('user_id')->constrained()->cascadeOnDelete();
            $table->foreignId('cycle_id')->constrained('ivf_cycles')->cascadeOnDelete();
            $table->string('kind', 16);                            // scan|retrieval|transfer|beta
            $table->foreignId('reminder_id')->constrained()->cascadeOnDelete();
            $table->timestamps();

            $table->unique(['cycle_id', 'kind']);
        });

        $now = now();
        $json = fn (?array $v) => $v === null ? null : json_encode($v, JSON_UNESCAPED_UNICODE | JSON_UNESCAPED_SLASHES);
        $row = fn (string $group, string $code, int $sort, array $title, ?array $body, ?array $meta) => [
            'group' => $group,
            'code' => $code,
            'sort_order' => $sort,
            'is_active' => 1,
            'audiences' => $json(['ttc']),
            'title' => $json($title),
            'body' => $json($body),
            'meta' => $json($meta),
            'needs_review' => 1,
            'created_at' => $now,
            'updated_at' => $now,
        ];

        DB::table('catalog_items')->insertOrIgnore([
            $row('ivf_stages', 'prep', 1, ['fa' => 'آماده‌سازی', 'en' => 'Preparation'], ['fa' => 'آزمایش‌ها و سونوی پایه', 'en' => 'Tests and a baseline scan'], null),
            $row('ivf_stages', 'stim', 2, ['fa' => 'تحریک تخمک‌گذاری', 'en' => 'Ovarian stimulation'], ['fa' => 'تزریق روزانه · حدود ۱۰ تا ۱۲ روز', 'en' => 'Daily injections · about 10 to 12 days'], null),
            $row('ivf_stages', 'retrieval', 3, ['fa' => 'تخمک‌کشی', 'en' => 'Egg retrieval'], ['fa' => 'حدود ۳۶ ساعت بعد از تزریق تریگر، طبق برنامه کلینیک', 'en' => 'About 36 hours after the trigger shot, as your clinic schedules it'], null),
            $row('ivf_stages', 'transfer', 4, ['fa' => 'انتقال جنین', 'en' => 'Embryo transfer'], ['fa' => 'تاریخ را کلینیک بر اساس رشد جنین تعیین می‌کند', 'en' => 'Your clinic sets the date from how the embryos develop'], null),
            $row('ivf_stages', 'tww', 5, ['fa' => 'انتظار دو هفته‌ای', 'en' => 'Two-week wait'], ['fa' => 'داروها را طبق نسخه ادامه بده', 'en' => 'Keep taking your medicines as prescribed'], null),
            $row('ivf_stages', 'test', 6, ['fa' => 'تست بارداری', 'en' => 'Pregnancy test'], ['fa' => 'آزمایش خون بتا در تاریخی که کلینیک گفته', 'en' => 'A beta blood test on the day your clinic gave you'], null),
            $row('ivf_protocols', 'antagonist', 1, ['fa' => 'آنتاگونیست', 'en' => 'Antagonist'], ['fa' => 'رایج‌ترین پروتکل؛ تحریک از روزهای اول سیکل و داروی جلوگیری از تخمک‌گذاری زودرس از حدود روز پنجم تحریک', 'en' => 'The most common protocol: stimulation from early in the cycle, with a medicine that prevents early ovulation from about day 5 of stimulation'], null),
            $row('ivf_protocols', 'long_agonist', 2, ['fa' => 'آگونیست طولانی', 'en' => 'Long agonist'], ['fa' => 'سرکوب هورمونی از سیکل قبل، بعد تحریک', 'en' => 'Hormones are switched off from the cycle before, then stimulation starts'], null),
            $row('ivf_protocols', 'short_agonist', 3, ['fa' => 'آگونیست کوتاه', 'en' => 'Short agonist'], ['fa' => 'آگونیست و تحریک تقریباً هم‌زمان شروع می‌شوند', 'en' => 'The agonist and stimulation start at about the same time'], null),
            $row('ivf_protocols', 'mild', 4, ['fa' => 'تحریک ملایم', 'en' => 'Mild stimulation'], ['fa' => 'دوز کمتر دارو و معمولاً تخمک کمتر', 'en' => 'Lower doses and usually fewer eggs'], null),
            $row('ivf_protocols', 'natural', 5, ['fa' => 'سیکل طبیعی', 'en' => 'Natural cycle'], ['fa' => 'بدون تحریک یا با داروی خیلی کم', 'en' => 'No or very little stimulation'], null),
            $row('ivf_protocols', 'frozen_transfer', 6, ['fa' => 'انتقال جنین منجمد', 'en' => 'Frozen embryo transfer'], ['fa' => 'آماده‌سازی رحم برای انتقال جنین فریزشده', 'en' => 'Preparing the womb lining to transfer a frozen embryo'], null),
            $row('ivf_injection_sites', 'abdomen_upper_right', 1, ['fa' => 'شکم · راست بالا', 'en' => 'Abdomen · upper right'], ['fa' => 'حداقل ۵ سانت دور از ناف؛ روی کبودی یا جای قبلی نزن', 'en' => 'At least 5 cm away from the navel; avoid bruises and the last spot'], ['region' => 'abdomen', 'side' => 'right']),
            $row('ivf_injection_sites', 'abdomen_upper_left', 2, ['fa' => 'شکم · چپ بالا', 'en' => 'Abdomen · upper left'], ['fa' => 'حداقل ۵ سانت دور از ناف؛ روی کبودی یا جای قبلی نزن', 'en' => 'At least 5 cm away from the navel; avoid bruises and the last spot'], ['region' => 'abdomen', 'side' => 'left']),
            $row('ivf_injection_sites', 'thigh_right', 3, ['fa' => 'ران · راست', 'en' => 'Thigh · right'], ['fa' => 'جلو و بیرون ران، میانه فاصله زانو تا لگن', 'en' => 'Front and outer thigh, halfway between knee and hip'], ['region' => 'thigh', 'side' => 'right']),
            $row('ivf_injection_sites', 'thigh_left', 4, ['fa' => 'ران · چپ', 'en' => 'Thigh · left'], ['fa' => 'جلو و بیرون ران، میانه فاصله زانو تا لگن', 'en' => 'Front and outer thigh, halfway between knee and hip'], ['region' => 'thigh', 'side' => 'left']),
            $row('ivf_injection_sites', 'abdomen_lower_right', 5, ['fa' => 'شکم · راست پایین', 'en' => 'Abdomen · lower right'], ['fa' => 'حداقل ۵ سانت دور از ناف؛ روی کبودی یا جای قبلی نزن', 'en' => 'At least 5 cm away from the navel; avoid bruises and the last spot'], ['region' => 'abdomen', 'side' => 'right']),
            $row('ivf_injection_sites', 'abdomen_lower_left', 6, ['fa' => 'شکم · چپ پایین', 'en' => 'Abdomen · lower left'], ['fa' => 'حداقل ۵ سانت دور از ناف؛ روی کبودی یا جای قبلی نزن', 'en' => 'At least 5 cm away from the navel; avoid bruises and the last spot'], ['region' => 'abdomen', 'side' => 'left']),
            $row('ivf_injection_sites', 'arm_right', 7, ['fa' => 'بازو · راست', 'en' => 'Arm · right'], ['fa' => 'پشت بازو؛ معمولاً وقتی کسی کمکت می‌کند', 'en' => 'Back of the upper arm; usually when someone helps you'], ['region' => 'arm', 'side' => 'right']),
            $row('ivf_injection_sites', 'arm_left', 8, ['fa' => 'بازو · چپ', 'en' => 'Arm · left'], ['fa' => 'پشت بازو؛ معمولاً وقتی کسی کمکت می‌کند', 'en' => 'Back of the upper arm; usually when someone helps you'], ['region' => 'arm', 'side' => 'left']),
            $row('ivf_med_presets', 'fsh', 1, ['fa' => 'هورمون تحریک (FSH)', 'en' => 'Stimulation hormone (FSH)'], null, ['role' => 'stimulation', 'route' => 'subcutaneous', 'unit' => 'iu', 'times' => ['20:00'], 'stock_unit' => 'pen']),
            $row('ivf_med_presets', 'hmg', 2, ['fa' => 'hMG (FSH + LH)', 'en' => 'hMG (FSH + LH)'], null, ['role' => 'stimulation', 'route' => 'subcutaneous', 'unit' => 'iu', 'times' => ['20:00'], 'stock_unit' => 'vial']),
            $row('ivf_med_presets', 'gnrh_antagonist', 3, ['fa' => 'داروی جلوگیری از تخمک‌گذاری زودرس (آنتاگونیست)', 'en' => 'Medicine to prevent early ovulation (antagonist)'], null, ['role' => 'suppression', 'route' => 'subcutaneous', 'unit' => 'mg', 'times' => ['08:00'], 'stock_unit' => 'prefilled_syringe']),
            $row('ivf_med_presets', 'gnrh_agonist', 4, ['fa' => 'آگونیست GnRH', 'en' => 'GnRH agonist'], null, ['role' => 'suppression', 'route' => 'subcutaneous', 'unit' => 'mg', 'times' => ['08:00'], 'stock_unit' => 'vial']),
            $row('ivf_med_presets', 'hcg_trigger', 5, ['fa' => 'تزریق تریگر (hCG)', 'en' => 'Trigger shot (hCG)'], null, ['role' => 'trigger', 'route' => 'subcutaneous', 'unit' => 'iu', 'stock_unit' => 'prefilled_syringe']),
            $row('ivf_med_presets', 'agonist_trigger', 6, ['fa' => 'تزریق تریگر (آگونیست)', 'en' => 'Trigger shot (agonist)'], null, ['role' => 'trigger', 'route' => 'subcutaneous', 'unit' => 'mg', 'stock_unit' => 'prefilled_syringe']),
            $row('ivf_med_presets', 'progesterone_vaginal', 7, ['fa' => 'پروژسترون واژینال', 'en' => 'Vaginal progesterone'], null, ['role' => 'luteal_support', 'route' => 'vaginal', 'unit' => 'mg', 'times' => ['08:00', '20:00'], 'stock_unit' => 'box']),
            $row('ivf_med_presets', 'progesterone_im', 8, ['fa' => 'پروژسترون تزریقی', 'en' => 'Progesterone injection'], null, ['role' => 'luteal_support', 'route' => 'intramuscular', 'unit' => 'mg', 'times' => ['20:00'], 'stock_unit' => 'ampoule']),
            $row('ivf_med_presets', 'estradiol', 9, ['fa' => 'استرادیول', 'en' => 'Estradiol'], null, ['role' => 'luteal_support', 'route' => 'oral', 'unit' => 'mg', 'times' => ['08:00', '20:00'], 'stock_unit' => 'box']),
            $row('ivf_guidance', 'trigger_timing', 1, ['fa' => 'تزریق تریگر (آخرین تزریق)', 'en' => 'Trigger shot (the last injection)'], ['fa' => 'وقتی پزشک اعلام کرد، دقیقاً در همان ساعت تزریق کن؛ زمان آن برای تخمک‌کشی مهم است.', 'en' => 'Inject at exactly the time your doctor gives you; its timing matters for the egg retrieval.'], ['placement' => 'meds']),
            $row('ivf_guidance', 'site_rotation', 2, ['fa' => 'محل تزریق را عوض کن', 'en' => 'Change the injection spot'], ['fa' => 'هر بار جای دیگری تزریق کن تا پوست تحریک و کبود نشود. در شکم حداقل ۵ سانت از ناف فاصله بگیر.', 'en' => 'Use a different spot each time so the skin doesn\'t get sore or bruised. On the abdomen keep at least 5 cm from the navel.'], ['placement' => 'meds']),
            $row('ivf_guidance', 'scan_interpretation', 3, ['fa' => 'تفسیر با پزشک است', 'en' => 'Your doctor interprets it'], ['fa' => 'تفسیر و تصمیم درباره زمان تخمک‌کشی با پزشکت است. ما فقط کمک می‌کنیم همه‌چیز یک‌جا ثبت شود.', 'en' => 'Interpreting the scan and deciding when to retrieve is your doctor\'s call. We only help you keep everything in one place.'], ['placement' => 'scan']),
            $row('ivf_guidance', 'tww_feelings', 4, ['fa' => 'هر حسی داری طبیعی است', 'en' => 'Whatever you feel is normal'], ['fa' => 'هر حسی داری طبیعی است. تست خانگی زودتر از موعد می‌تواند گمراه‌کننده باشد.', 'en' => 'Whatever you feel is normal. A home test taken too early can be misleading.'], ['placement' => 'tww']),
            $row('ivf_guidance', 'early_test', 5, ['fa' => 'تست خانگی زودهنگام', 'en' => 'Testing early at home'], ['fa' => 'داروهای تریگر و پروژسترون می‌توانند جواب تست خانگی را در روزهای اول اشتباه نشان دهند؛ منتظر آزمایش خون بتا بمان.', 'en' => 'Trigger and progesterone medicines can make an early home test misleading; wait for the beta blood test.'], ['placement' => 'tww']),
            $row('ivf_danger_signs', 'ohss', 1, ['fa' => 'اگر این‌ها را داشتی فوراً به پزشک خبر بده', 'en' => 'Tell your doctor straight away if you have'], ['fa' => 'نفخ شدید و سریع شکم، تنگی نفس، کم شدن ادرار، درد شدید شکم، یا خونریزی زیاد.', 'en' => 'Severe or fast-growing bloating, shortness of breath, passing much less urine, severe tummy pain, or heavy bleeding.'], ['severity' => 'urgent', 'placement' => ['home', 'tww'], 'hotlines' => ['115']]),
            $row('ivf_danger_signs', 'fever_after_procedure', 2, ['fa' => 'تب بعد از تخمک‌کشی یا انتقال', 'en' => 'Fever after retrieval or transfer'], ['fa' => 'تب، لرز یا ترشح بدبو بعد از تخمک‌کشی یا انتقال جنین را همان روز به کلینیک بگو.', 'en' => 'Report a fever, chills or smelly discharge after the retrieval or transfer to your clinic the same day.'], ['severity' => 'urgent', 'placement' => ['home', 'tww'], 'hotlines' => ['115']]),
        ]);
    }

    public function down(): void
    {
        DB::table('catalog_items')->where('group', 'ivf_stages')
            ->whereIn('code', ['prep', 'stim', 'retrieval', 'transfer', 'tww', 'test'])->delete();
        DB::table('catalog_items')->where('group', 'ivf_protocols')
            ->whereIn('code', ['antagonist', 'long_agonist', 'short_agonist', 'mild', 'natural', 'frozen_transfer'])->delete();
        DB::table('catalog_items')->where('group', 'ivf_injection_sites')
            ->whereIn('code', ['abdomen_upper_right', 'abdomen_upper_left', 'thigh_right', 'thigh_left', 'abdomen_lower_right', 'abdomen_lower_left', 'arm_right', 'arm_left'])->delete();
        DB::table('catalog_items')->where('group', 'ivf_med_presets')
            ->whereIn('code', ['fsh', 'hmg', 'gnrh_antagonist', 'gnrh_agonist', 'hcg_trigger', 'agonist_trigger', 'progesterone_vaginal', 'progesterone_im', 'estradiol'])->delete();
        DB::table('catalog_items')->where('group', 'ivf_guidance')
            ->whereIn('code', ['trigger_timing', 'site_rotation', 'scan_interpretation', 'tww_feelings', 'early_test'])->delete();
        DB::table('catalog_items')->where('group', 'ivf_danger_signs')
            ->whereIn('code', ['ohss', 'fever_after_procedure'])->delete();
        Schema::dropIfExists('ivf_reminders');
        Schema::dropIfExists('ivf_tww_logs');
        Schema::dropIfExists('ivf_scans');
        Schema::dropIfExists('ivf_dose_logs');
        Schema::dropIfExists('ivf_meds');
        Schema::dropIfExists('ivf_cycles');
    }
};
