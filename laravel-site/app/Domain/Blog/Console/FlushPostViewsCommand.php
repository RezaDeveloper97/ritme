<?php

declare(strict_types=1);

namespace App\Domain\Blog\Console;

use App\Domain\Blog\Actions\FlushPostViews;
use Illuminate\Console\Command;

final class FlushPostViewsCommand extends Command
{
    protected $signature = 'blog:flush-views';

    protected $description = 'Write the cached magazine view counters to the database';

    public function handle(FlushPostViews $flush): int
    {
        $count = $flush->handle();
        $this->components->info("Flushed views of {$count} post(s).");

        return self::SUCCESS;
    }
}
