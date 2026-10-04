<?php

declare(strict_types=1);

namespace App\Filament\Resources\Blog\Posts\Pages;

use App\Filament\Resources\Blog\Posts\EditorImages;

/**
 * Splits the post form state into post attributes and tag ids for SavePost.
 */
final class PostFormData
{
    /**
     * @param  array<string, mixed>  $data
     * @return array<string, mixed>
     */
    public static function content(array $data): array
    {
        unset($data['tag_ids']);
        if (array_key_exists('body', $data)) {
            $data['body'] = EditorImages::toStorage(is_string($data['body']) ? $data['body'] : '');
        }

        return $data;
    }

    /**
     * @param  array<string, mixed>  $data
     * @return list<int>|null
     */
    public static function tagIds(array $data): ?array
    {
        if (! array_key_exists('tag_ids', $data)) {
            return null;
        }

        return array_values(array_map(intval(...), array_filter((array) $data['tag_ids'], is_numeric(...))));
    }
}
