<?php

declare(strict_types=1);

namespace App\Filament\Resources\Blog\Posts\Pages;

use App\Domain\Blog\Actions\SavePost;
use App\Domain\Blog\Models\Post;
use App\Filament\Resources\Blog\Posts\EditorImages;
use App\Filament\Resources\Blog\Posts\PostResource;
use App\Filament\Resources\Blog\Posts\PostRevisions;
use Filament\Actions\Action;
use Filament\Actions\DeleteAction;
use Filament\Resources\Pages\EditRecord;
use Filament\Support\Icons\Heroicon;
use Illuminate\Database\Eloquent\Model;
use Illuminate\Support\HtmlString;

/**
 * Editing a post. SEO managers can open this page but only their SEO tab is saved: content fields are disabled in
 * the form and ignored here (BlogPolicy::canEditContent()), so a tampered Livewire payload changes nothing.
 *
 * @property Post $record
 */
final class EditPost extends EditRecord
{
    protected static string $resource = PostResource::class;

    /** @var array<string, string|null> */
    private array $revisionBefore = [];

    protected function mutateFormDataBeforeFill(array $data): array
    {
        $data['body'] = EditorImages::toEditor(is_string($data['body'] ?? null) ? $data['body'] : '');
        $data['tag_ids'] = $this->record->tags()->pluck('blog_tags.id')->map(intval(...))->all();

        return $data;
    }

    protected function beforeSave(): void
    {
        $this->revisionBefore = PostRevisions::snapshot($this->record);
    }

    protected function handleRecordUpdate(Model $record, array $data): Model
    {
        /** @var Post $record */
        if (! PostResource::canEditContent()) {
            return $record;
        }

        return app(SavePost::class)->handle($record, PostFormData::content($data), PostFormData::tagIds($data));
    }

    protected function afterSave(): void
    {
        PostRevisions::log($this->record, $this->revisionBefore);
    }

    protected function getHeaderActions(): array
    {
        return [
            PostResource::previewAction(),
            PostResource::duplicateAction(),
            Action::make('revisions')
                ->label('تاریخچه تغییرات')
                ->icon(Heroicon::OutlinedClock)
                ->color('gray')
                ->modalHeading('تاریخچه تغییرات')
                ->modalSubmitAction(false)
                ->modalCancelActionLabel('بستن')
                ->modalContent(fn (): HtmlString => PostRevisions::render($this->record)),
            DeleteAction::make(),
        ];
    }
}
