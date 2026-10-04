<?php

declare(strict_types=1);

namespace App\Domain\Media\Actions;

use App\Domain\Media\Jobs\OptimizeMedia;
use App\Domain\Media\Models\Media;
use Illuminate\Contracts\Bus\Dispatcher;

/**
 * Queues OptimizeMedia for one item (all automatic presets, or only `$presets` merged into the existing set). On
 * the sync queue the variants exist when this returns; otherwise the original is served until the worker runs.
 */
final class RegenerateMediaVariants
{
    public function __construct(private readonly Dispatcher $bus) {}

    /**
     * @param  list<string>|null  $presets
     */
    public function handle(Media $media, ?array $presets = null): void
    {
        $this->bus->dispatch((new OptimizeMedia($media->id, $presets))->afterCommit());
    }
}
