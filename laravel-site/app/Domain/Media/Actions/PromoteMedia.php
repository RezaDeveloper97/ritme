<?php

declare(strict_types=1);

namespace App\Domain\Media\Actions;

use App\Domain\Media\Enums\MediaFormat;
use App\Domain\Media\Jobs\OptimizeMedia;
use App\Domain\Media\Models\Media;
use Closure;
use Illuminate\Contracts\Bus\Dispatcher;
use Illuminate\Contracts\Config\Repository as Config;
use Illuminate\Contracts\Filesystem\Factory as Filesystems;
use Illuminate\Contracts\Filesystem\Filesystem;
use Illuminate\Database\ConnectionInterface;
use RuntimeException;
use Throwable;

/**
 * Moves not-yet-public media (e.g. join-request photos on the private `pending` disk, L9-04b) to the media disk, all or
 * nothing: every file is copied first (same relative path), then the rows switch disk inside one transaction together
 * with the caller's own writes (`$work`), and only after the commit are the old files removed and the variants queued.
 * Any failure deletes the copies already written and leaves rows, old files and the caller's writes untouched.
 */
final class PromoteMedia
{
    public function __construct(
        private readonly Filesystems $filesystems,
        private readonly ConnectionInterface $db,
        private readonly Dispatcher $bus,
        private readonly Config $config,
    ) {}

    /** The public media disk (`media.disk`). */
    public function mediaDisk(): string
    {
        return (string) $this->config->get('media.disk', 'public');
    }

    /**
     * @template TResult
     *
     * @param  iterable<Media>  $media  items on any disk; those already on the media disk are left alone
     * @param  Closure(): TResult  $work  runs inside the same transaction as the disk switch
     * @return TResult
     *
     * @throws RuntimeException when a file cannot be read or written (nothing is promoted then)
     */
    public function handle(iterable $media, Closure $work): mixed
    {
        $target = $this->mediaDisk();
        $pending = [];
        foreach ($media as $item) {
            if ($item->disk !== $target) {
                $pending[$item->id] = $item;
            }
        }

        if ($pending === []) {
            return $this->db->transaction($work);
        }

        $to = $this->filesystems->disk($target);
        $sources = array_map(static fn (Media $item): string => $item->disk, $pending);
        $written = [];

        try {
            foreach ($pending as $id => $item) {
                $this->copy($item, $this->filesystems->disk($sources[$id]), $to, $written);
            }

            $result = $this->db->transaction(function () use ($pending, $target, $work): mixed {
                foreach ($pending as $item) {
                    $item->forceFill(['disk' => $target])->save();
                }

                return $work();
            });
        } catch (Throwable $e) {
            foreach ($pending as $id => $item) {
                $item->disk = $sources[$id]; // the in-memory models mirror the rolled-back rows
            }
            $this->remove($to, $written);

            throw $e;
        }

        foreach ($pending as $id => $item) {
            $this->remove($this->filesystems->disk($sources[$id]), $item->allPaths());
            $this->queueOptimization($item);
        }

        return $result;
    }

    /**
     * @param  list<string>  $written
     */
    private function copy(Media $item, Filesystem $from, Filesystem $to, array &$written): void
    {
        foreach ($item->allPaths() as $path) {
            $stream = $from->readStream($path);
            if (! is_resource($stream)) {
                throw new RuntimeException("Media #{$item->id}: cannot read {$path} from its disk.");
            }

            try {
                if ($to->exists($path) || ! $to->writeStream($path, $stream)) {
                    throw new RuntimeException("Media #{$item->id}: cannot write {$path} to the media disk.");
                }
            } finally {
                if (get_resource_type($stream) === 'stream') { // a closed stream reports "Unknown"
                    fclose($stream);
                }
            }
            $written[] = $path;
        }
    }

    /**
     * Deletes files and then their directories when nothing else is left in them.
     *
     * @param  list<string>  $paths
     */
    private function remove(Filesystem $disk, array $paths): void
    {
        if ($paths === []) {
            return;
        }

        $disk->delete($paths);
        foreach (array_unique(array_map(dirname(...), $paths)) as $directory) {
            if ($directory !== '.' && $directory !== '' && $disk->allFiles($directory) === []) {
                $disk->deleteDirectory($directory);
            }
        }
    }

    private function queueOptimization(Media $media): void
    {
        if ($media->mime === MediaFormat::Svg->mime()) {
            $media->forceFill(['optimized_at' => now()])->save(); // vector: nothing to generate

            return;
        }

        try {
            $this->bus->dispatch(new OptimizeMedia($media->id));
        } catch (Throwable $e) {
            report($e); // sync queue: a failed optimisation must not fail the promotion — the original is served
        }
    }
}
