<?php

declare(strict_types=1);

namespace App\Domain\Media\Jobs;

use App\Domain\Media\Actions\GenerateMediaVariants;
use App\Domain\Media\Models\Media;
use Illuminate\Contracts\Queue\ShouldQueue;
use Illuminate\Foundation\Queue\Queueable;

/**
 * Writes the variants of one media item. Runs on the configured queue (database in production, drained by the
 * cron-driven scheduler; sync in tests). Until it has run, the original is served.
 */
final class OptimizeMedia implements ShouldQueue
{
    use Queueable;

    public int $tries = 3;

    public int $timeout = 300;

    /**
     * @param  list<string>|null  $presets  null = every automatic preset
     */
    public function __construct(public readonly int $mediaId, public readonly ?array $presets = null)
    {
        $connection = config('media.queue.connection');
        if (is_string($connection) && $connection !== '') {
            $this->onConnection($connection);
        }
        $this->onQueue((string) config('media.queue.name', 'default'));
    }

    public function handle(GenerateMediaVariants $generate): void
    {
        $media = Media::query()->find($this->mediaId);

        if ($media !== null) { // deleted before the job ran
            $generate->handle($media, $this->presets);
        }
    }
}
