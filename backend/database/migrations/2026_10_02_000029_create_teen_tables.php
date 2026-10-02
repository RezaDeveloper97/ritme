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
     * Twin of backend-go/db/migrations/00029_teen.sql
     * (docs/go-migration/migrations.md): the teen tables written by the
     * Go-only /api/v1/teen endpoints (CB-TEEN-01) plus the same
     * `catalog_items` seed rows (`teen_signs`, `teen_faq`, `teen_kit_items`,
     * needs clinical review), so `make schema-diff` row counts match. No model
     * or routes here. insertOrIgnore never overwrites an admin edit.
     */
    public function up(): void
    {
        Schema::create('teen_profiles', function (Blueprint $table) {
            $table->id();
            $table->foreignId('user_id')->unique()->constrained()->cascadeOnDelete();
            $table->string('age_band', 8);                       // 10_12|13_15|16_17
            $table->string('menarche', 16);                      // not_yet|under_1y|over_1y
            $table->string('parent_note', 280)->nullable();
            $table->timestamps();
        });

        Schema::create('teen_kit_checks', function (Blueprint $table) {
            $table->id();
            $table->foreignId('user_id')->constrained()->cascadeOnDelete();
            $table->string('item_code', 64);
            $table->timestamps();

            $table->unique(['user_id', 'item_code']);
        });

        $now = now();
        $json = fn (?array $v) => $v === null ? null : json_encode($v, JSON_UNESCAPED_UNICODE | JSON_UNESCAPED_SLASHES);
        $row = fn (string $group, string $code, int $sort, array $title, ?array $body, ?array $meta) => [
            'group' => $group,
            'code' => $code,
            'sort_order' => $sort,
            'is_active' => 1,
            'audiences' => $json(['teen']),
            'title' => $json($title),
            'body' => $json($body),
            'meta' => $json($meta),
            'needs_review' => 1,
            'created_at' => $now,
            'updated_at' => $now,
        ];
        $timing = [
            'fa' => 'هر بدنی زمان خودش را دارد. وقتی کیف اضطراری آماده باشد، غافلگیر نمی‌شوی.',
            'en' => "Every body has its own timing. With your emergency kit ready, you won't be caught off guard.",
        ];

        DB::table('catalog_items')->insertOrIgnore([
            $row('teen_signs', 'approaching_signs', 1, [
                'fa' => 'نشانه‌های نزدیک شدن اولین پریود',
                'en' => 'Signs your first period is getting close',
            ], [
                'fa' => 'معمولاً حدود ۲ سال بعد از شروع رشد سینه‌ها. ترشح سفید یا شفاف از چند ماه قبل هم طبیعی است.',
                'en' => 'It usually comes about 2 years after your breasts start to develop. White or clear discharge for a few months before is normal too.',
            ], ['kind' => 'sign', 'menarche' => ['not_yet']]),
            $row('teen_signs', 'growth_spurt', 2, ['fa' => 'جهش قد', 'en' => 'A growth spurt'], [
                'fa' => 'رشد ناگهانی قد و رویش مو زیر بغل معمولاً قبل از اولین پریود شروع می‌شود.',
                'en' => 'A sudden growth spurt and underarm hair usually start before the first period.',
            ], ['kind' => 'sign', 'menarche' => ['not_yet']]),
            $row('teen_signs', 'estimate_year_or_two', 3, [
                'fa' => 'بر اساس جواب‌هایت: احتمالاً در یکی دو سال آینده',
                'en' => 'Based on your answers: probably within the next year or two',
            ], $timing, ['kind' => 'estimate', 'menarche' => ['not_yet'], 'age_bands' => ['10_12']]),
            $row('teen_signs', 'estimate_coming_months', 4, [
                'fa' => 'بر اساس جواب‌هایت: احتمالاً در ماه‌های آینده',
                'en' => 'Based on your answers: probably in the coming months',
            ], $timing, ['kind' => 'estimate', 'menarche' => ['not_yet'], 'age_bands' => ['13_15']]),
            $row('teen_signs', 'estimate_talk', 5, [
                'fa' => 'بهتر است با مادرت یا پزشک صحبت کنی',
                'en' => "It's a good idea to talk to your mother or a doctor",
            ], [
                'fa' => 'بیشتر دخترها تا ۱۵ سالگی پریود می‌شوند. صحبت با پزشک کمک می‌کند مطمئن شوی همه چیز روبه‌راه است.',
                'en' => 'Most girls get their first period by 15. A doctor can help make sure everything is okay.',
            ], ['kind' => 'estimate', 'menarche' => ['not_yet'], 'age_bands' => ['16_17'], 'severity' => 'caution']),
            $row('teen_signs', 'estimate_first_year', 6, [
                'fa' => 'سال اول: نامنظم بودن طبیعی است',
                'en' => 'The first year: irregular is normal',
            ], [
                'fa' => 'در ۱ تا ۲ سال اول فاصله پریودها ممکن است خیلی فرق کند. ثبت کردن کمک می‌کند الگوی خودت را ببینی.',
                'en' => 'In the first 1–2 years the gap between periods can vary a lot. Logging helps you see your own pattern.',
            ], ['kind' => 'estimate', 'menarche' => ['under_1y']]),
            $row('teen_signs', 'estimate_settling', 7, [
                'fa' => 'چرخه‌ات در حال منظم شدن است',
                'en' => 'Your cycle is settling',
            ], [
                'fa' => 'ثبت کردن کمک می‌کند زمان پریود بعدی را زودتر بدانی و کیفت آماده باشد.',
                'en' => 'Logging helps you know when your next period is coming so your kit is ready.',
            ], ['kind' => 'estimate', 'menarche' => ['over_1y']]),
            $row('teen_signs', 'when_to_talk', 8, [
                'fa' => 'کی با مادرت یا پزشک صحبت کنی',
                'en' => 'When to talk to your mother or a doctor',
            ], [
                'fa' => 'اگر تا ۱۵ سالگی پریود نشدی، یا درد و خونریزی خیلی زیاد داری، با مادرت یا پزشک صحبت کن.',
                'en' => "If you haven't had a period by 15, or you have a lot of pain or very heavy bleeding, talk to your mother or a doctor.",
            ], ['kind' => 'talk']),
            $row('teen_faq', 'irregular', 1, [
                'fa' => 'پریودم نامنظم است، مشکلی دارد؟',
                'en' => 'My period is irregular — is something wrong?',
            ], [
                'fa' => 'در ۱ تا ۲ سال اول نامنظم بودن خیلی شایع و طبیعی است.',
                'en' => 'In the first 1–2 years irregular periods are very common and normal.',
            ], null),
            $row('teen_faq', 'how_much_bleeding', 2, [
                'fa' => 'چقدر خونریزی طبیعی است؟',
                'en' => 'How much bleeding is normal?',
            ], [
                'fa' => 'معمولاً ۳ تا ۷ روز. اگر هر ساعت نوار پر می‌شود، به مادرت یا پزشک بگو.',
                'en' => 'Usually 3 to 7 days. If you soak a pad every hour, tell your mother or a doctor.',
            ], null),
            $row('teen_faq', 'period_pain', 3, [
                'fa' => 'درد پریود چه کنم؟',
                'en' => 'What can I do about period pain?',
            ], [
                'fa' => 'گرما روی شکم، کمی تحرک و مسکن ساده با اجازه بزرگ‌ترها کمک می‌کند.',
                'en' => "Warmth on your tummy, some gentle movement and a simple painkiller with an adult's OK can help.",
            ], null),
            $row('teen_faq', 'pad_change', 4, [
                'fa' => 'هر چند وقت نوار را عوض کنم؟',
                'en' => 'How often should I change my pad?',
            ], [
                'fa' => 'معمولاً هر ۴ تا ۶ ساعت، یا زودتر اگر پر شده است.',
                'en' => 'Usually every 4 to 6 hours, or sooner if it is full.',
            ], null),
            $row('teen_faq', 'sports', 5, [
                'fa' => 'در پریود می‌توانم ورزش کنم؟',
                'en' => 'Can I do sports during my period?',
            ], [
                'fa' => 'بله. ورزش سبک حتی ممکن است درد را کمتر کند.',
                'en' => 'Yes. Gentle exercise may even ease cramps.',
            ], null),
            $row('teen_kit_items', 'pads', 1, ['fa' => '۲ نوار بهداشتی', 'en' => '2 sanitary pads'], null, null),
            $row('teen_kit_items', 'underwear', 2, ['fa' => 'یک لباس زیر اضافه', 'en' => 'A spare pair of underwear'], null, null),
            $row('teen_kit_items', 'wipes', 3, ['fa' => 'دستمال مرطوب', 'en' => 'Wet wipes'], null, null),
            $row('teen_kit_items', 'pouch', 4, ['fa' => 'کیسه کوچک', 'en' => 'A small pouch'], null, null),
        ]);
    }

    public function down(): void
    {
        DB::table('catalog_items')->where('group', 'teen_signs')
            ->whereIn('code', ['approaching_signs', 'growth_spurt', 'estimate_year_or_two', 'estimate_coming_months',
                'estimate_talk', 'estimate_first_year', 'estimate_settling', 'when_to_talk'])->delete();
        DB::table('catalog_items')->where('group', 'teen_faq')
            ->whereIn('code', ['irregular', 'how_much_bleeding', 'period_pain', 'pad_change', 'sports'])->delete();
        DB::table('catalog_items')->where('group', 'teen_kit_items')
            ->whereIn('code', ['pads', 'underwear', 'wipes', 'pouch'])->delete();
        Schema::dropIfExists('teen_kit_checks');
        Schema::dropIfExists('teen_profiles');
    }
};
