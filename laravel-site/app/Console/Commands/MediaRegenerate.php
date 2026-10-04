<?php

declare(strict_types=1);

namespace App\Console\Commands;

use App\Domain\Media\Actions\GenerateMediaVariants;
use App\Domain\Media\Models\Media;
use App\Domain\Media\Support\VariantPlanner;
use Illuminate\Console\Command;
use Throwable;

/**
 * Re-creates variants inline (no queue) — after changing config/media.php presets, formats or qualities, after a
 * focal-point change, or to add an on-demand preset (e.g. `--preset=square` for products).
 */
final class MediaRegenerate extends Command
{
    protected $signature = 'media:regenerate
        {--id=* : only these media ids}
        {--preset=* : only these presets (merged into existing variants); default = every automatic preset}';

    protected $description = 'Regenerate optimised image variants of the media library';

    public function handle(GenerateMediaVariants $generate, VariantPlanner $planner): int
    {
        $presets = array_values(array_map('strval', (array) $this->option('preset')));
        $unknown = array_diff($presets, $planner->presetNames());
        if ($unknown !== []) {
            $this->error('Unknown preset(s): '.implode(', ', $unknown).'. Known: '.implode(', ', $planner->presetNames()));

            return self::FAILURE;
        }

        $ids = array_values(array_map('intval', (array) $this->option('id')));

        $query = Media::query()->when($ids !== [], static fn ($q) => $q->whereIn('id', $ids));
        $total = (clone $query)->count();
        if ($total === 0) {
            $this->warn('No media found.');

            return $ids === [] ? self::SUCCESS : self::FAILURE;
        }

        $failed = 0;
        $bar = $this->output->createProgressBar($total);
        foreach ($query->lazyById(50) as $media) {
            try {
                $generate->handle($media, $presets === [] ? null : $presets);
            } catch (Throwable $e) {
                $failed++;
                $this->newLine();
                $this->error("#{$media->id} {$media->path()}: {$e->getMessage()}");
            }
            $bar->advance();
        }
        $bar->finish();
        $this->newLine();

        $this->info(sprintf('Regenerated %d of %d media.', $total - $failed, $total));

        return $failed === 0 ? self::SUCCESS : self::FAILURE;
    }
}
