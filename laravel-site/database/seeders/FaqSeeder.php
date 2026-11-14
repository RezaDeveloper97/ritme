<?php

declare(strict_types=1);

namespace Database\Seeders;

use App\Domain\Faq\Actions\InvalidateFaqCache;
use App\Domain\Faq\Models\FaqGroup;
use App\Domain\Faq\Models\FaqItem;
use App\Domain\Faq\Observers\FaqObserver;
use Illuminate\Database\Seeder;

/**
 * Every FAQ of the design (L3-09): the five /faq categories (listed), the contextual groups of index («قبل از نصب»),
 * plus, contact and directory-business, and the six stage groups `stage-<slug>` — the latter copied from the stage
 * lang files (lang/fa/stages/<slug>.php `faq.items`) so the stage pages look exactly as before.
 *
 * Idempotent: groups are matched by slug (title/flags refreshed); items are only written into a group that has none,
 * so re-seeding never overwrites admin edits. Bracketed design placeholders («[زمان بررسی]» …) are kept verbatim for
 * the editors to replace. Model events may be off (DatabaseSeeder), so answers are cleaned here explicitly.
 */
class FaqSeeder extends Seeder
{
    /** @var list<string> */
    public const STAGES = ['cycle', 'ttc', 'pregnancy', 'postpartum', 'menopause', 'teen'];

    public function run(): void
    {
        $sort = 0;
        foreach (self::groups() as $slug => $group) {
            $model = FaqGroup::query()->updateOrCreate(
                ['slug' => $slug],
                ['title' => $group['title'], 'is_listed' => $group['listed'], 'sort_order' => $sort += 10],
            );

            if (FaqItem::query()->where('faq_group_id', $model->id)->exists()) {
                continue;
            }

            foreach ($group['items'] as $index => [$question, $answer]) {
                FaqItem::query()->create([
                    'faq_group_id' => $model->id,
                    'question' => $question,
                    'answer' => FaqObserver::cleanAnswer($answer),
                    'is_published' => true,
                    'sort_order' => ($index + 1) * 10,
                ]);
            }
        }

        // Observers may be muted while seeding: invalidate the FAQ and full-page caches explicitly.
        app(InvalidateFaqCache::class)();
    }

    /**
     * @return array<string, array{title: string, listed: bool, items: list<array{0: string, 1: string}>}>
     */
    public static function groups(): array
    {
        $groups = [
            // design/html/faq.html — the /faq categories, in design order.
            'getting-started' => ['title' => 'شروع کار', 'listed' => true, 'items' => [
                ['ریتمی چیست؟', 'همراه سلامت زنان، از اولین پریود تا یائسگی؛ با ثبت ساده، تحلیل صادقانه و خدمات مراقبتی.'],
                ['برای کدام گوشی‌هاست؟', 'اندروید و iOS؛ [نسخه سبک برای گوشی‌های قدیمی‌تر].'],
                ['چطور مرحله‌ام را عوض کنم؟', 'از «من › حالت اپ» یا چیپ بالای صفحه امروز.'],
            ]],
            'accuracy-health' => ['title' => 'دقت و سلامت', 'listed' => true, 'items' => [
                ['پیش‌بینی‌ها چقدر دقیق است؟', 'هر پیش‌بینی با سطح اطمینانش نشان داده می‌شود؛ هرچه ثبت بیشتر باشد، تصویر روشن‌تر است.'],
                ['ریتمی تشخیص پزشکی می‌دهد؟', 'نه. اگر چیزی ارزش پیگیری داشت، پیشنهاد می‌کنیم با پزشک صحبت کنی.'],
            ]],
            'privacy' => ['title' => 'حریم خصوصی', 'listed' => true, 'items' => [
                ['داده‌ام فروخته می‌شود؟', 'نه، هرگز.'],
                ['همدم چه چیزهایی را می‌بیند؟', 'فقط چیزهایی که خودت اجازه بدهی؛ هر وقت بخواهی قطعش کن.'],
                ['چطور داده‌ام را حذف کنم؟', 'از «من › پشتیبان و خروجی داده › حذف حساب».'],
            ]],
            'payment' => ['title' => 'پرداخت', 'listed' => true, 'items' => [
                ['چه چیزهایی رایگان است؟', 'فهرست کامل در صفحه «رایگان و پلاس» آمده است.'],
                ['چطور اشتراک را لغو کنم؟', 'از «من › اشتراک»، با یک دکمه.'],
            ]],
            'services-shop' => ['title' => 'خدمات و فروشگاه', 'listed' => true, 'items' => [
                ['رزرو کلاس چطور است؟', 'از «خدمات › مادر و کودک» مجموعه را پیدا کن و وقت خالی را رزرو کن.'],
                ['فروشگاه از داده سلامتم استفاده می‌کند؟', 'نه؛ مگر خودت جداگانه اجازه بدهی، مثل یادآور خرید قبل از پریود.'],
            ]],

            // design/html/index.html «قبل از نصب».
            'home' => ['title' => 'قبل از نصب', 'listed' => false, 'items' => [
                ['ریتمی رایگان است؟', 'بخش‌های اصلی ریتمی همیشه رایگان است. امکانات پیشرفته اختیاری در «ریتمی پلاس» جدا و شفاف اعلام شده‌اند.'],
                ['ریتمی جای پزشک را می‌گیرد؟', 'نه. ریتمی تشخیص نمی‌دهد؛ کمک می‌کند بدنت را بهتر بشناسی و اگر چیزی ارزش پیگیری داشت، به پزشک برسی.'],
                ['داده‌هایم کجا می‌رود؟', 'داده‌ات فروخته نمی‌شود و هر وقت بخواهی می‌توانی خروجی بگیری یا حذفش کنی.'],
                ['با تغییر مرحله زندگی، داده قبلی‌ام چه می‌شود؟', 'همه ثبت‌هایت حفظ می‌شود؛ فقط صفحه‌ها و ابزارها با مرحله جدید تنظیم می‌شوند.'],
            ]],

            // design/Web_Pricing.html «سؤالات متداول».
            'plus' => ['title' => 'سؤالات متداول', 'listed' => false, 'items' => [
                ['اگر پلاس را لغو کنم، داده‌ام چه می‌شود؟', 'هیچ‌چیز پاک نمی‌شود. همه ثبت‌هایت سر جایش می‌ماند و به نسخه رایگان برمی‌گردی.'],
                ['هشدارهای سلامت پولی است؟', 'نه. علائم خطر، هشدارها و اورژانس در هر حالتی رایگان می‌مانند.'],
                ['همراه بدون پلاس چه می‌شود؟', 'بعد از پایان پلاس، دسترسی همراه قطع می‌شود؛ داده تو دست نمی‌خورد.'],
                ['پرداخت و لغو چطور است؟', 'پرداخت از درگاه بانکی و لغو با یک لمس در «من › ریتمی پلاس». [سیاست بازگشت وجه پس از بررسی حقوقی]'],
            ]],

            // design/html/contact.html «شاید جوابت اینجا باشد».
            'contact' => ['title' => 'شاید جوابت اینجا باشد', 'listed' => false, 'items' => [
                ['رمز یا قفل اپ را فراموش کرده‌ام', '<p>[راهنمای بازیابی]</p><p>موضوع «حریم خصوصی و داده» مسیر جدا دارد: [ایمیل مسئول داده] — درخواست‌های دسترسی، اصلاح و حذف داده مستقیماً به مسئول داده می‌رسد.</p>'],
                ['چطور داده‌ام را خروجی بگیرم یا حذف کنم؟', 'از «من › پشتیبان و خروجی داده»؛ حذف حساب هم از همان‌جاست.'],
                ['مجموعه‌ام را چطور در ریتمی ثبت کنم؟', 'از صفحه «برای کسب‌وکارها» درخواست ثبت بده.'],
            ]],

            // design/html/directory-business.html «سؤال‌های رایج».
            'directory-business' => ['title' => 'سؤال‌های رایج', 'listed' => false, 'items' => [
                ['ثبت مجموعه چقدر طول می‌کشد؟', 'پر کردن فرم حدود ۱۵ دقیقه است. بررسی مدارک [زمان بررسی] روز کاری طول می‌کشد.'],
                ['آیا می‌توانم جایگاه بالاتری در نتایج بخرم؟', 'نه. ترتیب نتایج فقط بر اساس فاصله، سن کودک و نظر مادرهاست و آگهی ویژه نداریم.'],
                ['اگر رزرو آنلاین نخواهم چه؟', 'می‌توانی «فقط تماس تلفنی» را انتخاب کنی تا به‌جای دکمه رزرو، شماره مجموعه نمایش داده شود.'],
                ['چه اطلاعاتی از مادرها به من می‌رسد؟', 'فقط نام، نام و سن کودک و شماره تماس. هیچ داده سلامتی از ریتمی به مجموعه‌ها نمی‌رسد.'],
            ]],
        ];

        foreach (self::STAGES as $stage) {
            $groups["stage-{$stage}"] = ['title' => self::stageName($stage), 'listed' => false, 'items' => self::stageItems($stage)];
        }

        return $groups;
    }

    private static function stageName(string $stage): string
    {
        $name = trans("stages/{$stage}.name");

        return is_string($name) && $name !== "stages/{$stage}.name" ? $name : $stage;
    }

    /**
     * @return list<array{0: string, 1: string}>
     */
    private static function stageItems(string $stage): array
    {
        $items = trans("stages/{$stage}.faq.items");
        if (! is_array($items)) {
            return [];
        }

        $faq = [];
        foreach ($items as $item) {
            if (is_array($item) && is_string($item['question'] ?? null) && is_string($item['answer'] ?? null)) {
                $faq[] = [$item['question'], $item['answer']];
            }
        }

        return $faq;
    }
}
