<?php

declare(strict_types=1);

namespace App\Filament\Resources\Faq;

use App\Domain\Faq\Actions\InvalidateFaqCache;
use App\Domain\Faq\Models\FaqItem;
use Filament\Actions\Action;
use Filament\Actions\BulkAction;
use Filament\Actions\CreateAction;
use Filament\Actions\DeleteAction;
use Filament\Actions\DeleteBulkAction;
use Filament\Actions\EditAction;
use Filament\Forms\Components\RichEditor;
use Filament\Forms\Components\TextInput;
use Filament\Forms\Components\Toggle;
use Filament\Resources\RelationManagers\RelationManager;
use Filament\Schemas\Schema;
use Filament\Support\Icons\Heroicon;
use Filament\Tables\Columns\TextColumn;
use Filament\Tables\Columns\ToggleColumn;
use Filament\Tables\Table;
use Illuminate\Database\Eloquent\Collection;
use Illuminate\Database\Eloquent\Model;
use Illuminate\Support\HtmlString;

/**
 * Questions of a FAQ group: drag-sort, publish toggle, preview (the answer as the site renders it). The answer is
 * sanitised on save (FaqObserver: links, lists, bold/italic only). Only published items reach the site and its
 * FAQPage JSON-LD.
 */
final class ItemsRelationManager extends RelationManager
{
    protected static string $relationship = 'items';

    protected static ?string $title = 'سؤال‌ها';

    protected static ?string $modelLabel = 'سؤال';

    protected static ?string $pluralModelLabel = 'سؤال‌ها';

    protected static ?string $recordTitleAttribute = 'question';

    public function form(Schema $schema): Schema
    {
        return $schema->columns(1)->components([
            TextInput::make('question')->label('سؤال')->required()->maxLength(255),
            RichEditor::make('answer')
                ->label('جواب')
                ->required()
                ->toolbarButtons([['bold', 'italic', 'link'], ['bulletList', 'orderedList'], ['undo', 'redo']])
                ->fileAttachments(false)
                ->helperText('بدون ادعای تشخیص و بدون «حتماً، قطعاً، دقیق‌ترین، تضمینی».'),
            Toggle::make('is_published')->label('منتشر شود')->default(true),
        ]);
    }

    public function table(Table $table): Table
    {
        return $table
            ->defaultSort('sort_order')
            ->reorderable('sort_order')
            ->afterReordering(static fn () => app(InvalidateFaqCache::class)())
            ->columns([
                TextColumn::make('question')->label('سؤال')->wrap()->searchable(),
                ToggleColumn::make('is_published')->label('منتشرشده'),
                TextColumn::make('updated_at')->label('آخرین ویرایش')
                    ->formatStateUsing(static fn ($state): ?string => $state ? jdate($state, 'Y/m/d H:i') : null),
            ])
            ->headerActions([CreateAction::make()->label('سؤال جدید')])
            ->recordActions([self::previewAction(), EditAction::make(), DeleteAction::make()])
            ->toolbarActions([
                BulkAction::make('publish')->label('انتشار')->icon(Heroicon::OutlinedEye)
                    ->action(static fn (Collection $records) => self::publish($records, true)),
                BulkAction::make('unpublish')->label('عدم انتشار')->icon(Heroicon::OutlinedEyeSlash)
                    ->action(static fn (Collection $records) => self::publish($records, false)),
                DeleteBulkAction::make(),
            ]);
    }

    /**
     * One save per item, so FaqObserver bumps the caches.
     *
     * @param  Collection<int, Model>  $records
     */
    private static function publish(Collection $records, bool $published): void
    {
        foreach ($records as $record) {
            if ($record instanceof FaqItem) {
                $record->update(['is_published' => $published]);
            }
        }
    }

    public static function previewAction(): Action
    {
        return Action::make('preview')
            ->label('پیش‌نمایش')
            ->icon(Heroicon::OutlinedEye)
            ->modalHeading(static fn (FaqItem $record): string => $record->question)
            ->modalContent(static fn (FaqItem $record): HtmlString => new HtmlString(
                '<div class="fi-prose">'.$record->answer.'</div>'
                .($record->is_published ? '' : '<p><strong>منتشر نشده — در سایت نمایش داده نمی‌شود.</strong></p>'),
            ))
            ->modalSubmitAction(false)
            ->modalCancelActionLabel('بستن');
    }
}
