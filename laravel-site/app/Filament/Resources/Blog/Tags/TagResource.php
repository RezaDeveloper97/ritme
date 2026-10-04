<?php

declare(strict_types=1);

namespace App\Filament\Resources\Blog\Tags;

use App\Domain\Blog\Models\Tag;
use App\Filament\Components\Seo\SeoFields;
use App\Filament\Resources\Blog\BlogPolicy;
use App\Filament\Resources\Blog\Tags\Pages\CreateTag;
use App\Filament\Resources\Blog\Tags\Pages\EditTag;
use App\Filament\Resources\Blog\Tags\Pages\ListTags;
use BackedEnum;
use Filament\Actions\DeleteBulkAction;
use Filament\Actions\EditAction;
use Filament\Facades\Filament;
use Filament\Forms\Components\TextInput;
use Filament\Resources\Resource;
use Filament\Schemas\Components\Tabs;
use Filament\Schemas\Components\Tabs\Tab;
use Filament\Schemas\Components\Utilities\Get;
use Filament\Schemas\Schema;
use Filament\Support\Icons\Heroicon;
use Filament\Tables\Columns\TextColumn;
use Filament\Tables\Table;
use Illuminate\Database\Eloquent\Builder;
use UnitEnum;

/**
 * Magazine tags with the SEO tab. Tags can also be created inline from the post editor. Access: TagPolicy.
 */
final class TagResource extends Resource
{
    protected static ?string $model = Tag::class;

    protected static ?string $slug = 'blog/tags';

    protected static string|BackedEnum|null $navigationIcon = Heroicon::OutlinedHashtag;

    protected static string|UnitEnum|null $navigationGroup = 'مجله';

    protected static ?int $navigationSort = 30;

    protected static ?string $modelLabel = 'برچسب';

    protected static ?string $pluralModelLabel = 'برچسب‌ها';

    protected static ?string $recordTitleAttribute = 'name';

    public static function form(Schema $schema): Schema
    {
        $locked = static fn (): bool => ! BlogPolicy::canEditContent(Filament::auth()->user());

        return $schema->columns(1)->components([
            Tabs::make('tag')->tabs([
                Tab::make('اطلاعات')->disabled($locked)->schema([
                    TextInput::make('name')->label('نام')->required()->maxLength(191)->live(onBlur: true),
                    TextInput::make('slug')->label('نامک (slug)')->maxLength(191)->live(onBlur: true)
                        ->unique(ignoreRecord: true)
                        ->helperText('خالی بماند، از نام ساخته می‌شود.'),
                ]),
                Tab::make('سئو')->icon(Heroicon::OutlinedMagnifyingGlass)->schema([
                    SeoFields::make()
                        ->titleFrom('name')
                        ->urlUsing(static fn (Get $get): string => url('/blog/tag/'.rawurlencode(trim((string) $get('slug')) ?: 'slug'))),
                ]),
            ]),
        ]);
    }

    public static function table(Table $table): Table
    {
        return $table
            ->modifyQueryUsing(static fn (Builder $query): Builder => $query->withCount('posts'))
            ->defaultSort('name')
            ->columns([
                TextColumn::make('name')->label('نام')->searchable()->sortable(),
                TextColumn::make('slug')->label('نامک'),
                TextColumn::make('posts_count')->label('مقاله‌ها')->numeric()->sortable(),
            ])
            ->recordActions([EditAction::make()])
            ->toolbarActions([DeleteBulkAction::make()]);
    }

    public static function getPages(): array
    {
        return [
            'index' => ListTags::route('/'),
            'create' => CreateTag::route('/create'),
            'edit' => EditTag::route('/{record}/edit'),
        ];
    }
}
