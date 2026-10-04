<?php

declare(strict_types=1);

namespace App\Domain\Media\Observers;

use App\Domain\Media\Models\Media;
use App\Support\Cache\CacheBumpingObserver;
use App\Support\Cache\NamespaceBumper;
use Illuminate\Contracts\Filesystem\Factory as Filesystems;
use Illuminate\Database\Eloquent\Model;

/**
 * Any media change invalidates cached media DTOs, SEO fragments that embed media (OG image, Organization logo) and
 * (via cacheaside.always_bump) the full-page cache. Deleting a row removes the original, every variant and the
 * media's own directory.
 */
final class MediaObserver extends CacheBumpingObserver
{
    public function __construct(NamespaceBumper $bumper, private readonly Filesystems $filesystems)
    {
        parent::__construct($bumper);
    }

    protected function namespaces(Model $model): array
    {
        return ['media', 'seo'];
    }

    public function deleted(Model $model): void
    {
        if ($model instanceof Media) {
            $disk = $this->filesystems->disk($model->disk);
            $disk->delete($model->allPaths());

            if ($model->directory !== '' && $disk->allFiles($model->directory) === []) {
                $disk->deleteDirectory($model->directory);
            }
        }

        parent::deleted($model);
    }
}
