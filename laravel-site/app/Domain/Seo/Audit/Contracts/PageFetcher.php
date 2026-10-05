<?php

declare(strict_types=1);

namespace App\Domain\Seo\Audit\Contracts;

use App\Domain\Seo\Audit\Crawl\FetchResult;
use App\Domain\Seo\Audit\Crawl\KernelPageFetcher;
use Illuminate\Container\Attributes\Bind;

/**
 * Renders one site path without the network (the audit never makes external HTTP requests). Redirects are returned,
 * not followed. `restore()` puts the caller's own request back once a crawl is over.
 */
#[Bind(KernelPageFetcher::class)]
interface PageFetcher
{
    public function fetch(string $path): FetchResult;

    public function restore(): void;
}
