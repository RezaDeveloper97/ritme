<?php

namespace Database\Seeders;

use App\Enums\CyclePhase;
use App\Models\Article;
use Illuminate\Database\Seeder;

class ArticleSeeder extends Seeder
{
    public function run(): void
    {
        $articles = [
            [
                'slug' => 'period-blood-clots',
                'title' => ['fa' => 'لخته‌های خون در دوران پریود؛ معنی آن چیست؟', 'en' => 'Blood clots during your period: what do they mean?'],
                'excerpt' => ['fa' => 'لخته‌های خون در پریود معمولاً طبیعی‌اند؛ اما چه زمانی باید نگران شد؟', 'en' => 'Period clots are usually normal — but when should you worry?'],
                'cycle_phases' => [CyclePhase::MENSTRUATION->value],
                'category' => 'period_health',
                'read_time_minutes' => 4,
                'sort_order' => 1,
            ],
            [
                'slug' => 'period-pain-relief',
                'title' => ['fa' => 'راه‌های کاهش درد پریود', 'en' => 'Ways to relieve period pain'],
                'excerpt' => ['fa' => 'از گرما درمانی تا تغذیه؛ روش‌های مؤثر برای کاهش دردهای قاعدگی', 'en' => 'From heat therapy to nutrition: effective ways to ease cramps'],
                'cycle_phases' => [CyclePhase::MENSTRUATION->value],
                'category' => 'period_health',
                'read_time_minutes' => 5,
                'sort_order' => 2,
            ],
            [
                'slug' => 'follicular-energy',
                'title' => ['fa' => 'فاز فولیکولار؛ زمان اوج انرژی', 'en' => 'The follicular phase: your energy peak'],
                'excerpt' => ['fa' => 'چرا بعد از پریود احساس انرژی بیشتری داری و چطور از آن استفاده کنی', 'en' => 'Why you feel more energetic after your period and how to use it'],
                'cycle_phases' => [CyclePhase::FOLLICULAR->value],
                'category' => 'cycle_science',
                'read_time_minutes' => 3,
                'sort_order' => 1,
            ],
            [
                'slug' => 'ovulation-signs',
                'title' => ['fa' => 'نشانه‌های تخمک‌گذاری را بشناس', 'en' => 'Know the signs of ovulation'],
                'excerpt' => ['fa' => 'تغییرات ترشحات، دمای بدن و علائمی که تخمک‌گذاری را نشان می‌دهند', 'en' => 'Discharge changes, body temperature and signs that signal ovulation'],
                'cycle_phases' => [CyclePhase::OVULATION->value],
                'category' => 'fertility',
                'read_time_minutes' => 4,
                'sort_order' => 1,
            ],
            [
                'slug' => 'pms-management',
                'title' => ['fa' => 'مدیریت علائم پی‌ام‌اس', 'en' => 'Managing PMS symptoms'],
                'excerpt' => ['fa' => 'راهکارهایی برای کنترل تغییرات خلقی و علائم جسمی پیش از پریود', 'en' => 'Strategies to manage mood and physical symptoms before your period'],
                'cycle_phases' => [CyclePhase::LUTEAL->value],
                'category' => 'wellbeing',
                'read_time_minutes' => 5,
                'sort_order' => 1,
            ],
            [
                'slug' => 'cycle-nutrition-basics',
                'title' => ['fa' => 'تغذیه متناسب با چرخه قاعدگی', 'en' => 'Eating in sync with your cycle'],
                'excerpt' => ['fa' => 'در هر فاز چه بخوریم تا حال بهتری داشته باشیم', 'en' => 'What to eat in each phase to feel your best'],
                'cycle_phases' => null,
                'category' => 'nutrition',
                'read_time_minutes' => 6,
                'sort_order' => 1,
            ],
        ];

        foreach ($articles as $article) {
            Article::updateOrCreate(
                ['slug' => $article['slug']],
                $article + ['is_published' => true, 'published_at' => now()]
            );
        }
    }
}
