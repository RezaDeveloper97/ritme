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
     * Schema-only twin of backend-go/db/migrations/00003_checkups.sql
     * (docs/go-migration/migrations.md): the periodic-checkups catalog, the
     * user's records and per-type settings, served by the Go-only
     * /api/v1/checkups and /api/admin/v1/checkup-types endpoints (T-M4-01,
     * docs/checkups/README.md). No models here.
     *
     * The default catalog rows are seeded here too, with the same values as the
     * goose migration, so `make schema-diff` row counts match. The copy needs a
     * medical review before production.
     */
    public function up(): void
    {
        Schema::create('checkup_types', function (Blueprint $table) {
            $table->id();
            $table->string('key', 64)->nullable()->unique();   // null for a user's custom checkup
            $table->foreignId('user_id')->nullable()->constrained()->cascadeOnDelete();
            $table->string('category', 20);                    // monthly|six_monthly|annual|multi_year|age_based|custom
            $table->json('title');                             // {"<lang>": "…"}
            $table->json('subtitle')->nullable();
            $table->json('why')->nullable();
            $table->string('performed_by', 10);                // self|doctor|lab|dentist
            $table->string('icon', 40)->nullable();
            $table->string('tone', 10)->default('neutral');    // rose|violet|amber|teal|green|neutral
            $table->unsignedSmallInteger('interval_months');
            $table->unsignedSmallInteger('interval_months_max')->nullable();
            $table->unsignedTinyInteger('age_min')->nullable();
            $table->unsignedTinyInteger('age_max')->nullable();
            $table->unsignedTinyInteger('cycle_day_from')->nullable();
            $table->unsignedTinyInteger('cycle_day_to')->nullable();
            $table->unsignedSmallInteger('remind_lead_days')->default(7);
            $table->json('prep_steps')->nullable();
            $table->json('guide_steps')->nullable();
            $table->json('finding_options')->nullable();
            $table->boolean('hide_in_pregnancy')->default(false);
            $table->boolean('is_active')->default(true);
            $table->integer('sort_order')->default(0);
            $table->text('source_note')->nullable();
            $table->timestamps();

            $table->index(['is_active', 'sort_order']);
        });

        Schema::create('checkup_records', function (Blueprint $table) {
            $table->id();
            $table->foreignId('user_id')->constrained()->cascadeOnDelete();
            $table->foreignId('checkup_type_id')->constrained()->cascadeOnDelete();
            $table->date('done_on');
            $table->string('result', 10);                      // normal|follow_up|pending
            $table->json('findings')->nullable();              // finding_options keys (self-exam)
            $table->text('note')->nullable();
            $table->boolean('has_attachment')->default(false); // the file itself stays on the phone
            $table->date('next_due_on')->nullable();           // user override of the computed next due
            $table->timestamps();

            $table->index(['user_id', 'checkup_type_id', 'done_on']);
        });

        Schema::create('user_checkup_settings', function (Blueprint $table) {
            $table->id();
            $table->foreignId('user_id')->constrained()->cascadeOnDelete();
            $table->foreignId('checkup_type_id')->constrained()->cascadeOnDelete();
            $table->boolean('enabled')->default(true);
            $table->boolean('remind')->default(true);
            $table->timestamps();

            $table->unique(['user_id', 'checkup_type_id']);
        });

        $now = now();

        DB::table('checkup_types')->insertOrIgnore(array_map(
            fn (array $row): array => $row + ['created_at' => $now, 'updated_at' => $now],
            [
                [
                    'key' => 'breast_self_exam',
                    'category' => 'monthly',
                    'title' => '{"fa":"خودآزمایی سینه","en":"Breast self-exam"}',
                    'subtitle' => '{"fa":"چند دقیقه در خانه، بعد از پریود","en":"A few minutes at home, after your period"}',
                    'why' => '{"fa":"وقتی حالت طبیعی سینه‌هایت را بشناسی، هر تغییری را زودتر می‌بینی. بهترین زمان چند روز بعد از پریود است؛ سینه‌ها کمتر حساس و متورم‌اند.","en":"Knowing how your breasts normally feel helps you notice any change early. The best time is a few days after your period, when breasts are less tender and swollen."}',
                    'performed_by' => 'self',
                    'icon' => 'ribbon',
                    'tone' => 'rose',
                    'interval_months' => 1,
                    'interval_months_max' => null,
                    'age_min' => null,
                    'age_max' => null,
                    'cycle_day_from' => 7,
                    'cycle_day_to' => 10,
                    'remind_lead_days' => 3,
                    'prep_steps' => '[{"fa":"روز ۷ تا ۱۰ سیکل، چند روز بعد از پریود را انتخاب کن","en":"Pick cycle day 7 to 10, a few days after your period"},{"fa":"حدود ۵ دقیقه وقت آرام و یک آینه کافی است","en":"About 5 quiet minutes and a mirror are enough"}]',
                    'guide_steps' => '[{"title":{"fa":"جلوی آینه","en":"In front of a mirror"},"body":{"fa":"با دست‌ها پایین و بعد بالا، به تغییر شکل، اندازه، فرورفتگی یا تغییر پوست نگاه کن.","en":"With your arms down and then raised, look for changes in shape, size, dimpling or skin."}},{"title":{"fa":"ایستاده یا زیر دوش","en":"Standing or in the shower"},"body":{"fa":"با سه انگشت میانی و فشار ملایم تا محکم، به‌صورت دایره‌ای کل سینه و زیر بغل را لمس کن.","en":"Using your three middle fingers and light to firm pressure, feel the whole breast and armpit in small circles."}},{"title":{"fa":"دراز کشیده","en":"Lying down"},"body":{"fa":"بالشی زیر شانه بگذار و همان حرکت را تکرار کن؛ نوک سینه را هم به‌آرامی فشار بده.","en":"Put a pillow under your shoulder and repeat the same motion; gently squeeze the nipple too."}}]',
                    'finding_options' => '[{"key":"none","exclusive":true,"label":{"fa":"چیزی متفاوت نبود","en":"Nothing different"}},{"key":"lump","label":{"fa":"توده یا سفتی","en":"Lump or thickening"}},{"key":"skin_change","label":{"fa":"تغییر پوست","en":"Skin change"}},{"key":"discharge","label":{"fa":"ترشح","en":"Discharge"}},{"key":"pain","label":{"fa":"درد","en":"Pain"}}]',
                    'hide_in_pregnancy' => true,
                    'is_active' => true,
                    'sort_order' => 1,
                    'source_note' => 'Default catalog (T-M4-01). Needs medical review before production.',
                ],
                [
                    'key' => 'clinical_breast_exam',
                    'category' => 'annual',
                    'title' => '{"fa":"معاینه بالینی سینه","en":"Clinical breast exam"}',
                    'subtitle' => '{"fa":"توسط پزشک","en":"By a doctor"}',
                    'why' => '{"fa":"پزشک یا ماما در معاینه بالینی تغییراتی را بررسی می‌کند که ممکن است در خودآزمایی دیده نشوند. معمولاً سالی یک‌بار، همراه با چکاپ عمومی انجام می‌شود.","en":"In a clinical exam a doctor or midwife checks for changes you may not notice yourself. It is usually done once a year, together with a general checkup."}',
                    'performed_by' => 'doctor',
                    'icon' => 'breast',
                    'tone' => 'violet',
                    'interval_months' => 12,
                    'interval_months_max' => null,
                    'age_min' => null,
                    'age_max' => null,
                    'cycle_day_from' => null,
                    'cycle_day_to' => null,
                    'remind_lead_days' => 30,
                    'prep_steps' => '[{"fa":"بهترین زمان حدود یک هفته بعد از پریود است","en":"The best time is about a week after your period"},{"fa":"هر تغییری که در خودآزمایی دیدی را یادداشت کن و همراه ببر","en":"Note any change you found in your self-exams and bring it along"}]',
                    'guide_steps' => null,
                    'finding_options' => null,
                    'hide_in_pregnancy' => false,
                    'is_active' => true,
                    'sort_order' => 2,
                    'source_note' => 'Default catalog (T-M4-01). Needs medical review before production.',
                ],
                [
                    'key' => 'pap_smear',
                    'category' => 'multi_year',
                    'title' => '{"fa":"پاپ‌اسمیر / HPV","en":"Pap smear / HPV"}',
                    'subtitle' => '{"fa":"غربالگری دهانه رحم","en":"Cervical screening"}',
                    'why' => '{"fa":"پاپ‌اسمیر تغییرات سلول‌های دهانه رحم را پیش از آنکه به مشکل جدی تبدیل شوند نشان می‌دهد. برای بیشتر زنان ۲۱ تا ۶۵ ساله هر ۳ سال یک‌بار توصیه می‌شود؛ همراه با تست HPV می‌تواند هر ۵ سال شود.","en":"A Pap smear shows changes in cervical cells before they become a serious problem. For most women aged 21 to 65 it is recommended every 3 years; together with an HPV test it can be every 5 years."}',
                    'performed_by' => 'doctor',
                    'icon' => 'flask',
                    'tone' => 'violet',
                    'interval_months' => 36,
                    'interval_months_max' => null,
                    'age_min' => 21,
                    'age_max' => 65,
                    'cycle_day_from' => 10,
                    'cycle_day_to' => 20,
                    'remind_lead_days' => 30,
                    'prep_steps' => '[{"fa":"بهترین زمان: وسط سیکل، نه در روزهای پریود","en":"Best time: mid-cycle, not during your period"},{"fa":"۴۸ ساعت قبل از رابطه، دوش واژینال و کرم‌های واژینال پرهیز کن","en":"Avoid sex, douching and vaginal creams for 48 hours before"},{"fa":"جواب معمولاً ۱ تا ۳ هفته بعد آماده می‌شود","en":"Results are usually ready 1 to 3 weeks later"}]',
                    'guide_steps' => null,
                    'finding_options' => null,
                    'hide_in_pregnancy' => false,
                    'is_active' => true,
                    'sort_order' => 3,
                    'source_note' => 'Default catalog (T-M4-01). Needs medical review before production.',
                ],
                [
                    'key' => 'blood_test',
                    'category' => 'annual',
                    'title' => '{"fa":"آزمایش خون کامل","en":"Full blood test"}',
                    'subtitle' => '{"fa":"CBC، تیروئید، ویتامین D، آهن","en":"CBC, thyroid, vitamin D, iron"}',
                    'why' => '{"fa":"آزمایش خون سالانه کم‌خونی، کمبود آهن و ویتامین D و مشکلات تیروئید را نشان می‌دهد که در زنان شایع‌اند و اغلب بی‌علامت شروع می‌شوند.","en":"A yearly blood test shows anaemia, low iron and vitamin D, and thyroid problems, which are common in women and often start without symptoms."}',
                    'performed_by' => 'lab',
                    'icon' => 'blood',
                    'tone' => 'amber',
                    'interval_months' => 12,
                    'interval_months_max' => null,
                    'age_min' => null,
                    'age_max' => null,
                    'cycle_day_from' => null,
                    'cycle_day_to' => null,
                    'remind_lead_days' => 14,
                    'prep_steps' => '[{"fa":"اگر پزشک گفته، ۱۰ تا ۱۲ ساعت ناشتا باش","en":"If your doctor asked, fast for 10 to 12 hours"},{"fa":"فهرست داروها و مکمل‌هایت را همراه داشته باش","en":"Bring a list of your medicines and supplements"},{"fa":"آب کافی بنوش تا خون‌گیری راحت‌تر شود","en":"Drink enough water so the blood draw is easier"}]',
                    'guide_steps' => null,
                    'finding_options' => null,
                    'hide_in_pregnancy' => false,
                    'is_active' => true,
                    'sort_order' => 4,
                    'source_note' => 'Default catalog (T-M4-01). Needs medical review before production.',
                ],
                [
                    'key' => 'dentist',
                    'category' => 'six_monthly',
                    'title' => '{"fa":"دندان‌پزشکی","en":"Dentist"}',
                    'subtitle' => '{"fa":"معاینه و جرم‌گیری","en":"Check-up and cleaning"}',
                    'why' => '{"fa":"معاینه و جرم‌گیری منظم از پوسیدگی و بیماری لثه پیشگیری می‌کند. تغییرات هورمونی سیکل و بارداری هم می‌تواند لثه‌ها را حساس‌تر کند.","en":"Regular check-ups and cleaning prevent decay and gum disease. Hormonal changes during the cycle and pregnancy can also make gums more sensitive."}',
                    'performed_by' => 'dentist',
                    'icon' => 'tooth',
                    'tone' => 'teal',
                    'interval_months' => 6,
                    'interval_months_max' => null,
                    'age_min' => null,
                    'age_max' => null,
                    'cycle_day_from' => null,
                    'cycle_day_to' => null,
                    'remind_lead_days' => 14,
                    'prep_steps' => '[{"fa":"اگر جایی از دندان یا لثه‌ات درد یا حساسیت دارد یادداشت کن","en":"Note any tooth or gum that hurts or feels sensitive"},{"fa":"اگر باردار هستی یا احتمالش را می‌دهی به دندان‌پزشک بگو","en":"Tell your dentist if you are or might be pregnant"}]',
                    'guide_steps' => null,
                    'finding_options' => null,
                    'hide_in_pregnancy' => false,
                    'is_active' => true,
                    'sort_order' => 5,
                    'source_note' => 'Default catalog (T-M4-01). Needs medical review before production.',
                ],
                [
                    'key' => 'mammography',
                    'category' => 'age_based',
                    'title' => '{"fa":"ماموگرافی","en":"Mammography"}',
                    'subtitle' => '{"fa":"غربالگری سرطان سینه","en":"Breast cancer screening"}',
                    'why' => '{"fa":"ماموگرافی می‌تواند توده‌هایی را سال‌ها پیش از آنکه لمس شوند نشان دهد. برای بیشتر زنان از ۴۰ سالگی هر ۱ تا ۲ سال توصیه می‌شود؛ با سابقه خانوادگی ممکن است پزشک زودتر شروع کند.","en":"A mammogram can show lumps years before they can be felt. For most women it is recommended every 1 to 2 years from age 40; with a family history your doctor may start earlier."}',
                    'performed_by' => 'lab',
                    'icon' => 'shieldCheck',
                    'tone' => 'rose',
                    'interval_months' => 12,
                    'interval_months_max' => 24,
                    'age_min' => 40,
                    'age_max' => null,
                    'cycle_day_from' => null,
                    'cycle_day_to' => null,
                    'remind_lead_days' => 30,
                    'prep_steps' => '[{"fa":"روز معاینه دئودورانت، پودر یا لوسیون روی سینه و زیر بغل نزن","en":"On the day, skip deodorant, powder or lotion on your breasts and underarms"},{"fa":"حدود یک هفته بعد از پریود که سینه‌ها کمتر حساس‌اند بهترین زمان است","en":"About a week after your period, when breasts are less tender, is the best time"},{"fa":"اگر ماموگرافی قبلی داری همراه ببر","en":"Bring any previous mammograms"}]',
                    'guide_steps' => null,
                    'finding_options' => null,
                    'hide_in_pregnancy' => true,
                    'is_active' => true,
                    'sort_order' => 6,
                    'source_note' => 'Default catalog (T-M4-01). Needs medical review before production.',
                ],
            ],
        ));
    }

    public function down(): void
    {
        Schema::dropIfExists('user_checkup_settings');
        Schema::dropIfExists('checkup_records');
        Schema::dropIfExists('checkup_types');
    }
};
