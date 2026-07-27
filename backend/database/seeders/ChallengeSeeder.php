<?php

namespace Database\Seeders;

use App\Models\Challenge;
use Illuminate\Database\Seeder;

/**
 * The daily-challenge pool.
 *
 * {@see \App\Services\Challenges\DailyChallengeService} narrows this pool by
 * cycle phase, then by the category the user's recent logs point at, then by
 * the difficulty their streak has unlocked — so every (category, difficulty)
 * combination needs a few entries or the pick falls back to the whole pool.
 *
 * Idempotent: keyed by `slug`, so re-running only adds what's missing and
 * refreshes wording. Admin-authored challenges (no slug) are never touched.
 */
class ChallengeSeeder extends Seeder
{
    public function run(): void
    {
        $this->adoptLegacyRows();

        foreach ($this->challenges() as $index => $challenge) {
            Challenge::updateOrCreate(
                ['slug' => $challenge['slug']],
                $challenge + [
                    'cycle_phase' => null,
                    'is_active' => true,
                    'sort_order' => $index + 1,
                ],
            );
        }
    }

    /**
     * Installs seeded before `slug` existed hold the same challenges without
     * one. Claim those rows by Persian title so re-seeding updates them in
     * place instead of creating a duplicate — and so the completions already
     * recorded against them stay attached.
     */
    private function adoptLegacyRows(): void
    {
        $legacy = Challenge::query()->whereNull('slug')->get();

        if ($legacy->isEmpty()) {
            return;
        }

        foreach ($this->challenges() as $challenge) {
            if (Challenge::query()->where('slug', $challenge['slug'])->exists()) {
                continue;
            }

            $match = $legacy->first(
                fn (Challenge $row) => ($row->title['fa'] ?? null) === $challenge['title']['fa'],
            );

            if ($match) {
                $match->update(['slug' => $challenge['slug']]);
                $legacy = $legacy->except($match->id);
            }
        }
    }

    /**
     * @return array<int, array<string, mixed>>
     */
    private function challenges(): array
    {
        return [
            // ── tracking ────────────────────────────────────────────────
            [
                'slug' => 'log-basal-temperature',
                'title' => ['fa' => 'ثبت دمای بدن', 'en' => 'Log your body temperature'],
                'description' => ['fa' => 'امروز دمای پایه بدنت را ثبت کن تا روند سیکل دقیق‌تر شود', 'en' => 'Log your basal body temperature today for a more accurate cycle'],
                'difficulty' => 'easy',
                'category' => 'tracking',
            ],
            [
                'slug' => 'log-mood',
                'title' => ['fa' => 'حال امروزت را ثبت کن', 'en' => 'Log how you feel today'],
                'description' => ['fa' => 'یک دقیقه وقت بگذار و حال و حوصله‌ی امروزت را ثبت کن', 'en' => "Take a minute to log today's mood"],
                'difficulty' => 'easy',
                'category' => 'tracking',
            ],
            [
                'slug' => 'log-symptoms',
                'title' => ['fa' => 'علائم امروز را کامل کن', 'en' => "Complete today's symptoms"],
                'description' => ['fa' => 'علائم امروزت را ثبت کن تا الگوهای بدنت واضح‌تر شود', 'en' => "Log today's symptoms so your patterns become clearer"],
                'difficulty' => 'medium',
                'category' => 'tracking',
            ],
            [
                'slug' => 'log-seven-days',
                'title' => ['fa' => 'هفت روز پیاپی ثبت کن', 'en' => 'Log seven days in a row'],
                'description' => ['fa' => 'امروز هم ثبت کن تا زنجیره‌ی هفت‌روزه‌ات کامل شود', 'en' => 'Log today to complete your seven-day run'],
                'difficulty' => 'hard',
                'category' => 'tracking',
            ],

            // ── mindfulness ─────────────────────────────────────────────
            [
                'slug' => 'meditate-10-minutes',
                'title' => ['fa' => '۱۰ دقیقه مدیتیشن', 'en' => '10 minutes of meditation'],
                'description' => ['fa' => 'برای کاهش استرس، ۱۰ دقیقه مدیتیشن کن', 'en' => 'Meditate for 10 minutes to reduce stress'],
                'difficulty' => 'easy',
                'category' => 'mindfulness',
            ],
            [
                'slug' => 'deep-breathing',
                'title' => ['fa' => 'پنج نفس عمیق', 'en' => 'Five deep breaths'],
                'description' => ['fa' => 'هر وقت فشار را حس کردی، پنج نفس عمیق و آرام بکش', 'en' => 'Take five slow, deep breaths whenever tension builds'],
                'difficulty' => 'easy',
                'category' => 'mindfulness',
            ],
            [
                'slug' => 'gratitude-note',
                'title' => ['fa' => 'سه چیز خوب امروز', 'en' => 'Three good things'],
                'description' => ['fa' => 'سه چیز که امروز بابتشان سپاسگزاری، برای خودت بنویس', 'en' => "Write down three things you're grateful for today"],
                'difficulty' => 'medium',
                'category' => 'mindfulness',
            ],
            [
                'slug' => 'screen-free-hour',
                'title' => ['fa' => 'یک ساعت بدون صفحه‌نمایش', 'en' => 'One screen-free hour'],
                'description' => ['fa' => 'یک ساعت از امروز را بدون گوشی و نمایشگر بگذران', 'en' => 'Spend one hour today away from phones and screens'],
                'difficulty' => 'hard',
                'category' => 'mindfulness',
            ],

            // ── nutrition ───────────────────────────────────────────────
            [
                'slug' => 'healthy-meal',
                'title' => ['fa' => 'یک وعده‌ی سالم بخور', 'en' => 'Eat one healthy meal'],
                'description' => ['fa' => 'امروز یک وعده‌ی متعادل و مغذی برای خودت آماده کن', 'en' => 'Prepare one balanced, nutritious meal today'],
                'difficulty' => 'medium',
                'category' => 'nutrition',
            ],
            [
                'slug' => 'drink-water',
                'title' => ['fa' => 'هشت لیوان آب', 'en' => 'Eight glasses of water'],
                'description' => ['fa' => 'امروز هشت لیوان آب بنوش؛ کم‌آبی خستگی را بیشتر می‌کند', 'en' => 'Drink eight glasses of water today — dehydration deepens fatigue'],
                'difficulty' => 'easy',
                'category' => 'nutrition',
            ],
            [
                'slug' => 'iron-rich-food',
                'title' => ['fa' => 'یک خوراکی آهن‌دار', 'en' => 'One iron-rich food'],
                'description' => ['fa' => 'خوراکی آهن‌دار مثل عدس، اسفناج یا خرما به وعده‌ات اضافه کن', 'en' => 'Add an iron-rich food like lentils, spinach or dates to a meal'],
                'difficulty' => 'easy',
                'category' => 'nutrition',
            ],
            [
                'slug' => 'no-added-sugar',
                'title' => ['fa' => 'یک روز بدون قند افزوده', 'en' => 'A day without added sugar'],
                'description' => ['fa' => 'امروز را بدون نوشیدنی و خوراکی شیرین‌شده بگذران', 'en' => 'Go today without sweetened drinks or snacks'],
                'difficulty' => 'hard',
                'category' => 'nutrition',
            ],

            // ── exercise ────────────────────────────────────────────────
            [
                'slug' => 'walk-20-minutes',
                'title' => ['fa' => '۲۰ دقیقه پیاده‌روی', 'en' => 'Walk for 20 minutes'],
                'description' => ['fa' => 'یک پیاده‌روی سبک ۲۰ دقیقه‌ای انجام بده', 'en' => 'Go for a light 20-minute walk'],
                'difficulty' => 'medium',
                'category' => 'exercise',
            ],
            [
                'slug' => 'gentle-stretch',
                'title' => ['fa' => '۵ دقیقه کشش', 'en' => '5 minutes of stretching'],
                'description' => ['fa' => 'چند حرکت کششی ملایم برای کمر و لگن انجام بده', 'en' => 'Do a few gentle stretches for your lower back and hips'],
                'difficulty' => 'easy',
                'category' => 'exercise',
                'cycle_phase' => 'menstruation',
            ],
            [
                'slug' => 'strength-session',
                'title' => ['fa' => 'یک تمرین قدرتی کوتاه', 'en' => 'A short strength session'],
                'description' => ['fa' => 'انرژی امروزت خوب است؛ ۲۰ دقیقه تمرین قدرتی انجام بده', 'en' => 'Your energy is up today — do 20 minutes of strength work'],
                'difficulty' => 'hard',
                'category' => 'exercise',
            ],

            // ── sleep ───────────────────────────────────────────────────
            [
                'slug' => 'sleep-earlier',
                'title' => ['fa' => 'زودتر بخواب', 'en' => 'Go to bed earlier'],
                'description' => ['fa' => 'امشب ۳۰ دقیقه زودتر بخواب تا خواب بهتری داشته باشی', 'en' => 'Sleep 30 minutes earlier tonight for better rest'],
                'difficulty' => 'easy',
                'category' => 'sleep',
            ],
            [
                'slug' => 'no-caffeine-after-four',
                'title' => ['fa' => 'بعد از عصر بدون کافئین', 'en' => 'No caffeine after 4pm'],
                'description' => ['fa' => 'امروز بعد از ساعت ۴ چای و قهوه نخور تا شب راحت‌تر بخوابی', 'en' => 'Skip coffee and tea after 4pm for an easier night'],
                'difficulty' => 'medium',
                'category' => 'sleep',
            ],
            [
                'slug' => 'phone-out-of-bedroom',
                'title' => ['fa' => 'گوشی بیرون از اتاق خواب', 'en' => 'Phone out of the bedroom'],
                'description' => ['fa' => 'امشب گوشی را بیرون از اتاق بگذار و بدون آن بخواب', 'en' => 'Leave your phone outside the bedroom tonight'],
                'difficulty' => 'hard',
                'category' => 'sleep',
            ],
        ];
    }
}
