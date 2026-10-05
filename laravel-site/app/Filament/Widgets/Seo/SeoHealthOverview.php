<?php

declare(strict_types=1);

namespace App\Filament\Widgets\Seo;

use App\Domain\Seo\Audit\Models\AuditRun;
use App\Domain\Seo\Audit\Queries\AuditHistory;
use App\Filament\Pages\Seo\AuditReport;
use Filament\Support\Icons\Heroicon;
use Filament\Widgets\StatsOverviewWidget;
use Filament\Widgets\StatsOverviewWidget\Stat;

/**
 * SEO health at a glance (L7-05): score of the latest audit (with its trend), errors / warnings / notices, pages
 * without a share image and when the last audit ran.
 */
final class SeoHealthOverview extends StatsOverviewWidget
{
    use SeoWidgetAccess;

    protected static ?int $sort = 10;

    protected ?string $pollingInterval = null;

    protected ?string $heading = 'سلامت سئو';

    protected function getStats(): array
    {
        $history = app(AuditHistory::class);
        $run = $history->latest();
        $report = AuditReport::getUrl();

        if ($run === null) {
            return [
                Stat::make('امتیاز سلامت سئو', '—')
                    ->description('هنوز ممیزی‌ای اجرا نشده است.')
                    ->icon(Heroicon::OutlinedHeart)
                    ->url($report),
            ];
        }

        $trend = $history->trend()->map(static fn (AuditRun $r): float => (float) $r->score)->all();
        $score = (int) $run->score;
        $missingOg = $history->countCode($run, 'og.image');

        return [
            Stat::make('امتیاز سلامت سئو', fa_digits($score).' از ۱۰۰')
                ->description(fa_digits($run->pages_count).' صفحه بررسی شد')
                ->icon(Heroicon::OutlinedHeart)
                ->color($score >= 90 ? 'success' : ($score >= 70 ? 'warning' : 'danger'))
                ->chart(count($trend) > 1 ? $trend : null)
                ->url($report),
            Stat::make('خطا / هشدار / نکته', fa_digits($run->errors_count).' / '.fa_digits($run->warnings_count).' / '.fa_digits($run->notices_count))
                ->description($run->errors_count > 0 ? 'خطاها را اول برطرف کنید' : 'خطایی پیدا نشد')
                ->color($run->errors_count > 0 ? 'danger' : 'success')
                ->icon(Heroicon::OutlinedExclamationTriangle)
                ->url($report),
            Stat::make('صفحه‌های بدون تصویر اشتراک', fa_digits($missingOg))
                ->description('og:image برای اشتراک در شبکه‌ها')
                ->color($missingOg > 0 ? 'warning' : 'success')
                ->icon(Heroicon::OutlinedPhoto)
                ->url($report),
            Stat::make('آخرین ممیزی', $run->finished_at === null ? '—' : jdate($run->finished_at, 'j F Y، H:i'))
                ->description(($run->finished_at?->diffForHumans() ?? '').($run->truncated ? ' · نیمه‌کاره (محدودیت زمان/صفحه)' : ''))
                ->icon(Heroicon::OutlinedClock)
                ->url($report),
        ];
    }
}
