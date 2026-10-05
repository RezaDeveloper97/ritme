<?php

declare(strict_types=1);

namespace App\Domain\Seo\Redirects\Actions;

use App\Domain\Seo\Redirects\Models\NotFoundLog;
use Illuminate\Support\Carbon;

/**
 * Deletes 404 rows not seen for `$days` days (scheduler, daily; 90 days by default).
 */
final class PurgeNotFoundLogs
{
    public const DAYS = 90;

    public function handle(int $days = self::DAYS): int
    {
        return NotFoundLog::query()->where('last_seen_at', '<', Carbon::now()->subDays(max(1, $days)))->delete();
    }
}
