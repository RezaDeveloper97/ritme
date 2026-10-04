<?php

declare(strict_types=1);

namespace App\Filament\Resources\Blog\Authors;

use App\Domain\Blog\Models\Author;
use App\Filament\Components\Seo\SeoFields;
use App\Filament\Forms\Components\MediaPicker;
use App\Filament\Resources\Blog\Authors\Pages\CreateAuthor;
use App\Filament\Resources\Blog\Authors\Pages\EditAuthor;
use App\Filament\Resources\Blog\Authors\Pages\ListAuthors;
use App\Filament\Resources\Blog\BlogPolicy;
use BackedEnum;
use Filament\Actions\DeleteBulkAction;
use Filament\Actions\EditAction;
use Filament\Facades\Filament;
use Filament\Forms\Components\TagsInput;
use Filament\Forms\Components\Textarea;
use Filament\Forms\Components\TextInput;
use Filament\Forms\Components\Toggle;
use Filament\Resources\Resource;
use Filament\Schemas\Components\Grid;
use Filament\Schemas\Components\Tabs;
use Filament\Schemas\Components\Tabs\Tab;
use Filament\Schemas\Components\Utilities\Get;
use Filament\Schemas\Schema;
use Filament\Support\Icons\Heroicon;
use Filament\Tables\Columns\IconColumn;
use Filament\Tables\Columns\TextColumn;
use Filament\Tables\Filters\TernaryFilter;
use Filament\Tables\Table;
use Illuminate\Database\Eloquent\Builder;
use UnitEnum;

/**
 * Authors and medical reviewers (E-E-A-T): credentials and sameAs profile URLs feed the Person JSON-LD node.
 * Access: AuthorPolicy.
 */
final class AuthorResource extends Resource
{
    protected static ?string $model = Author::class;

    protected static ?string $slug = 'blog/authors';

    protected static string|BackedEnum|null $navigationIcon = Heroicon::OutlinedUserCircle;

    protected static string|UnitEnum|null $navigationGroup = 'مجله';

    protected static ?int $navigationSort = 40;

    protected static ?string $modelLabel = 'نویسنده';

    protected static ?string $pluralModelLabel = 'نویسندگان و بازبین‌ها';

    protected static ?string $recordTitleAttribute = 'name';

    public static function form(Schema $schema): Schema
    {
        $locked = static fn (): bool => ! BlogPolicy::canEditContent(Filament::auth()->user());

        return $schema->columns(1)->components([
            Tabs::make('author')->tabs([
                Tab::make('اطلاعات')->disabled($locked)->schema([
                    Grid::make(2)->schema([
                        TextInput::make('name')->label('نام')->required()->maxLength(191)->live(onBlur: true),
                        TextInput::make('slug')->label('نامک (slug)')->maxLength(191)->live(onBlur: true)
                            ->unique(ignoreRecord: true)
                            ->helperText('خالی بماند، از نام ساخته می‌شود.'),
                        TextInput::make('job_title')->label('عنوان شغلی')->maxLength(191),
                        TextInput::make('credentials')->label('مدرک و تخصص')->maxLength(255)
                            ->helperText('مثلاً «متخصص زنان و زایمان»؛ در کادر نویسنده و داده ساختاریافته نمایش داده می‌شود.'),
                    ]),
                    Toggle::make('is_medical_reviewer')->label('بازبین پزشکی')
                        ->helperText('فقط این افراد در فیلد «بازبین پزشکی» مقاله‌ها قابل انتخاب‌اند.'),
                    Textarea::make('bio')->label('معرفی کوتاه')->rows(4)->live(onBlur: true),
                    MediaPicker::make('avatar_media_id')->label('تصویر'),
                    TagsInput::make('same_as')
                        ->label('پروفایل‌های معتبر (sameAs)')
                        ->placeholder('https://…')
                        ->nestedRecursiveRules(['url:https,http', 'max:500'])
                        ->helperText('نشانی کامل صفحه‌های رسمی (نظام پزشکی، لینکدین …)؛ هر نشانی را وارد و Enter بزنید.'),
                ]),
                Tab::make('سئو')->icon(Heroicon::OutlinedMagnifyingGlass)->schema([
                    SeoFields::make()
                        ->titleFrom('name')
                        ->descriptionFrom('bio')
                        ->imageFrom('avatar_media_id')
                        ->urlUsing(static fn (Get $get): string => url('/blog/author/'.rawurlencode(trim((string) $get('slug')) ?: 'slug'))),
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
                TextColumn::make('credentials')->label('تخصص')->placeholder('—')->limit(40),
                IconColumn::make('is_medical_reviewer')->label('بازبین پزشکی')->boolean(),
                TextColumn::make('posts_count')->label('مقاله‌ها')->numeric(),
            ])
            ->filters([TernaryFilter::make('is_medical_reviewer')->label('بازبین پزشکی')])
            ->recordActions([EditAction::make()])
            ->toolbarActions([DeleteBulkAction::make()]);
    }

    public static function getPages(): array
    {
        return [
            'index' => ListAuthors::route('/'),
            'create' => CreateAuthor::route('/create'),
            'edit' => EditAuthor::route('/{record}/edit'),
        ];
    }
}
