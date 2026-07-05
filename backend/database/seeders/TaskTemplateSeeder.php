<?php

namespace Database\Seeders;

use App\Enums\TaskCategory;
use App\Models\TaskTemplate;
use Illuminate\Database\Seeder;

class TaskTemplateSeeder extends Seeder
{
    public function run(): void
    {
        $templates = [
            [
                'key' => 'log_bbt',
                'title' => ['fa' => 'ثبت دمای بدن', 'en' => 'Log body temperature'],
                'description' => ['fa' => 'دمای پایه بدنت را هر روز صبح ثبت کن', 'en' => 'Record your basal body temperature each morning'],
                'category' => TaskCategory::TRACKING->value,
                'sort_order' => 1,
            ],
            [
                'key' => 'drink_water',
                'title' => ['fa' => 'نوشیدن ۸ لیوان آب', 'en' => 'Drink 8 glasses of water'],
                'description' => ['fa' => 'در طول روز به اندازه کافی آب بنوش', 'en' => 'Stay hydrated throughout the day'],
                'category' => TaskCategory::HYDRATION->value,
                'sort_order' => 2,
            ],
            [
                'key' => 'vitamin_d',
                'title' => ['fa' => 'مصرف ویتامین D', 'en' => 'Take vitamin D'],
                'description' => ['fa' => 'مکمل ویتامین D خود را مصرف کن', 'en' => 'Take your vitamin D supplement'],
                'category' => TaskCategory::SUPPLEMENT->value,
                'sort_order' => 3,
            ],
            [
                'key' => 'walk_30',
                'title' => ['fa' => '۳۰ دقیقه پیاده‌روی', 'en' => '30 minutes walking'],
                'description' => ['fa' => 'یک پیاده‌روی سبک ۳۰ دقیقه‌ای داشته باش', 'en' => 'Take a light 30-minute walk'],
                'category' => TaskCategory::EXERCISE->value,
                'sort_order' => 4,
            ],
            [
                'key' => 'breathing_5',
                'title' => ['fa' => '۵ دقیقه تنفس عمیق', 'en' => '5 minutes of deep breathing'],
                'description' => ['fa' => 'برای آرامش ذهن، تمرین تنفس عمیق انجام بده', 'en' => 'Practice deep breathing to calm your mind'],
                'category' => TaskCategory::MINDFULNESS->value,
                'sort_order' => 5,
            ],
        ];

        foreach ($templates as $template) {
            TaskTemplate::updateOrCreate(
                ['key' => $template['key']],
                $template + ['is_active' => true]
            );
        }
    }
}
