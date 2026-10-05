<?php

declare(strict_types=1);

namespace App\Domain\Seo\Indexing\Observers;

use App\Domain\Seo\Indexing\IndexNow\IndexNow;
use App\Domain\Seo\Indexing\IndexNow\IndexNowUrls;
use App\Domain\Seo\Indexing\Jobs\SubmitToIndexNow;
use Illuminate\Contracts\Bus\Dispatcher;
use Illuminate\Database\Eloquent\Model;

/**
 * Queues an IndexNow submission (after commit) when a post, product or place is published, updated, unpublished or
 * deleted — only while IndexNow is active (admin switch + production). Registered in SeoServiceProvider.
 */
final class IndexNowObserver
{
    public function __construct(
        private readonly IndexNow $indexNow,
        private readonly IndexNowUrls $urls,
        private readonly Dispatcher $bus,
    ) {}

    public function saved(Model $model): void
    {
        $this->queue($model, false);
    }

    public function deleted(Model $model): void
    {
        $this->queue($model, true);
    }

    private function queue(Model $model, bool $deleted): void
    {
        if (! $this->indexNow->active()) {
            return;
        }

        $urls = $this->urls->forChange($model, $deleted);
        if ($urls !== []) {
            $this->bus->dispatch((new SubmitToIndexNow($urls))->afterCommit());
        }
    }
}
