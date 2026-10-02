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
     * Twin of backend-go/db/migrations/00028_loss.sql
     * (docs/go-migration/migrations.md): the pregnancy-loss tables written
     * by the Go-only /api/v1/loss endpoints (CB-LOSS-01) plus the same
     * `catalog_items` seed rows (loss_types, loss_warning_signs,
     * loss_hotlines, loss_followups, loss_moods, loss_support,
     * loss_next_steps — needs clinical review), so `make schema-diff` row
     * counts match. No model or routes here. insertOrIgnore never overwrites
     * an admin edit.
     */
    public function up(): void
    {
        Schema::create('pregnancy_losses', function (Blueprint $table) {
            $table->id();
            $table->foreignId('user_id')->constrained()->cascadeOnDelete();
            $table->string('loss_type', 32)->default('unspecified'); // early_miscarriage|late_miscarriage|ectopic|chemical|unspecified
            $table->date('occurred_on')->nullable();
            $table->boolean('notify_companion')->default(false);
            $table->timestamp('companion_notified_at')->nullable();
            $table->timestamp('content_stopped_at')->nullable();
            $table->json('paused_reminders')->nullable();             // reminder ids paused by the loss
            $table->date('bleeding_stopped_on')->nullable();
            $table->date('beta_next_on')->nullable();
            $table->date('beta_negative_on')->nullable();
            $table->foreignId('beta_reminder_id')->nullable()->constrained('reminders')->nullOnDelete();
            $table->foreignId('visit_reminder_id')->nullable()->constrained('reminders')->nullOnDelete();
            $table->string('next_step', 16)->nullable();             // cycle|ttc|nothing
            $table->timestamp('next_step_at')->nullable();
            $table->text('private_note')->nullable();                // AES-256-GCM ciphertext (Go)
            $table->timestamp('note_updated_at')->nullable();
            $table->timestamps();

            $table->index(['user_id', 'created_at']);
        });

        Schema::create('pregnancy_loss_moods', function (Blueprint $table) {
            $table->id();
            $table->foreignId('user_id')->constrained()->cascadeOnDelete();
            $table->foreignId('loss_id')->constrained('pregnancy_losses')->cascadeOnDelete();
            $table->date('log_date');
            $table->string('mood', 16);                              // sad|numb|angry|a_bit_better
            $table->timestamps();

            $table->unique(['loss_id', 'log_date']);
            $table->index(['user_id', 'log_date']);
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
            $row('loss_types', 'early_miscarriage', 1, ['fa' => 'سقط در ۳ ماه اول', 'en' => 'Miscarriage in the first 3 months'], null, null),
            $row('loss_types', 'late_miscarriage', 2, ['fa' => 'سقط بعد از ۳ ماه اول', 'en' => 'Miscarriage after the first 3 months'], null, null),
            $row('loss_types', 'ectopic', 3, ['fa' => 'حاملگی خارج از رحم', 'en' => 'Ectopic pregnancy'], null, null),
            $row('loss_types', 'chemical', 4, ['fa' => 'بارداری شیمیایی یا تست مثبت کوتاه', 'en' => 'Chemical pregnancy or a brief positive test'], null, null),
            $row('loss_types', 'unspecified', 5, ['fa' => 'ترجیح می‌دهم نگویم', 'en' => 'I\'d rather not say'], null, null),
            $row('loss_warning_signs', 'heavy_bleeding', 1, ['fa' => 'خونریزی خیلی زیاد', 'en' => 'Very heavy bleeding'], ['fa' => 'پر شدن ۲ نوار در ساعت، ۲ ساعت پشت سر هم', 'en' => 'Soaking 2 pads an hour, 2 hours in a row'], ['severity' => 'urgent', 'hotline' => '115']),
            $row('loss_warning_signs', 'fever', 2, ['fa' => 'تب ۳۸ درجه یا بیشتر', 'en' => 'A fever of 38 °C or higher'], null, ['severity' => 'urgent', 'hotline' => '115']),
            $row('loss_warning_signs', 'severe_pain', 3, ['fa' => 'درد شدید شکم', 'en' => 'Severe pain in the abdomen'], null, ['severity' => 'urgent', 'hotline' => '115']),
            $row('loss_warning_signs', 'dizziness_fainting', 4, ['fa' => 'سرگیجه یا غش', 'en' => 'Dizziness or fainting'], null, ['severity' => 'urgent', 'hotline' => '115']),
            $row('loss_warning_signs', 'foul_discharge', 5, ['fa' => 'ترشح بدبو', 'en' => 'Foul-smelling discharge'], null, ['severity' => 'urgent', 'hotline' => '115']),
            $row('loss_hotlines', 'emergency', 1, ['fa' => 'اورژانس ۱۱۵', 'en' => 'Emergency 115'], ['fa' => 'برای علائم خطر جسمی، همین حالا تماس بگیر', 'en' => 'For any of the warning signs, call now'], ['number' => '115', 'kind' => 'medical']),
            $row('loss_hotlines', 'counselling', 2, ['fa' => 'صدای مشاور ۱۴۸۰', 'en' => 'Counselling line 1480'], ['fa' => 'مشاوره تلفنی برای وقتی که حالت سنگین است', 'en' => 'Phone counselling for when things feel heavy'], ['number' => '1480', 'kind' => 'mental_health']),
            $row('loss_hotlines', 'social_emergency', 3, ['fa' => 'اورژانس اجتماعی ۱۲۳', 'en' => 'Social emergency 123'], ['fa' => 'اگر به آسیب زدن به خودت فکر می‌کنی', 'en' => 'If you are thinking about hurting yourself'], ['number' => '123', 'kind' => 'crisis']),
            $row('loss_followups', 'bleeding', 1, ['fa' => 'خونریزی', 'en' => 'Bleeding'], ['fa' => 'ثبت روزانه تا قطع شدن', 'en' => 'Log it every day until it stops'], null),
            $row('loss_followups', 'beta', 2, ['fa' => 'آزمایش بتا تا منفی شدن', 'en' => 'Beta hCG test until negative'], ['fa' => 'تکرار آزمایش را با پزشکت هماهنگ کن', 'en' => 'Plan the repeat tests with your doctor'], null),
            $row('loss_followups', 'visit', 3, ['fa' => 'ویزیت پیگیری', 'en' => 'Follow-up visit'], ['fa' => 'حدود ۲ هفته بعد نوبت بگیر', 'en' => 'Book one for about 2 weeks later'], null),
            $row('loss_moods', 'sad', 1, ['fa' => 'غمگین', 'en' => 'Sad'], null, null),
            $row('loss_moods', 'numb', 2, ['fa' => 'بی‌حس', 'en' => 'Numb'], null, null),
            $row('loss_moods', 'angry', 3, ['fa' => 'عصبانی', 'en' => 'Angry'], null, null),
            $row('loss_moods', 'a_bit_better', 4, ['fa' => 'کمی بهتر', 'en' => 'A bit better'], null, null),
            $row('loss_support', 'mood_support', 1, ['fa' => 'حال دلت', 'en' => 'How you feel'], ['fa' => 'غم، احساس گناه یا بی‌حسی بعد از سقط طبیعی است و تقصیر تو نیست.', 'en' => 'Sadness, guilt or numbness after a pregnancy loss is natural, and it is not your fault.'], null),
            $row('loss_support', 'crisis', 2, ['fa' => 'اگر غم خیلی سنگین است', 'en' => 'If the sadness feels too heavy'], ['fa' => 'اگر غم خیلی سنگین است یا به آسیب زدن به خودت فکر می‌کنی، همین حالا با صدای مشاور ۱۴۸۰ یا اورژانس اجتماعی ۱۲۳ تماس بگیر.', 'en' => 'If the sadness feels too heavy or you are thinking about hurting yourself, call the 1480 counselling line or the 123 social emergency line right now.'], ['severity' => 'urgent', 'hotlines' => [['number' => '1480', 'label' => ['fa' => 'صدای مشاور', 'en' => 'Counselling line']], ['number' => '123', 'label' => ['fa' => 'اورژانس اجتماعی', 'en' => 'Social emergency']]]]),
            $row('loss_support', 'recurrent_hint', 3, ['fa' => 'اگر سقط دوم یا سوم بود', 'en' => 'If this was a second or third loss'], ['fa' => 'اگر سقط دوم یا سوم بود، درباره آزمایش‌های بررسی علت با پزشکت صحبت کن.', 'en' => 'If this was a second or third loss, talk with your doctor about tests to look for a cause.'], null),
            $row('loss_support', 'companion_notice', 4, ['fa' => '{name} خبر داد که بارداری ادامه ندارد.', 'en' => '{name} let you know that the pregnancy is not continuing.'], null, ['someone' => ['fa' => 'همراهت', 'en' => 'Your partner']]),
            $row('loss_next_steps', 'cycle', 1, ['fa' => 'فعلاً فقط پیگیری سیکل', 'en' => 'Just track my cycle for now'], ['fa' => 'اولین پریود معمولاً ۴ تا ۶ هفته بعد می‌آید', 'en' => 'The first period usually comes 4 to 6 weeks later'], ['life_mode' => 'cycle']),
            $row('loss_next_steps', 'ttc', 2, ['fa' => 'دوباره اقدام به بارداری', 'en' => 'Try to conceive again'], ['fa' => 'زمان مناسب را با پزشکت هماهنگ کن', 'en' => 'Agree on the right time with your doctor'], ['life_mode' => 'ttc']),
            $row('loss_next_steps', 'nothing', 3, ['fa' => 'فعلاً هیچ‌چیز', 'en' => 'Nothing for now'], ['fa' => 'فقط یادآور پیگیری‌های پزشکی می‌ماند', 'en' => 'Only the medical follow-up reminders stay'], ['life_mode' => 'cycle']),
        ]);
    }

    public function down(): void
    {
        $seeded = [
            'loss_types' => ['early_miscarriage', 'late_miscarriage', 'ectopic', 'chemical', 'unspecified'],
            'loss_warning_signs' => ['heavy_bleeding', 'fever', 'severe_pain', 'dizziness_fainting', 'foul_discharge'],
            'loss_hotlines' => ['emergency', 'counselling', 'social_emergency'],
            'loss_followups' => ['bleeding', 'beta', 'visit'],
            'loss_moods' => ['sad', 'numb', 'angry', 'a_bit_better'],
            'loss_support' => ['mood_support', 'crisis', 'recurrent_hint', 'companion_notice'],
            'loss_next_steps' => ['cycle', 'ttc', 'nothing'],
        ];
        foreach ($seeded as $group => $codes) {
            DB::table('catalog_items')->where('group', $group)->whereIn('code', $codes)->delete();
        }
        Schema::dropIfExists('pregnancy_loss_moods');
        Schema::dropIfExists('pregnancy_losses');
    }
};
