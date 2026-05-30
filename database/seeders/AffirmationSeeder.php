<?php

namespace Database\Seeders;

use App\Models\Affirmation;
use Illuminate\Database\Seeder;

class AffirmationSeeder extends Seeder
{
    public function run(): void
    {
        if (Affirmation::query()->exists()) {
            return; // already seeded
        }

        $affirmations = [
            ['fa' => 'هر روز فرصتی تازه برای مراقبت از خودت است.', 'en' => 'Every day is a fresh chance to care for yourself.'],
            ['fa' => 'بدن من شایسته‌ی مهربانی و احترام است.', 'en' => 'My body deserves kindness and respect.'],
            ['fa' => 'من به ریتم بدنم گوش می‌دهم و آن را می‌پذیرم.', 'en' => 'I listen to and honor my body’s rhythm.'],
            ['fa' => 'آرامش من از درون شروع می‌شود.', 'en' => 'My calm begins within.'],
            ['fa' => 'من به اندازه کافی هستم، همین حالا.', 'en' => 'I am enough, just as I am.'],
            ['fa' => 'با هر نفس، انرژی تازه‌ای می‌گیرم.', 'en' => 'With every breath, I welcome fresh energy.'],
            ['fa' => 'مراقبت از خودم یک انتخاب قدرتمند است.', 'en' => 'Caring for myself is a powerful choice.'],
        ];

        foreach ($affirmations as $index => $text) {
            Affirmation::create([
                'text' => $text,
                'cycle_phase' => null,
                'is_active' => true,
                'sort_order' => $index + 1,
            ]);
        }
    }
}
