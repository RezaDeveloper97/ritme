<?php

namespace Database\Seeders;

use App\Models\Challenge;
use Illuminate\Database\Seeder;

/**
 * The daily-challenge pool.
 *
 * {@see \App\Services\Challenges\DailyChallengeService} narrows this pool by
 * cycle day, then by the category the user's recent logs point at — so every
 * category needs a few entries or the pick falls back to the whole pool.
 *
 * Most entries are untargeted (`cycle_day_from`/`to` null = any day) and act as
 * that fallback; the day-targeted ones are challenges that only make sense at a
 * point in the cycle, and they win on the days they cover.
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
                    'cycle_day_from' => null,
                    'cycle_day_to' => null,
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
                'category' => 'tracking',
            ],
            [
                'slug' => 'log-mood',
                'title' => ['fa' => 'حال امروزت را ثبت کن', 'en' => 'Log how you feel today'],
                'description' => ['fa' => 'یک دقیقه وقت بگذار و حال و حوصله‌ی امروزت را ثبت کن', 'en' => "Take a minute to log today's mood"],
                'category' => 'tracking',
            ],
            [
                'slug' => 'log-symptoms',
                'title' => ['fa' => 'علائم امروز را کامل کن', 'en' => "Complete today's symptoms"],
                'description' => ['fa' => 'علائم امروزت را ثبت کن تا الگوهای بدنت واضح‌تر شود', 'en' => "Log today's symptoms so your patterns become clearer"],
                'category' => 'tracking',
            ],
            [
                'slug' => 'log-seven-days',
                'title' => ['fa' => 'هفت روز پیاپی ثبت کن', 'en' => 'Log seven days in a row'],
                'description' => ['fa' => 'امروز هم ثبت کن تا زنجیره‌ی هفت‌روزه‌ات کامل شود', 'en' => 'Log today to complete your seven-day run'],
                'category' => 'tracking',
            ],

            // ── mindfulness ─────────────────────────────────────────────
            [
                'slug' => 'meditate-10-minutes',
                'title' => ['fa' => '۱۰ دقیقه مدیتیشن', 'en' => '10 minutes of meditation'],
                'description' => ['fa' => 'برای کاهش استرس، ۱۰ دقیقه مدیتیشن کن', 'en' => 'Meditate for 10 minutes to reduce stress'],
                'category' => 'mindfulness',
            ],
            [
                'slug' => 'deep-breathing',
                'title' => ['fa' => 'پنج نفس عمیق', 'en' => 'Five deep breaths'],
                'description' => ['fa' => 'هر وقت فشار را حس کردی، پنج نفس عمیق و آرام بکش', 'en' => 'Take five slow, deep breaths whenever tension builds'],
                'category' => 'mindfulness',
            ],
            [
                'slug' => 'gratitude-note',
                'title' => ['fa' => 'سه چیز خوب امروز', 'en' => 'Three good things'],
                'description' => ['fa' => 'سه چیز که امروز بابتشان سپاسگزاری، برای خودت بنویس', 'en' => "Write down three things you're grateful for today"],
                'category' => 'mindfulness',
            ],
            [
                'slug' => 'screen-free-hour',
                'title' => ['fa' => 'یک ساعت بدون صفحه‌نمایش', 'en' => 'One screen-free hour'],
                'description' => ['fa' => 'یک ساعت از امروز را بدون گوشی و نمایشگر بگذران', 'en' => 'Spend one hour today away from phones and screens'],
                'category' => 'mindfulness',
            ],

            // ── nutrition ───────────────────────────────────────────────
            [
                'slug' => 'healthy-meal',
                'title' => ['fa' => 'یک وعده‌ی سالم بخور', 'en' => 'Eat one healthy meal'],
                'description' => ['fa' => 'امروز یک وعده‌ی متعادل و مغذی برای خودت آماده کن', 'en' => 'Prepare one balanced, nutritious meal today'],
                'category' => 'nutrition',
            ],
            [
                'slug' => 'drink-water',
                'title' => ['fa' => 'هشت لیوان آب', 'en' => 'Eight glasses of water'],
                'description' => ['fa' => 'امروز هشت لیوان آب بنوش؛ کم‌آبی خستگی را بیشتر می‌کند', 'en' => 'Drink eight glasses of water today — dehydration deepens fatigue'],
                'category' => 'nutrition',
            ],
            [
                'slug' => 'iron-rich-food',
                'title' => ['fa' => 'یک خوراکی آهن‌دار', 'en' => 'One iron-rich food'],
                'description' => ['fa' => 'خوراکی آهن‌دار مثل عدس، اسفناج یا خرما به وعده‌ات اضافه کن', 'en' => 'Add an iron-rich food like lentils, spinach or dates to a meal'],
                'category' => 'nutrition',
            ],
            [
                'slug' => 'no-added-sugar',
                'title' => ['fa' => 'یک روز بدون قند افزوده', 'en' => 'A day without added sugar'],
                'description' => ['fa' => 'امروز را بدون نوشیدنی و خوراکی شیرین‌شده بگذران', 'en' => 'Go today without sweetened drinks or snacks'],
                'category' => 'nutrition',
            ],

            // ── exercise ────────────────────────────────────────────────
            [
                'slug' => 'walk-20-minutes',
                'title' => ['fa' => '۲۰ دقیقه پیاده‌روی', 'en' => 'Walk for 20 minutes'],
                'description' => ['fa' => 'یک پیاده‌روی سبک ۲۰ دقیقه‌ای انجام بده', 'en' => 'Go for a light 20-minute walk'],
                'category' => 'exercise',
            ],
            [
                'slug' => 'gentle-stretch',
                'title' => ['fa' => '۵ دقیقه کشش', 'en' => '5 minutes of stretching'],
                'description' => ['fa' => 'چند حرکت کششی ملایم برای کمر و لگن انجام بده', 'en' => 'Do a few gentle stretches for your lower back and hips'],
                'category' => 'exercise',
                'cycle_day_from' => 1,
                'cycle_day_to' => 5,
            ],
            [
                'slug' => 'strength-session',
                'title' => ['fa' => 'یک تمرین قدرتی کوتاه', 'en' => 'A short strength session'],
                'description' => ['fa' => 'انرژی امروزت خوب است؛ ۲۰ دقیقه تمرین قدرتی انجام بده', 'en' => 'Your energy is up today — do 20 minutes of strength work'],
                'category' => 'exercise',
            ],

            // ── sleep ───────────────────────────────────────────────────
            [
                'slug' => 'sleep-earlier',
                'title' => ['fa' => 'زودتر بخواب', 'en' => 'Go to bed earlier'],
                'description' => ['fa' => 'امشب ۳۰ دقیقه زودتر بخواب تا خواب بهتری داشته باشی', 'en' => 'Sleep 30 minutes earlier tonight for better rest'],
                'category' => 'sleep',
            ],
            [
                'slug' => 'no-caffeine-after-four',
                'title' => ['fa' => 'بعد از عصر بدون کافئین', 'en' => 'No caffeine after 4pm'],
                'description' => ['fa' => 'امروز بعد از ساعت ۴ چای و قهوه نخور تا شب راحت‌تر بخوابی', 'en' => 'Skip coffee and tea after 4pm for an easier night'],
                'category' => 'sleep',
            ],
            [
                'slug' => 'phone-out-of-bedroom',
                'title' => ['fa' => 'گوشی بیرون از اتاق خواب', 'en' => 'Phone out of the bedroom'],
                'description' => ['fa' => 'امشب گوشی را بیرون از اتاق بگذار و بدون آن بخواب', 'en' => 'Leave your phone outside the bedroom tonight'],
                'category' => 'sleep',
            ],

            // ── day-targeted ────────────────────────────────────────────
            // Challenges that only land at a point in the cycle. Days are the
            // canonical 28-day defaults (bleeding 1..5, ovulation ~14) — a
            // range is deliberately wide so it still fits shorter/longer cycles.
            [
                'slug' => 'warm-compress-period',
                'title' => ['fa' => 'کیسه‌ی آب گرم روی شکم', 'en' => 'A warm compress on your belly'],
                'description' => ['fa' => 'روزهای اول قاعدگی، ۱۵ دقیقه کیسه‌ی آب گرم دردِ شکم را کم می‌کند', 'en' => 'In the first days of your period, 15 minutes of heat eases cramps'],
                'category' => 'mindfulness',
                'cycle_day_from' => 1,
                'cycle_day_to' => 3,
            ],
            [
                'slug' => 'rest-day-period',
                'title' => ['fa' => 'یک استراحت واقعی', 'en' => 'A real rest'],
                'description' => ['fa' => 'امروز یک کار از لیستت را حذف کن و به بدنت استراحت بده', 'en' => 'Drop one thing from your list today and let your body rest'],
                'category' => 'sleep',
                'cycle_day_from' => 1,
                'cycle_day_to' => 4,
            ],
            [
                'slug' => 'iron-after-period',
                'title' => ['fa' => 'جبران آهن بعد از پریود', 'en' => 'Replenish iron after your period'],
                'description' => ['fa' => 'بعد از خون‌ریزی، یک وعده‌ی آهن‌دار همراه ویتامین C بخور', 'en' => 'After bleeding, pair an iron-rich meal with vitamin C'],
                'category' => 'nutrition',
                'cycle_day_from' => 5,
                'cycle_day_to' => 9,
            ],
            [
                'slug' => 'energy-peak-workout',
                'title' => ['fa' => 'از اوج انرژی استفاده کن', 'en' => 'Use your energy peak'],
                'description' => ['fa' => 'این روزها معمولاً پرانرژی‌ترین روزهای چرخه‌اند؛ یک تمرین جدی‌تر انجام بده', 'en' => 'These are usually your highest-energy days — take on a harder workout'],
                'category' => 'exercise',
                'cycle_day_from' => 8,
                'cycle_day_to' => 14,
            ],
            [
                'slug' => 'log-fertility-signs',
                'title' => ['fa' => 'نشانه‌های تخمک‌گذاری را ثبت کن', 'en' => 'Log your ovulation signs'],
                'description' => ['fa' => 'ترشحات و دمای بدنت را این روزها ثبت کن تا پنجره‌ی باروری دقیق‌تر شود', 'en' => 'Log discharge and temperature these days for a sharper fertile window'],
                'category' => 'tracking',
                'cycle_day_from' => 12,
                'cycle_day_to' => 17,
            ],
            [
                'slug' => 'magnesium-before-period',
                'title' => ['fa' => 'منیزیم قبل از پریود', 'en' => 'Magnesium before your period'],
                'description' => ['fa' => 'خوراکی‌های پرمنیزیم مثل بادام و شکلات تلخ به سندرم پیش از قاعدگی کمک می‌کنند', 'en' => 'Magnesium-rich foods like almonds and dark chocolate help with PMS'],
                'category' => 'nutrition',
                'cycle_day_from' => 20,
                'cycle_day_to' => 28,
            ],
            [
                'slug' => 'gentle-with-yourself-pms',
                'title' => ['fa' => 'با خودت مهربان باش', 'en' => 'Be gentle with yourself'],
                'description' => ['fa' => 'روزهای پیش از قاعدگی نوسان خلق طبیعی است؛ امروز یک کار آرام‌بخش برای خودت بکن', 'en' => 'Mood swings before your period are normal — do one calming thing for yourself today'],
                'category' => 'mindfulness',
                'cycle_day_from' => 24,
                'cycle_day_to' => 35,
            ],
        ];
    }
}
