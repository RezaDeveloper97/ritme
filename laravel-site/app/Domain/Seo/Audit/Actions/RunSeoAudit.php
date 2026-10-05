<?php

declare(strict_types=1);

namespace App\Domain\Seo\Audit\Actions;

use App\Domain\Seo\Audit\AuditedPage;
use App\Domain\Seo\Audit\AuditOptions;
use App\Domain\Seo\Audit\AuditResult;
use App\Domain\Seo\Audit\Enums\RunStatus;
use App\Domain\Seo\Audit\Enums\RunTrigger;
use App\Domain\Seo\Audit\Models\AuditRun;
use App\Domain\Seo\Audit\Models\AuditRunPage;
use App\Domain\Seo\Audit\Queries\AuditHistory;
use App\Domain\Seo\Audit\SeoAuditEngine;
use App\Domain\Seo\Audit\Severity;
use Illuminate\Database\ConnectionInterface;
use Illuminate\Support\Carbon;
use Throwable;

/**
 * Runs one audit and (unless `$store` is false) stores it: the run row (given — queued from the admin — or new),
 * its pages and findings with fix links, totals and the health score. Full crawls audit the previous run's failing
 * pages first (incremental-friendly within the run's bounds). Keeps the last KEEP_RUNS runs.
 */
final class RunSeoAudit
{
    public const KEEP_RUNS = 30;

    public function __construct(
        private readonly SeoAuditEngine $engine,
        private readonly AuditHistory $history,
        private readonly ConnectionInterface $db,
    ) {}

    /**
     * @return array{0: AuditResult, 1: AuditRun|null}
     */
    public function handle(AuditOptions $options, RunTrigger $trigger = RunTrigger::Cli, ?AuditRun $run = null, bool $store = true): array
    {
        if ($options->isFullCrawl() && $options->priority === []) {
            $options = $options->withPriority($this->history->priorityPaths());
        }

        $record = null;
        if ($store) {
            $record = $run ?? new AuditRun(['trigger' => $trigger]);
            $record->fill([
                'status' => RunStatus::Running,
                'max_pages' => $options->maxPages,
                'time_limit' => $options->timeLimit,
                'started_at' => Carbon::now(),
                'message' => null,
            ])->save();
        }

        try {
            $result = $this->engine->run($options);
        } catch (Throwable $e) {
            $record?->forceFill(['status' => RunStatus::Failed, 'finished_at' => Carbon::now(), 'message' => mb_substr($e->getMessage(), 0, 1000)])->save();

            throw $e;
        }

        if ($record !== null) {
            $this->store($record, $result);
            $this->prune();
        }

        return [$result, $record];
    }

    private function store(AuditRun $run, AuditResult $result): void
    {
        $this->db->transaction(function () use ($run, $result): void {
            $run->pages()->delete();
            $issues = [];
            foreach ($result->pages as $page) {
                $record = AuditRunPage::query()->create($this->pageRow($run, $page));
                foreach ($page->audit->issues as $issue) {
                    $issues[] = [
                        'run_id' => $run->id,
                        'page_id' => $record->id,
                        'path' => mb_substr($page->path(), 0, 2048),
                        'severity' => $issue->severity->value,
                        'code' => mb_substr($issue->code, 0, 64),
                        'message' => mb_substr($issue->message, 0, 1000),
                        'fix_url' => $this->engine->fixUrl($issue->code, $page->editUrl),
                    ];
                }
            }
            foreach (array_chunk($issues, 200) as $chunk) {
                $run->issues()->insert($chunk);
            }

            $run->forceFill([
                'status' => RunStatus::Completed,
                'pages_count' => count($result->pages),
                'errors_count' => $result->count(Severity::Error),
                'warnings_count' => $result->count(Severity::Warning),
                'notices_count' => $result->count(Severity::Notice),
                'score' => $result->score(),
                'truncated' => $result->truncated,
                'truncated_by' => $result->truncatedBy,
                'duration_ms' => $result->milliseconds,
                'finished_at' => Carbon::now(),
            ])->save();
        });
    }

    /**
     * @return array<string, mixed>
     */
    private function pageRow(AuditRun $run, AuditedPage $page): array
    {
        $audit = $page->audit;

        return [
            'run_id' => $run->id,
            'path' => mb_substr($page->path(), 0, 2048),
            'path_hash' => sha1($page->path()),
            'title' => $audit->title === null ? null : mb_substr($audit->title, 0, 500),
            'status' => $page->status,
            'source' => $page->source,
            'route_name' => $page->routeName,
            'content_type' => $page->contentType?->value,
            'indexable' => $audit->indexable,
            'in_sitemap' => $page->inSitemap,
            'response_ms' => $page->milliseconds,
            'html_bytes' => $page->htmlBytes,
            'requests_count' => min(65535, $page->requests),
            'inbound_links' => $page->inbound,
            'analysis_score' => $page->analysisScore,
            'score' => $audit->score(),
            'errors_count' => min(65535, $audit->errorCount()),
            'warnings_count' => min(65535, $audit->warningCount()),
            'notices_count' => min(65535, $audit->noticeCount()),
            'content_hash' => $page->contentHash,
            'edit_url' => $page->editUrl,
        ];
    }

    private function prune(): void
    {
        $keep = AuditRun::query()->latest('id')->limit(self::KEEP_RUNS)->pluck('id')->all();
        if ($keep !== []) {
            AuditRun::query()->whereNotIn('id', $keep)->delete();
        }
    }
}
