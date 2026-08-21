<?php

namespace Database\Seeders;

use App\Models\InfoSection;
use Illuminate\Database\Seeder;

/**
 * Seeds the app's text screens — راهنما و پشتیبانی, حریم خصوصی, قوانین,
 * درباره ما — with the copy that used to live in the frontend's `profileInfo`
 * messages, so each screen looks identical the day it becomes admin-managed and
 * admins have real boxes to edit rather than a blank page.
 *
 * Idempotent and non-destructive: rows are matched on their stable
 * (group, key) pair and only ever inserted, so re-running never overwrites
 * edited text or re-enables a box an admin deliberately switched off.
 */
class InfoSectionSeeder extends Seeder
{
    public function run(): void
    {
        foreach ($this->rows() as $group => $rows) {
            foreach ($rows as $index => $row) {
                InfoSection::firstOrCreate(
                    ['group' => $group, 'key' => $row['key']],
                    [
                        'heading' => $row['heading'],
                        'body' => $row['body'],
                        'link_label' => $row['link_label'] ?? null,
                        'link_url' => $row['link_url'] ?? null,
                        // Leave gaps so an admin can slot a box in between two
                        // shipped ones without renumbering the rest.
                        'sort_order' => ($index + 1) * 10,
                        'is_active' => true,
                    ]
                );
            }
        }
    }

    /** @return array<string, array<int, array<string, mixed>>> */
    private function rows(): array
    {
        return [
            'help' => [
                [
                    'key' => 'home',
                    'heading' => ['fa' => 'خانه', 'en' => 'Home'],
                    'body' => ['fa' => 'توی صفحه خانه، خلاصه‌ای از وضعیت امروزت رو می‌بینی: روز چرخه یا هفته بارداری، پیش‌بینی‌های نزدیک و پیام روزانه‌ای که مخصوص شرایط خودته.', 'en' => 'The home screen shows a summary of your day: your cycle day or pregnancy week, upcoming predictions, and a daily message tailored to you.'],
                ],
                [
                    'key' => 'calendar',
                    'heading' => ['fa' => 'تقویم', 'en' => 'Calendar'],
                    'body' => ['fa' => 'تقویم شمسی، دوره‌هایی که ثبت کردی و روزهای پیش‌بینی‌شده رو نشون می‌ده. با زدن روی هر روز می‌تونی جزئیات همون روز رو ببینی یا ویرایشش کنی.', 'en' => 'The Jalali calendar displays your logged periods and predicted days. Tap any day to see its details or edit its information.'],
                ],
                [
                    'key' => 'daily-log',
                    'heading' => ['fa' => 'ثبت روزانه', 'en' => 'Daily log'],
                    'body' => ['fa' => 'از بخش ثبت می‌تونی هر روز حال عمومی، علائم، میزان خونریزی و بقیه چیزها رو ثبت کنی. هرچی منظم‌تر ثبت کنی، پیش‌بینی‌ها دقیق‌تر می‌شن.', 'en' => 'From the log section you can record your general mood, symptoms, flow, and more each day. Logging regularly improves the accuracy of predictions.'],
                ],
                [
                    'key' => 'profile',
                    'heading' => ['fa' => 'پروفایل', 'en' => 'Profile'],
                    'body' => ['fa' => 'توی پروفایل می‌تونی اطلاعات شخصی و سلامتیت رو ویرایش کنی، زبان اپ رو عوض کنی، یه نسخه از داده‌هات رو بگیری یا حسابت رو حذف کنی.', 'en' => 'In your profile you can edit your personal and health information, change the app language, download a copy of your data, or delete your account.'],
                ],
                [
                    'key' => 'support',
                    'heading' => ['fa' => 'پشتیبانی', 'en' => 'Support'],
                    'body' => ['fa' => 'اگه سؤالی داری یا به مشکلی خوردی، برامون بنویس؛ هر چه زودتر جوابت رو می‌دیم.', 'en' => 'If you have a question or run into a problem, write to us and we\'ll get back to you as soon as we can.'],
                    'link_label' => ['fa' => 'ارسال ایمیل به پشتیبانی', 'en' => 'Email support'],
                    'link_url' => 'mailto:support@ritmesalamat.com',
                ],
            ],
            'privacy' => [
                [
                    'key' => 'commitment',
                    'heading' => ['fa' => 'قولی که بهت می‌دیم', 'en' => 'Our commitment to you'],
                    'body' => ['fa' => 'ریتمی جاییه که خصوصی‌ترین اطلاعات سلامتی‌ات رو ثبت می‌کنی؛ برای همین حریم خصوصی پیش‌فرض ماست، نه یه گزینه. ما فقط همون حداقل اطلاعاتی رو جمع می‌کنیم که اپ بدون اون کار نمی‌کنه و هیچ داده‌ای رو «برای روز مبادا» نگه نمی‌داریم.', 'en' => 'Ritme is built to hold some of your most personal health information, so privacy is our default, not an option. We collect only the minimum data the app needs to work, and we never store anything "just in case".'],
                ],
                [
                    'key' => 'never-sold',
                    'heading' => ['fa' => 'داده‌هات فروخته نمی‌شه', 'en' => 'Your data is never sold'],
                    'body' => ['fa' => 'اطلاعاتت هیچ‌وقت به هیچ شخص یا شرکتی فروخته یا اجاره داده نمی‌شه و برای تبلیغات هم استفاده نمی‌شه. داده‌هات فقط برای این‌که ریتمی به خودت سرویس بده به کار می‌ره.', 'en' => 'Your information is never sold or rented to any person or company, and it is not used for advertising. Your data is used solely to provide Ritme\'s services to you.'],
                ],
                [
                    'key' => 'health-confidential',
                    'heading' => ['fa' => 'داده‌های سلامتی‌ات محرمانه می‌مونه', 'en' => 'Health data stays confidential'],
                    'body' => ['fa' => 'تاریخ‌های دوره، علائم، وضعیت بارداری و پیش‌بینی‌ها هیچ‌وقت توی گزارش‌های فنی، ابزارهای تحلیلی یا گزارش خطا ثبت نمی‌شن و با هیچ سرویس شخص ثالثی به اشتراک گذاشته نمی‌شن. فقط با ورود امن خودت به حساب کاربری می‌شه به این داده‌ها رسید.', 'en' => 'Period dates, symptoms, pregnancy status, and predictions are never written to technical logs, analytics tools, or error reports, and they are never shared with third-party services. This data is only accessible through your own secure sign-in.'],
                ],
                [
                    'key' => 'export-delete',
                    'heading' => ['fa' => 'هر وقت خواستی، داده‌هات رو بگیر یا پاک کن', 'en' => 'Your right to export and delete'],
                    'body' => ['fa' => 'داده‌ها مال خودته. هر وقت بخوای می‌تونی از بخش پروفایل، یه نسخه کامل از اطلاعاتت رو توی یه فایل بگیری یا حسابت و همه داده‌هاش رو برای همیشه حذف کنی. حذف حساب فوریه و برگشتی نداره.', 'en' => 'You own your data. At any time, you can download a complete copy of your information as a file from your profile, or permanently delete your account and everything in it. Account deletion is immediate and irreversible.'],
                ],
            ],
            'terms' => [
                [
                    'key' => 'informational-not-medical-advice',
                    'heading' => ['fa' => 'اطلاع‌رسانیه، نه توصیه پزشکی', 'en' => 'Informational, not medical advice'],
                    'body' => ['fa' => 'محتوا و پیش‌بینی‌های ریتمی — از جمله پیش‌بینی زمان دوره، تخمک‌گذاری و محتوای هفته‌های بارداری — فقط جنبه اطلاع‌رسانی داره و جای معاینه، تشخیص یا توصیه پزشک رو نمی‌گیره. برای هر تصمیمی که به سلامتیت مربوطه، با پزشک یا ماما مشورت کن.', 'en' => 'Ritme\'s content and predictions — including period and ovulation predictions and weekly pregnancy content — are for informational purposes only and are not a substitute for examination, diagnosis, or advice from a doctor. For any health-related decision, please consult a physician or midwife.'],
                ],
                [
                    'key' => 'acceptable-use',
                    'heading' => ['fa' => 'استفاده درست از اپ', 'en' => 'Acceptable use'],
                    'body' => ['fa' => 'ریتمی برای استفاده شخصی خودته. اطلاعات رو صادقانه و برای حساب خودت ثبت کن، از حساب دیگران استفاده نکن و سراغ دسترسی غیرمجاز به سامانه یا داده‌های بقیه کاربرها نرو.', 'en' => 'Ritme is for your personal use. Please log information honestly and only for your own account, do not use someone else\'s account, and do not attempt unauthorized access to the system or to other users\' data.'],
                ],
                [
                    'key' => 'your-account-and-one-time-codes',
                    'heading' => ['fa' => 'حساب کاربری و ورود با کد یک‌بارمصرف', 'en' => 'Your account and one-time codes'],
                    'body' => ['fa' => 'ورود به ریتمی با شماره موبایل و کد یک‌بارمصرف (OTP) انجام می‌شه. امنیت شماره موبایل و گوشیت با خودته؛ کد ورود رو در اختیار کسی نذار.', 'en' => 'Signing in to Ritme uses your mobile number and a one-time code (OTP). Keeping your mobile number and device secure is your responsibility; never share your sign-in code with anyone.'],
                ],
                [
                    'key' => 'deleting-your-data',
                    'heading' => ['fa' => 'حذف اطلاعات', 'en' => 'Deleting your data'],
                    'body' => ['fa' => 'هر وقت بخوای می‌تونی حسابت رو از بخش پروفایل حذف کنی. با حذف حساب، همه داده‌هات از سرورهای ما پاک می‌شه و نشست‌های فعالت هم باطل می‌شه.', 'en' => 'You can delete your account from the profile section at any time. When you delete your account, all of your data is erased from our servers and your active sessions are revoked.'],
                ],
            ],
            'about' => [
                [
                    'key' => 'what-is-ritme',
                    'heading' => ['fa' => 'ریتمی چیه؟', 'en' => 'What is Ritme?'],
                    'body' => ['fa' => 'ریتمی یه همراه خصوصی برای سلامت زنانه؛ اپی که کمکت می‌کنه ریتم بدنت رو بهتر بشناسی، هر روز حال و علائمت رو ثبت کنی و پیش‌بینی‌ها و محتوایی متناسب با شرایط خودت بگیری.', 'en' => 'Ritme is a private companion for women\'s health — an app for understanding your body\'s rhythm, logging how you feel each day, and receiving predictions and content tailored to your situation.'],
                ],
                [
                    'key' => 'two-modes-cycle-and-pregnancy',
                    'heading' => ['fa' => 'دو تا حالت: چرخه و بارداری', 'en' => 'Two modes: cycle and pregnancy'],
                    'body' => ['fa' => 'توی حالت چرخه، ریتمی دوره‌های قاعدگی‌ات رو ثبت می‌کنه و زمان تقریبی دوره بعدی و تخمک‌گذاری رو پیش‌بینی می‌کنه. توی حالت بارداری، هفته‌به‌هفته کنارته: محتوای مخصوص هر هفته، یادآورها و ثبت علائم و حرکات جنین. این پیش‌بینی‌ها و محتواها فقط اطلاع‌رسانیه و جای نظر پزشک رو نمی‌گیره.', 'en' => 'In cycle mode, Ritme tracks your menstrual cycles and estimates when your next period and ovulation are likely to occur. In pregnancy mode, it accompanies you week by week with stage-appropriate content, reminders, and tools to log symptoms and fetal movement. These predictions and content are informational and do not replace a doctor\'s advice.'],
                ],
                [
                    'key' => 'app-version',
                    'heading' => ['fa' => 'نسخه اپ', 'en' => 'App version'],
                    'body' => ['fa' => 'الان داری از نسخه ۱٫۰٫۰ ریتمی استفاده می‌کنی. ما مرتب اپ رو بهتر می‌کنیم؛ نظرت توی این مسیر خیلی برامون ارزش داره.', 'en' => 'You are using Ritme version 1.0.0. We improve the app regularly — your feedback is invaluable along the way.'],
                ],
            ],
        ];
    }
}
