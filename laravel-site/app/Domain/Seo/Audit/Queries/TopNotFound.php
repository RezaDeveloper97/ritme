<?php

declare(strict_types=1);

namespace App\Domain\Seo\Audit\Queries;

use App\Domain\Seo\Redirects\Enums\AgentClass;
use App\Domain\Seo\Redirects\Models\NotFoundLog;
use Illuminate\Database\Eloquent\Builder;
use Illuminate\Support\Carbon;

/**
 * The most hit 404 paths of the last days (404 monitor, L7-03), people before bots.
 */
final class TopNotFound
{
    public const DAYS = 30;

    /**
     * @return Builder<NotFoundLog>
     */
    public function query(int $days = self::DAYS): Builder
    {
        return NotFoundLog::query()
            ->where('last_seen_at', '>=', Carbon::now()->subDays($days))
            ->orderByRaw('CASE WHEN agent = ? THEN 0 ELSE 1 END', [AgentClass::Human->value])
            ->orderByDesc('hits');
    }
}
