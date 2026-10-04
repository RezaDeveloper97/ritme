<?php

declare(strict_types=1);

namespace App\Filament\Resources\Blog\Posts;

use App\Domain\Media\Contracts\MediaRepository;
use App\Filament\Forms\Components\MediaPicker;
use Filament\Actions\Action;
use Filament\Forms\Components\RichEditor;
use Filament\Forms\Components\RichEditor\EditorCommand;
use Filament\Forms\Components\RichEditor\RichEditorTool;
use Filament\Support\Enums\Width;
use Filament\Support\Icons\Heroicon;

/**
 * Rich editor tool «تصویر از کتابخانه»: pick (or upload) an image through the media library picker and insert it
 * at the cursor as <img data-id="{media id}" src="/media/…" alt="…">; EditorImages stores it as data-media-id.
 */
final class InsertMediaAction
{
    public const NAME = 'insertMedia';

    public static function tool(): RichEditorTool
    {
        return RichEditorTool::make(self::NAME)
            ->label('تصویر از کتابخانه رسانه')
            ->icon(Heroicon::Photo)
            ->action();
    }

    public static function make(): Action
    {
        return Action::make(self::NAME)
            ->label('تصویر از کتابخانه رسانه')
            ->modalHeading('درج تصویر از کتابخانه رسانه')
            ->modalWidth(Width::Large)
            ->schema([
                MediaPicker::make('media_id')->label('تصویر')->required(),
            ])
            ->action(static function (array $arguments, array $data, RichEditor $component): void {
                $media = is_numeric($data['media_id'] ?? null) ? app(MediaRepository::class)->find((int) $data['media_id']) : null;
                if ($media === null) {
                    return;
                }

                $component->runCommands(
                    [
                        EditorCommand::make('insertContent', arguments: [[
                            'type' => 'image',
                            'attrs' => [
                                'id' => (string) $media->id,
                                'src' => $media->url,
                                'alt' => (string) ($media->alt ?? ''),
                            ],
                        ]]),
                    ],
                    editorSelection: is_array($arguments['editorSelection'] ?? null) ? $arguments['editorSelection'] : null,
                );
            });
    }
}
