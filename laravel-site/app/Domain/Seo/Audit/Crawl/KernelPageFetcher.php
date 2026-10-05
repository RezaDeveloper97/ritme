<?php

declare(strict_types=1);

namespace App\Domain\Seo\Audit\Crawl;

use App\Domain\Seo\Audit\Contracts\PageFetcher;
use Illuminate\Contracts\Config\Repository as Config;
use Illuminate\Contracts\Foundation\Application;
use Illuminate\Contracts\Http\Kernel as HttpKernel;
use Illuminate\Routing\Route;
use Illuminate\Support\Facades\Facade;
use Illuminate\Support\Facades\Request as RequestFactory;
use Throwable;

/**
 * Sends the path through the application's HTTP kernel in-process, like a cookie-less first visit by a crawler:
 * `Cache-Control: no-cache` skips the guest page cache (so the audit measures and inspects a real render and never
 * stores its own variant), and the user agent is classed as a bot, so audit 404s stay out of the 404 monitor.
 * The request is built through the Request facade so the domain does not depend on the HTTP layer's classes.
 */
final class KernelPageFetcher implements PageFetcher
{
    public const USER_AGENT = 'RitmeSeoAuditBot/1.0 (in-process)';

    private mixed $original = null;

    private bool $captured = false;

    public function __construct(
        private readonly Application $app,
        private readonly HttpKernel $kernel,
        private readonly Config $config,
    ) {}

    public function fetch(string $path): FetchResult
    {
        if (! $this->captured) {
            $this->original = $this->app->bound('request') ? $this->app->make('request') : null;
            $this->captured = true;
        }

        $root = rtrim((string) $this->config->get('app.url', 'http://localhost'), '/');
        $request = RequestFactory::create($root.$path, 'GET', [], [], [], [
            'HTTP_ACCEPT' => 'text/html,application/xhtml+xml',
            'HTTP_ACCEPT_LANGUAGE' => 'fa',
            'HTTP_USER_AGENT' => self::USER_AGENT,
            'HTTP_CACHE_CONTROL' => 'no-cache',
        ]);

        $started = hrtime(true);
        try {
            $response = $this->kernel->handle($request);
            $milliseconds = (int) round((hrtime(true) - $started) / 1_000_000);
            $this->kernel->terminate($request, $response);
        } catch (Throwable $e) {
            return new FetchResult($path, 0, milliseconds: (int) round((hrtime(true) - $started) / 1_000_000), error: $e->getMessage());
        }

        $route = $request->route();
        // The router caches controller instances on the Route; a reused controller would keep the previous
        // request's scoped SeoManager / SchemaGraph. One fresh controller per request, like PHP-FPM.
        if ($route instanceof Route) {
            $route->flushController();
        }
        $parameters = [];
        if ($route instanceof Route) {
            foreach ((array) $route->originalParameters() as $key => $value) {
                if (is_scalar($value)) {
                    $parameters[(string) $key] = (string) $value;
                }
            }
        }
        $type = (string) $response->headers->get('Content-Type', '');
        $status = $response->getStatusCode();
        $content = $status === 200 && str_contains(strtolower($type), 'text/html') ? $response->getContent() : '';

        return new FetchResult(
            path: $path,
            status: $status,
            contentType: $type,
            location: $response->headers->get('Location'),
            html: is_string($content) ? $content : '',
            milliseconds: $milliseconds,
            routeName: $route instanceof Route ? $route->getName() : null,
            routeParameters: $parameters,
        );
    }

    public function restore(): void
    {
        if (! $this->captured) {
            return;
        }
        if ($this->original !== null) {
            $this->app->instance('request', $this->original);
        }
        Facade::clearResolvedInstance('request');
        $this->original = null;
        $this->captured = false;
    }
}
