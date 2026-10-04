<?php

declare(strict_types=1);

namespace App\Domain\Blog\Console;

use App\Domain\Blog\Actions\PublishScheduledPosts;
use Illuminate\Console\Command;

final class PublishScheduledPostsCommand extends Command
{
    protected $signature = 'blog:publish-scheduled';

    protected $description = 'Publish scheduled magazine posts whose publish time has come';

    public function handle(PublishScheduledPosts $publish): int
    {
        $count = $publish->handle();
        $this->components->info("Published {$count} scheduled post(s).");

        return self::SUCCESS;
    }
}
