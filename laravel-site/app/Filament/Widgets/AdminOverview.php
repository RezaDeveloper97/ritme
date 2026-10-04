<?php

declare(strict_types=1);

namespace App\Filament\Widgets;

use App\Models\User;
use Filament\Widgets\StatsOverviewWidget;
use Filament\Widgets\StatsOverviewWidget\Stat;
use Spatie\Activitylog\Models\Activity;

/**
 * Placeholder dashboard counts (L1-08). Domain tasks add their own stats; real SEO widgets arrive in L7-05.
 */
final class AdminOverview extends StatsOverviewWidget
{
    protected static ?int $sort = 1;

    protected ?string $pollingInterval = null;

    protected function getStats(): array
    {
        return [
            Stat::make('کاربران فعال مدیریت', User::query()->where('is_active', true)->count()),
            Stat::make('فعالیت‌های ۷ روز اخیر', Activity::query()->where('created_at', '>=', now()->subDays(7))->count()),
        ];
    }
}
