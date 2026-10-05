<?php

declare(strict_types=1);

namespace App\Domain\Seo\Redirects\Console;

use App\Domain\Seo\Redirects\Actions\FlushRedirectStats;
use Illuminate\Console\Command;

final class FlushRedirectStatsCommand extends Command
{
    protected $signature = 'seo:flush-redirect-stats';

    protected $description = 'Write the buffered redirect hits and 404 counts to the database';

    public function handle(FlushRedirectStats $flush): int
    {
        $result = $flush->handle();
        $this->components->info("Flushed {$result['redirects']} redirect(s) and {$result['not_found']} 404 path(s).");

        return self::SUCCESS;
    }
}
