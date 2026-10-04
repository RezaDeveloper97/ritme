<?php

declare(strict_types=1);

namespace App\Filament\Resources\Blog\Posts;

use App\Domain\Blog\Models\Post;
use App\Domain\Seo\SeoManager;
use Illuminate\Http\Response;
use Illuminate\Support\Facades\URL;
use Illuminate\Support\Facades\View;

/**
 * Draft preview behind a temporary signed URL (shareable with a reviewer who has no admin account). Registered in
 * AdminPanelProvider under the panel path (never page-cached, see AdminPaths), always noindex (meta + X-Robots-Tag)
 * and private/no-store. Renders on the site layout with the stored (sanitised) body.
 */
final class PostPreviewController
{
    public const ROUTE = 'blog.posts.preview';

    public const VIEW_NAMESPACE = 'ritme-admin-blog';

    public const TTL_HOURS = 24;

    public static function url(Post $post): string
    {
        return URL::temporarySignedRoute('filament.admin.'.self::ROUTE, now()->addHours(self::TTL_HOURS), ['post' => $post->getKey()]);
    }

    public function __invoke(Post $post, SeoManager $seo): Response
    {
        View::replaceNamespace(self::VIEW_NAMESPACE, dirname(__DIR__).'/views');

        $post->load(['author', 'reviewer', 'category']);
        $seo->for($post)->title('پیش‌نمایش: '.$post->title)->noindex();

        $response = response()->view(self::VIEW_NAMESPACE.'::post-preview', [
            'title' => $post->title,
            'excerpt' => $post->excerpt,
            'body' => $post->body,
            'status' => $post->status->label(),
            'coverId' => $post->cover_media_id,
            'coverMobileId' => $post->cover_mobile_media_id,
            'author' => $post->author?->name,
            'reviewer' => $post->reviewer?->name,
            'category' => $post->category?->name,
            'readingTime' => $post->reading_time,
        ]);

        $response->headers->set('X-Robots-Tag', 'noindex, nofollow');
        $response->headers->set('Cache-Control', 'private, no-store');

        return $response;
    }
}
