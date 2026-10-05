<?php

declare(strict_types=1);

namespace App\Domain\Seo\Audit\Jobs;

use App\Domain\Seo\Audit\Actions\RunSeoAudit;
use App\Domain\Seo\Audit\AuditOptions;
use App\Domain\Seo\Audit\Enums\RunStatus;
use App\Domain\Seo\Audit\Models\AuditRun;
use Illuminate\Contracts\Config\Repository as Config;
use Illuminate\Contracts\Queue\ShouldQueue;
use Illuminate\Foundation\Queue\Queueable;
use Illuminate\Support\Carbon;
use Throwable;

/**
 * Runs a queued audit (QueueSeoAudit) with the configured bounds (`seo.audit.max_pages`, `seo.audit.time_limit`).
 * One attempt: a failed crawl is recorded on the run, the next weekly run starts fresh.
 */
final class RunSeoAuditJob implements ShouldQueue
{
    use Queueable;

    public int $tries = 1;

    public int $timeout = 600;

    public function __construct(public readonly int $runId) {}

    public function handle(RunSeoAudit $audit, Config $config): void
    {
        $run = AuditRun::query()->find($this->runId);
        if ($run === null || $run->status !== RunStatus::Queued) {
            return;
        }

        $audit->handle(new AuditOptions(
            maxPages: max(1, (int) $config->get('seo.audit.max_pages', AuditOptions::DEFAULT_MAX_PAGES)),
            timeLimit: max(10, (int) $config->get('seo.audit.time_limit', AuditOptions::DEFAULT_TIME_LIMIT)),
        ), $run->trigger, $run);
    }

    public function failed(?Throwable $e): void
    {
        AuditRun::query()
            ->whereKey($this->runId)
            ->whereIn('status', [RunStatus::Queued->value, RunStatus::Running->value])
            ->update(['status' => RunStatus::Failed->value, 'finished_at' => Carbon::now(), 'message' => mb_substr((string) $e?->getMessage(), 0, 1000)]);
    }
}
