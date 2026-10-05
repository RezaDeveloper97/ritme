<?php

declare(strict_types=1);

namespace App\Domain\Seo\Indexing\Jobs;

use App\Domain\Seo\Indexing\IndexNow\IndexNow;
use Illuminate\Contracts\Queue\ShouldQueue;
use Illuminate\Foundation\Queue\Queueable;

/**
 * Submits changed URLs to IndexNow from the queue (database queue drained by the cron scheduler in production). The
 * switch is checked again when the job runs, so turning IndexNow off also stops queued submissions.
 */
final class SubmitToIndexNow implements ShouldQueue
{
    use Queueable;

    public int $tries = 3;

    public int $timeout = 30;

    /**
     * @param  list<string>  $urls
     */
    public function __construct(public readonly array $urls) {}

    /**
     * @return list<int>
     */
    public function backoff(): array
    {
        return [60, 600];
    }

    public function handle(IndexNow $indexNow): void
    {
        $indexNow->submit($this->urls);
    }
}
