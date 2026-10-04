<?php

declare(strict_types=1);

namespace App\Filament\Resources\Faq;

use App\Domain\Faq\Actions\InvalidateFaqCache;
use App\Domain\Faq\Models\FaqGroup;
use App\Filament\Resources\Faq\Pages\CreateFaqGroup;
use App\Filament\Resources\Faq\Pages\EditFaqGroup;
use App\Filament\Resources\Faq\Pages\ListFaqGroups;
use BackedEnum;
use Filament\Actions\Action;
use Filament\Actions\DeleteBulkAction;
use Filament\Actions\EditAction;
use Filament\Forms\Components\TextInput;
use Filament\Forms\Components\Toggle;
use Filament\Resources\Resource;
use Filament\Schemas\Components\Section;
use Filament\Schemas\Schema;
use Filament\Support\Icons\Heroicon;
use Filament\Tables\Columns\IconColumn;
use Filament\Tables\Columns\TextColumn;
use Filament\Tables\Table;
use Illuminate\Database\Eloquent\Builder;
use Illuminate\Support\Facades\Route;
use UnitEnum;

/**
 * FAQ groups (L3-09): title, stable slug, «show on /faq», drag-sort; their questions are edited in the
 * ItemsRelationManager on the edit page (drag-sort, publish toggle, preview). Access: FaqPolicy.
 * Saves bump `faq` + `pages` (FaqObserver; drag-sort via InvalidateFaqCache), so the site updates on the next request.
 */
final class FaqGroupResource extends Resource
{
    protected static ?string $model = FaqGroup::class;

    protected static ?string $slug = 'faq/groups';

    protected static string|BackedEnum|null $navigationIcon = Heroicon::OutlinedQuestionMarkCircle;

    protected static string|UnitEnum|null $navigationGroup = 'محتوای سایت';

    protected static ?int $navigationSort = 30;

    protected static ?string $modelLabel = 'گروه سؤال';

    protected static ?string $pluralModelLabel = 'سؤالات متداول';

    protected static ?string $recordTitleAttribute = 'title';

    public static function form(Schema $schema): Schema
    {
        return $schema->columns(1)->components([
            Section::make()->columns(2)->schema([
                TextInput::make('title')->label('عنوان')->required()->maxLength(191)
                    ->helperText('عنوان دسته در صفحه سؤالات متداول؛ در صفحه اصلی و پلاس همان تیتر بخش است.'),
                TextInput::make('slug')->label('شناسه (slug)')->maxLength(64)->unique(ignoreRecord: true)
                    ->helperText('کلید ثابتی که صفحه‌ها با آن گروه را می‌خوانند (home، plus، stage-cycle …) و لنگر #… در صفحه سؤالات. برای گروه‌های موجود تغییرش نده.'),
                Toggle::make('is_listed')->label('نمایش در صفحه سؤالات متداول (/faq)')->default(false),
            ]),
        ]);
    }

    public static function table(Table $table): Table
    {
        return $table
            ->modifyQueryUsing(static fn (Builder $query): Builder => $query
                ->withCount(['items', 'items as published_items_count' => static fn (Builder $q): Builder => $q->where('is_published', true)]))
            ->defaultSort('sort_order')
            ->reorderable('sort_order')
            ->afterReordering(static fn () => app(InvalidateFaqCache::class)())
            ->columns([
                TextColumn::make('title')->label('عنوان')->searchable(),
                TextColumn::make('slug')->label('شناسه'),
                IconColumn::make('is_listed')->label('در /faq')->boolean(),
                TextColumn::make('published_items_count')->label('منتشرشده')->numeric(),
                TextColumn::make('items_count')->label('همه سؤال‌ها')->numeric(),
            ])
            ->recordActions([EditAction::make(), self::viewOnSiteAction()])
            ->toolbarActions([DeleteBulkAction::make()]);
    }

    public static function getRelations(): array
    {
        return [ItemsRelationManager::class];
    }

    public static function getPages(): array
    {
        return [
            'index' => ListFaqGroups::route('/'),
            'create' => CreateFaqGroup::route('/create'),
            'edit' => EditFaqGroup::route('/{record}/edit'),
        ];
    }

    public static function viewOnSiteAction(): Action
    {
        return Action::make('viewOnSite')
            ->label('مشاهده در سایت')
            ->icon(Heroicon::OutlinedArrowTopRightOnSquare)
            ->url(static fn (FaqGroup $record): ?string => self::publicUrl($record), shouldOpenInNewTab: true)
            ->visible(static fn (FaqGroup $record): bool => self::publicUrl($record) !== null);
    }

    /**
     * Where the group is shown: /faq#slug for listed groups, else the page that renders it (FaqServiceProvider
     * composers, stage pages). null when no page uses the slug.
     */
    public static function publicUrl(FaqGroup $group): ?string
    {
        if ($group->is_listed) {
            return route('faq').'#'.$group->slug;
        }

        $route = match (true) {
            $group->slug === 'home' => 'home',
            $group->slug === 'plus' => 'plus',
            $group->slug === 'contact' => 'contact',
            $group->slug === 'directory-business' => 'directory.business',
            str_starts_with($group->slug, 'stage-') => 'stage.'.substr($group->slug, 6),
            default => null,
        };

        return $route !== null && Route::has($route) ? route($route) : null;
    }
}
