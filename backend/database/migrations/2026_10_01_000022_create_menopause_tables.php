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
     * Twin of backend-go/db/migrations/00022_menopause.sql
     * (docs/go-migration/migrations.md): the menopause tables written by the
     * Go-only /api/v1/menopause endpoints (CB-MENO-01..03), the
     * checkup_types.audiences column, and the same `catalog_items` /
     * `checkup_types` seed rows (needs clinical review), so `make schema-diff`
     * row counts match. No models or routes here. insertOrIgnore never
     * overwrites an admin edit. Seed rows are generated from one source for
     * both migrations (docs/canvas-build/menopause.md).
     */
    public function up(): void
    {
        Schema::table('checkup_types', function (Blueprint $table) {
            $table->json('audiences')->nullable()->after('hide_in_pregnancy'); // life modes, null = everyone
        });

        Schema::create('hot_flashes', function (Blueprint $table) {
            $table->id();
            $table->foreignId('user_id')->constrained()->cascadeOnDelete();
            $table->dateTime('started_at');
            $table->unsignedInteger('duration_s')->nullable();   // null = timer running
            $table->unsignedTinyInteger('severity')->nullable(); // 1–4
            $table->boolean('night')->default(false);
            $table->boolean('sweat')->default(false);
            $table->json('triggers')->nullable();
            $table->timestamps();

            $table->index(['user_id', 'started_at']);
        });

        Schema::create('menopause_scores', function (Blueprint $table) {
            $table->id();
            $table->foreignId('user_id')->constrained()->cascadeOnDelete();
            $table->date('month');                               // first day of the Jalali month
            $table->json('answers');                             // {item code: 0–4}
            $table->unsignedTinyInteger('total');
            $table->unsignedTinyInteger('somatic');
            $table->unsignedTinyInteger('psychological');
            $table->unsignedTinyInteger('urogenital');
            $table->timestamps();

            $table->unique(['user_id', 'month']);
        });

        Schema::create('treatment_items', function (Blueprint $table) {
            $table->id();
            $table->foreignId('user_id')->constrained()->cascadeOnDelete();
            $table->string('kind', 16);                          // hrt|supplement|lifestyle
            $table->string('name', 120);
            $table->string('dose', 120)->nullable();
            $table->string('schedule', 16)->nullable();          // morning|noon|evening|night|weekly
            $table->date('started_on')->nullable();
            $table->date('review_on')->nullable();
            $table->unsignedSmallInteger('weekly_goal')->nullable();
            $table->string('goal_unit', 16)->nullable();         // sessions|minutes
            $table->date('stopped_on')->nullable();
            $table->foreignId('reminder_id')->nullable()->constrained()->nullOnDelete();
            $table->integer('sort_order')->default(0);
            $table->timestamps();

            $table->index(['user_id', 'kind']);
        });

        Schema::create('treatment_intakes', function (Blueprint $table) {
            $table->id();
            $table->foreignId('user_id')->constrained()->cascadeOnDelete();
            $table->foreignId('treatment_item_id')->constrained()->cascadeOnDelete();
            $table->date('intake_date');
            $table->unsignedSmallInteger('amount')->nullable();  // minutes/sessions (lifestyle)
            $table->dateTime('taken_at');
            $table->timestamps();

            $table->unique(['treatment_item_id', 'intake_date']);
            $table->index(['user_id', 'intake_date']);
        });

        Schema::create('side_effect_logs', function (Blueprint $table) {
            $table->id();
            $table->foreignId('user_id')->constrained()->cascadeOnDelete();
            $table->foreignId('treatment_item_id')->nullable()->constrained()->nullOnDelete();
            $table->date('log_date');
            $table->string('code', 32);
            $table->timestamps();

            $table->unique(['user_id', 'log_date', 'code']);
        });

        $now = now();
        DB::table('catalog_items')->insertOrIgnore([
            ['group' => 'meno_score_items', 'code' => 'hot_flashes', 'sort_order' => 1, 'is_active' => 1, 'audiences' => '["menopause"]',
                'title' => '{"fa":"گرگرفتگی و تعریق","en":"Hot flashes and sweating"}',
                'body' => '{"fa":"موج‌های گرما و تعریق، در روز یا شب","en":"Waves of heat and sweating, by day or night"}',
                'meta' => '{"domain":"somatic","max":4,"log":["symptoms.general.hot_flashes","symptoms.general.night_sweats"]}',
                'needs_review' => 1, 'created_at' => $now, 'updated_at' => $now],
            ['group' => 'meno_score_items', 'code' => 'heart_discomfort', 'sort_order' => 2, 'is_active' => 1, 'audiences' => '["menopause"]',
                'title' => '{"fa":"ناراحتی قلبی","en":"Heart discomfort"}',
                'body' => '{"fa":"تپش قلب، تند یا نامنظم زدن قلب، احساس فشار در سینه","en":"Palpitations, a racing or skipping heartbeat, chest tightness"}',
                'meta' => '{"domain":"somatic","max":4,"log":["symptoms.general.palpitations"]}',
                'needs_review' => 1, 'created_at' => $now, 'updated_at' => $now],
            ['group' => 'meno_score_items', 'code' => 'sleep_problems', 'sort_order' => 3, 'is_active' => 1, 'audiences' => '["menopause"]',
                'title' => '{"fa":"مشکلات خواب","en":"Sleep problems"}',
                'body' => '{"fa":"سخت خوابیدن، بیدار شدن شبانه یا زود بیدار شدن","en":"Trouble falling asleep, waking at night or waking too early"}',
                'meta' => '{"domain":"somatic","max":4,"log":["symptoms.general.insomnia"]}',
                'needs_review' => 1, 'created_at' => $now, 'updated_at' => $now],
            ['group' => 'meno_score_items', 'code' => 'joint_muscle', 'sort_order' => 4, 'is_active' => 1, 'audiences' => '["menopause"]',
                'title' => '{"fa":"درد مفاصل و عضلات","en":"Joint and muscle pain"}',
                'body' => '{"fa":"درد یا خشکی مفاصل و عضلات","en":"Aching or stiff joints and muscles"}',
                'meta' => '{"domain":"somatic","max":4,"log":["pain.location.joints"]}',
                'needs_review' => 1, 'created_at' => $now, 'updated_at' => $now],
            ['group' => 'meno_score_items', 'code' => 'depressive_mood', 'sort_order' => 5, 'is_active' => 1, 'audiences' => '["menopause"]',
                'title' => '{"fa":"غمگینی یا بی‌حوصلگی","en":"Low mood"}',
                'body' => '{"fa":"احساس غم، بی‌حوصلگی، زود گریه کردن یا بی‌انگیزگی","en":"Feeling down, sad, tearful or unmotivated"}',
                'meta' => '{"domain":"psychological","max":4,"log":["symptoms.general.low_mood"]}',
                'needs_review' => 1, 'created_at' => $now, 'updated_at' => $now],
            ['group' => 'meno_score_items', 'code' => 'irritability', 'sort_order' => 6, 'is_active' => 1, 'audiences' => '["menopause"]',
                'title' => '{"fa":"تحریک‌پذیری","en":"Irritability"}',
                'body' => '{"fa":"عصبی بودن، زود از کوره در رفتن","en":"Feeling tense or quick to anger"}',
                'meta' => '{"domain":"psychological","max":4,"log":["symptoms.general.irritability"]}',
                'needs_review' => 1, 'created_at' => $now, 'updated_at' => $now],
            ['group' => 'meno_score_items', 'code' => 'anxiety', 'sort_order' => 7, 'is_active' => 1, 'audiences' => '["menopause"]',
                'title' => '{"fa":"اضطراب","en":"Anxiety"}',
                'body' => '{"fa":"بی‌قراری، دلشوره یا احساس ترس","en":"Restlessness, worry or feeling panicky"}',
                'meta' => '{"domain":"psychological","max":4,"log":["symptoms.general.anxiety"]}',
                'needs_review' => 1, 'created_at' => $now, 'updated_at' => $now],
            ['group' => 'meno_score_items', 'code' => 'exhaustion', 'sort_order' => 8, 'is_active' => 1, 'audiences' => '["menopause"]',
                'title' => '{"fa":"خستگی جسمی و ذهنی","en":"Physical and mental exhaustion"}',
                'body' => '{"fa":"کم شدن انرژی، تمرکز و حافظه","en":"Less energy, focus and memory"}',
                'meta' => '{"domain":"psychological","max":4,"log":["symptoms.general.fatigue","symptoms.general.brain_fog"]}',
                'needs_review' => 1, 'created_at' => $now, 'updated_at' => $now],
            ['group' => 'meno_score_items', 'code' => 'sexual_problems', 'sort_order' => 9, 'is_active' => 1, 'audiences' => '["menopause"]',
                'title' => '{"fa":"مشکلات جنسی","en":"Sexual problems"}',
                'body' => '{"fa":"تغییر در میل، رابطه یا رضایت جنسی","en":"Changes in sexual desire, activity or satisfaction"}',
                'meta' => '{"domain":"urogenital","max":4,"log":["urogenital.symptoms.low_libido"]}',
                'needs_review' => 1, 'created_at' => $now, 'updated_at' => $now],
            ['group' => 'meno_score_items', 'code' => 'bladder_problems', 'sort_order' => 10, 'is_active' => 1, 'audiences' => '["menopause"]',
                'title' => '{"fa":"مشکلات ادراری","en":"Bladder problems"}',
                'body' => '{"fa":"تکرر، فوریت یا نشت ادرار","en":"Needing to pass urine often or urgently, or leaking"}',
                'meta' => '{"domain":"urogenital","max":4,"log":["urogenital.symptoms.bladder_symptoms"]}',
                'needs_review' => 1, 'created_at' => $now, 'updated_at' => $now],
            ['group' => 'meno_score_items', 'code' => 'vaginal_dryness', 'sort_order' => 11, 'is_active' => 1, 'audiences' => '["menopause"]',
                'title' => '{"fa":"خشکی واژن","en":"Vaginal dryness"}',
                'body' => '{"fa":"احساس خشکی یا سوزش، یا درد هنگام رابطه","en":"Dryness or burning, or pain during sex"}',
                'meta' => '{"domain":"urogenital","max":4,"log":["urogenital.symptoms.vaginal_dryness"]}',
                'needs_review' => 1, 'created_at' => $now, 'updated_at' => $now],
            ['group' => 'meno_score_bands', 'code' => 'none', 'sort_order' => 1, 'is_active' => 1, 'audiences' => '["menopause"]',
                'title' => '{"fa":"بدون علامت","en":"No symptoms"}',
                'body' => null,
                'meta' => '{"min":0,"max":4}',
                'needs_review' => 1, 'created_at' => $now, 'updated_at' => $now],
            ['group' => 'meno_score_bands', 'code' => 'mild', 'sort_order' => 2, 'is_active' => 1, 'audiences' => '["menopause"]',
                'title' => '{"fa":"خفیف","en":"Mild"}',
                'body' => null,
                'meta' => '{"min":5,"max":8}',
                'needs_review' => 1, 'created_at' => $now, 'updated_at' => $now],
            ['group' => 'meno_score_bands', 'code' => 'moderate', 'sort_order' => 3, 'is_active' => 1, 'audiences' => '["menopause"]',
                'title' => '{"fa":"متوسط","en":"Moderate"}',
                'body' => null,
                'meta' => '{"min":9,"max":16}',
                'needs_review' => 1, 'created_at' => $now, 'updated_at' => $now],
            ['group' => 'meno_score_bands', 'code' => 'severe', 'sort_order' => 4, 'is_active' => 1, 'audiences' => '["menopause"]',
                'title' => '{"fa":"شدید","en":"Severe"}',
                'body' => null,
                'meta' => '{"min":17,"max":44}',
                'needs_review' => 1, 'created_at' => $now, 'updated_at' => $now],
            ['group' => 'meno_alerts', 'code' => 'postmenopausal_bleeding', 'sort_order' => 1, 'is_active' => 1, 'audiences' => '["menopause"]',
                'title' => '{"fa":"خونریزی بعد از یائسگی","en":"Bleeding after menopause"}',
                'body' => '{"fa":"هر خونریزی یا لکه‌بینی، حتی کم، بعد از ۱۲ ماه قطع پریود باید بررسی شود. بیشتر وقت‌ها علت ساده‌ای دارد، اما بررسی زودهنگام مهم است.","en":"Any bleeding or spotting, even a little, 12 months or more after your last period should be checked. Usually the cause is simple, but an early check matters."}',
                'meta' => '{"severity":"urgent","primary":true,"stages":["meno","post"],"cta":{"fa":"این مورد را به پزشک بگو","en":"Tell your doctor about this"}}',
                'needs_review' => 1, 'created_at' => $now, 'updated_at' => $now],
            ['group' => 'meno_alerts', 'code' => 'heavy_perimenopause_bleeding', 'sort_order' => 2, 'is_active' => 1, 'audiences' => '["menopause"]',
                'title' => '{"fa":"خونریزی خیلی زیاد یا طولانی در پیش‌یائسگی","en":"Very heavy or long bleeding in perimenopause"}',
                'body' => '{"fa":"بیش از ۷ روز، یا پر شدن نوار در کمتر از ۲ ساعت","en":"More than 7 days, or soaking a pad in under 2 hours"}',
                'meta' => '{"severity":"caution","stages":["peri","unsure"]}',
                'needs_review' => 1, 'created_at' => $now, 'updated_at' => $now],
            ['group' => 'meno_alerts', 'code' => 'chest_pain_palpitations', 'sort_order' => 3, 'is_active' => 1, 'audiences' => '["menopause"]',
                'title' => '{"fa":"درد قفسه سینه یا تپش قلب شدید","en":"Chest pain or a severe racing heart"}',
                'body' => '{"fa":"اگر ناگهانی و شدید است، با اورژانس (۱۱۵) تماس بگیر","en":"If it is sudden and severe, call emergency services (115)"}',
                'meta' => '{"severity":"urgent","emergency":true,"hotline":"115"}',
                'needs_review' => 1, 'created_at' => $now, 'updated_at' => $now],
            ['group' => 'meno_alerts', 'code' => 'one_sided_leg_swelling', 'sort_order' => 4, 'is_active' => 1, 'audiences' => '["menopause"]',
                'title' => '{"fa":"درد و ورم یک‌طرفه ساق پا","en":"Pain and swelling in one calf"}',
                'body' => '{"fa":"به‌خصوص اگر هورمون‌درمانی می‌کنی","en":"Especially if you take hormone therapy"}',
                'meta' => '{"severity":"urgent","hrt":true}',
                'needs_review' => 1, 'created_at' => $now, 'updated_at' => $now],
            ['group' => 'meno_alerts', 'code' => 'breast_change', 'sort_order' => 5, 'is_active' => 1, 'audiences' => '["menopause"]',
                'title' => '{"fa":"توده یا تغییر در پستان","en":"A lump or change in the breast"}',
                'body' => '{"fa":"هر تغییر تازه را به پزشک نشان بده","en":"Have any new change checked by a doctor"}',
                'meta' => '{"severity":"caution"}',
                'needs_review' => 1, 'created_at' => $now, 'updated_at' => $now],
            ['group' => 'meno_alerts', 'code' => 'persistent_low_mood', 'sort_order' => 6, 'is_active' => 1, 'audiences' => '["menopause"]',
                'title' => '{"fa":"غم عمیق یا ناامیدی چند هفته‌ای","en":"Deep sadness or hopelessness for weeks"}',
                'body' => '{"fa":"حرف زدن با پزشک یا مشاور کمک می‌کند","en":"Talking to a doctor or counsellor helps"}',
                'meta' => '{"severity":"caution"}',
                'needs_review' => 1, 'created_at' => $now, 'updated_at' => $now],
            ['group' => 'meno_alerts', 'code' => 'fragility_fracture', 'sort_order' => 7, 'is_active' => 1, 'audiences' => '["menopause"]',
                'title' => '{"fa":"شکستگی با ضربه یا زمین خوردن ساده","en":"A fracture from a minor knock or fall"}',
                'body' => '{"fa":"ممکن است نشانه پوکی استخوان باشد","en":"It may be a sign of osteoporosis"}',
                'meta' => '{"severity":"caution"}',
                'needs_review' => 1, 'created_at' => $now, 'updated_at' => $now],
            ['group' => 'meno_tips', 'code' => 'stage_peri', 'sort_order' => 1, 'is_active' => 1, 'audiences' => '["menopause"]',
                'title' => '{"fa":"پیش‌یائسگی","en":"Perimenopause"}',
                'body' => '{"fa":"پریودها ممکن است نامنظم، کم یا زیاد شوند و علائم تازه بیایند. ثبت پریود و علائم کمک می‌کند الگوها را ببینی و با پزشک دقیق‌تر حرف بزنی.","en":"Periods may become irregular, lighter or heavier, and new symptoms may start. Logging periods and symptoms helps you see patterns and talk to your doctor in detail."}',
                'meta' => '{"placement":"home","stages":["peri"]}',
                'needs_review' => 1, 'created_at' => $now, 'updated_at' => $now],
            ['group' => 'meno_tips', 'code' => 'stage_meno', 'sort_order' => 2, 'is_active' => 1, 'audiences' => '["menopause"]',
                'title' => '{"fa":"یائسگی","en":"Menopause"}',
                'body' => '{"fa":"از ۱۲ ماه گذشته، یعنی وارد یائسگی شده‌ای. از این به بعد هر خونریزی یا لکه‌بینی را ثبت کن و به پزشک خبر بده.","en":"It has been more than 12 months, which means you have reached menopause. From now on, log any bleeding or spotting and tell your doctor."}',
                'meta' => '{"placement":"home","stages":["meno"]}',
                'needs_review' => 1, 'created_at' => $now, 'updated_at' => $now],
            ['group' => 'meno_tips', 'code' => 'stage_post', 'sort_order' => 3, 'is_active' => 1, 'audiences' => '["menopause"]',
                'title' => '{"fa":"پس از یائسگی","en":"Postmenopause"}',
                'body' => '{"fa":"در این سال‌ها مراقبت از استخوان و قلب مهم‌تر می‌شود. چکاپ‌های منظم را دنبال کن و هر خونریزی یا لکه‌بینی را به پزشک بگو.","en":"In these years caring for your bones and heart matters more. Keep up with regular checkups and tell your doctor about any bleeding or spotting."}',
                'meta' => '{"placement":"home","stages":["post"]}',
                'needs_review' => 1, 'created_at' => $now, 'updated_at' => $now],
            ['group' => 'meno_tips', 'code' => 'stage_unsure', 'sort_order' => 4, 'is_active' => 1, 'audiences' => '["menopause"]',
                'title' => '{"fa":"مطمئن نیستم","en":"Not sure"}',
                'body' => '{"fa":"اگر مطمئن نیستی کجای مسیر هستی، تاریخ تقریبی آخرین پریود کمک می‌کند. ۱۲ ماه بدون پریود یعنی یائسگی؛ پزشک می‌تواند دقیق‌تر بگوید.","en":"If you are not sure where you are, the approximate date of your last period helps. 12 months without a period means menopause; your doctor can tell you more."}',
                'meta' => '{"placement":"home","stages":["unsure"]}',
                'needs_review' => 1, 'created_at' => $now, 'updated_at' => $now],
            ['group' => 'meno_tips', 'code' => 'log_bleeding_note', 'sort_order' => 5, 'is_active' => 1, 'audiences' => '["menopause"]',
                'title' => '{"fa":"خونریزی بعد از یائسگی","en":"Bleeding after menopause"}',
                'body' => '{"fa":"بعد از یائسگی هر خونریزی باید به پزشک گفته شود؛ اگر ثبت کنی راهنمایی‌ات می‌کنیم.","en":"After menopause any bleeding should be reported to a doctor; if you log it, we will guide you."}',
                'meta' => '{"placement":"log","stages":["meno","post"]}',
                'needs_review' => 1, 'created_at' => $now, 'updated_at' => $now],
            ['group' => 'meno_tips', 'code' => 'hot_flash_breathing', 'sort_order' => 6, 'is_active' => 1, 'audiences' => '["menopause"]',
                'title' => '{"fa":"در لحظه گرگرفتگی","en":"During a hot flash"}',
                'body' => '{"fa":"نفس عمیق و آرام (۶ بار در دقیقه) و خنک کردن مچ و گردن در لحظه کمک می‌کند.","en":"Slow, deep breathing (6 breaths a minute) and cooling your wrists and neck help in the moment."}',
                'meta' => '{"placement":"hot_flash"}',
                'needs_review' => 1, 'created_at' => $now, 'updated_at' => $now],
            ['group' => 'meno_tips', 'code' => 'hrt_review', 'sort_order' => 7, 'is_active' => 1, 'audiences' => '["menopause"]',
                'title' => '{"fa":"بازبینی با پزشک","en":"Review with your doctor"}',
                'body' => '{"fa":"معمولاً ۳ ماه بعد از شروع، اثر و عوارض بررسی می‌شود.","en":"Effects and side effects are usually reviewed about 3 months after starting."}',
                'meta' => '{"placement":"treatment","review_after_months":3}',
                'needs_review' => 1, 'created_at' => $now, 'updated_at' => $now],
            ['group' => 'meno_tips', 'code' => 'hrt_spotting', 'sort_order' => 8, 'is_active' => 1, 'audiences' => '["menopause"]',
                'title' => '{"fa":"لکه‌بینی در ماه‌های اول","en":"Spotting in the first months"}',
                'body' => '{"fa":"لکه‌بینی در ماه‌های اول شایع است، اما اگر ادامه داشت یا زیاد بود به پزشک بگو.","en":"Spotting is common in the first months, but tell your doctor if it continues or is heavy."}',
                'meta' => '{"placement":"treatment"}',
                'needs_review' => 1, 'created_at' => $now, 'updated_at' => $now],
            ['group' => 'meno_tips', 'code' => 'doctor_only', 'sort_order' => 9, 'is_active' => 1, 'audiences' => '["menopause"]',
                'title' => '{"fa":"فقط با نظر پزشک","en":"Only with your doctor"}',
                'body' => '{"fa":"شروع، قطع یا تغییر دوز هر دارو فقط با نظر پزشک.","en":"Start, stop or change the dose of any medicine only with your doctor."}',
                'meta' => '{"placement":"treatment"}',
                'needs_review' => 1, 'created_at' => $now, 'updated_at' => $now],
            ['group' => 'meno_tips', 'code' => 'lifestyle_resistance', 'sort_order' => 10, 'is_active' => 1, 'audiences' => '["menopause"]',
                'title' => '{"fa":"ورزش مقاومتی","en":"Resistance exercise"}',
                'body' => '{"fa":"برای استخوان و عضله","en":"For bones and muscles"}',
                'meta' => '{"placement":"treatment_lifestyle","weekly_goal":2,"goal_unit":"sessions"}',
                'needs_review' => 1, 'created_at' => $now, 'updated_at' => $now],
            ['group' => 'meno_tips', 'code' => 'lifestyle_brisk_walk', 'sort_order' => 11, 'is_active' => 1, 'audiences' => '["menopause"]',
                'title' => '{"fa":"پیاده‌روی تند","en":"Brisk walking"}',
                'body' => '{"fa":"۱۵۰ دقیقه در هفته","en":"150 minutes a week"}',
                'meta' => '{"placement":"treatment_lifestyle","weekly_goal":150,"goal_unit":"minutes"}',
                'needs_review' => 1, 'created_at' => $now, 'updated_at' => $now],
            ['group' => 'meno_tips', 'code' => 'lifestyle_relaxation', 'sort_order' => 12, 'is_active' => 1, 'audiences' => '["menopause"]',
                'title' => '{"fa":"تمرین آرام‌سازی","en":"Relaxation practice"}',
                'body' => '{"fa":"۱۰ دقیقه، برای گرگرفتگی و خواب","en":"10 minutes, for hot flashes and sleep"}',
                'meta' => '{"placement":"treatment_lifestyle","weekly_goal":70,"goal_unit":"minutes"}',
                'needs_review' => 1, 'created_at' => $now, 'updated_at' => $now],
            ['group' => 'meno_tips', 'code' => 'checkups_intro', 'sort_order' => 13, 'is_active' => 1, 'audiences' => '["menopause"]',
                'title' => '{"fa":"چکاپ‌ها و آزمایش‌ها","en":"Checkups and tests"}',
                'body' => '{"fa":"بعد از یائسگی خطر پوکی استخوان و بیماری قلبی بیشتر می‌شود. این فهرست یادآور است؛ زمان‌بندی دقیق را پزشکت تعیین می‌کند.","en":"After menopause the risk of osteoporosis and heart disease rises. This list is a reminder; your doctor sets the exact timing."}',
                'meta' => '{"placement":"checkups"}',
                'needs_review' => 1, 'created_at' => $now, 'updated_at' => $now],
            ['group' => 'meno_tips', 'code' => 'patterns_disclaimer', 'sort_order' => 14, 'is_active' => 1, 'audiences' => '["menopause"]',
                'title' => '{"fa":"الگوهایی که دیدیم","en":"Patterns we noticed"}',
                'body' => '{"fa":"بر اساس ثبت‌های خودت · تشخیص پزشکی نیست","en":"Based on your own logs · not a medical diagnosis"}',
                'meta' => '{"placement":"score"}',
                'needs_review' => 1, 'created_at' => $now, 'updated_at' => $now],
            ['group' => 'meno_checkup_groups', 'code' => 'heart_metabolic', 'sort_order' => 1, 'is_active' => 1, 'audiences' => '["menopause"]',
                'title' => '{"fa":"قلب و متابولیسم","en":"Heart and metabolism"}',
                'body' => null,
                'meta' => '{"checkups":["meno_blood_pressure","meno_blood_sugar","meno_lipids","meno_weight_waist"]}',
                'needs_review' => 1, 'created_at' => $now, 'updated_at' => $now],
            ['group' => 'meno_checkup_groups', 'code' => 'bone', 'sort_order' => 2, 'is_active' => 1, 'audiences' => '["menopause"]',
                'title' => '{"fa":"استخوان","en":"Bones"}',
                'body' => null,
                'meta' => '{"checkups":["meno_bone_density","meno_vitamin_d_calcium"]}',
                'needs_review' => 1, 'created_at' => $now, 'updated_at' => $now],
            ['group' => 'meno_checkup_groups', 'code' => 'cancer_screening', 'sort_order' => 3, 'is_active' => 1, 'audiences' => '["menopause"]',
                'title' => '{"fa":"غربالگری سرطان","en":"Cancer screening"}',
                'body' => null,
                'meta' => '{"checkups":["mammography","pap_smear","meno_colon_screening"]}',
                'needs_review' => 1, 'created_at' => $now, 'updated_at' => $now],
            ['group' => 'meno_checkup_groups', 'code' => 'other', 'sort_order' => 4, 'is_active' => 1, 'audiences' => '["menopause"]',
                'title' => '{"fa":"سایر","en":"Other"}',
                'body' => null,
                'meta' => '{"checkups":["meno_thyroid","meno_eye_exam","dentist"]}',
                'needs_review' => 1, 'created_at' => $now, 'updated_at' => $now],
        ]);

        DB::table('checkup_types')->insertOrIgnore([
            ['key' => 'meno_blood_pressure', 'category' => 'monthly', 'title' => '{"fa":"فشار خون","en":"Blood pressure"}',
                'subtitle' => '{"fa":"ماهانه در خانه، سالانه نزد پزشک","en":"Monthly at home, yearly with your doctor"}',
                'why' => '{"fa":"بعد از یائسگی خطر فشار خون بالا بیشتر می‌شود و اغلب بی‌علامت است. اندازه‌گیری منظم در خانه تغییرها را زود نشان می‌دهد.","en":"After menopause the risk of high blood pressure rises and it often has no symptoms. Regular home readings show changes early."}',
                'performed_by' => 'self', 'icon' => 'heart', 'tone' => 'rose', 'interval_months' => 1, 'interval_months_max' => null,
                'age_min' => null, 'remind_lead_days' => 3, 'prep_steps' => '[{"fa":"۵ دقیقه آرام بنشین و بعد اندازه بگیر","en":"Sit quietly for 5 minutes before measuring"}]',
                'hide_in_pregnancy' => 1, 'audiences' => '["menopause"]', 'is_active' => 0, 'sort_order' => 101,
                'source_note' => 'Menopause catalog (CB-MENO-01). [needs clinical review]', 'created_at' => $now, 'updated_at' => $now],
            ['key' => 'meno_blood_sugar', 'category' => 'multi_year', 'title' => '{"fa":"قند خون ناشتا یا HbA1c","en":"Fasting blood sugar or HbA1c"}',
                'subtitle' => '{"fa":"معمولاً هر ۱ تا ۳ سال","en":"Usually every 1 to 3 years"}',
                'why' => '{"fa":"با بالا رفتن سن و تغییرات هورمونی، خطر دیابت بیشتر می‌شود. آزمایش منظم آن را زود نشان می‌دهد.","en":"With age and hormonal change the risk of diabetes rises. A regular test shows it early."}',
                'performed_by' => 'lab', 'icon' => 'blood', 'tone' => 'amber', 'interval_months' => 12, 'interval_months_max' => 36,
                'age_min' => null, 'remind_lead_days' => 14, 'prep_steps' => '[{"fa":"برای قند ناشتا ۸ تا ۱۰ ساعت چیزی جز آب نخور","en":"For fasting sugar, have nothing but water for 8 to 10 hours"}]',
                'hide_in_pregnancy' => 1, 'audiences' => '["menopause"]', 'is_active' => 0, 'sort_order' => 102,
                'source_note' => 'Menopause catalog (CB-MENO-01). [needs clinical review]', 'created_at' => $now, 'updated_at' => $now],
            ['key' => 'meno_lipids', 'category' => 'multi_year', 'title' => '{"fa":"چربی خون (کلسترول و تری‌گلیسرید)","en":"Blood lipids (cholesterol and triglycerides)"}',
                'subtitle' => '{"fa":"معمولاً هر ۱ تا ۵ سال","en":"Usually every 1 to 5 years"}',
                'why' => '{"fa":"بعد از یائسگی کلسترول معمولاً بالا می‌رود و خطر بیماری قلبی بیشتر می‌شود.","en":"After menopause cholesterol usually rises and so does the risk of heart disease."}',
                'performed_by' => 'lab', 'icon' => 'flask', 'tone' => 'amber', 'interval_months' => 12, 'interval_months_max' => 60,
                'age_min' => null, 'remind_lead_days' => 14, 'prep_steps' => '[{"fa":"اگر پزشک گفته، ۹ تا ۱۲ ساعت ناشتا باش","en":"If your doctor asked, fast for 9 to 12 hours"}]',
                'hide_in_pregnancy' => 1, 'audiences' => '["menopause"]', 'is_active' => 0, 'sort_order' => 103,
                'source_note' => 'Menopause catalog (CB-MENO-01). [needs clinical review]', 'created_at' => $now, 'updated_at' => $now],
            ['key' => 'meno_weight_waist', 'category' => 'monthly', 'title' => '{"fa":"وزن و دور کمر","en":"Weight and waist"}',
                'subtitle' => '{"fa":"ماهانه","en":"Monthly"}',
                'why' => '{"fa":"در یائسگی چربی بیشتر دور شکم جمع می‌شود که با خطر قلبی همراه است. اندازه‌گیری ماهانه تغییرها را نشان می‌دهد.","en":"In menopause more fat gathers around the waist, which is linked to heart risk. A monthly measurement shows changes."}',
                'performed_by' => 'self', 'icon' => 'note', 'tone' => 'neutral', 'interval_months' => 1, 'interval_months_max' => null,
                'age_min' => null, 'remind_lead_days' => 3, 'prep_steps' => null,
                'hide_in_pregnancy' => 1, 'audiences' => '["menopause"]', 'is_active' => 0, 'sort_order' => 104,
                'source_note' => 'Menopause catalog (CB-MENO-01). [needs clinical review]', 'created_at' => $now, 'updated_at' => $now],
            ['key' => 'meno_bone_density', 'category' => 'age_based', 'title' => '{"fa":"سنجش تراکم استخوان (DEXA)","en":"Bone density scan (DEXA)"}',
                'subtitle' => '{"fa":"معمولاً از ۶۵ سالگی، یا زودتر با عامل خطر","en":"Usually from 65, or earlier with a risk factor"}',
                'why' => '{"fa":"بعد از یائسگی استخوان‌ها سریع‌تر تحلیل می‌روند. این سنجش پوکی استخوان را پیش از شکستگی نشان می‌دهد. اگر عامل خطر داری، زودتر با پزشک هماهنگ کن.","en":"After menopause bone is lost faster. This scan shows osteoporosis before a fracture. If you have a risk factor, ask your doctor about testing sooner."}',
                'performed_by' => 'lab', 'icon' => 'shield', 'tone' => 'violet', 'interval_months' => 24, 'interval_months_max' => null,
                'age_min' => 65, 'remind_lead_days' => 30, 'prep_steps' => null,
                'hide_in_pregnancy' => 1, 'audiences' => '["menopause"]', 'is_active' => 0, 'sort_order' => 105,
                'source_note' => 'Menopause catalog (CB-MENO-01). [needs clinical review]', 'created_at' => $now, 'updated_at' => $now],
            ['key' => 'meno_vitamin_d_calcium', 'category' => 'annual', 'title' => '{"fa":"ویتامین D و کلسیم","en":"Vitamin D and calcium"}',
                'subtitle' => '{"fa":"طبق نظر پزشک","en":"As your doctor advises"}',
                'why' => '{"fa":"ویتامین D و کلسیم کافی برای استخوان مهم است؛ پزشک تصمیم می‌گیرد آزمایش یا مکمل لازم است یا نه.","en":"Enough vitamin D and calcium matters for bone; your doctor decides whether a test or supplement is needed."}',
                'performed_by' => 'lab', 'icon' => 'pill', 'tone' => 'amber', 'interval_months' => 12, 'interval_months_max' => null,
                'age_min' => null, 'remind_lead_days' => 14, 'prep_steps' => null,
                'hide_in_pregnancy' => 1, 'audiences' => '["menopause"]', 'is_active' => 0, 'sort_order' => 106,
                'source_note' => 'Menopause catalog (CB-MENO-01). [needs clinical review]', 'created_at' => $now, 'updated_at' => $now],
            ['key' => 'meno_colon_screening', 'category' => 'age_based', 'title' => '{"fa":"غربالگری سرطان روده","en":"Bowel cancer screening"}',
                'subtitle' => '{"fa":"از ۴۵ سالگی، طبق روش انتخابی","en":"From 45, depending on the method"}',
                'why' => '{"fa":"غربالگری روده می‌تواند پولیپ یا سرطان را زود پیدا کند. فاصله تکرار به روش (آزمایش مدفوع یا کولونوسکوپی) بستگی دارد.","en":"Bowel screening can find polyps or cancer early. How often depends on the method (stool test or colonoscopy)."}',
                'performed_by' => 'doctor', 'icon' => 'shieldCheck', 'tone' => 'violet', 'interval_months' => 12, 'interval_months_max' => 120,
                'age_min' => 45, 'remind_lead_days' => 30, 'prep_steps' => null,
                'hide_in_pregnancy' => 1, 'audiences' => '["menopause"]', 'is_active' => 0, 'sort_order' => 107,
                'source_note' => 'Menopause catalog (CB-MENO-01). [needs clinical review]', 'created_at' => $now, 'updated_at' => $now],
            ['key' => 'meno_thyroid', 'category' => 'annual', 'title' => '{"fa":"تیروئید (TSH)","en":"Thyroid (TSH)"}',
                'subtitle' => '{"fa":"اگر خستگی یا تغییر وزن داری","en":"If you have tiredness or weight change"}',
                'why' => '{"fa":"مشکلات تیروئید در این سن شایع‌اند و علائمشان می‌تواند شبیه یائسگی باشد.","en":"Thyroid problems are common at this age and can look like menopause symptoms."}',
                'performed_by' => 'lab', 'icon' => 'flask', 'tone' => 'amber', 'interval_months' => 12, 'interval_months_max' => null,
                'age_min' => null, 'remind_lead_days' => 14, 'prep_steps' => null,
                'hide_in_pregnancy' => 1, 'audiences' => '["menopause"]', 'is_active' => 0, 'sort_order' => 108,
                'source_note' => 'Menopause catalog (CB-MENO-01). [needs clinical review]', 'created_at' => $now, 'updated_at' => $now],
            ['key' => 'meno_eye_exam', 'category' => 'annual', 'title' => '{"fa":"معاینه چشم","en":"Eye exam"}',
                'subtitle' => '{"fa":"سالانه","en":"Yearly"}',
                'why' => '{"fa":"معاینه منظم چشم تغییرات بینایی و بیماری‌هایی مثل آب سیاه را زود نشان می‌دهد.","en":"A regular eye exam shows vision changes and conditions such as glaucoma early."}',
                'performed_by' => 'doctor', 'icon' => 'stetho', 'tone' => 'green', 'interval_months' => 12, 'interval_months_max' => null,
                'age_min' => null, 'remind_lead_days' => 30, 'prep_steps' => null,
                'hide_in_pregnancy' => 1, 'audiences' => '["menopause"]', 'is_active' => 0, 'sort_order' => 109,
                'source_note' => 'Menopause catalog (CB-MENO-01). [needs clinical review]', 'created_at' => $now, 'updated_at' => $now],
        ]);
    }

    public function down(): void
    {
        DB::table('catalog_items')->where('group', 'meno_score_items')
            ->whereIn('code', ['hot_flashes', 'heart_discomfort', 'sleep_problems', 'joint_muscle', 'depressive_mood', 'irritability', 'anxiety', 'exhaustion', 'sexual_problems', 'bladder_problems', 'vaginal_dryness'])->delete();
        DB::table('catalog_items')->where('group', 'meno_score_bands')
            ->whereIn('code', ['none', 'mild', 'moderate', 'severe'])->delete();
        DB::table('catalog_items')->where('group', 'meno_alerts')
            ->whereIn('code', ['postmenopausal_bleeding', 'heavy_perimenopause_bleeding', 'chest_pain_palpitations', 'one_sided_leg_swelling', 'breast_change', 'persistent_low_mood', 'fragility_fracture'])->delete();
        DB::table('catalog_items')->where('group', 'meno_tips')
            ->whereIn('code', ['stage_peri', 'stage_meno', 'stage_post', 'stage_unsure', 'log_bleeding_note', 'hot_flash_breathing', 'hrt_review', 'hrt_spotting', 'doctor_only', 'lifestyle_resistance', 'lifestyle_brisk_walk', 'lifestyle_relaxation', 'checkups_intro', 'patterns_disclaimer'])->delete();
        DB::table('catalog_items')->where('group', 'meno_checkup_groups')
            ->whereIn('code', ['heart_metabolic', 'bone', 'cancer_screening', 'other'])->delete();
        DB::table('checkup_types')->whereNull('user_id')
            ->whereIn('key', ['meno_blood_pressure', 'meno_blood_sugar', 'meno_lipids', 'meno_weight_waist', 'meno_bone_density', 'meno_vitamin_d_calcium', 'meno_colon_screening', 'meno_thyroid', 'meno_eye_exam'])->delete();
        Schema::dropIfExists('side_effect_logs');
        Schema::dropIfExists('treatment_intakes');
        Schema::dropIfExists('treatment_items');
        Schema::dropIfExists('menopause_scores');
        Schema::dropIfExists('hot_flashes');
        Schema::table('checkup_types', function (Blueprint $table) {
            $table->dropColumn('audiences');
        });
    }
};
