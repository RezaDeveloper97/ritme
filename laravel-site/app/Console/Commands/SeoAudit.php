<?php

declare(strict_types=1);

namespace App\Console\Commands;

use App\Domain\Seo\Audit\Actions\QueueSeoAudit;
use App\Domain\Seo\Audit\Actions\RunSeoAudit;
use App\Domain\Seo\Audit\AuditOptions;
use App\Domain\Seo\Audit\Enums\RunTrigger;
use App\Domain\Seo\Audit\Severity;
use Illuminate\Console\Command;
use Illuminate\Contracts\Config\Repository as Config;
use Illuminate\Support\Facades\Schema;

/**
 * seo:audit — the SEO audit engine (L7-05) from the command line / CI. Renders pages in-process (no HTTP, no network):
 * every parameterless GET route, every sitemap URL and the internal links found on them (bounded by --max-pages and
 * --time-limit), or only the given --path values. Exits non-zero when any error is found (--strict: any warning).
 * A full crawl is stored as a run for the admin report (--no-store to skip); --queue hands it to the queue instead.
 */
final class SeoAudit extends Command
{
    protected $signature = 'seo:audit
        {--path=* : audit only these paths (e.g. --path=/ --path=/cycle)}
        {--strict : treat warnings as errors}
        {--max-pages= : maximum HTML pages to audit (default seo.audit.max_pages or 500)}
        {--time-limit= : stop crawling after this many seconds (default seo.audit.time_limit or 240)}
        {--no-store : do not store the run for the admin report}
        {--as-is : keep this environment\'s robots (outside production every page is noindex)}
        {--queue : queue a stored run instead of crawling now}
        {--notices : also print notices}';

    protected $description = 'Crawl the site in-process and check titles, descriptions, h1, canonical, robots vs sitemap, links, images, OG, JSON-LD, redirects and speed';

    public function handle(RunSeoAudit $audit, QueueSeoAudit $queue, Config $config): int
    {
        if ((bool) $this->option('queue')) {
            $run = $queue->handle(RunTrigger::Cli);
            $run === null ? $this->warn('An audit is already queued or running.') : $this->info("Audit #{$run->id} queued.");

            return self::SUCCESS;
        }

        /** @var list<string> $paths */
        $paths = array_values(array_filter((array) $this->option('path'), 'is_string'));
        $options = new AuditOptions(
            paths: $paths,
            maxPages: max(1, (int) ($this->option('max-pages') ?? $config->get('seo.audit.max_pages', AuditOptions::DEFAULT_MAX_PAGES))),
            timeLimit: max(1, (int) ($this->option('time-limit') ?? $config->get('seo.audit.time_limit', AuditOptions::DEFAULT_TIME_LIMIT))),
            assumeProduction: ! (bool) $this->option('as-is'),
        );
        $store = $paths === [] && ! (bool) $this->option('no-store') && Schema::hasTable('seo_audit_runs');

        [$result, $run] = $audit->handle($options, RunTrigger::Cli, store: $store);

        foreach ($result->skipped as $skip) {
            $this->line("<fg=gray>-</> {$skip['path']} skipped (HTTP {$skip['status']}, {$skip['type']})");
        }

        if ($result->pages === []) {
            $this->warn('No routes to audit.');

            return self::SUCCESS;
        }

        $strict = (bool) $this->option('strict');
        $notices = (bool) $this->option('notices');
        $errors = 0;
        $warnings = 0;
        foreach ($result->pages as $page) {
            $audit = $page->audit;
            $pageErrors = $audit->errorCount() + ($strict ? $audit->warningCount() : 0);
            $errors += $pageErrors;
            $warnings += $strict ? 0 : $audit->warningCount();

            $this->line(($pageErrors > 0 ? '<fg=red>✘</>' : '<fg=green>✔</>').' '.$page->path());
            foreach ($audit->issues as $issue) {
                if ($issue->severity === Severity::Notice && ! $notices) {
                    continue;
                }
                $label = match (true) {
                    $issue->severity === Severity::Error, $strict && $issue->severity === Severity::Warning => '<fg=red>error</>',
                    $issue->severity === Severity::Warning => '<fg=yellow>warning</>',
                    default => '<fg=gray>notice</>',
                };
                $this->line("    {$label} [{$issue->code}] {$issue->message}");
            }
        }

        $this->newLine();
        if ($result->truncated) {
            $this->warn('Stopped early ('.($result->truncatedBy === 'time' ? 'time limit' : 'max pages').'): orphan-page check skipped.');
        }
        $summary = count($result->pages).' page(s), '.$errors.' error(s), '.$warnings.' warning(s).';
        $details = 'Score '.$result->score().'/100, '.$result->count(Severity::Notice).' notice(s), '.$result->fetches.' render(s) in '.$result->milliseconds.' ms'
            .($run !== null ? ", stored as run #{$run->id}." : '.');
        if ($errors > 0) {
            $this->error($summary);
            $this->line($details);

            return self::FAILURE;
        }

        $this->info($summary);
        $this->line($details);

        return self::SUCCESS;
    }
}
