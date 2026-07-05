<?php

namespace Database\Seeders;

use App\Models\Challenge;
use Illuminate\Database\Seeder;

class ChallengeSeeder extends Seeder
{
    public function run(): void
    {
        if (Challenge::query()->exists()) {
            return; // already seeded
        }

        $challenges = [
            [
                'title' => ['fa' => 'ثبت دمای بدن', 'en' => 'Log your body temperature'],
                'description' => ['fa' => 'امروز دمای پایه بدنت را ثبت کن تا روند سیکل دقیق‌تر شود', 'en' => 'Log your basal body temperature today for a more accurate cycle'],
                'difficulty' => 'easy',
                'category' => 'tracking',
            ],
            [
                'title' => ['fa' => '۱۰ دقیقه مدیتیشن', 'en' => '10 minutes of meditation'],
                'description' => ['fa' => 'برای کاهش استرس، ۱۰ دقیقه مدیتیشن کن', 'en' => 'Meditate for 10 minutes to reduce stress'],
                'difficulty' => 'easy',
                'category' => 'mindfulness',
            ],
            [
                'title' => ['fa' => 'یک وعده‌ی سالم بخور', 'en' => 'Eat one healthy meal'],
                'description' => ['fa' => 'امروز یک وعده‌ی متعادل و مغذی برای خودت آماده کن', 'en' => 'Prepare one balanced, nutritious meal today'],
                'difficulty' => 'medium',
                'category' => 'nutrition',
            ],
            [
                'title' => ['fa' => '۲۰ دقیقه پیاده‌روی', 'en' => 'Walk for 20 minutes'],
                'description' => ['fa' => 'یک پیاده‌روی سبک ۲۰ دقیقه‌ای انجام بده', 'en' => 'Go for a light 20-minute walk'],
                'difficulty' => 'medium',
                'category' => 'exercise',
            ],
            [
                'title' => ['fa' => 'زودتر بخواب', 'en' => 'Go to bed earlier'],
                'description' => ['fa' => 'امشب ۳۰ دقیقه زودتر بخواب تا خواب بهتری داشته باشی', 'en' => 'Sleep 30 minutes earlier tonight for better rest'],
                'difficulty' => 'easy',
                'category' => 'sleep',
            ],
        ];

        foreach ($challenges as $index => $challenge) {
            Challenge::create($challenge + [
                'cycle_phase' => null,
                'is_active' => true,
                'sort_order' => $index + 1,
            ]);
        }
    }
}
