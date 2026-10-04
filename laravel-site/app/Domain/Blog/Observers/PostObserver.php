<?php

declare(strict_types=1);

namespace App\Domain\Blog\Observers;

use App\Domain\Blog\Models\Post;
use App\Domain\Blog\Support\PostContent;
use App\Domain\Blog\Support\PostSchedule;
use App\Domain\Blog\Support\PostSlugger;
use App\Support\Cache\CacheBumpingObserver;
use App\Support\Cache\NamespaceBumper;
use Illuminate\Database\Eloquent\Model;

/**
 * Prepares a post on save (slug + history, sanitised body, reading time, schedule normalisation) and invalidates
 * the magazine caches (`blog`), sitemaps (`sitemap`) and, via cacheaside.always_bump, the full-page cache (`pages`).
 */
final class PostObserver extends CacheBumpingObserver
{
    public function __construct(
        NamespaceBumper $bumper,
        private readonly PostSlugger $slugger,
        private readonly PostContent $content,
    ) {
        parent::__construct($bumper);
    }

    protected function namespaces(Model $model): array
    {
        return ['blog', 'sitemap'];
    }

    public function saving(Post $post): void
    {
        $this->content->prepare($post);
        PostSchedule::normalize($post);
        $this->slugger->assign($post);
    }

    public function updated(Post $post): void
    {
        $this->slugger->recordChange($post);
    }
}
