<?php

declare(strict_types=1);

namespace App\Filament\Resources\Blog\Posts;

use App\Domain\Media\Contracts\MediaRepository;
use App\Support\Html\HtmlFragment;

/**
 * Bridges the Filament rich editor and the stored article HTML. The editor tracks an image by `data-id` (here: the
 * media id); the site stores `data-media-id` so the article renderer can swap in the responsive <x-picture> variants.
 * Images keep their `src` (a /media/ URL) so the stored HTML still works without the renderer.
 */
final class EditorImages
{
    /**
     * Stored HTML → editor HTML.
     */
    public static function toEditor(?string $html): string
    {
        return self::rename((string) $html, 'data-media-id', 'data-id');
    }

    /**
     * Editor HTML → stored HTML (sanitised afterwards by PostObserver).
     */
    public static function toStorage(?string $html): string
    {
        return self::rename((string) $html, 'data-id', 'data-media-id');
    }

    /**
     * URL of a media item for the editor canvas (null when it no longer exists).
     */
    public static function url(mixed $mediaId): ?string
    {
        if (! is_numeric($mediaId)) {
            return null;
        }

        return app(MediaRepository::class)->find((int) $mediaId)?->url;
    }

    private static function rename(string $html, string $from, string $to): string
    {
        if (trim($html) === '' || ! str_contains($html, $from)) {
            return $html;
        }

        $fragment = HtmlFragment::load($html);
        foreach ($fragment->elements('img') as $img) {
            $id = trim($img->getAttribute($from));
            $img->removeAttribute($from);
            if (preg_match('/^[1-9]\d{0,18}$/', $id) === 1) {
                $img->setAttribute($to, $id);
            }
        }

        return $fragment->html();
    }
}
