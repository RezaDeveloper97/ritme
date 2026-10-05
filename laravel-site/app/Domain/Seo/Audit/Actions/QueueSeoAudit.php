<?php

declare(strict_types=1);

namespace App\Domain\Seo\Audit\Actions;

use App\Domain\Seo\Audit\Enums\RunStatus;
use App\Domain\Seo\Audit\Enums\RunTrigger;
use App\Domain\Seo\Audit\Jobs\RunSeoAuditJob;
use App\Domain\Seo\Audit\Models\AuditRun;
use App\Domain\Seo\Audit\Queries\AuditHistory;
use Illuminate\Contracts\Config\Repository as Config;

/**
 * "Run now" / the weekly schedule: records a queued run and dispatches RunSeoAuditJob, unless a run is already queued
 * or running (returns null then). On the `sync` queue the job runs after the response, never inside the admin
 * request (the crawl swaps the container's request while it renders pages).
 */
final class QueueSeoAudit
{
    public function __construct(private readonly AuditHistory $history, private readonly Config $config) {}

    public function handle(RunTrigger $trigger, ?int $userId = null): ?AuditRun
    {
        if ($this->history->active() !== null) {
            return null;
        }

        $run = AuditRun::query()->create([
            'status' => RunStatus::Queued,
            'trigger' => $trigger,
            'triggered_by' => $userId,
        ]);

        if ($this->config->get('queue.default') === 'sync') {
            RunSeoAuditJob::dispatchAfterResponse($run->id);
        } else {
            RunSeoAuditJob::dispatch($run->id);
        }

        return $run;
    }
}
