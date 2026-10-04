<?php

declare(strict_types=1);

namespace App\Providers\Domain;

use App\Domain\Blog\Console\FlushPostViewsCommand;
use App\Domain\Blog\Console\PublishScheduledPostsCommand;
use App\Domain\Blog\Contracts\AuthorRepository;
use App\Domain\Blog\Contracts\CategoryRepository;
use App\Domain\Blog\Contracts\PostRepository;
use App\Domain\Blog\Contracts\TagRepository;
use App\Domain\Blog\Models\Author;
use App\Domain\Blog\Models\Category;
use App\Domain\Blog\Models\Post;
use App\Domain\Blog\Models\Tag;
use App\Domain\Blog\Observers\PostObserver;
use App\Domain\Blog\Observers\TaxonomyObserver;
use App\Domain\Blog\Repositories\CachedAuthorRepository;
use App\Domain\Blog\Repositories\CachedCategoryRepository;
use App\Domain\Blog\Repositories\CachedPostRepository;
use App\Domain\Blog\Repositories\CachedTagRepository;
use App\Domain\Blog\Repositories\EloquentAuthorRepository;
use App\Domain\Blog\Repositories\EloquentCategoryRepository;
use App\Domain\Blog\Repositories\EloquentPostRepository;
use App\Domain\Blog\Repositories\EloquentTagRepository;
use App\Domain\Blog\Support\PostContent;
use App\Domain\Media\Actions\FindMediaUsages;
use App\Providers\DomainServiceProvider;
use Illuminate\Console\Scheduling\Schedule;
use Illuminate\Contracts\Foundation\Application;
use Illuminate\Database\Eloquent\Relations\Relation;

final class BlogServiceProvider extends DomainServiceProvider
{
    protected array $repositories = [
        PostRepository::class => [EloquentPostRepository::class, CachedPostRepository::class],
        CategoryRepository::class => [EloquentCategoryRepository::class, CachedCategoryRepository::class],
        TagRepository::class => [EloquentTagRepository::class, CachedTagRepository::class],
        AuthorRepository::class => [EloquentAuthorRepository::class, CachedAuthorRepository::class],
    ];

    protected array $observers = [
        Post::class => PostObserver::class,
        Category::class => TaxonomyObserver::class,
        Tag::class => TaxonomyObserver::class,
        Author::class => TaxonomyObserver::class,
    ];

    public function register(): void
    {
        parent::register();

        $this->app->singleton(PostContent::class, static fn (Application $app): PostContent => PostContent::fromConfig($app['config']));
    }

    public function boot(): void
    {
        parent::boot();

        // Stable morph names for seo_meta.seoable_type (class names may move).
        Relation::morphMap([
            'blog_post' => Post::class,
            'blog_category' => Category::class,
            'blog_tag' => Tag::class,
            'blog_author' => Author::class,
        ]);

        // Media in use must never be offered for bulk deletion (admin media library, L2-03).
        FindMediaUsages::column('blog_posts', 'cover_media_id', 'تصویر شاخص مقاله', 'title');
        FindMediaUsages::column('blog_posts', 'cover_mobile_media_id', 'تصویر شاخص موبایل مقاله', 'title');
        FindMediaUsages::column('blog_authors', 'avatar_media_id', 'تصویر نویسنده', 'name');

        if ($this->app->runningInConsole()) {
            $this->commands([PublishScheduledPostsCommand::class, FlushPostViewsCommand::class]);
        }

        // cPanel cron runs `schedule:run` every minute.
        $this->callAfterResolving(Schedule::class, static function (Schedule $schedule): void {
            $schedule->command('blog:publish-scheduled')->everyMinute()->withoutOverlapping(5);
            $schedule->command('blog:flush-views')->everyFiveMinutes()->withoutOverlapping(10);
        });
    }
}
