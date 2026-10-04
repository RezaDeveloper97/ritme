<?php

declare(strict_types=1);

namespace App\Console\Commands;

use App\Domain\Seo\Audit\AuditIssue;
use App\Domain\Seo\Audit\PageAudit;
use App\Domain\Seo\Audit\SeoAuditor;
use App\Domain\Seo\Audit\Severity;
use Illuminate\Console\Command;
use Illuminate\Contracts\Http\Kernel as HttpKernel;
use Illuminate\Http\Request;
use Illuminate\Routing\Route;
use Illuminate\Routing\Router;
use Illuminate\Support\Str;
use Throwable;

/**
 * seo:audit v1 — renders every GET route without required parameters in-process (no HTTP, no network) and runs
 * the SeoAuditor rules on each HTML response. Exits non-zero when any error is found. Extended in L7-05.
 */
final class SeoAudit extends Command
{
    /** URI patterns (Str::is) that are not public pages. */
    public const EXCLUDED_URIS = [
        'up', 'admin', 'admin/*', 'livewire/*', 'filament/*', 'storage/*', '_ignition/*', 'sanctum/*',
        'sitemap.xml', 'sitemaps/*', 'robots.txt', 'manifest.webmanifest', 'sw.js', 'build/*',
    ];

    protected $signature = 'seo:audit
        {--path=* : audit only these paths (e.g. --path=/ --path=/cycle)}
        {--strict : treat warnings as errors}';

    protected $description = 'Render every public route in-process and check titles, descriptions, h1, canonical, OG, images and links';

    public function handle(HttpKernel $kernel, Router $router, SeoAuditor $auditor): int
    {
        $paths = $this->paths($router);
        if ($paths === []) {
            $this->warn('No routes to audit.');

            return self::SUCCESS;
        }

        $pages = [];
        foreach ($paths as $path) {
            $page = $this->render($kernel, $auditor, $path);
            if ($page !== null) {
                $pages[] = $page;
            }
        }

        $pages = $auditor->crossCheck($pages);
        $strict = (bool) $this->option('strict');
        $errors = 0;
        $warnings = 0;

        foreach ($pages as $page) {
            $pageErrors = $page->errorCount() + ($strict ? $page->warningCount() : 0);
            $errors += $pageErrors;
            $warnings += $strict ? 0 : $page->warningCount();

            $this->line(($pageErrors > 0 ? '<fg=red>✘</>' : '<fg=green>✔</>').' '.$page->url);
            foreach ($page->issues as $issue) {
                $isError = $strict || $issue->severity === Severity::Error;
                $this->line('    '.($isError ? '<fg=red>error</>' : '<fg=yellow>warning</>')." [{$issue->code}] {$issue->message}");
            }
        }

        $this->newLine();
        $summary = count($pages).' page(s), '.$errors.' error(s), '.$warnings.' warning(s).';
        if ($errors > 0) {
            $this->error($summary);

            return self::FAILURE;
        }

        $this->info($summary);

        return self::SUCCESS;
    }

    /**
     * @return list<string>
     */
    private function paths(Router $router): array
    {
        /** @var list<string> $given */
        $given = array_values(array_filter((array) $this->option('path'), 'is_string'));
        if ($given !== []) {
            return array_values(array_unique(array_map(static fn (string $p): string => '/'.ltrim($p, '/'), $given)));
        }

        $paths = [];
        foreach ($router->getRoutes()->getRoutes() as $route) {
            if ($this->auditable($route)) {
                $paths[] = '/'.ltrim($route->uri(), '/');
            }
        }

        return array_values(array_unique($paths));
    }

    private function auditable(Route $route): bool
    {
        if (! in_array('GET', $route->methods(), true) || Str::is(self::EXCLUDED_URIS, $route->uri())) {
            return false;
        }

        // Only routes without required parameters ({id} but not {id?}).
        return preg_match('/\{[^}?]+\}/', $route->uri()) !== 1;
    }

    private function render(HttpKernel $kernel, SeoAuditor $auditor, string $path): ?PageAudit
    {
        $request = Request::create(url($path), 'GET', server: ['HTTP_ACCEPT' => 'text/html']);

        try {
            $response = $kernel->handle($request);
            $kernel->terminate($request, $response);
        } catch (Throwable $e) {
            return new PageAudit($path, null, null, false, [AuditIssue::error('render', $e->getMessage())]);
        }

        $status = $response->getStatusCode();
        $type = (string) $response->headers->get('Content-Type');
        if ($status >= 400) {
            return new PageAudit($path, null, null, false, [AuditIssue::error('render', "HTTP {$status}.")]);
        }
        if ($status !== 200 || ! str_contains($type, 'text/html')) {
            $this->line("<fg=gray>-</> {$path} skipped (HTTP {$status}, {$type})");

            return null;
        }

        return $auditor->auditPage($path, (string) $response->getContent());
    }
}
