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
     * Twin of backend-go/db/migrations/00052_assistant_clinic.sql
     * (docs/go-migration/migrations.md): the AI assistant clinic (bloom
     * B-N7-06) — departments, safety terms, profile sharing, conversations,
     * messages and doctor handoffs — used only by the Go internal/assistant
     * package, plus the seeded departments and safety terms so
     * `make schema-diff` row counts match. No model or routes here.
     */
    public function up(): void
    {
        Schema::create('assistant_departments', function (Blueprint $table) {
            $table->id();
            $table->string('code', 32)->unique();
            $table->string('icon', 32)->nullable();
            $table->integer('sort_order')->default(0);
            $table->boolean('is_active')->default(true);
            $table->json('name');
            $table->json('tagline')->nullable();
            $table->json('intro')->nullable();
            $table->json('does_not_do')->nullable();
            $table->json('quick_replies')->nullable();
            $table->text('system_prompt');
            $table->text('guardrails')->nullable();
            $table->boolean('needs_review')->default(true);
            $table->timestamps();

            $table->index(['is_active', 'sort_order']);
        });

        Schema::create('assistant_safety_terms', function (Blueprint $table) {
            $table->id();
            $table->string('kind', 16);                         // emergency|diagnosis
            $table->string('locale', 8);
            $table->string('term', 100);
            $table->boolean('is_active')->default(true);
            $table->timestamps();

            $table->unique(['kind', 'locale', 'term']);
        });

        Schema::create('assistant_profile_settings', function (Blueprint $table) {
            $table->id();
            $table->foreignId('user_id')->unique()->constrained()->cascadeOnDelete();
            $table->json('shared');
            $table->timestamps();
        });

        Schema::create('assistant_conversations', function (Blueprint $table) {
            $table->id();
            $table->foreignId('user_id')->constrained()->cascadeOnDelete();
            $table->foreignId('department_id')->constrained('assistant_departments');
            $table->string('title', 120)->nullable();
            $table->string('triage_level', 16)->nullable();     // self_care|non_urgent|soon|emergency
            $table->boolean('emergency')->default(false);
            $table->unsignedInteger('message_count')->default(0);
            $table->text('summary')->nullable();
            $table->dateTime('summary_at')->nullable();
            $table->unsignedInteger('summary_message_count')->default(0);
            $table->dateTime('last_message_at')->nullable();
            $table->timestamps();

            $table->index(['user_id', 'last_message_at']);
        });

        Schema::create('assistant_messages', function (Blueprint $table) {
            $table->id();
            $table->foreignId('conversation_id')->constrained('assistant_conversations')->cascadeOnDelete();
            $table->string('role', 16);                         // user|assistant|card
            $table->text('body');
            $table->foreignId('photo_file_id')->nullable()->constrained('files')->nullOnDelete();
            $table->string('card', 16)->nullable();
            $table->string('triage_level', 16)->nullable();
            $table->string('guardrail', 16)->nullable();
            $table->json('chips')->nullable();
            $table->string('status', 12)->default('complete');
            $table->timestamps();
        });

        Schema::create('assistant_handoffs', function (Blueprint $table) {
            $table->id();
            $table->foreignId('conversation_id')->constrained('assistant_conversations')->cascadeOnDelete();
            $table->foreignId('user_id')->constrained()->cascadeOnDelete();
            $table->foreignId('booking_id')->constrained('telemed_bookings')->cascadeOnDelete();
            $table->text('summary');
            $table->string('triage_level', 16)->nullable();
            $table->timestamps();

            $table->unique(['conversation_id', 'booking_id']);
        });

        $now = now();
        $dept = fn (string $code, string $icon, int $sort, string $name, string $tagline, string $intro, string $notDo, string $chips, string $prompt, string $guard) => [
            'code' => $code, 'icon' => $icon, 'sort_order' => $sort, 'is_active' => 1,
            'name' => $name, 'tagline' => $tagline, 'intro' => $intro, 'does_not_do' => $notDo, 'quick_replies' => $chips,
            'system_prompt' => $prompt, 'guardrails' => $guard, 'needs_review' => 1, 'created_at' => $now, 'updated_at' => $now,
        ];
        DB::table('assistant_departments')->insertOrIgnore([
            $dept('general', 'stethoscope', 1, '{"fa":"مشاور پزشکی عمومی","en":"General health advisor"}', '{"fa":"علائم، دارو، آزمایش","en":"Symptoms, medicines, lab tests"}', '{"fa":"راهنمایی کلی درباره علائم روزمره، داروهایی که مصرف می‌کنی و معنای ساده نتایج آزمایش. کمک می‌کند بفهمی چه زمانی باید به پزشک مراجعه کنی.","en":"General guidance on everyday symptoms, the medicines you take and what lab results mean in plain words. Helps you decide when to see a doctor."}', '{"fa":["تشخیص بیماری","تجویز دارو یا تعیین دوز","جایگزین ویزیت پزشک"],"en":["Diagnosing a disease","Prescribing medicines or doses","Replacing a doctor\'s visit"]}', '{"fa":["سردرد","تب","سرماخوردگی","تفسیر آزمایش","چیز دیگر"],"en":["Headache","Fever","Cold","Lab results","Something else"]}', 'You are the general health advisor of the Ritme women\'s health app. Help with everyday symptoms, how common medicines are usually used, and what lab values mean in plain words. Ask one or two short clarifying questions before giving guidance. Suggest when a doctor visit is sensible and how soon.', 'Never name a diagnosis as a fact; describe patterns as possibilities. Never give a medicine dose or start/stop advice for prescription drugs. Pregnancy or breastfeeding changes medicine safety: always say so when relevant.'),
            $dept('gynecology', 'flower', 2, '{"fa":"مشاوره زنان","en":"Gynecology advisor"}', '{"fa":"سیکل، باروری، بارداری، یائسگی","en":"Cycle, fertility, pregnancy, menopause"}', '{"fa":"پاسخ به پرسش‌هایت درباره سیکل قاعدگی، لکه‌بینی، باروری، بارداری و یائسگی با در نظر گرفتن داده‌هایی که خودت اجازه داده‌ای.","en":"Answers about your menstrual cycle, spotting, fertility, pregnancy and menopause, taking into account the data you chose to share."}', '{"fa":["تشخیص بیماری یا بارداری","تجویز دارو یا هورمون","تفسیر قطعی سونوگرافی"],"en":["Diagnosing a disease or a pregnancy","Prescribing medicines or hormones","Definitive ultrasound readings"]}', '{"fa":["پریود نامنظم","لکه‌بینی","درد پریود","اقدام به بارداری","چیز دیگر"],"en":["Irregular period","Spotting","Period pain","Trying to conceive","Something else"]}', 'You are the gynecology advisor of the Ritme women\'s health app. Help with menstrual cycle questions, spotting, period pain, fertility, pregnancy and menopause. Use the shared cycle and life-stage data when relevant. Ask short clarifying questions (cycle day, duration, severity) before guidance.', 'Heavy bleeding, severe one-sided pelvic pain, fever with pelvic pain, or bleeding in pregnancy need urgent care: say so clearly. Never confirm or exclude a pregnancy; suggest a test. Never recommend hormones or contraceptive brands.'),
            $dept('sexual_health', 'heart', 3, '{"fa":"مشاوره جنسی","en":"Sexual health advisor"}', '{"fa":"رابطه، میل، درد، پیشگیری","en":"Intimacy, desire, pain, contraception"}', '{"fa":"فضایی امن و بدون قضاوت برای پرسش درباره میل جنسی، درد هنگام رابطه، پیشگیری و عفونت‌ها.","en":"A safe, non-judgemental space for questions about desire, pain during sex, contraception and infections."}', '{"fa":["تشخیص عفونت","تجویز دارو یا روش پیشگیری","قضاوت درباره رابطه‌ات"],"en":["Diagnosing an infection","Prescribing medicines or a contraceptive method","Judging your relationship"]}', '{"fa":["درد هنگام رابطه","کاهش میل","پیشگیری","ترشحات","چیز دیگر"],"en":["Pain during sex","Low desire","Contraception","Discharge","Something else"]}', 'You are the sexual health advisor of the Ritme women\'s health app. Be warm, respectful and non-judgemental. Help with desire, pain during intercourse, contraception options in general terms, and signs of infection that need a check-up.', 'Never ask for explicit details beyond what is medically useful. Sexual violence or coercion: respond with care and point to professional help. Possible infection: recommend testing, never a specific drug.'),
            $dept('skin_beauty', 'sparkle', 4, '{"fa":"زیبایی و پوست","en":"Skin & beauty"}', '{"fa":"آکنه، لک، مراقبت پوست و مو","en":"Acne, dark spots, skin and hair care"}', '{"fa":"راهنمایی روتین مراقبت، آکنه، لک، خشکی، ریزش مو و اثر هورمون‌ها و سیکل روی پوست. با عکس هم می‌توانی توضیح بدهی.","en":"Guidance on care routines, acne, dark spots, dryness, hair loss and how hormones and your cycle affect your skin. You can also explain with a photo."}', '{"fa":["تشخیص بیماری پوستی","تجویز دارو","تفسیر قطعی عکس"],"en":["Diagnosing a skin disease","Prescribing medicines","Definitive photo readings"]}', '{"fa":["آکنه","لک و تیرگی","خشکی","ریزش مو","چیز دیگر"],"en":["Acne","Dark spots","Dryness","Hair loss","Something else"]}', 'You are the skin and beauty advisor of the Ritme women\'s health app. Help with skincare routines, acne, pigmentation, dryness and hair loss, and how the cycle, pregnancy and hormones affect skin. A photo may be attached: describe only what is visible and say a photo is not a diagnosis.', 'If the user is pregnant, trying to conceive or breastfeeding, warn against retinoids, hydroquinone and other ingredients unsafe in pregnancy unless a doctor approves. A changing mole, a bleeding lesion or a fast-spreading rash needs a doctor soon.'),
            $dept('nutrition', 'leaf', 5, '{"fa":"تغذیه","en":"Nutrition"}', '{"fa":"رژیم، مکمل، وزن","en":"Diet, supplements, weight"}', '{"fa":"پیشنهادهای ساده برای تغذیه سالم در هر مرحله از زندگی: سیکل، اقدام به بارداری، بارداری، شیردهی و یائسگی.","en":"Simple ideas for healthy eating at every life stage: cycle, trying to conceive, pregnancy, breastfeeding and menopause."}', '{"fa":["رژیم درمانی برای بیماری","تعیین دوز مکمل","برنامه کاهش وزن شدید"],"en":["Therapeutic diets for a disease","Supplement doses","Extreme weight-loss plans"]}', '{"fa":["تغذیه در بارداری","آهن و کم‌خونی","کاهش وزن","مکمل‌ها","چیز دیگر"],"en":["Pregnancy nutrition","Iron and anaemia","Weight loss","Supplements","Something else"]}', 'You are the nutrition advisor of the Ritme women\'s health app. Give practical, culturally familiar (Iranian) food suggestions for the user\'s life stage. Keep advice balanced and sustainable.', 'Never give supplement doses; say a doctor or dietitian should set them. Signs of an eating disorder: respond with care and suggest professional help. No diets below safe energy intake.'),
            $dept('mental_health', 'moon', 6, '{"fa":"سلامت روان","en":"Mental health"}', '{"fa":"خلق، خواب، استرس","en":"Mood, sleep, stress"}', '{"fa":"گفت‌وگویی آرام درباره خلق، استرس، خواب و ارتباط آن‌ها با سیکل و مراحل زندگی، همراه با تمرین‌های ساده.","en":"A calm conversation about mood, stress and sleep and how they relate to your cycle and life stage, with simple exercises."}', '{"fa":["تشخیص اختلال روانی","تجویز دارو","جایگزین روان‌درمانگر یا اورژانس"],"en":["Diagnosing a mental disorder","Prescribing medicines","Replacing a therapist or emergency care"]}', '{"fa":["استرس","بی‌خوابی","نوسان خلق قبل از پریود","غم بعد از زایمان","چیز دیگر"],"en":["Stress","Insomnia","Mood swings before my period","Sadness after birth","Something else"]}', 'You are the mental health companion of the Ritme women\'s health app. Listen first, reflect feelings, and offer simple evidence-based coping exercises (breathing, sleep hygiene, journaling). Consider cycle-related mood changes, postpartum mood and menopause.', 'Any sign of self-harm or suicidal thoughts: respond with empathy, urge contacting emergency services (115) or a trusted person right now, and do not continue with routine advice. Never name a psychiatric diagnosis or a medicine.'),
        ]);
        $term = fn (string $kind, string $locale, string $text) => [
            'kind' => $kind, 'locale' => $locale, 'term' => $text, 'is_active' => 1, 'created_at' => $now, 'updated_at' => $now,
        ];
        DB::table('assistant_safety_terms')->insertOrIgnore([
            $term('emergency', 'fa', 'خونریزی شدید'),
            $term('emergency', 'fa', 'خونریزی زیاد در بارداری'),
            $term('emergency', 'fa', 'بیهوش'),
            $term('emergency', 'fa', 'از هوش رفتم'),
            $term('emergency', 'fa', 'تشنج'),
            $term('emergency', 'fa', 'درد قفسه سینه'),
            $term('emergency', 'fa', 'درد سینه شدید'),
            $term('emergency', 'fa', 'نفس نمی‌کشم'),
            $term('emergency', 'fa', 'تنگی نفس شدید'),
            $term('emergency', 'fa', 'خودکشی'),
            $term('emergency', 'fa', 'خودم را بکشم'),
            $term('emergency', 'fa', 'خودمو بکشم'),
            $term('emergency', 'fa', 'درد شدید شکم'),
            $term('emergency', 'fa', 'مسمومیت'),
            $term('emergency', 'fa', 'اوردوز'),
            $term('emergency', 'fa', 'حرکت جنین را حس نمی‌کنم'),
            $term('emergency', 'fa', 'کیسه آب پاره'),
            $term('emergency', 'en', 'heavy bleeding'),
            $term('emergency', 'en', 'unconscious'),
            $term('emergency', 'en', 'fainted'),
            $term('emergency', 'en', 'seizure'),
            $term('emergency', 'en', 'chest pain'),
            $term('emergency', 'en', 'can\'t breathe'),
            $term('emergency', 'en', 'cannot breathe'),
            $term('emergency', 'en', 'suicide'),
            $term('emergency', 'en', 'kill myself'),
            $term('emergency', 'en', 'severe abdominal pain'),
            $term('emergency', 'en', 'overdose'),
            $term('emergency', 'en', 'poisoning'),
            $term('emergency', 'en', 'baby stopped moving'),
            $term('emergency', 'en', 'water broke'),
            $term('diagnosis', 'fa', 'تشخیص بده'),
            $term('diagnosis', 'fa', 'تشخیص بدی'),
            $term('diagnosis', 'fa', 'چه بیماری'),
            $term('diagnosis', 'fa', 'چه مریضی'),
            $term('diagnosis', 'fa', 'چه دارویی'),
            $term('diagnosis', 'fa', 'چه قرصی'),
            $term('diagnosis', 'fa', 'نسخه بنویس'),
            $term('diagnosis', 'fa', 'دوز'),
            $term('diagnosis', 'fa', 'چند میلی'),
            $term('diagnosis', 'en', 'diagnose'),
            $term('diagnosis', 'en', 'what disease'),
            $term('diagnosis', 'en', 'what illness'),
            $term('diagnosis', 'en', 'which medicine'),
            $term('diagnosis', 'en', 'what medicine'),
            $term('diagnosis', 'en', 'prescribe'),
            $term('diagnosis', 'en', 'prescription'),
            $term('diagnosis', 'en', 'dosage'),
            $term('diagnosis', 'en', 'what dose'),
        ]);
    }

    /**
     * Reverse the migrations.
     */
    public function down(): void
    {
        Schema::dropIfExists('assistant_handoffs');
        Schema::dropIfExists('assistant_messages');
        Schema::dropIfExists('assistant_conversations');
        Schema::dropIfExists('assistant_profile_settings');
        Schema::dropIfExists('assistant_safety_terms');
        Schema::dropIfExists('assistant_departments');
    }
};
