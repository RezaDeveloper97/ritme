<?php

use Illuminate\Database\Migrations\Migration;
use Illuminate\Support\Facades\DB;

return new class extends Migration
{
    /**
     * Twin of backend-go/db/migrations/00047_catalog_services_hub.sql (docs/go-migration/migrations.md): the
     * `catalog_items` rows of the Go-only «خدمات» hub (bloom B-N7-01, GET /api/v1/services) — groups
     * services_sections, services_care and services_programs — so `make schema-diff` row counts match. Data only;
     * no model or routes here. insertOrIgnore never overwrites an admin edit.
     */
    public function up(): void
    {
        $now = now();
        $json = fn (?array $v) => $v === null ? null : json_encode($v, JSON_UNESCAPED_UNICODE | JSON_UNESCAPED_SLASHES);
        $row = fn (string $group, string $code, int $sort, ?array $audiences, array $title, ?array $body, ?array $meta) => [
            'group' => $group,
            'code' => $code,
            'sort_order' => $sort,
            'is_active' => 1,
            'audiences' => $json($audiences),
            'title' => $json($title),
            'body' => $json($body),
            'meta' => $json($meta),
            'needs_review' => 0,
            'created_at' => $now,
            'updated_at' => $now,
        ];

        DB::table('catalog_items')->insertOrIgnore([
            $row('services_sections', 'search', 1, null, ['fa' => 'پزشک، آزمایش، کلاس یا برنامه…', 'en' => 'Doctors, labs, classes or programs…'], null, ['href' => '/search']),
            $row('services_sections', 'booking', 2, null, ['fa' => 'نوبت پیش رو', 'en' => 'Upcoming visit'], null, null),
            $row('services_sections', 'care', 3, null, ['fa' => 'مراقبت سلامت', 'en' => 'Health care'], null, null),
            $row('services_sections', 'checkups', 4, null, ['fa' => 'چکاپ‌های دوره‌ای و یادآور دارو', 'en' => 'Routine checkups & medication reminders'], null, ['href' => '/checkups']),
            $row('services_sections', 'programs', 5, null, ['fa' => 'برنامه‌های مراقبتی', 'en' => 'Care programs'], ['fa' => 'بر اساس چیزی که خودت فعال کنی', 'en' => 'Based on what you turn on'], null),
            $row('services_sections', 'mother_child', 6, null, ['fa' => 'برای مادر و کودک', 'en' => 'For mother & child'], null, ['caption' => ['fa' => 'کلاس، استخر و خانه بازی نزدیک تو', 'en' => 'Classes, pools and play centres near you']]),
            $row('services_sections', 'learning', 7, null, ['fa' => 'آموزش', 'en' => 'Learning'], ['fa' => 'دوره‌ها و کلاس‌های آنلاین', 'en' => 'Online courses and classes'], ['caption' => ['fa' => 'آمادگی زایمان، شیردهی، یائسگی', 'en' => 'Birth prep, breastfeeding, menopause']]),
            $row('services_sections', 'shop', 8, null, ['fa' => 'فروشگاه', 'en' => 'Shop'], ['fa' => 'فروشگاه از داده سلامت تو جداست و پیشنهادهایش بر اساس ثبت‌هایت نیست.', 'en' => 'The shop is separate from your health data and its suggestions are not based on what you log.'], ['categories' => [['code' => 'layette', 'icon' => 'bottle', 'title' => ['fa' => 'سیسمونی و نوزاد', 'en' => 'Layette & baby']], ['code' => 'beauty', 'icon' => 'dropLine', 'title' => ['fa' => 'آرایشی و بهداشتی', 'en' => 'Beauty & hygiene']]]]),
            $row('services_sections', 'emergency', 9, null, ['fa' => 'اورژانس است؟', 'en' => 'Is it an emergency?'], ['fa' => 'ریتمی جایگزین اورژانس نیست', 'en' => 'Ritme is not a substitute for emergency care'], ['phone' => '115']),
            $row('services_care', 'assistant', 1, null, ['fa' => 'دستیار سلامت', 'en' => 'Health assistant'], ['fa' => 'پاسخ و ارجاع', 'en' => 'Answers and referrals'], ['icon' => 'sparkle', 'tone' => 'brand']),
            $row('services_care', 'doctors', 2, null, ['fa' => 'پزشک و ماما', 'en' => 'Doctors & midwives'], ['fa' => 'ویدیویی، حضوری', 'en' => 'Video or in person'], ['icon' => 'stetho', 'tone' => 'data']),
            $row('services_care', 'record', 3, null, ['fa' => 'پرونده سلامت', 'en' => 'Health record'], ['fa' => 'سوابق و مدارک', 'en' => 'History and documents'], ['icon' => 'fileDoc', 'tone' => 'brand', 'href' => '/record', 'counter' => 'record_documents']),
            $row('services_care', 'labs', 4, null, ['fa' => 'تحلیل آزمایش', 'en' => 'Lab analysis'], ['fa' => 'عکس برگه آزمایش', 'en' => 'Photo of your lab sheet'], ['icon' => 'flask', 'tone' => 'period', 'href' => '/labs']),
            $row('services_care', 'vitals', 5, null, ['fa' => 'علائم حیاتی', 'en' => 'Vital signs'], ['fa' => 'فشار، قند، ضربان', 'en' => 'Blood pressure, sugar, pulse'], ['icon' => 'heartLine', 'tone' => 'period', 'href' => '/vitals']),
            $row('services_care', 'insurance', 6, null, ['fa' => 'بیمه', 'en' => 'Insurance'], ['fa' => 'پوشش و خسارت', 'en' => 'Coverage and claims'], ['icon' => 'shield', 'tone' => 'data']),
            $row('services_programs', 'pain_endometriosis', 1, null, ['fa' => 'درد و اندومتریوز', 'en' => 'Pain & endometriosis'], ['fa' => 'دفترچه درد و گزارش', 'en' => 'Pain diary and report'], ['icon' => 'flame', 'tone' => 'period']),
            $row('services_programs', 'pmdd', 2, null, ['fa' => 'PMDD و خلق', 'en' => 'PMDD & mood'], ['fa' => 'پرسشنامه ماهانه', 'en' => 'Monthly questionnaire'], ['icon' => 'smile', 'tone' => 'brand']),
            $row('services_programs', 'heavy_bleeding', 3, null, ['fa' => 'خونریزی زیاد', 'en' => 'Heavy bleeding'], ['fa' => 'جدول خونریزی', 'en' => 'Bleeding chart'], ['icon' => 'drop', 'tone' => 'period']),
            $row('services_programs', 'pelvic_floor', 4, null, ['fa' => 'کف لگن', 'en' => 'Pelvic floor'], ['fa' => 'برنامه ۸ هفته‌ای', 'en' => '8-week program'], ['icon' => 'target', 'tone' => 'data']),
            $row('services_programs', 'contraception', 5, ['cycle', 'postpartum', 'menopause'], ['fa' => 'پیشگیری', 'en' => 'Contraception'], ['fa' => 'قرص و روش‌ها', 'en' => 'Pill and methods'], ['icon' => 'pill', 'tone' => 'brand', 'href' => '/contraception']),
        ]);
    }

    public function down(): void
    {
        DB::table('catalog_items')->where('group', 'services_sections')
            ->whereIn('code', ['search', 'booking', 'care', 'checkups', 'programs', 'mother_child', 'learning', 'shop', 'emergency'])->delete();
        DB::table('catalog_items')->where('group', 'services_care')
            ->whereIn('code', ['assistant', 'doctors', 'record', 'labs', 'vitals', 'insurance'])->delete();
        DB::table('catalog_items')->where('group', 'services_programs')
            ->whereIn('code', ['pain_endometriosis', 'pmdd', 'heavy_bleeding', 'pelvic_floor', 'contraception'])->delete();
    }
};
