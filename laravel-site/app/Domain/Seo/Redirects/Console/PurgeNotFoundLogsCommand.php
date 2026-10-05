<?php

declare(strict_types=1);

namespace App\Domain\Seo\Redirects\Console;

use App\Domain\Seo\Redirects\Actions\PurgeNotFoundLogs;
use Illuminate\Console\Command;

final class PurgeNotFoundLogsCommand extends Command
{
    protected $signature = 'seo:purge-404 {--days=90 : Delete paths not seen for this many days}';

    protected $description = 'Delete old rows of the 404 monitor';

    public function handle(PurgeNotFoundLogs $purge): int
    {
        $deleted = $purge->handle((int) $this->option('days'));
        $this->components->info("Deleted {$deleted} 404 row(s).");

        return self::SUCCESS;
    }
}
