<?php

declare(strict_types=1);

namespace App\Filament\Resources\Blog\Posts\Pages;

use App\Domain\Blog\Actions\SavePost;
use App\Domain\Blog\Models\Post;
use App\Filament\Resources\Blog\Posts\PostResource;
use App\Filament\Resources\Blog\Posts\PostRevisions;
use Filament\Resources\Pages\CreateRecord;
use Illuminate\Database\Eloquent\Model;

/**
 * @property Post $record
 */
final class CreatePost extends CreateRecord
{
    protected static string $resource = PostResource::class;

    protected function handleRecordCreation(array $data): Model
    {
        return app(SavePost::class)->handle(new Post, PostFormData::content($data), PostFormData::tagIds($data));
    }

    protected function afterCreate(): void
    {
        PostRevisions::log($this->record, [], 'created');
    }

    protected function getRedirectUrl(): string
    {
        return PostResource::getUrl('edit', ['record' => $this->record]);
    }
}
