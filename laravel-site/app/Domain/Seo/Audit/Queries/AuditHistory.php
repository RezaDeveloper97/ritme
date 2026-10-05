<?php

declare(strict_types=1);

namespace App\Domain\Seo\Audit\Queries;

use App\Domain\Seo\Audit\Enums\RunStatus;
use App\Domain\Seo\Audit\Models\AuditRun;
use App\Domain\Seo\Audit\Models\AuditRunIssue;
use App\Domain\Seo\Audit\Models\AuditRunPage;
use App\Domain\Seo\Audit\Severity;
use Illuminate\Support\Carbon;
use Illuminate\Support\Collection;

/**
 * Read side of the stored audit runs, for the report page, the dashboard widgets and incremental runs.
 */
final class AuditHistory
{
    /** A queued / running run older than this is considered dead (worker killed) and no longer blocks a new one. */
    public const STALE_MINUTES = 30;

    public function latest(): ?AuditRun
    {
        return AuditRun::query()->where('status', RunStatus::Completed->value)->latest('finished_at')->latest('id')->first();
    }

    public function active(): ?AuditRun
    {
        return AuditRun::query()
            ->whereIn('status', [RunStatus::Queued->value, RunStatus::Running->value])
            ->where('updated_at', '>=', Carbon::now()->subMinutes(self::STALE_MINUTES))
            ->latest('id')
            ->first();
    }

    /**
     * Completed runs, oldest first (score trend).
     *
     * @return Collection<int, AuditRun>
     */
    public function trend(int $limit = 12): Collection
    {
        return AuditRun::query()
            ->where('status', RunStatus::Completed->value)
            ->latest('finished_at')->latest('id')
            ->limit($limit)
            ->get()
            ->reverse()
            ->values();
    }

    /**
     * Finding codes of one run, most frequent first.
     *
     * @return list<array{code: string, severity: Severity, count: int}>
     */
    public function topIssues(AuditRun $run, int $limit = 8): array
    {
        $rows = AuditRunIssue::query()
            ->selectRaw('code, severity, COUNT(*) AS aggregate')
            ->where('run_id', $run->id)
            ->groupBy('code', 'severity')
            ->get();

        $issues = [];
        foreach ($rows as $row) {
            $issues[] = ['code' => $row->code, 'severity' => $row->severity, 'count' => (int) $row->getAttribute('aggregate')];
        }
        $rank = static fn (Severity $s): int => match ($s) {
            Severity::Error => 0,
            Severity::Warning => 1,
            Severity::Notice => 2,
        };
        usort($issues, static fn (array $a, array $b): int => [$rank($a['severity']), -$a['count']] <=> [$rank($b['severity']), -$b['count']]);

        return array_slice($issues, 0, $limit);
    }

    public function countCode(AuditRun $run, string $code): int
    {
        return AuditRunIssue::query()->where('run_id', $run->id)->where('code', $code)->count();
    }

    /**
     * Content pages whose analyser score is under the threshold, weakest first.
     *
     * @return Collection<int, AuditRunPage>
     */
    public function contentNeedingWork(AuditRun $run, int $threshold, int $limit = 10): Collection
    {
        return AuditRunPage::query()
            ->where('run_id', $run->id)
            ->whereNotNull('analysis_score')
            ->where('analysis_score', '<', $threshold)
            ->orderBy('analysis_score')
            ->limit($limit)
            ->get();
    }

    /**
     * Paths the next run should audit first: pages of the latest run that had errors or warnings (newly seeded
     * pages are prepended by the engine's own order anyway).
     *
     * @return list<string>
     */
    public function priorityPaths(int $limit = 200): array
    {
        $run = $this->latest();
        if ($run === null) {
            return [];
        }

        /** @var list<string> $paths */
        $paths = AuditRunPage::query()
            ->where('run_id', $run->id)
            ->where(static fn ($q) => $q->where('errors_count', '>', 0)->orWhere('warnings_count', '>', 0))
            ->orderByDesc('errors_count')
            ->limit($limit)
            ->pluck('path')
            ->all();

        return $paths;
    }
}
