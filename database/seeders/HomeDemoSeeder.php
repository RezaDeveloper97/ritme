<?php

namespace Database\Seeders;

use App\Enums\NotificationType;
use App\Enums\ReminderType;
use App\Models\Reminder;
use App\Models\User;
use App\Models\UserNotification;
use Carbon\Carbon;
use Illuminate\Database\Seeder;

/**
 * Per-user demo data (doctor/medication reminders + notifications) so the home
 * page renders fully in development. Safe to re-run: only seeds the first user
 * when it has none.
 */
class HomeDemoSeeder extends Seeder
{
    public function run(): void
    {
        $user = User::query()->first();
        if (!$user) {
            return;
        }

        if (!$user->reminders()->exists()) {
            Reminder::create([
                'user_id' => $user->id,
                'type' => ReminderType::DOCTOR->value,
                'title' => 'دکتر عابدزاده',
                'subtitle' => 'متخصص زنان و زایمان',
                'scheduled_at' => Carbon::tomorrow()->setTime(13, 30),
                'recurrence' => 'none',
                'is_active' => true,
                'meta' => ['location' => 'مطب مرکزی'],
            ]);

            Reminder::create([
                'user_id' => $user->id,
                'type' => ReminderType::MEDICATION->value,
                'title' => 'امگا ۳',
                'subtitle' => '۱ عدد',
                'recurrence' => 'daily',
                'recurrence_time' => '16:00:00',
                'starts_on' => Carbon::today(),
                'is_active' => true,
            ]);
        }

        if (!$user->appNotifications()->exists()) {
            UserNotification::create([
                'user_id' => $user->id,
                'type' => NotificationType::TIP->value,
                'title' => ['fa' => 'نکته‌ی امروز', 'en' => 'Today’s tip'],
                'body' => ['fa' => 'فراموش نکن امروز آب کافی بنوشی.', 'en' => 'Don’t forget to stay hydrated today.'],
            ]);

            UserNotification::create([
                'user_id' => $user->id,
                'type' => NotificationType::REMINDER->value,
                'title' => ['fa' => 'یادآور دارو', 'en' => 'Medication reminder'],
                'body' => ['fa' => 'ساعت ۱۶ زمان مصرف امگا ۳ است.', 'en' => 'Omega-3 is due at 16:00.'],
                'read_at' => now(),
            ]);
        }
    }
}
